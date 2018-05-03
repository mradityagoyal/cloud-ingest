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
	"os"
	"path/filepath"
	"sort"

	"cloud.google.com/go/storage"
	"github.com/GoogleCloudPlatform/cloud-ingest/gcloud"
	"github.com/GoogleCloudPlatform/cloud-ingest/helpers"
	"github.com/golang/glog"

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
	return fileInfos, nil
}

func (h *ListHandler) Do(ctx context.Context, taskReqMsg *taskpb.TaskReqMsg) *taskpb.TaskRespMsg {
	listSpec := taskReqMsg.Spec.GetListSpec()
	if listSpec == nil {
		err := errors.New("ListHandler.Do taskReqMsg.Spec is not ListSpec")
		return buildTaskRespMsg(taskReqMsg, nil, nil, err)
	}

	logFields := LogFields{
		"worker_id":        workerID,
		"file_stat_errors": 0,
	}

	w := h.gcs.NewWriterWithCondition(ctx, listSpec.DstListResultBucket, listSpec.DstListResultObject,
		helpers.GetGCSGenerationNumCondition(listSpec.ExpectedGenerationNum))

	// Set the resumable upload chunk size.
	if t, ok := w.(*storage.Writer); ok {
		t.ChunkSize = h.resumableChunkSize
	}

	if _, err := fmt.Fprintln(w, taskReqMsg.TaskRelRsrcName); err != nil {
		w.CloseWithError(err)
		return buildTaskRespMsg(taskReqMsg, nil, logFields, err)
	}

	fileInfos, err := listDirectory(listSpec.SrcDirectory)
	if err != nil {
		w.CloseWithError(err)
		return buildTaskRespMsg(taskReqMsg, nil, logFields, err)
	}
	var bytesFound, filesFound, dirsFound int64
	for _, fileInfo := range fileInfos {
		fullPath := filepath.Join(listSpec.SrcDirectory, fileInfo.Name())
		listFileEntry := ListFileEntry{fileInfo.IsDir(), fullPath}
		if _, err := fmt.Fprintln(w, listFileEntry); err != nil {
			w.CloseWithError(err)
			return buildTaskRespMsg(taskReqMsg, nil, logFields, err)
		}
		if fileInfo.IsDir() {
			dirsFound++
		} else {
			filesFound++
			bytesFound += fileInfo.Size()
		}
	}

	if err := w.Close(); err != nil {
		return buildTaskRespMsg(taskReqMsg, nil, logFields, err)
	}

	logFields["files_found"] = filesFound
	logFields["bytes_found"] = bytesFound
	logFields["dirs_found"] = dirsFound

	return buildTaskRespMsg(taskReqMsg, nil, logFields, nil)
}
