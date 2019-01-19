/*
Copyright 2018 Google Inc. All Rights Reserved.
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package agent

import (
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"cloud.google.com/go/storage"
	"github.com/GoogleCloudPlatform/cloud-ingest/gcloud"
	"github.com/GoogleCloudPlatform/cloud-ingest/helpers"
	listpb "github.com/GoogleCloudPlatform/cloud-ingest/proto/listfile_go_proto"
	taskpb "github.com/GoogleCloudPlatform/cloud-ingest/proto/task_go_proto"
)

// DepthFirstListHandler is responsible for handling depth-first list tasks.
type DepthFirstListHandler struct {
	gcs                   gcloud.GCS
	resumableChunkSize    int
	listFileSizeThreshold int
	allowedDirBytes       int
}

func NewDepthFirstListHandler(storageClient *storage.Client, resumableChunkSize int, listFileSizeThreshold, allowedDirBytes int) *DepthFirstListHandler {
	return &DepthFirstListHandler{gcloud.NewGCSClient(storageClient), resumableChunkSize, listFileSizeThreshold, allowedDirBytes}
}

// processDirectory lists a single directory. It adds any directories it finds to the given dirStore
// and returns the discovered files sorted in case sensitive alphabetical order by path.
// The given listMD is updated with the number of files/dirs found.
func processDirectory(dir string, dirStore *DirectoryInfoStore, listMD *listingFileMetadata) ([]listpb.FileInfo, error) {
	f, err := os.Open(dir)
	if err != nil {
		return nil, err
	}
	osFileInfos, err := f.Readdir(-1)
	f.Close()
	if err != nil {
		return nil, err
	}

	pbFileInfos := make([]listpb.FileInfo, 0)
	for _, osFileInfo := range osFileInfos {
		if strings.Contains(osFileInfo.Name(), "\n") {
			return nil, errors.New("the listing contains file with newlines")
		}
		path := filepath.Join(dir, osFileInfo.Name())
		if osFileInfo.IsDir() {
			dirInfo := listpb.DirectoryInfo{Path: path}
			err := dirStore.Add(dirInfo)
			if err != nil {
				return nil, err
			}
			listMD.dirsDiscovered++
		} else {
			pbFileInfo := listpb.FileInfo{
				Path:             path,
				LastModifiedTime: osFileInfo.ModTime().Unix(),
				Size:             osFileInfo.Size(),
			}
			pbFileInfos = append(pbFileInfos, pbFileInfo)
			listMD.files++
			listMD.bytes += pbFileInfo.Size
		}
	}

	// Readdir returns the entries in "directory order", so they must be sorted
	// to meet our expectations of lexicographical order.
	sort.Slice(pbFileInfos, func(i, j int) bool {
		return pbFileInfos[i].Path < pbFileInfos[j].Path
	})

	return pbFileInfos, nil
}

// writeDirectories writes all of the directories stored in dirStore to the given writer
// in case sensitive alphabetical order by path.
func writeDirectories(w io.Writer, dirStore *DirectoryInfoStore) error {
	for _, dirInfo := range dirStore.DirectoryInfos() {
		entry := listpb.ListFileEntry{Entry: &listpb.ListFileEntry_DirectoryInfo{DirectoryInfo: &dirInfo}}
		if err := writeProtobuf(w, &entry); err != nil {
			return err
		}
	}
	return nil
}

// processDirectories lists directories until it has found enough files or it has used too much memory.
// For each directory it processes, it adds the discovered directories to the given dirStore and
// stores metadata about each file in memory. When it is finished with a directory, the file metadata
// is sorted alphabetically by path and written to the given writer.
func processDirectories(w io.Writer, dirStore *DirectoryInfoStore, listFileSizeThreshold, maxDirBytes int) (*listingFileMetadata, error) {
	totalFiles := 0
	listMD := &listingFileMetadata{}

	// Ensure that at least one directory is listed. Without the firstTime flag, the initial list
	// of directories could exceed the memory limit, resulting in no directories being listed.
	firstTime := true
	for firstTime || (dirStore.Size() < maxDirBytes && totalFiles+dirStore.Len() < listFileSizeThreshold) {
		dirToProcess := dirStore.RemoveFirst()
		if dirToProcess == nil {
			break
		}
		pbFileInfos, err := processDirectory(dirToProcess.Path, dirStore, listMD)
		if err != nil {
			return nil, err
		}
		for _, pbFileInfo := range pbFileInfos {
			entry := listpb.ListFileEntry{Entry: &listpb.ListFileEntry_FileInfo{FileInfo: &pbFileInfo}}
			if err := writeProtobuf(w, &entry); err != nil {
				return nil, err
			}
		}
		totalFiles += len(pbFileInfos)
		firstTime = false
		listMD.dirsListed++
	}
	return listMD, nil
}

// listDirectoriesAndWriteListFile lists starting at the specified directories in case sensitive
// alphabetical, depth first order. It continues listing until it finds listFileSizeThreshold or
// uses more than maxDirBytes to store unexplored directories.
func listDirectoriesAndWriteListFile(w io.Writer, listSpec *taskpb.ListSpec, listFileSizeThreshold, maxDirBytes int) (*listingFileMetadata, error) {
	// Add directories from list spec into the DirStore.
	// Directories will be explored in alphabetical, depth first order.
	dirStore := NewDirectoryInfoStore()
	for _, dirPath := range listSpec.SrcDirectories {
		if err := dirStore.Add(listpb.DirectoryInfo{Path: dirPath}); err != nil {
			return nil, err
		}
	}

	listMD, err := processDirectories(w, dirStore, listFileSizeThreshold, maxDirBytes)
	if err != nil {
		return nil, err
	}

	if err = writeDirectories(w, dirStore); err != nil {
		return nil, err
	}

	return listMD, nil
}

func (h *DepthFirstListHandler) Do(ctx context.Context, taskReqMsg *taskpb.TaskReqMsg) *taskpb.TaskRespMsg {
	listSpec := taskReqMsg.Spec.GetListSpec()
	if listSpec == nil {
		err := errors.New("ListHandler.Do taskReqMsg.Spec is not ListSpec")
		return buildTaskRespMsg(taskReqMsg, nil, nil, err)
	}

	log := &taskpb.Log{
		Log: &taskpb.Log_ListLog{ListLog: &taskpb.ListLog{}},
	}

	w := h.gcs.NewWriterWithCondition(ctx, listSpec.DstListResultBucket, listSpec.DstListResultObject,
		helpers.GetGCSGenerationNumCondition(listSpec.ExpectedGenerationNum))

	// Set the resumable upload chunk size.
	if t, ok := w.(*storage.Writer); ok {
		t.ChunkSize = h.resumableChunkSize
	}

	listMD, err := listDirectoriesAndWriteListFile(w, listSpec, h.listFileSizeThreshold, h.allowedDirBytes)
	if err != nil {
		w.CloseWithError(err)
		return buildTaskRespMsg(taskReqMsg, nil, log, err)
	}

	if err := w.Close(); err != nil {
		return buildTaskRespMsg(taskReqMsg, nil, log, err)
	}

	ll := log.GetListLog()
	ll.FilesFound = listMD.files
	ll.BytesFound = listMD.bytes
	ll.DirsFound = listMD.dirsDiscovered
	ll.DirsListed = listMD.dirsListed

	return buildTaskRespMsg(taskReqMsg, nil, log, nil)
}
