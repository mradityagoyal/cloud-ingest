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
	"context"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"strconv"
	"testing"

	"cloud.google.com/go/storage"
	"github.com/golang/mock/gomock"
	"strings"
)

var (
	jobConfigId string = "job_config_id_A"
	jobRunId    string = "job_run_id_A"
	taskId      string = "task_id_A"
)

func listSuccessCompletionMessage() *TaskCompletionMessage {
	return &TaskCompletionMessage{
		FullTaskId: jobConfigId + ":" + jobRunId + ":" + taskId,
		Status:     "SUCCESS",
		TaskParams: map[string]interface{}{
			"dst_list_result_bucket":  "bucket1",
			"dst_list_result_object":  "object",
			"src_directory":           "dir",
			"expected_generation_num": 0,
		},
		LogEntry: map[string]interface{}{"logkey": "logval"},
	}
}

func TestListProgressMessageHandlerInvalidCompletionMessage(t *testing.T) {
	handler := ListProgressMessageHandler{}

	taskCompletionMessage := listSuccessCompletionMessage()
	taskCompletionMessage.FullTaskId = "garbage"
	log.SetOutput(ioutil.Discard) // Suppress the log spam.
	_, err := handler.HandleMessage(nil /* jobSpec */, taskCompletionMessage)
	defer log.SetOutput(os.Stdout) // Reenable logging.
	if err == nil {
		t.Error("error is nil, expected error: can not parse full task id...")
	} else if !strings.Contains(err.Error(), "can not parse") {
		t.Errorf("expected error: %s, found: %s.", "can not parse full task id", err.Error())
	}
}

func TestListProgressMessageHandlerMissingParams(t *testing.T) {
	handler := ListProgressMessageHandler{}

	taskCompletionMessage := listSuccessCompletionMessage()
	taskCompletionMessage.TaskParams = TaskParams{}

	_, err := handler.HandleMessage(nil /* jobSpec */, taskCompletionMessage)
	if err == nil {
		t.Error("error is nil, expected error: missing params...")
	} else if !strings.Contains(err.Error(), "missing params") {
		t.Errorf("expected error: %s, found: %s.", "missing params", err.Error())
	}
}

func TestListProgressMessageHandlerFailReadingGenNum(t *testing.T) {
	// Read the successful task, but fail to pick up on the metadata.
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	errorMsg := "failed to read metadata"
	mockObjectMetadataReader := NewMockObjectMetadataReader(mockCtrl)
	mockObjectMetadataReader.EXPECT().
		GetMetadata(gomock.Any(), gomock.Any(), gomock.Any()).
		Return(nil, errors.New(errorMsg))

	handler := ListProgressMessageHandler{
		ObjectMetadataReader: mockObjectMetadataReader,
	}

	log.SetOutput(ioutil.Discard) // Suppress the log spam.
	_, err := handler.HandleMessage(nil /* jobSpec */, listSuccessCompletionMessage())
	defer log.SetOutput(os.Stdout) // Reenable logging.
	if err == nil {
		t.Errorf("error is nil, expected error: %s.", errorMsg)
	} else if err.Error() != errorMsg {
		t.Errorf("expected error: %s, found: %s.", errorMsg, err.Error())
	}
}

func TestListProgressMessageHandlerFailReadingListResult(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	mockListReader := NewMockListingResultReader(mockCtrl)
	errorMsg := "Failed reading listing result."
	mockListReader.EXPECT().ReadListResult(context.Background(), "bucket1", "object").Return(nil, errors.New(errorMsg))

	mockObjectMetadataReader := NewMockObjectMetadataReader(mockCtrl)
	mockObjectMetadataReader.EXPECT().
		GetMetadata(gomock.Any(), gomock.Any(), gomock.Any()).
		Return(&ObjectMetadata{GenerationNumber: 1}, nil)

	handler := ListProgressMessageHandler{
		ListingResultReader:  mockListReader,
		ObjectMetadataReader: mockObjectMetadataReader,
	}

	log.SetOutput(ioutil.Discard) // Suppress the log spam.
	_, err := handler.HandleMessage(nil /* jobSpec */, listSuccessCompletionMessage())
	defer log.SetOutput(os.Stdout) // Reenable logging.
	if err == nil {
		t.Errorf("error is nil, expected error: %s.", errorMsg)
	} else if err.Error() != errorMsg {
		t.Errorf("expected error: %s, found: %s.", errorMsg, err.Error())
	}
}

