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
	"bytes"
	"context"
	"fmt"
	"os"
	"reflect"
	"sort"
	"testing"

	"github.com/GoogleCloudPlatform/cloud-ingest/dcp"
	"github.com/GoogleCloudPlatform/cloud-ingest/dcp/proto"
	"github.com/GoogleCloudPlatform/cloud-ingest/gcloud"
	"github.com/GoogleCloudPlatform/cloud-ingest/helpers"
	"github.com/golang/mock/gomock"
)

func TestListNoTaskParams(t *testing.T) {
	h := ListHandler{}
	taskParams := dcp.TaskParams{}
	msg := h.Do(context.Background(), "task", taskParams)
	checkForInvalidTaskParamsArguments("task", msg, t)
}

func TestListMissingOneTaskParams(t *testing.T) {
	h := &ListHandler{}
	taskParams := dcp.TaskParams{
		"dst_list_result_bucket":  "bucket",
		"dst_list_result_object":  "object",
		"src_directory":           "dir",
		"expected_generation_num": 0,
	}
	testMissingOneTaskParams(h, taskParams, t)
}

func TestListInvalidGenerationNum(t *testing.T) {
	h := ListHandler{}
	taskParams := dcp.TaskParams{
		"dst_list_result_bucket":  "bucket",
		"dst_list_result_object":  "object",
		"src_directory":           "dir",
		"expected_generation_num": "not a number",
	}
	msg := h.Do(context.Background(), "task", taskParams)
	checkForInvalidTaskParamsArguments("task", msg, t)
}

func TestDirNotFound(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	writer := helpers.NewStringWriteCloser(nil)
	mockGCS := gcloud.NewMockGCS(mockCtrl)
	mockGCS.EXPECT().NewWriterWithCondition(
		context.Background(), "bucket", "object", gomock.Any()).Return(writer)

	h := ListHandler{gcs: mockGCS}
	taskParams := dcp.TaskParams{
		"dst_list_result_bucket":  "bucket",
		"dst_list_result_object":  "object",
		"src_directory":           "dir does not exist",
		"expected_generation_num": 0,
	}
	msg := h.Do(context.Background(), "task", taskParams)
	checkFailureWithType("task", proto.TaskFailureType_FILE_NOT_FOUND_FAILURE, msg, t)
	if writer.WrittenString() != "" {
		t.Errorf("expected nothing written but found: %s", writer.WrittenString())
	}
}

func TestListSuccessEmptyDir(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	writer := &helpers.StringWriteCloser{}

	taskRRName := "projects/project_A/jobConfigs/config_B/jobRuns/run_C/tasks/task_D"
	var expectedListResult bytes.Buffer
	expectedListResult.WriteString(fmt.Sprintln(taskRRName))

	tmpDir := helpers.CreateTmpDir("", "test-list-agent-")
	defer os.RemoveAll(tmpDir)

	mockGCS := gcloud.NewMockGCS(mockCtrl)
	mockGCS.EXPECT().NewWriterWithCondition(
		context.Background(), "bucket", "object", gomock.Any()).Return(writer)

	h := ListHandler{gcs: mockGCS}
	taskParams := dcp.TaskParams{
		"dst_list_result_bucket":  "bucket",
		"dst_list_result_object":  "object",
		"src_directory":           tmpDir,
		"expected_generation_num": 0,
	}
	msg := h.Do(context.Background(), taskRRName, taskParams)
	checkSuccessMsg(taskRRName, msg, t)
	if writer.WrittenString() != expectedListResult.String() {
		t.Errorf("expected to write \"%s\", found: \"%s\"",
			expectedListResult.String(), writer.WrittenString())
	}
	// Check the log entry fields.
	if msg.LogEntry["files_found"].(int64) != int64(0) {
		t.Errorf("expected 0 files but found %d", msg.LogEntry["files_found"])
	}
	if msg.LogEntry["bytes_found"].(int64) != int64(0) {
		t.Errorf("expected 0 bytes but found %d", msg.LogEntry["bytes_found"])
	}
}

