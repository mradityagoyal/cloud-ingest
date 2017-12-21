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
	"crypto/md5"
	"fmt"
	"io"
	"os"
	"path"
	"reflect"

	"cloud.google.com/go/storage"
	"github.com/GoogleCloudPlatform/cloud-ingest/dcp"
	"github.com/GoogleCloudPlatform/cloud-ingest/dcp/proto"
	"github.com/GoogleCloudPlatform/cloud-ingest/gcloud"
	"github.com/GoogleCloudPlatform/cloud-ingest/helpers"
)

// CopyHandler is responsible for handling copy tasks.
type CopyHandler struct {
	gcs                gcloud.GCS
	resumableChunkSize int
}

func NewCopyHandler(
	storageClient *storage.Client, resumableChunkSize int) *CopyHandler {
	return &CopyHandler{gcloud.NewGCSClient(storageClient), resumableChunkSize}
}

func checkForFileChanges(beforeStats os.FileInfo, f *os.File) error {
	afterStats, err := f.Stat()
	if err != nil {
		return err
	}
	if beforeStats.Size() != afterStats.Size() || beforeStats.ModTime() != afterStats.ModTime() {
		return AgentError{
			Msg: fmt.Sprintf(
				"File has been changed during the copy. Before stats:%+v, after stats: %+v",
				beforeStats, afterStats),
			FailureType: proto.TaskFailureType_FILE_MODIFIED_FAILURE,
		}
	}
	return nil
}

func (h *CopyHandler) Do(ctx context.Context, taskRRName string,
	taskParams dcp.TaskParams) dcp.TaskCompletionMessage {

	srcPath, srcPathOK := taskParams["src_file"].(string)
	bucketName, bucketNameOK := taskParams["dst_bucket"].(string)
	objectName, objectNameOK := taskParams["dst_object"].(string)
	generationNum, err := helpers.ToInt64(taskParams["expected_generation_num"])

	dstPath := fmt.Sprint("gs://", path.Join(bucketName, objectName))
	logEntry := dcp.LogEntry{
		"worker_id": workerID,
		"src_file":  srcPath,
		"dst_file":  dstPath,
	}

	if !srcPathOK || !bucketNameOK || !objectNameOK || err != nil {
		return buildTaskCompletionMessage(
			taskRRName, taskParams, logEntry, NewInvalidTaskParamsError(taskParams))
	}

	// TODO(b/70808741): Preserve the POSIX mtime of the file as GCS metadata
	// (goog-reserved-file-mtime)
	w := h.gcs.NewWriterWithCondition(ctx, bucketName, objectName,
		helpers.GetGCSGenerationNumCondition(generationNum))

	// Set the resumable upload chunk size.
	if t, ok := w.(*storage.Writer); ok {
		t.ChunkSize = h.resumableChunkSize
	}

	srcFile, err := os.Open(srcPath)
	if err != nil {
		return buildTaskCompletionMessage(taskRRName, taskParams, logEntry, err)
	}
	defer srcFile.Close()
	stats, err := srcFile.Stat()
	if err != nil {
		return buildTaskCompletionMessage(taskRRName, taskParams, logEntry, err)
	}
	logEntry["src_bytes"] = stats.Size()
	logEntry["src_modified_time"] = stats.ModTime()

	buffer := make([]byte, h.resumableChunkSize)
	hash := md5.New()
	for {
		n, err := srcFile.Read(buffer)
		if err != nil {
			if err == io.EOF {
				break
			}
			w.CloseWithError(err)
			return buildTaskCompletionMessage(taskRRName, taskParams, logEntry, err)
		}
		_, err = w.Write(buffer[:n])
		if err != nil {
			w.CloseWithError(err)
			return buildTaskCompletionMessage(taskRRName, taskParams, logEntry, err)
		}

		_, err = hash.Write(buffer[:n])
		if err != nil {
			w.CloseWithError(err)
			return buildTaskCompletionMessage(taskRRName, taskParams, logEntry, err)
		}
	}

	if err := w.Close(); err != nil {
		return buildTaskCompletionMessage(taskRRName, taskParams, logEntry, err)
	}

	if err := checkForFileChanges(stats, srcFile); err != nil {
		return buildTaskCompletionMessage(taskRRName, taskParams, logEntry, err)
	}

	srcMD5 := hash.Sum(nil)
	logEntry["src_md5"] = srcMD5

	// TODO(b/70814620): Find a way to get partial object attributes. Only few
	// attributes are needed here.
	attrs := w.Attrs()
	logEntry["dst_md5"] = attrs.MD5
	logEntry["dst_bytes"] = attrs.Size
	logEntry["dst_modified_time"] = attrs.Updated

	if !reflect.DeepEqual(srcMD5, attrs.MD5) {
		err := AgentError{
			Msg: fmt.Sprintf("MD5 mismatch for file %s (%s) against object %s (%s)",
				srcPath, srcMD5, dstPath, attrs.MD5),
			FailureType: proto.TaskFailureType_MD5_MISMATCH_FAILURE,
		}
		return buildTaskCompletionMessage(taskRRName, taskParams, logEntry, err)
	}
	return buildTaskCompletionMessage(taskRRName, taskParams, logEntry, nil)
}