func TestListProgressMessageHandlerEmptyChannel(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	mockListReader := NewMockListingResultReader(mockCtrl)
	c := make(chan string)
	close(c)
	mockListReader.EXPECT().ReadListResult(context.Background(), "bucket1", "object").Return(c, nil)

	mockObjectMetadataReader := NewMockObjectMetadataReader(mockCtrl)
	mockObjectMetadataReader.EXPECT().
		GetMetadata(gomock.Any(), gomock.Any(), gomock.Any()).
		Return(&ObjectMetadata{GenerationNumber: 1}, nil)

	handler := ListProgressMessageHandler{
		ListingResultReader:  mockListReader,
		ObjectMetadataReader: mockObjectMetadataReader,
	}

	jobSpec := &JobSpec{
		GCSBucket: "bucket2",
	}

	_, err := handler.HandleMessage(jobSpec, listSuccessCompletionMessage())
	errorMsg := fmt.Sprintf(noTaskIdInListOutput, "job_config_id_A:job_run_id_A:task_id_A", "")
	if err == nil {
		t.Errorf("error is nil, expected error: %s.", errorMsg)
	} else if err.Error() != errorMsg {
		t.Errorf("expected error: %s, found: %s.", errorMsg, err.Error())
	}
}

func TestListProgressMessageHandlerMismatchedTask(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	mockListReader := NewMockListingResultReader(mockCtrl)
	c := make(chan string)
	go func() {
		defer close(c)
		c <- "task_id_B"
	}()
	mockListReader.EXPECT().ReadListResult(context.Background(), "bucket1", "object").Return(c, nil)

	mockObjectMetadataReader := NewMockObjectMetadataReader(mockCtrl)
	mockObjectMetadataReader.EXPECT().
		GetMetadata(gomock.Any(), gomock.Any(), gomock.Any()).
		Return(&ObjectMetadata{GenerationNumber: 1}, nil)

	handler := ListProgressMessageHandler{
		ListingResultReader:  mockListReader,
		ObjectMetadataReader: mockObjectMetadataReader,
	}

	jobSpec := &JobSpec{
		GCSBucket: "bucket2",
	}

	_, err := handler.HandleMessage(jobSpec, listSuccessCompletionMessage())
	errorMsg := fmt.Sprintf(noTaskIdInListOutput, "job_config_id_A:job_run_id_A:task_id_A", "task_id_B")
	if err == nil {
		t.Errorf("error is nil, expected error: %s.", errorMsg)
	} else if err.Error() != errorMsg {
		t.Errorf("expected error: %s, found: %s.", errorMsg, err.Error())
	}
}

func TestListProgressMessageHandlerMetadataError(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	// Setup ListingResultReader
	mockListReader := NewMockListingResultReader(mockCtrl)
	c := make(chan string)
	go func() {
		defer close(c)
		c <- "job_config_id_A:job_run_id_A:task_id_A"
		c <- "dir/file0"
	}()
	mockListReader.EXPECT().ReadListResult(context.Background(), "bucket1", "object").Return(c, nil)

	// Setup ObjectMetadataReader - file0 doesn't exist, file1 is at generation 1.
	expectedError := "Some transient gcs metadata error"
	mockObjectMetadataReader := NewMockObjectMetadataReader(mockCtrl)
	mockObjectMetadataReader.EXPECT().
		GetMetadata(gomock.Any(), gomock.Any(), "object").
		Return(&ObjectMetadata{GenerationNumber: 1}, nil)
	mockObjectMetadataReader.EXPECT().
		GetMetadata(gomock.Any(), gomock.Any(), "file0").
		Return(nil, errors.New(expectedError))

	handler := ListProgressMessageHandler{
		ListingResultReader:  mockListReader,
		ObjectMetadataReader: mockObjectMetadataReader,
	}

	jobSpec := &JobSpec{
		GCSBucket: "bucket2",
	}

	log.SetOutput(ioutil.Discard) // Suppress the log spam.
	_, err := handler.HandleMessage(jobSpec, listSuccessCompletionMessage())
	defer log.SetOutput(os.Stdout) // Reenable logging.
	if err == nil {
		t.Errorf("expected error: %v.", expectedError)
	} else if err.Error() != expectedError {
		t.Errorf("expected error: %v, found: %v.", expectedError, err)
	}
}

