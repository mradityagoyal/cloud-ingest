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

package list

import (
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"

	"cloud.google.com/go/storage"
	"github.com/GoogleCloudPlatform/cloud-ingest/agent/gcloud"
	"github.com/GoogleCloudPlatform/cloud-ingest/agent/tasks/common"

	listfilepb "github.com/GoogleCloudPlatform/cloud-ingest/proto/listfile_go_proto"
	taskpb "github.com/GoogleCloudPlatform/cloud-ingest/proto/task_go_proto"
)

// DepthFirstListHandler is responsible for handling depth-first list tasks.
type DepthFirstListHandler struct {
	gcs                   gcloud.GCS
	resumableChunkSize    int
	listFileSizeThreshold int
	allowedDirBytes       int
}

// NewDepthFirstListHandler returns a new DepthFirstListHandler.
func NewDepthFirstListHandler(storageClient *storage.Client) *DepthFirstListHandler {
	// Convert maxMemoryForListingDirectories to bytes and divide it equally between
	// the list task processing threads.
	allowedDirBytes := *maxMemoryForListingDirectories * 1024 * 1024 / *NumberConcurrentListTasks
	return &DepthFirstListHandler{
		gcs:                   gcloud.NewGCSClient(storageClient),
		resumableChunkSize:    *listTaskChunkSize,
		listFileSizeThreshold: *listFileSizeThreshold,
		allowedDirBytes:       allowedDirBytes,
	}
}

// processDir lists the contents of a single directory. It adds any directories it finds to the
// given dirStore.
// It returns the discovered files (and directories if writeDirs is true) sorted in case sensitive
// alphabetical order by path. The given listMD is updated with the number of files/dirs found.
func processDir(dir string, dirStore *DirectoryInfoStore, listMD *listingFileMetadata, writeDirs bool) ([]*listfilepb.ListFileEntry, error) {
	f, err := os.Open(dir)
	if err != nil {
		return nil, err
	}
	osFileInfos, err := f.Readdir(-1)
	f.Close()
	if err != nil {
		return nil, err
	}

	var entries []*listfilepb.ListFileEntry
	for _, osFileInfo := range osFileInfos {
		if strings.Contains(osFileInfo.Name(), "\n") {
			return nil, errors.New("the listing contains file with newlines")
		}
		path := filepath.Join(dir, osFileInfo.Name())
		if osFileInfo.IsDir() {
			dirInfo := listfilepb.DirectoryInfo{Path: path}
			err := dirStore.Add(dirInfo)
			if err != nil {
				return nil, err
			}
			listMD.dirsDiscovered++
			if writeDirs {
				entries = append(entries, &listfilepb.ListFileEntry{Entry: &listfilepb.ListFileEntry_DirectoryInfo{DirectoryInfo: &dirInfo}})
			}
		} else {
			pbFileInfo := listfilepb.FileInfo{
				Path:             path,
				LastModifiedTime: osFileInfo.ModTime().Unix(),
				Size:             osFileInfo.Size(),
			}
			entries = append(entries, &listfilepb.ListFileEntry{Entry: &listfilepb.ListFileEntry_FileInfo{FileInfo: &pbFileInfo}})
			listMD.files++
			listMD.bytes += pbFileInfo.Size
		}
	}

	err = sortListFileEntries(entries)
	return entries, err
}

// writeDirectories writes all of the directories stored in dirStore to the given writer
// in case sensitive alphabetical order by path.
func writeDirectories(w io.Writer, dirStore *DirectoryInfoStore) error {
	for _, dirInfo := range dirStore.DirectoryInfos() {
		entry := listfilepb.ListFileEntry{Entry: &listfilepb.ListFileEntry_DirectoryInfo{DirectoryInfo: &dirInfo}}
		if err := writeProtobuf(w, &entry); err != nil {
			return err
		}
	}
	return nil
}

