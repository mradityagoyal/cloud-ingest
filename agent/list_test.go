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
	"sort"
	"testing"

	"github.com/GoogleCloudPlatform/cloud-ingest/gcloud"
	"github.com/GoogleCloudPlatform/cloud-ingest/helpers"
	"github.com/golang/mock/gomock"

	taskpb "github.com/GoogleCloudPlatform/cloud-ingest/proto/task_go_proto"
)

func testListSpec(srcDir string) *taskpb.Spec {
	return &taskpb.Spec{
		Spec: &taskpb.Spec_ListSpec{
			ListSpec: &taskpb.ListSpec{
				DstListResultBucket:   "bucket",
				DstListResultObject:   "object",
				SrcDirectory:          srcDir,
				ExpectedGenerationNum: 0,
			},
		},
	}
}

func testListTaskReqMsg(taskRelRsrcName, srcDir string) *taskpb.TaskReqMsg {
	return &taskpb.TaskReqMsg{
		TaskRelRsrcName: taskRelRsrcName,
		Spec:            testListSpec(srcDir),
	}
}

func TestDirNotFound(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	writer := helpers.NewStringWriteCloser(nil)
	mockGCS := gcloud.NewMockGCS(mockCtrl)
	mockGCS.EXPECT().NewWriterWithCondition(
		context.Background(), "bucket", "object", gomock.Any()).Return(writer)

	h := ListHandler{gcs: mockGCS}
	taskReqParams := testListTaskReqMsg("task", "dir does not exist")
	taskRespMsg := h.Do(context.Background(), taskReqParams)
	checkFailureWithType("task", taskpb.FailureType_FILE_NOT_FOUND_FAILURE, taskRespMsg, t)
	if writer.WrittenString() != "" {
		t.Errorf("expected nothing written but found: %s", writer.WrittenString())
	}
}

func TestListSuccessEmptyDir(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	writer := &helpers.StringWriteCloser{}

	taskRelRsrcName := "projects/project_A/jobConfigs/config_B/jobRuns/run_C/tasks/task_D"
	var expectedListResult bytes.Buffer
	expectedListResult.WriteString(fmt.Sprintln(taskRelRsrcName))

	tmpDir := helpers.CreateTmpDir("", "test-list-agent-")
	defer os.RemoveAll(tmpDir)

	mockGCS := gcloud.NewMockGCS(mockCtrl)
	mockGCS.EXPECT().NewWriterWithCondition(
		context.Background(), "bucket", "object", gomock.Any()).Return(writer)

	h := ListHandler{gcs: mockGCS}
	taskReqParams := testListTaskReqMsg(taskRelRsrcName, tmpDir)
	taskRespMsg := h.Do(context.Background(), taskReqParams)
	checkSuccessMsg(taskRelRsrcName, taskRespMsg, t)
	if writer.WrittenString() != expectedListResult.String() {
		t.Errorf("expected to write \"%s\", found: \"%s\"",
			expectedListResult.String(), writer.WrittenString())
	}
	// Check the log fields.
	/* TODO(b/78596512): Proto~ify the AgentLogFields (aka log entry) in the schema.
	if taskRespMsg.AgentLogFields["files_found"].(int64) != int64(0) {
		t.Errorf("expected 0 files but found %d", taskRespMsg.AgentLogFields["files_found"])
	}
	if taskRespMsg.AgentLogFields["bytes_found"].(int64) != int64(0) {
		t.Errorf("expected 0 bytes but found %d", taskRespMsg.AgentLogFields["bytes_found"])
	}
	*/
}