func TestListSuccessFlatDir(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	writer := &helpers.StringWriteCloser{}

	taskRRName := "projects/project_A/jobConfigs/config_B/jobRuns/run_C/tasks/task_D"
	var expectedListResult bytes.Buffer
	expectedListResult.WriteString(fmt.Sprintln(taskRRName))

	tmpDir := helpers.CreateTmpDir("", "test-list-agent-")
	defer os.RemoveAll(tmpDir)

	fileContent := "0123456789"
	filePaths := make([]string, 10)
	for i := 0; i < 10; i++ {
		filePaths[i] = helpers.CreateTmpFile(tmpDir, "test-file-", fileContent)
	}
	// The result of the list are sorted.
	sort.Strings(filePaths)
	for _, path := range filePaths {
		expectedListResult.WriteString(fmt.Sprintln(dcp.ListFileEntry{false, path}))
	}

	mockGCS := gcloud.NewMockGCS(mockCtrl)
	mockGCS.EXPECT().NewWriterWithCondition(
		context.Background(), "bucket", "object", gomock.Any()).Return(writer)

	h := ListHandler{gcs: mockGCS}
	taskParams := dcp.TaskParams{
		"dst_list_result_bucket":  "bucket",
		"dst_list_result_object":  "object",
		"src_directory":           tmpDir,
		"expected_generation_num": 0,
	}
	msg := h.Do(context.Background(), taskRRName, taskParams)
	checkSuccessMsg(taskRRName, msg, t)
	if writer.WrittenString() != expectedListResult.String() {
		t.Errorf("expected to write \"%s\", found: \"%s\"",
			expectedListResult.String(), writer.WrittenString())
	}
	// Check the log entry fields.
	if msg.LogEntry["files_found"].(int64) != int64(10) {
		t.Errorf("expected 0 files but found %d", msg.LogEntry["files_found"])
	}
	if msg.LogEntry["bytes_found"].(int64) != int64(100) {
		t.Errorf("expected 0 bytes but found %d", msg.LogEntry["bytes_found"])
	}
}

func TestListSuccessNestedDir(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	writer := &helpers.StringWriteCloser{}

	taskRRName := "projects/project_A/jobConfigs/config_B/jobRuns/run_C/tasks/task_D"
	var expectedListResult bytes.Buffer
	expectedListResult.WriteString(fmt.Sprintln(taskRRName))

	tmpDir := helpers.CreateTmpDir("", "test-list-agent-")
	nestedTmpDir := helpers.CreateTmpDir(tmpDir, "sub-dir-")
	emptyDir := helpers.CreateTmpDir(tmpDir, "empty-dir-")
	defer os.RemoveAll(tmpDir)

	expectedListResult.WriteString(fmt.Sprintln(dcp.ListFileEntry{true, emptyDir}))
	expectedListResult.WriteString(fmt.Sprintln(dcp.ListFileEntry{true, nestedTmpDir}))

	fileContent := "0123456789"
	filePaths := make([]string, 0)

	for i := 0; i < 10; i++ {
		filePaths = append(filePaths, helpers.CreateTmpFile(tmpDir, "test-file-", fileContent))
	}
	// The result of the list are in sorted order.
	sort.Strings(filePaths)
	for _, path := range filePaths {
		expectedListResult.WriteString(fmt.Sprintln(dcp.ListFileEntry{false, path}))
	}
	// Create some files in the sub-dir. These should not be in the list output.
	for i := 0; i < 10; i++ {
		filePaths = append(filePaths, helpers.CreateTmpFile(nestedTmpDir, "test-file-", fileContent))
	}

	mockGCS := gcloud.NewMockGCS(mockCtrl)
	mockGCS.EXPECT().NewWriterWithCondition(
		context.Background(), "bucket", "object", gomock.Any()).Return(writer)

	h := ListHandler{gcs: mockGCS}
	taskParams := dcp.TaskParams{
		"dst_list_result_bucket":  "bucket",
		"dst_list_result_object":  "object",
		"src_directory":           tmpDir,
		"expected_generation_num": 0,
	}
	msg := h.Do(context.Background(), taskRRName, taskParams)
	checkSuccessMsg(taskRRName, msg, t)
	if writer.WrittenString() != expectedListResult.String() {
		t.Errorf("expected to write \"%s\", found: \"%s\"",
			expectedListResult.String(), writer.WrittenString())
	}

	expectedLogEntry := dcp.LogEntry{
		"worker_id":        workerID,
		"file_stat_errors": 0,
		"files_found":      int64(10),
		"bytes_found":      int64(100),
		"dirs_found":       int64(2),
	}
	if !reflect.DeepEqual(msg.LogEntry, expectedLogEntry) {
		t.Errorf("expected log entry: %+v, but found: %+v", expectedLogEntry, msg.LogEntry)
	}
}
