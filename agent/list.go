/*
Copyright 2017 Google Inc. All Rights Reserved.
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
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"cloud.google.com/go/storage"
	"github.com/GoogleCloudPlatform/cloud-ingest/gcloud"
	"github.com/GoogleCloudPlatform/cloud-ingest/helpers"
	"github.com/golang/glog"
	"google.golang.org/api/googleapi"

	taskpb "github.com/GoogleCloudPlatform/cloud-ingest/proto/task_go_proto"
)

// ListHandler is responsible for handling list tasks.
type ListHandler struct {
	gcs                gcloud.GCS
	resumableChunkSize int
}

func NewListHandler(storageClient *storage.Client, resumableChunkSize int) *ListHandler {
	return &ListHandler{gcloud.NewGCSClient(storageClient), resumableChunkSize}
}

func listDirectory(dir string) ([]os.FileInfo, error) {
	f, err := os.Open(dir)
	if err != nil {
		glog.Errorf("error opening dir %v: %v\n", dir, err)
		return nil, err
	}
	fileInfos, err := f.Readdir(-1)
	f.Close()
	if err != nil {
		glog.Errorf("error reading dir %v: %v\n", dir, err)
		return nil, err
	}
	// Readdir returns the entries in "directory order", so they must be sorted
	// to meet our expectations of lexicographical order.
	sort.Slice(fileInfos, func(i, j int) bool {
		return fileInfos[i].Name() < fileInfos[j].Name()
	})
	for _, fileInfo := range fileInfos {
		if strings.Contains(fileInfo.Name(), "\n") {
			return nil, errors.New("The listing contains file with newlines.")
		}
	}
	return fileInfos, nil
}

func newListFileEntry(fileInfo os.FileInfo, srcDir string) *ListFileEntry {
	fullPath := filepath.Join(srcDir, fileInfo.Name())
	return &ListFileEntry{fileInfo.IsDir(), fullPath}
}

// getListingUploadChunkSize decides how big the write buffer needs to be for uploading
// the listing file to GCS, based on the loaded listing.
func getListingUploadChunkSize(fileInfos []os.FileInfo, srcDir string, maxSize int) (int, error) {
	if maxSize < googleapi.MinUploadChunkSize {
		return 0, fmt.Errorf("invalid max chunk size %d", maxSize)
	}
	result := 0
	lineOverhead := len(srcDir) + 4 // file type, comma, path sep, newline.
	for _, fileInfo := range fileInfos {
		result += len(fileInfo.Name()) + lineOverhead
		if result >= maxSize {
			return maxSize, nil
		}
	}

	// Always allocate at least the minimum.
	if result < googleapi.MinUploadChunkSize {
		result = googleapi.MinUploadChunkSize
	}

	return result, nil
}

func writeListingFile(fileInfos []os.FileInfo, srcDir string, w io.Writer) (*listingFileMetadata, error) {
	var bytesFound, filesFound, dirsFound int64
	for _, fileInfo := range fileInfos {
		listFileEntry := newListFileEntry(fileInfo, srcDir)
		if _, err := fmt.Fprintln(w, listFileEntry); err != nil {
			return nil, err
		}
		if fileInfo.IsDir() {
			dirsFound++
		} else {
			filesFound++
			bytesFound += fileInfo.Size()
		}
	}

	return &listingFileMetadata{bytes: bytesFound, files: filesFound, dirsDiscovered: dirsFound}, nil
}

func (h *ListHandler) Do(ctx context.Context, taskReqMsg *taskpb.TaskReqMsg) *taskpb.TaskRespMsg {
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

	if len(listSpec.SrcDirectories) == 0 {
		return buildTaskRespMsg(taskReqMsg, nil, log, errors.New("list spec did not contain any source directories"))
	}
	fileInfos, err := listDirectory(listSpec.SrcDirectories[0])
	if err != nil {
		w.CloseWithError(err)
		return buildTaskRespMsg(taskReqMsg, nil, log, err)
	}

	// Set the resumable upload chunk size.
	if t, ok := w.(*storage.Writer); ok {
		t.ChunkSize, err = getListingUploadChunkSize(fileInfos, listSpec.SrcDirectories[0], h.resumableChunkSize)
		if err != nil {
			w.CloseWithError(err)
			return buildTaskRespMsg(taskReqMsg, nil, log, err)
		}
	}

	if _, err := fmt.Fprintln(w, taskReqMsg.TaskRelRsrcName); err != nil {
		w.CloseWithError(err)
		return buildTaskRespMsg(taskReqMsg, nil, log, err)
	}

	listMD, err := writeListingFile(fileInfos, listSpec.SrcDirectories[0], w)
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

	return buildTaskRespMsg(taskReqMsg, nil, log, nil)
}
