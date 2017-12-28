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
	"os"
	"reflect"
	"strings"
	"testing"
	"time"

	"cloud.google.com/go/storage"
	"github.com/GoogleCloudPlatform/cloud-ingest/dcp"
	"github.com/GoogleCloudPlatform/cloud-ingest/dcp/proto"
	"github.com/GoogleCloudPlatform/cloud-ingest/gcloud"
	"github.com/GoogleCloudPlatform/cloud-ingest/helpers"
	"github.com/golang/mock/gomock"
	"golang.org/x/sync/semaphore"
)

func TestNoTaskParams(t *testing.T) {
	h := CopyHandler{}
	taskParams := dcp.TaskParams{}
	msg := h.Do(context.Background(), "task", taskParams)
	checkForInvalidTaskParamsArguments("task", msg, t)
}

func TestCopyMissingOneTaskParams(t *testing.T) {
	h := &CopyHandler{}
	taskParams := dcp.TaskParams{
		"src_file":                "file",
		"dst_bucket":              "bucket",
		"dst_object":              "object",
		"expected_generation_num": 0,
	}
	testMissingOneTaskParams(h, taskParams, t)
}

func TestCopyInvalidGenerationNum(t *testing.T) {
	h := CopyHandler{}
	taskParams := dcp.TaskParams{
		"src_file":                "file",
		"dst_bucket":              "bucket",
		"dst_object":              "object",
		"expected_generation_num": "not a number",
	}
	msg := h.Do(context.Background(), "task", taskParams)
	checkForInvalidTaskParamsArguments("task", msg, t)
}

func TestSourceNotFound(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	writer := helpers.NewStringWriteCloser(nil)
	mockGCS := gcloud.NewMockGCS(mockCtrl)
	mockGCS.EXPECT().NewWriterWithCondition(
		context.Background(), "bucket", "object", gomock.Any()).Return(writer)

	h := CopyHandler{gcs: mockGCS}
	taskParams := dcp.TaskParams{
		"src_file":                "file does not exist",
		"dst_bucket":              "bucket",
		"dst_object":              "object",
		"expected_generation_num": 0,
	}
	msg := h.Do(context.Background(), "task", taskParams)
	checkFailureWithType("task", proto.TaskFailureType_FILE_NOT_FOUND_FAILURE, msg, t)
}

func TestAcquireBufferMemoryFail(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	writer := helpers.NewStringWriteCloser(&storage.ObjectAttrs{})

	tmpFile := helpers.CreateTmpFile("", "test-agent", "File content.")
	defer os.Remove(tmpFile)

	mockGCS := gcloud.NewMockGCS(mockCtrl)
	mockGCS.EXPECT().NewWriterWithCondition(
		context.Background(), "bucket", "object", gomock.Any()).Return(writer)

	copyMemoryLimit = 5
	h := CopyHandler{mockGCS, 10, semaphore.NewWeighted(5)}
	taskParams := dcp.TaskParams{
		"src_file":                tmpFile,
		"dst_bucket":              "bucket",
		"dst_object":              "object",
		"expected_generation_num": 0,
	}
	msg := h.Do(context.Background(), "task", taskParams)
	checkFailureWithType("task", proto.TaskFailureType_UNKNOWN, msg, t)
	if !strings.Contains(msg.FailureMessage, "total memory buffer limit for copy task") {
		t.Errorf("expected \"total memory buffer limit for copy task is\" in failure message, found: \"%s\"",
			msg.FailureMessage)
	}
}

func TestMD5Mismtach(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	writer := helpers.NewStringWriteCloser(&storage.ObjectAttrs{
		MD5: []byte{10, 11}, // Invalid MD5
	})

	tmpFile := helpers.CreateTmpFile("", "test-agent", "File content.")
	defer os.Remove(tmpFile)

	mockGCS := gcloud.NewMockGCS(mockCtrl)
	mockGCS.EXPECT().NewWriterWithCondition(
		context.Background(), "bucket", "object", gomock.Any()).Return(writer)

	h := CopyHandler{mockGCS, 5, semaphore.NewWeighted(defaultCopyMemoryLimit)}
	taskParams := dcp.TaskParams{
		"src_file":                tmpFile,
		"dst_bucket":              "bucket",
		"dst_object":              "object",
		"expected_generation_num": 0,
	}
	msg := h.Do(context.Background(), "task", taskParams)
	checkFailureWithType("task", proto.TaskFailureType_MD5_MISMATCH_FAILURE, msg, t)
}

func TestCopySuccess(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	content := "File content."
	contentMD5 := []byte{
		0x46, 0x89, 0xA0, 0x3F, 0xCA, 0x99, 0x9B, 0x7F,
		0xE5, 0xA6, 0xC2, 0xF1, 0x36, 0x40, 0xF6, 0x77}
	gcsModTime := time.Now()

	writer := helpers.NewStringWriteCloser(&storage.ObjectAttrs{
		MD5:     contentMD5,
		Size:    int64(len(content)),
		Updated: gcsModTime,
	})

	tmpFile := helpers.CreateTmpFile("", "test-agent", content)
	defer os.Remove(tmpFile)

	mockGCS := gcloud.NewMockGCS(mockCtrl)
	mockGCS.EXPECT().NewWriterWithCondition(
		context.Background(), "bucket", "object", gomock.Any()).Return(writer)

	h := CopyHandler{mockGCS, 5, semaphore.NewWeighted(copyMemoryLimit)}
	taskParams := dcp.TaskParams{
		"src_file":                tmpFile,
		"dst_bucket":              "bucket",
		"dst_object":              "object",
		"expected_generation_num": 0,
	}
	msg := h.Do(context.Background(), "task", taskParams)
	checkSuccessMsg("task", msg, t)
	if writer.WrittenString() != content {
		t.Errorf("expected to write \"%s\", found: \"%s\"",
			content, writer.WrittenString())
	}

	srcStats, _ := os.Stat(tmpFile)
	expectedLogEntry := dcp.LogEntry{
		"worker_id":         workerID,
		"src_md5":           contentMD5,
		"dst_md5":           contentMD5,
		"src_bytes":         int64(len(content)),
		"dst_bytes":         int64(len(content)),
		"src_file":          tmpFile,
		"dst_file":          "gs://bucket/object",
		"src_modified_time": srcStats.ModTime(),
		"dst_modified_time": gcsModTime,
	}
	if !reflect.DeepEqual(msg.LogEntry, expectedLogEntry) {
		t.Errorf("expected log entry: %+v, but found: %+v", expectedLogEntry, msg.LogEntry)
	}
}
