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
	"fmt"
	"os"
	"path/filepath"

	"cloud.google.com/go/storage"
	"github.com/GoogleCloudPlatform/cloud-ingest/dcp"
	"github.com/GoogleCloudPlatform/cloud-ingest/gcloud"
	"github.com/GoogleCloudPlatform/cloud-ingest/helpers"
)

// ListHandler is responsible for handling list tasks.
type ListHandler struct {
	gcs                gcloud.GCS
	resumableChunkSize int
}

func NewListHandler(
	storageClient *storage.Client, resumableChunkSize int) *ListHandler {
	return &ListHandler{gcloud.NewGCSClient(storageClient), resumableChunkSize}
}

type FileInfo struct {
	os.FileInfo
	fullPath string
	err      error
}

func listDirectory(ctx context.Context, srcDir string) <-chan FileInfo {
	c := make(chan FileInfo)
	go func() {
		defer close(c)
		filepath.Walk(srcDir, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				c <- FileInfo{info, path, err}
				return filepath.SkipDir
			}
			if ctx.Err() != nil {
				// The context is already closed. Stop execution.
				return filepath.SkipDir
			}
			if info.IsDir() {
				// Ignoring directories.
				return nil
			}
			c <- FileInfo{info, path, nil}
			return nil
		})
	}()
	return c
}

func (h *ListHandler) Do(ctx context.Context, taskRRName string,
	taskParams dcp.TaskParams) dcp.TaskCompletionMessage {

	bucketName, bucketNameOK := taskParams["dst_list_result_bucket"].(string)
	objectName, objectNameOK := taskParams["dst_list_result_object"].(string)
	srcDirectory, srcDirectoryOK := taskParams["src_directory"].(string)
	generationNum, err := helpers.ToInt64(taskParams["expected_generation_num"])

	logEntry := dcp.LogEntry{
		"worker_id":        workerID,
		"file_stat_errors": 0,
	}

	if !bucketNameOK || !objectNameOK || !srcDirectoryOK || err != nil {
		return buildTaskCompletionMessage(
			taskRRName, taskParams, logEntry, NewInvalidTaskParamsError(taskParams))
	}

	w := h.gcs.NewWriterWithCondition(ctx, bucketName, objectName,
		helpers.GetGCSGenerationNumCondition(generationNum))

	// Set the resumable upload chunk size.
	if t, ok := w.(*storage.Writer); ok {
		t.ChunkSize = h.resumableChunkSize
	}

	if _, err := fmt.Fprintln(w, taskRRName); err != nil {
		w.CloseWithError(err)
		return buildTaskCompletionMessage(taskRRName, taskParams, logEntry, err)
	}

	var bytesFound, filesFound int64
	ctxWithCancel, cancelFn := context.WithCancel(ctx)
	for file := range listDirectory(ctxWithCancel, srcDirectory) {
		if file.err != nil {
			cancelFn()
			w.CloseWithError(file.err)
			return buildTaskCompletionMessage(taskRRName, taskParams, logEntry, file.err)
		}
		if _, err := fmt.Fprintln(w, file.fullPath); err != nil {
			cancelFn()
			w.CloseWithError(err)
			return buildTaskCompletionMessage(taskRRName, taskParams, logEntry, err)
		}
		filesFound++
		bytesFound += file.Size()
	}

	if err := w.Close(); err != nil {
		return buildTaskCompletionMessage(taskRRName, taskParams, logEntry, err)
	}

	logEntry["files_found"] = filesFound
	logEntry["bytes_found"] = bytesFound

	return buildTaskCompletionMessage(taskRRName, taskParams, logEntry, nil)
}
