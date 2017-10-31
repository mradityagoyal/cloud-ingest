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
	"encoding/json"
	"fmt"
	"github.com/GoogleCloudPlatform/cloud-ingest/dcp/proto"
	"reflect"
	"testing"
)

/*******************************************************************************
TaskUpdateCollection Tests
*******************************************************************************/

func TestAddTaskUpdateEmptyCollection(t *testing.T) {
	tc := &TaskUpdateCollection{}

	expectedUpdate := &TaskUpdate{
		Task: &Task{
			JobConfigId: "dummy-config",
			JobRunId:    "dummy-run",
			TaskId:      "dummy-task",
		},
	}
	tc.AddTaskUpdate(expectedUpdate)
	retUpdate := tc.GetTaskUpdate(expectedUpdate.Task.getTaskFullId())
	if retUpdate != expectedUpdate {
		t.Errorf("task mismatch, expected: %v, returned: %v", expectedUpdate, retUpdate)
	}
	if tc.Size() != 1 {
		t.Errorf("expected 1 task update in the colleection, found %v", tc.Size())
	}
}

func TestAddTaskUpdateIgnoreTask(t *testing.T) {
	tc := &TaskUpdateCollection{}

	expectedUpdate := &TaskUpdate{
		Task: &Task{
			JobConfigId: "dummy-config",
			JobRunId:    "dummy-run",
			TaskId:      "dummy-task",
			Status:      Success,
		},
	}
	tc.AddTaskUpdate(expectedUpdate)

	newTask := *expectedUpdate.Task
	duplicateUpdate := &TaskUpdate{
		Task: &newTask,
	}
	tc.AddTaskUpdate(duplicateUpdate)

	newTask = *expectedUpdate.Task
	newTask.Status = Failed
	duplicateUpdate = &TaskUpdate{
		Task: &newTask,
	}
	tc.AddTaskUpdate(duplicateUpdate)

	retUpdate := tc.GetTaskUpdate(expectedUpdate.Task.getTaskFullId())
	if tc.Size() != 1 {
		t.Errorf("expected 1 task update in the collection, found %v", tc.Size())
	}
	if retUpdate != expectedUpdate {
		t.Errorf("task mismatch, expected: %v, returned: %v", expectedUpdate, retUpdate)
	}
}

func TestAddTaskUpdateOverrideTask(t *testing.T) {
	tc := &TaskUpdateCollection{}

	taskUpdate := &TaskUpdate{
		Task: &Task{
			JobConfigId: "dummy-config",
			JobRunId:    "dummy-run",
			TaskId:      "dummy-task",
			Status:      Failed,
		},
	}
	tc.AddTaskUpdate(taskUpdate)

	newTask := *taskUpdate.Task
	newTask.Status = Success
	expectedUpdate := &TaskUpdate{
		Task: &newTask,
	}
	tc.AddTaskUpdate(expectedUpdate)

	retUpdate := tc.GetTaskUpdate(taskUpdate.Task.getTaskFullId())
	if tc.Size() != 1 {
		t.Errorf("expected 1 task update in the colleection, found %v", tc.Size())
	}
	if retUpdate != expectedUpdate {
		t.Errorf("task mismatch, expected: %v, returned: %v", expectedUpdate, retUpdate)
	}
}

func TestGetTaskUpdatesEmptyCollection(t *testing.T) {
	tc := &TaskUpdateCollection{}
	count := 0
	for range tc.GetTaskUpdates() {
		count++
	}
	if count != 0 {
		t.Errorf("expected 0 updates, found", count)
	}
}

func TestGetTaskUpdates(t *testing.T) {
	tc := &TaskUpdateCollection{}

	taskUpdate1 := &TaskUpdate{
		Task: &Task{
			JobConfigId: "dummy-config",
			JobRunId:    "dummy-run",
			TaskId:      "dummy-task-1",
		},
	}

	taskUpdate2 := &TaskUpdate{
		Task: &Task{
			JobConfigId: "dummy-config",
			JobRunId:    "dummy-run",
			TaskId:      "dummy-task-2",
		},
	}

	tc.AddTaskUpdate(taskUpdate1)
	tc.AddTaskUpdate(taskUpdate2)

	count := 0
	for range tc.GetTaskUpdates() {
		count++
	}
	if count != 2 {
		t.Errorf("expected 2 updates, found", count)
	}

	taskUpdates := tc.GetTaskUpdates()
	retTask1 := <-taskUpdates
	retTask2 := <-taskUpdates
	if retTask1 == retTask2 ||
		(retTask1 != taskUpdate1 && retTask1 != taskUpdate2) ||
		(retTask2 != taskUpdate1 && retTask2 != taskUpdate2) {
		t.Errorf("expected 2 task updates (%v, %v), but found (%v, %v)",
			taskUpdate1, taskUpdate2, retTask1, retTask2)
	}
}

/*******************************************************************************
Task Methods Tests
*******************************************************************************/

