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
	"github.com/golang/mock/gomock"
	"io/ioutil"
	"log"
	"os"
	"strings"
	"testing"
)

func copySuccessCompletionMessage() *TaskCompletionMessage {
	return &TaskCompletionMessage{
		FullTaskId: jobConfigId + ":" + jobRunId + ":" + taskId,
		Status:     "SUCCESS",
		TaskParams: map[string]interface{}{
			"src_file":                "file",
			"dst_bucket":              "bucket",
			"dst_object":              "object",
			"expected_generation_num": 1,
		},
		LogEntry: map[string]interface{}{"logkey": "logval"},
	}
}

func TestUploadGCSProgressMessageHandlerInvalidCompletionMessage(t *testing.T) {
	handler := UploadGCSProgressMessageHandler{}

	jobSpec := &JobSpec{}
	taskCompletionMessage := copySuccessCompletionMessage()
	taskCompletionMessage.FullTaskId = "garbage"
	log.SetOutput(ioutil.Discard) // Suppress the log spam.
	_, err := handler.HandleMessage(jobSpec, taskCompletionMessage)
	defer log.SetOutput(os.Stdout) // Reenable logging.

	if err == nil {
		t.Errorf("error is nil, expected error: %v.", errInvalidCompletionMessage)
	}
}

func TestUploadGCSProgressMessageHandlerMissingParams(t *testing.T) {
	handler := UploadGCSProgressMessageHandler{}

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

func TestUploadGCSProgressMessageHandlerFailReadingGenNum(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	errorMsg := "failed to read metadata"
	mockObjectMetadataReader := NewMockObjectMetadataReader(mockCtrl)
	mockObjectMetadataReader.EXPECT().
		GetMetadata(gomock.Any(), gomock.Any()).
		Return(nil, errors.New(errorMsg))

	handler := UploadGCSProgressMessageHandler{
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

func TestUploadGCSProgressMessageHandlerSuccess(t *testing.T) {
	uploadGCSTask := &Task{
		JobConfigId: jobConfigId,
		JobRunId:    jobRunId,
		TaskType:    uploadGCSTaskType,
		TaskId:      taskId,
		Status:      Success,
	}

	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	mockObjectMetadataReader := NewMockObjectMetadataReader(mockCtrl)
	mockObjectMetadataReader.EXPECT().
		GetMetadata(gomock.Any(), gomock.Any()).
		Return(&ObjectMetadata{GenerationNumber: 1}, nil)

	handler := UploadGCSProgressMessageHandler{
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
		Task:     uploadGCSTask,
		LogEntry: NewLogEntry(map[string]interface{}{"logkey": "logval"}),
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
