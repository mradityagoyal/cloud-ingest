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
	"io/ioutil"
	"log"
	"os"
	"strings"
	"testing"

	"github.com/golang/mock/gomock"
)

var (
	testProjectID     string = "project_id_A"
	testJobConfigID   string = "job_config_id_A"
	testJobRunID      string = "job_run_id_A"
	testTaskID        string = "task_id_A"
	testTaskFullIDStr string = NewTaskFullID(
		testProjectID, testJobConfigID, testJobRunID, testTaskID).String()
)

func listSuccessCompletionMessage() *TaskCompletionMessage {
	return &TaskCompletionMessage{
		TaskFullIDStr: testTaskFullIDStr,
		Status:        "SUCCESS",
		TaskParams: map[string]interface{}{
			"dst_list_result_bucket":  "bucket1",
			"dst_list_result_object":  "object",
			"src_directory":           "dir",
			"expected_generation_num": 0,
		},
		LogEntry: LogEntry{"logkey": "logval"},
	}
}

func TestListProgressMessageHandlerInvalidCompletionMessage(t *testing.T) {
	handler := ListProgressMessageHandler{}

	taskCompletionMessage := listSuccessCompletionMessage()
	taskCompletionMessage.TaskFullIDStr = "garbage"
	log.SetOutput(ioutil.Discard) // Suppress the log spam.
	_, err := handler.HandleMessage(nil /* jobSpec */, taskCompletionMessage)
	defer log.SetOutput(os.Stdout) // Reenable logging.
	if err == nil {
		t.Error("error is nil, expected error: cannot parse task id...")
	} else if !strings.Contains(err.Error(), "cannot parse task id") {
		t.Errorf(
			"expected error to contain %s, found: %s.", "cannot parse task id",
			err.Error())
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

func TestListProgressMessageHandlerSuccess(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	taskFullID := NewTaskFullID(testProjectID, testJobConfigID, testJobRunID, testTaskID)
	listTask := &Task{
		TaskFullID: *taskFullID,
		TaskType:   listTaskType,
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

	handler := ListProgressMessageHandler{
		ObjectMetadataReader: mockObjectMetadataReader,
	}

	taskUpdate, err := handler.HandleMessage(nil, listSuccessCompletionMessage())
	if err != nil {
		t.Errorf("expecting success, found error: %v.", err)
	}

	expectedTaskUpdate := &TaskUpdate{
		Task:     listTask,
		LogEntry: LogEntry{"logkey": "logval"},
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

	if len(taskUpdate.NewTasks) != 1 {
		t.Errorf("expecting 1 new task, found %d.", len(taskUpdate.NewTasks))
	}

	// Handle the task spec JSON separately, since it doesn't play well with equality checks.
	expectedNewTaskSpec := `{
		"dst_list_result_bucket": "bucket1",
		"dst_list_result_object": "object",
		"src_directory":          "dir",
		"byte_offset":            0
	}`

	if !AreEqualJSON(expectedNewTaskSpec, taskUpdate.NewTasks[0].TaskSpec) {
		t.Errorf("expected task spec: %s, found: %s",
			expectedNewTaskSpec, taskUpdate.NewTasks[0].TaskSpec)
	}
	// Blow it away.
	taskUpdate.NewTasks[0].TaskSpec = ""

	// Add task (sans spec) to our expected update.
	expectedNewTask := &Task{
		TaskFullID: TaskFullID{
			JobRunFullID: taskFullID.JobRunFullID,
			TaskID:       GetProcessListTaskID("bucket1", "object"),
		},
		TaskType: processListTaskType,
	}

	expectedTaskUpdate.NewTasks = append(expectedTaskUpdate.NewTasks, expectedNewTask)

	DeepEqualCompare("TaskUpdate", expectedTaskUpdate, taskUpdate, t)

	// Check pieces one at a time, for convenient visualization.
	DeepEqualCompare("TaskUpdate.Task", expectedTaskUpdate.Task, taskUpdate.Task, t)
	DeepEqualCompare("TaskUpdate.LogEntry", expectedTaskUpdate.LogEntry, taskUpdate.LogEntry, t)
	DeepEqualCompare("TaskUpdate.OriginalTaskParams",
		expectedTaskUpdate.OriginalTaskParams, taskUpdate.OriginalTaskParams, t)
	DeepEqualCompare("NewTasks", expectedTaskUpdate.NewTasks[0], taskUpdate.NewTasks[0], t)
}