func TestTaskCompletionMessageJsonToTaskUpdate(t *testing.T) {
	var tests = map[string]int64{
		"SUCCESS": Success,
		"FAILED":  Failed,
	}

	msgFormat := `{
		"task_id": "job_config_id_A:job_run_id_A:A",
		"status": "%s"
	}`

	for jsonStatusString, taskStatus := range tests {
		want := Task{
			JobConfigId: jobConfigId,
			JobRunId:    jobRunId,
			TaskId:      "A",
			Status:      taskStatus,
		}
		msg := []byte(fmt.Sprintf(msgFormat, jsonStatusString))

		taskUpdate, err := TaskCompletionMessageJsonToTaskUpdate(msg)
		task := taskUpdate.Task
		if err != nil {
			t.Errorf("error converting task msg JSON to Task, error: %v.", err)
		}
		if !reflect.DeepEqual(*task, want) {
			t.Errorf("result: %v, wanted: %v", *task, want)
		}
	}
}

func TestTaskCompletionMessageJsonToTaskUpdateFailureMsg(t *testing.T) {
	want := Task{
		JobConfigId:    jobConfigId,
		JobRunId:       jobRunId,
		TaskId:         "A",
		Status:         Failed,
		FailureType:    proto.TaskFailureType_UNUSED,
		FailureMessage: "Failure",
	}
	msg := []byte(`{
		"task_id": "job_config_id_A:job_run_id_A:A",
		"status": "FAILED",
		"failure_message": "Failure"
	}`)
	taskUpdate, err := TaskCompletionMessageJsonToTaskUpdate(msg)
	task := taskUpdate.Task
	if err != nil {
		t.Errorf("error converting task msg JSON to Task, error: %v.", err)
	}
	if !reflect.DeepEqual(*task, want) {
		t.Errorf("result: %v, wanted: %v", *task, want)
	}
}

func TestTaskCompletionMessageJsonToTaskUpdateFailureMsgFailParse(t *testing.T) {
	msg := []byte("Invalid JSON.")
	if _, err := TaskCompletionMessageJsonToTaskUpdate(msg); err == nil {
		t.Errorf("invalid JSON: %s. Method should not be able to parse", string(msg))
	}
}

func TestTaskCompletionMessageJsonToTaskUpdateMissingId(t *testing.T) {
	msg := []byte(`{
		"status": "FAILED",
		"failure_message": "Failure"
	}`)
	if _, err := TaskCompletionMessageJsonToTaskUpdate(msg); err == nil {
		t.Errorf(
			"task id does not exist in the message: %s, expected error.", string(msg))
	}
}

func TestTaskCompletionMessageJsonToTaskUpdateMissingStatus(t *testing.T) {
	msg := []byte(`{
		"task_id": "A"
	}`)
	if _, err := TaskCompletionMessageJsonToTaskUpdate(msg); err == nil {
		t.Errorf(
			"status does not exist in the message: %s, expected error.", string(msg))
	}
}

func TestTaskCompletionMessageJsonToTaskUpdateIncorrectStatus(t *testing.T) {
	msg := []byte(`{
		"task_id": "A",
		"status": "Incorrect status"
	}`)
	if _, err := TaskCompletionMessageJsonToTaskUpdate(msg); err == nil {
		t.Errorf(
			"incorrect status in the message: %s, expected error.", string(msg))
	}
}

func TestConstructPubSubTaskMsgSuccess(t *testing.T) {
	task := &Task{
		JobConfigId: "job_config",
		JobRunId:    "job_run",
		TaskId:      "task_id",
		TaskSpec:    "{\"foo\": \"bar\", \"nest1\": {\"nest2\": \"nested_val\"}}",
	}

	want := map[string]interface{}{
		"task_id": "job_config:job_run:task_id",
		"task_params": map[string]interface{}{
			"foo": "bar",
			"nest1": map[string]interface{}{
				"nest2": "nested_val",
			},
		},
	}

	// Construct the message
	msg, err := constructPubSubTaskMsg(task)
	if err != nil {
		t.Errorf("Wanted no error, but got %v", err)
	}

	// Unmarshall it into an arbitrary map and compare
	result := make(map[string]interface{})
	err = json.Unmarshal(msg, &result)
	if err != nil {
		t.Errorf("Wanted no error while unmarshalling expected result, but got %v", err)
	}

	if !reflect.DeepEqual(want, result) {
		t.Errorf("Wanted %v but got %v", want, result)
	}
}

func TestConstructPubSubTaskMsgFailure(t *testing.T) {
	task := &Task{
		JobConfigId: "job_config",
		JobRunId:    "job_run",
		TaskId:      "task_id",
		TaskSpec:    "{\"foo\": \"barBROOOOOOKEN!",
	}

	// Construct the message
	_, err := constructPubSubTaskMsg(task)
	if err == nil {
		t.Error("Expecting error, got no error.")
	}
}