// processDirectories lists directories until it has hit the list file size threshold or it has
// used too much memory. For each directory it processes, it writes any files to the list file and
// adds any directories to the list of directories to be listed. If includeDirs is true, both files
// and directories are written to the list file.
// processDirectories returns listing file metadata gathered while processing directories.
func processDirectories(w io.Writer, dirStore *DirectoryInfoStore, listFileSizeThreshold, maxDirBytes int, includeDirs bool) (*listingFileMetadata, error) {
	totalEntries := 0
	listMD := &listingFileMetadata{}

	// Ensure that at least one directory is listed. Without the firstTime flag, the initial list
	// of directories could exceed the memory limit, resulting in no directories being listed.
	firstTime := true
	for firstTime || (dirStore.Size() < maxDirBytes && totalEntries+dirStore.Len() < listFileSizeThreshold) {
		dirToProcess := dirStore.RemoveFirst()
		if dirToProcess == nil {
			break
		}
		entries, err := processDir(dirToProcess.Path, dirStore, listMD, includeDirs)
		if err != nil {
			return nil, err
		}
		for _, entry := range entries {
			if err := writeProtobuf(w, entry); err != nil {
				return nil, err
			}
		}
		totalEntries += len(entries)
		firstTime = false
		listMD.dirsListed++
	}
	listMD.dirsNotListed = int64(dirStore.Len())
	return listMD, nil
}

// listDirectoriesAndWriteResults lists starting at the specified directories in case sensitive
// alphabetical, depth first order. It continues listing until it finds listFileSizeThreshold or
// uses more than maxDirBytes to store unexplored directories. It writes the list results using the
// given writer. If writeDirs is true, both directories and files are sorted and written to the list
// file. Otherwise, just files are written.
// Unlisted directories (any directories that were found or included in the list spec but weren't
// listed) are stored in the returned directory info store.
func listDirectoriesAndWriteResults(w io.Writer, listSpec *taskpb.ListSpec, listFileSizeThreshold, maxDirBytes int, writeDirs bool) (*listingFileMetadata, *DirectoryInfoStore, error) {
	// Add directories from list spec into the DirStore.
	// Directories will be explored in alphabetical, depth first order.
	dirStore := NewDirectoryInfoStore()
	for _, dirPath := range listSpec.SrcDirectories {
		if err := dirStore.Add(listfilepb.DirectoryInfo{Path: dirPath}); err != nil {
			return nil, nil, err
		}
	}

	listMD, err := processDirectories(w, dirStore, listFileSizeThreshold, maxDirBytes, writeDirs)
	if err != nil {
		return nil, nil, err
	}

	return listMD, dirStore, nil
}

func (h *DepthFirstListHandler) Do(ctx context.Context, taskReqMsg *taskpb.TaskReqMsg) *taskpb.TaskRespMsg {
	listSpec := taskReqMsg.Spec.GetListSpec()
	if listSpec == nil {
		err := errors.New("ListHandler.Do taskReqMsg.Spec is not ListSpec")
		return common.BuildTaskRespMsg(taskReqMsg, nil, nil, err)
	}

	log := &taskpb.Log{
		Log: &taskpb.Log_ListLog{ListLog: &taskpb.ListLog{}},
	}

	w := gcsWriterWithCondition(ctx, h.gcs, listSpec.DstListResultBucket, listSpec.DstListResultObject, listSpec.ExpectedGenerationNum, h.resumableChunkSize)

	listMD, unlistedDirs, err := listDirectoriesAndWriteResults(w, listSpec, h.listFileSizeThreshold, h.allowedDirBytes, false /* writeDirs */)
	if err != nil {
		w.CloseWithError(err)
		return common.BuildTaskRespMsg(taskReqMsg, nil, log, err)
	}

	if err = writeDirectories(w, unlistedDirs); err != nil {
		w.CloseWithError(err)
		return common.BuildTaskRespMsg(taskReqMsg, nil, log, err)
	}

	if err := w.Close(); err != nil {
		return common.BuildTaskRespMsg(taskReqMsg, nil, log, err)
	}

	setListLog(log, listMD)

	return common.BuildTaskRespMsg(taskReqMsg, nil, log, nil)
}
