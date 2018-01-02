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

package dcp

import (
	"errors"
	"strings"
	"testing"

	"github.com/golang/mock/gomock"
)

func copySuccessCompletionMessage() *TaskCompletionMessage {
	return &TaskCompletionMessage{
		TaskRRName: testTaskRRName,
		Status:     "SUCCESS",
		TaskParams: map[string]interface{}{
			"src_file":                "file",
			"dst_bucket":              "bucket",
			"dst_object":              "object",
			"expected_generation_num": 1,
		},
		LogEntry: LogEntry{"logkey": "logval"},
	}
}

func TestCopyProgressMessageHandlerInvalidCompletionMessage(t *testing.T) {
	handler := CopyProgressMessageHandler{}

	jobSpec := &JobSpec{}
	taskCompletionMessage := copySuccessCompletionMessage()
	taskCompletionMessage.TaskRRName = "garbage"
	_, err := handler.HandleMessage(jobSpec, taskCompletionMessage)

	if err == nil {
		t.Errorf("error is nil, expected error: %v.", errInvalidCompletionMessage)
	}
}

func TestCopyProgressMessageHandlerMissingParams(t *testing.T) {
	handler := CopyProgressMessageHandler{}

	jobSpec := &JobSpec{}
	taskCompletionMessage := copySuccessCompletionMessage()
	taskCompletionMessage.TaskParams = TaskParams{}
	_, err := handler.HandleMessage(jobSpec, taskCompletionMessage)

	if err == nil {
		t.Error("error is nil, expected error: missing params...")
	} else if !strings.Contains(err.Error(), "missing params") {
		t.Errorf("expected error: %s, found: %s.", "missing params", err.Error())
	}
}

func TestCopyProgressMessageHandlerFailReadingGenNum(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	errorMsg := "failed to read metadata"
	mockObjectMetadataReader := NewMockObjectMetadataReader(mockCtrl)
	mockObjectMetadataReader.EXPECT().
		GetMetadata(gomock.Any(), gomock.Any(), gomock.Any()).
		Return(nil, errors.New(errorMsg))

	handler := CopyProgressMessageHandler{
		ObjectMetadataReader: mockObjectMetadataReader,
	}

	jobSpec := &JobSpec{}
	_, err := handler.HandleMessage(jobSpec, copySuccessCompletionMessage())

	if err == nil {
		t.Errorf("error is nil, expected error: %s.", errorMsg)
	} else if err.Error() != errorMsg {
		t.Errorf("expected error: %s, found: %s.", errorMsg, err.Error())
	}
}

func TestCopyProgressMessageHandlerSuccess(t *testing.T) {
	copyTask := &Task{
		TaskRRStruct: *NewTaskRRStruct(testProjectID, testJobConfigID, testJobRunID, testTaskID),
		TaskType:     copyTaskType,
		Status:       Success,
	}

	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	mockObjectMetadataReader := NewMockObjectMetadataReader(mockCtrl)
	mockObjectMetadataReader.EXPECT().
		GetMetadata(gomock.Any(), gomock.Any(), gomock.Any()).
		Return(&ObjectMetadata{GenerationNumber: 1}, nil)

	handler := CopyProgressMessageHandler{
		ObjectMetadataReader: mockObjectMetadataReader,
	}

	jobSpec := &JobSpec{}
	taskUpdate, err := handler.HandleMessage(jobSpec, copySuccessCompletionMessage())

	if err != nil {
		t.Errorf("expecting success, found error: %v.", err)
	}

	if taskUpdate.NewTasks != nil && len(taskUpdate.NewTasks) != 0 {
		t.Errorf("new tasks should be an empty array, new tasks: %v", taskUpdate.NewTasks)
	}

	expectedTaskUpdate := &TaskUpdate{
		Task:     copyTask,
		LogEntry: LogEntry{"logkey": "logval"},
		OriginalTaskParams: TaskParams{
			"src_file":                "file",
			"dst_bucket":              "bucket",
			"dst_object":              "object",
			"expected_generation_num": 1,
		},
		TransactionalSemantics: &FileIntegritySemantics{ExpectedGenerationNum: 1},
	}

	DeepEqualCompare("TaskUpdate", expectedTaskUpdate, taskUpdate, t)

	// Check pieces one at a time, for convenient visualization.
	DeepEqualCompare("TaskUpdate.Task", expectedTaskUpdate.Task, taskUpdate.Task, t)
	DeepEqualCompare("TaskUpdate.LogEntry", expectedTaskUpdate.LogEntry, taskUpdate.LogEntry, t)
	DeepEqualCompare("TaskUpdate.OriginalTaskParams",
		expectedTaskUpdate.OriginalTaskParams, taskUpdate.OriginalTaskParams, t)
}