func TestListSuccessFlatDir(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	writer := &helpers.StringWriteCloser{}

	taskRelRsrcName := "projects/project_A/jobConfigs/config_B/jobRuns/run_C/tasks/task_D"
	var expectedListResult bytes.Buffer
	expectedListResult.WriteString(fmt.Sprintln(taskRelRsrcName))

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
		expectedListResult.WriteString(fmt.Sprintln(ListFileEntry{false, path}))
	}

	mockGCS := gcloud.NewMockGCS(mockCtrl)
	mockGCS.EXPECT().NewWriterWithCondition(
		context.Background(), "bucket", "object", gomock.Any()).Return(writer)

	h := ListHandler{gcs: mockGCS}
	taskReqParams := testListTaskReqMsg(taskRelRsrcName, tmpDir)
	taskRespMsg := h.Do(context.Background(), taskReqParams)
	checkSuccessMsg(taskRelRsrcName, taskRespMsg, t)
	if writer.WrittenString() != expectedListResult.String() {
		t.Errorf("expected to write \"%s\", found: \"%s\"",
			expectedListResult.String(), writer.WrittenString())
	}
	// Check the log entry fields.
	/* TODO(b/78596512): Proto~ify the AgentLogFields (aka log entry) in the schema.
	if taskRespMsg.AgentLogFields["files_found"].(int64) != int64(10) {
		t.Errorf("expected 0 files but found %d", taskRespMsg.AgentLogFields["files_found"])
	}
	if taskRespMsg.AgentLogFields["bytes_found"].(int64) != int64(100) {
		t.Errorf("expected 0 bytes but found %d", taskRespMsg.AgentLogFields["bytes_found"])
	}
	*/
}

func TestListSuccessNestedDir(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	writer := &helpers.StringWriteCloser{}

	taskRelRsrcName := "projects/project_A/jobConfigs/config_B/jobRuns/run_C/tasks/task_D"
	var expectedListResult bytes.Buffer
	expectedListResult.WriteString(fmt.Sprintln(taskRelRsrcName))

	tmpDir := helpers.CreateTmpDir("", "test-list-agent-")
	nestedTmpDir := helpers.CreateTmpDir(tmpDir, "sub-dir-")
	emptyDir := helpers.CreateTmpDir(tmpDir, "empty-dir-")
	defer os.RemoveAll(tmpDir)

	expectedListResult.WriteString(fmt.Sprintln(ListFileEntry{true, emptyDir}))
	expectedListResult.WriteString(fmt.Sprintln(ListFileEntry{true, nestedTmpDir}))

	fileContent := "0123456789"
	filePaths := make([]string, 0)

	for i := 0; i < 10; i++ {
		filePaths = append(filePaths, helpers.CreateTmpFile(tmpDir, "test-file-", fileContent))
	}
	// The result of the list are in sorted order.
	sort.Strings(filePaths)
	for _, path := range filePaths {
		expectedListResult.WriteString(fmt.Sprintln(ListFileEntry{false, path}))
	}
	// Create some files in the sub-dir. These should not be in the list output.
	for i := 0; i < 10; i++ {
		filePaths = append(filePaths, helpers.CreateTmpFile(nestedTmpDir, "test-file-", fileContent))
	}

	mockGCS := gcloud.NewMockGCS(mockCtrl)
	mockGCS.EXPECT().NewWriterWithCondition(
		context.Background(), "bucket", "object", gomock.Any()).Return(writer)

	h := ListHandler{gcs: mockGCS}
	taskReqParams := testListTaskReqMsg(taskRelRsrcName, tmpDir)
	taskRespMsg := h.Do(context.Background(), taskReqParams)
	checkSuccessMsg(taskRelRsrcName, taskRespMsg, t)
	if writer.WrittenString() != expectedListResult.String() {
		t.Errorf("expected to write \"%s\", found: \"%s\"",
			expectedListResult.String(), writer.WrittenString())
	}

	wantLogFields := LogFields{
		"worker_id":        workerID,
		"file_stat_errors": 0,
		"files_found":      int64(10),
		"bytes_found":      int64(100),
		"dirs_found":       int64(2),
	}
	if taskRespMsg.AgentLogFields != wantLogFields.String() {
		t.Errorf("got logFields: %+v, want: %+v", taskRespMsg.AgentLogFields, wantLogFields)
	}
}