func TestListProgressMessageHandlerSuccess(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	// Setup ListingResultReader
	mockListReader := NewMockListingResultReader(mockCtrl)
	c := make(chan string)
	go func() {
		defer close(c)
		c <- "job_config_id_A:job_run_id_A:task_id_A"
		c <- "dir/file0"
		c <- "dir/file1"
	}()
	mockListReader.EXPECT().ReadListResult(context.Background(), "bucket1", "object").Return(c, nil)

	listTask := &Task{
		JobConfigId: jobConfigId,
		JobRunId:    jobRunId,
		TaskId:      "task_id_A",
		TaskType:    listTaskType,
		TaskSpec: `{
			"dst_list_result_bucket": "bucket1",
			"dst_list_result_object": "object",
			"src_directory": "dir",
			"expected_generation_num": 0
		}`,
		Status: Success,
	}

	// Setup ObjectMetadataReader - file0 doesn't exist, file1 is at generation 1.
	mockObjectMetadataReader := NewMockObjectMetadataReader(mockCtrl)
	mockObjectMetadataReader.EXPECT().
		GetMetadata(gomock.Any(), gomock.Any(), "object").
		Return(&ObjectMetadata{GenerationNumber: 1}, nil)
	mockObjectMetadataReader.EXPECT().
		GetMetadata(gomock.Any(), gomock.Any(), "file0").
		Return(nil, storage.ErrObjectNotExist)
	mockObjectMetadataReader.EXPECT().
		GetMetadata(gomock.Any(), gomock.Any(), "file1").
		Return(&ObjectMetadata{GenerationNumber: 1}, nil)

	handler := ListProgressMessageHandler{
		ListingResultReader:  mockListReader,
		ObjectMetadataReader: mockObjectMetadataReader,
	}

	jobSpec := &JobSpec{
		GCSBucket: "bucket2",
	}

	taskUpdate, err := handler.HandleMessage(jobSpec, listSuccessCompletionMessage())
	if err != nil {
		t.Errorf("expecting success, found error: %v.", err)
	}

	expectedTaskUpdate := &TaskUpdate{
		Task:     listTask,
		LogEntry: NewLogEntry(map[string]interface{}{"logkey": "logval"}),
		OriginalTaskParams: TaskParams{
			"dst_list_result_bucket":  "bucket1",
			"dst_list_result_object":  "object",
			"expected_generation_num": 0,
			"src_directory":           "dir",
		},
		TransactionalSemantics: &FileIntegritySemantics{ExpectedGenerationNum: 1},
	}

	// No task spec on TaskUpdate.
	listTask.TaskSpec = ""

	if len(taskUpdate.NewTasks) != 2 {
		t.Errorf("expecting 2 tasks, found %d.", len(taskUpdate.NewTasks))
	}

	for i := 0; i < 2; i++ {
		// Handle the task spec json separately, since it doesn't play well with equality checks.
		expectedNewTaskSpec := fmt.Sprintf(`{
				"dst_bucket": "bucket2",
				"dst_object": "file%d",
			  "expected_generation_num": %d,
				"src_file": "dir/file%d"
			}`, i, i, i)

		if !AreEqualJSON(expectedNewTaskSpec, taskUpdate.NewTasks[i].TaskSpec) {
			t.Errorf("expected task spec: %s, found: %s", expectedNewTaskSpec, taskUpdate.NewTasks[i].TaskSpec)
		}

		// Blow it away.
		taskUpdate.NewTasks[i].TaskSpec = ""

		// Add task (sans spec) to our expected update.
		expectedNewTask := &Task{
			JobConfigId: jobConfigId,
			JobRunId:    jobRunId,
			TaskType:    uploadGCSTaskType,
			TaskId:      GetUploadGCSTaskId("dir/file" + strconv.Itoa(i)),
		}

		expectedTaskUpdate.NewTasks = append(expectedTaskUpdate.NewTasks, expectedNewTask)
	}

	DeepEqualCompare("TaskUpdate", expectedTaskUpdate, taskUpdate, t)

	// Check pieces one at a time, for convenient visualization.
	DeepEqualCompare("TaskUpdate.Task", expectedTaskUpdate.Task, taskUpdate.Task, t)
	DeepEqualCompare("TaskUpdate.LogEntry", expectedTaskUpdate.LogEntry, taskUpdate.LogEntry, t)
	DeepEqualCompare("TaskUpdate.OriginalTaskParams",
		expectedTaskUpdate.OriginalTaskParams, taskUpdate.OriginalTaskParams, t)
	for i := 0; i < 2; i++ {
		DeepEqualCompare("NewTasks", expectedTaskUpdate.NewTasks[i], taskUpdate.NewTasks[i], t)
	}
}
