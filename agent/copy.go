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
	"flag"
	"fmt"
	"io"
	"math"
	"os"
	"path"
	"reflect"
	"strconv"

	"golang.org/x/time/rate"

	"cloud.google.com/go/storage"
	"github.com/GoogleCloudPlatform/cloud-ingest/dcp"
	"github.com/GoogleCloudPlatform/cloud-ingest/dcp/proto"
	"github.com/GoogleCloudPlatform/cloud-ingest/gcloud"
	"github.com/GoogleCloudPlatform/cloud-ingest/helpers"
	"golang.org/x/sync/semaphore"
)

const (
	defaultCopyMemoryLimit int64 = 1 << 30 // Default memory limit is 1 GB.
)

var (
	copyMemoryLimit int64
)

func init() {
	flag.Int64Var(&copyMemoryLimit, "copy-max-memory", defaultCopyMemoryLimit,
		"Max memory buffer (in bytes) consumed by the copy tasks.")
}

// CopyHandler is responsible for handling copy tasks.
type CopyHandler struct {
	gcs                gcloud.GCS
	resumableChunkSize int
	memoryLimiter      *semaphore.Weighted
}

func NewCopyHandler(
	storageClient *storage.Client, resumableChunkSize int) *CopyHandler {
	return &CopyHandler{
		gcs:                gcloud.NewGCSClient(storageClient),
		resumableChunkSize: resumableChunkSize,
		memoryLimiter:      semaphore.NewWeighted(copyMemoryLimit),
	}
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

	w := h.gcs.NewWriterWithCondition(ctx, bucketName, objectName,
		helpers.GetGCSGenerationNumCondition(generationNum))

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

	bufferSize := stats.Size()
	if t, ok := w.(*storage.Writer); ok {
		t.Metadata = map[string]string{
			dcp.MTIME_ATTR_NAME: strconv.FormatInt(stats.ModTime().Unix(), 10),
		}
		if bufferSize > int64(h.resumableChunkSize) {
			bufferSize = int64(h.resumableChunkSize)
			t.ChunkSize = h.resumableChunkSize
		}
	}

	if bufferSize > copyMemoryLimit {
		err := fmt.Errorf(
			"total memory buffer limit for copy task is %d bytes, but task requires "+
				"%d bytes (resumeableChunkSize)",
			copyMemoryLimit, bufferSize)
		return buildTaskCompletionMessage(taskRRName, taskParams, logEntry, err)
	}
	if err := h.memoryLimiter.Acquire(ctx, bufferSize); err != nil {
		return buildTaskCompletionMessage(taskRRName, taskParams, logEntry, err)
	}
	defer h.memoryLimiter.Release(bufferSize)
	buffer := make([]byte, bufferSize)

	hash := md5.New()

	bandwidth, ok := taskParams["bandwidth"].(int)
	maxBucketSize := math.MaxInt32
	limiter := rate.NewLimiter(rate.Limit(bandwidth), maxBucketSize)
	if !ok || bandwidth <= 0 {
		limiter = rate.NewLimiter(rate.Inf, maxBucketSize)
	}
	// The rate limiter starts with a full token bucket, we need to empty it before copying.
	if err := limiter.WaitN(ctx, maxBucketSize); err != nil {
		return buildTaskCompletionMessage(taskRRName, taskParams, logEntry, err)
	}
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
		if err := limiter.WaitN(ctx, n); err != nil {
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
