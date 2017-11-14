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
	"github.com/GoogleCloudPlatform/cloud-ingest/dcp/proto"
	"reflect"
	"strings"
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
Task Spec tests
*******************************************************************************/
func TestNewListTaskSpecFromMap(t *testing.T) {
	var tests = []struct {
		bucket interface{}
		object interface{}
		srcDir interface{}
		genNum interface{}
	}{
		{"bucket", "object", "srcdir", int64(1)},
		{"bucket", "object", "srcdir", int(1)},
		{"bucket", "object", "srcdir", json.Number("1")},
		{nil, "object", "srcdir", int(1)},
		{"bucket", nil, "srcdir", int(1)},
		{"bucket", "object", nil, int(1)},
		{"bucket", "object", "srcdir", nil},
	}

	for _, tc := range tests {
		params := make(map[string]interface{})
		if tc.bucket != nil {
			params["dst_list_result_bucket"] = tc.bucket
		}
		if tc.object != nil {
			params["dst_list_result_object"] = tc.object
		}
		if tc.srcDir != nil {
			params["src_directory"] = tc.srcDir
		}
		if tc.genNum != nil {
			params["expected_generation_num"] = tc.genNum
		}

		result, err := NewListTaskSpecFromMap(params)

		if tc.bucket != nil && tc.object != nil && tc.srcDir != nil && tc.genNum != nil {
			// All values populated, should be working result (always same values).
			expected := &ListTaskSpec{
				DstListResultBucket:   "bucket",
				DstListResultObject:   "object",
				SrcDirectory:          "srcdir",
				ExpectedGenerationNum: 1,
			}

			if err != nil {
				t.Errorf("expected nil error, got: %v", err)
			}

			DeepEqualCompare("listTaskSpec construction from map", expected, result, t)
		} else {
			// Any missing parameter should result in error.
			if err == nil {
				t.Error("wanted: missing params, but got nil error")
			} else if !strings.Contains(err.Error(), "missing params") {
				t.Errorf("wanted: missing params, but got: %v", err)
			}
		}
	}
}

func TestNewUploadGCSTaskSpecFromMap(t *testing.T) {
	var tests = []struct {
		srcFile interface{}
		bucket  interface{}
		object  interface{}
		genNum  interface{}
	}{
		{"srcfile", "bucket", "object", int64(1)},
		{"srcfile", "bucket", "object", int(1)},
		{"srcfile", "bucket", "object", json.Number("1")},
		{nil, "bucket", "object", int64(1)},
		{"srcfile", nil, "object", int64(1)},
		{"srcfile", "bucket", nil, int64(1)},
		{"srcfile", "bucket", "object", nil},
	}

	for _, tc := range tests {
		params := make(map[string]interface{})
		if tc.srcFile != nil {
			params["src_file"] = tc.srcFile
		}
		if tc.bucket != nil {
			params["dst_bucket"] = tc.bucket
		}
		if tc.object != nil {
			params["dst_object"] = tc.object
		}
		if tc.genNum != nil {
			params["expected_generation_num"] = tc.genNum
		}

		result, err := NewUploadGCSTaskSpecFromMap(params)

		if tc.srcFile != nil && tc.bucket != nil && tc.object != nil && tc.genNum != nil {
			// All values populated, should be working result (always same values).
			expected := &UploadGCSTaskSpec{
				SrcFile:               "srcfile",
				DstBucket:             "bucket",
				DstObject:             "object",
				ExpectedGenerationNum: 1,
			}

			if err != nil {
				t.Errorf("expected nil error, got: %v", err)
			}

			DeepEqualCompare("uploadGCSTaskSpec construction from map", expected, result, t)
		} else {
			// Any missing parameter should result in error.
			if err == nil {
				t.Error("wanted: missing params, but got nil error")
			} else if !strings.Contains(err.Error(), "missing params") {
				t.Errorf("wanted: missing params, but got: %v", err)
			}
		}
	}
}

/*******************************************************************************
Task status transition tests
*******************************************************************************/
func TestCanChangeTaskStatus(t *testing.T) {
	// Capturing various cases in one test, since it's simple and additive.
	if canChangeTaskStatus(Success, Unqueued) {
		t.Error("successful jobs cannot change state")
	}
	if canChangeTaskStatus(Success, Failed) {
		t.Error("successful jobs cannot change state")
	}
	if canChangeTaskStatus(Failed, Queued) {
		t.Error("job state cannot flow backwards")
	}
	if !canChangeTaskStatus(Failed, Unqueued) {
		t.Error("unsuccessful jobs should be reissuable")
	}

	// Go through standard forward flow
	if !canChangeTaskStatus(Unqueued, Queued) {
		t.Error("standard forwards job flow should work")
	}
	if !canChangeTaskStatus(Queued, Success) {
		t.Error("standard forwards job flow should work")
	}
	if !canChangeTaskStatus(Queued, Failed) {
		t.Error("standard forwards job flow should work")
	}
	if !canChangeTaskStatus(Failed, Success) {
		t.Error("standard forwards job flow should work")
	}

}

/*******************************************************************************
Task Methods Tests
*******************************************************************************/
func TestTaskCompletionMessageFromJsonFailureMessage(t *testing.T) {
	msg := []byte(`{
		"task_id": "job_config_id_A:job_run_id_A:A",
		"status": "FAILED",
		"failure_reason": 5,
		"failure_message": "Failure",
		"log_entry": {
			"logkey1": "logval1",
			"lognum": 42
		},
		"task_params": {
			"paramkey1": "paramval1",
			"paramnum": 42
		}
	}`)

	taskCompletionMessage, err := TaskCompletionMessageFromJson(msg)
	if err != nil {
		t.Errorf("Error converting completion msg JSON to TaskCompletionMessage: %v", err)
	}

	want := TaskCompletionMessage{
		FullTaskId:     "job_config_id_A:job_run_id_A:A",
		Status:         "FAILED",
		FailureType:    proto.TaskFailureType_FILE_NOT_FOUND_FAILURE,
		FailureMessage: "Failure",
		LogEntry: map[string]interface{}{
			"logkey1": "logval1",
			"lognum":  json.Number("42"),
		},
		TaskParams: map[string]interface{}{
			"paramkey1": "paramval1",
			"paramnum":  json.Number("42"),
		},
	}

	if !reflect.DeepEqual(*taskCompletionMessage, want) {
		t.Errorf("result: %v, wanted: %v", *taskCompletionMessage, want)
	}
}

func TestTaskCompletionMessageFromJsonSuccessMessage(t *testing.T) {
	msg := []byte(`{
		"task_id": "job_config_id_A:job_run_id_A:A",
		"status": "SUCCESS",
		"log_entry": {
			"logkey1": "logval1"
		},
		"task_params": {
			"paramkey1": "paramval1"
		}
	}`)

	taskCompletionMessage, err := TaskCompletionMessageFromJson(msg)
	if err != nil {
		t.Errorf("error converting completion msg JSON to TaskCompletionMessage: %v", err)
	}

	want := TaskCompletionMessage{
		FullTaskId: "job_config_id_A:job_run_id_A:A",
		Status:     "SUCCESS",
		LogEntry: map[string]interface{}{
			"logkey1": "logval1",
		},
		TaskParams: map[string]interface{}{
			"paramkey1": "paramval1",
		},
	}

	if !reflect.DeepEqual(*taskCompletionMessage, want) {
		t.Errorf("result: %v, wanted: %v", *taskCompletionMessage, want)
	}
}

func TestTaskCompletionMessageToTaskUpdateNilMessage(t *testing.T) {
	_, err := TaskCompletionMessageToTaskUpdate(nil)
	if err == nil {
		t.Error("nil input should have resulted in an error.")
	}
}

func TestTaskCompletionMessageToTaskUpdateBadTask(t *testing.T) {
	taskCompletionMessage := TaskCompletionMessage{
		FullTaskId: "invalid",
	}

	_, err := TaskCompletionMessageToTaskUpdate(&taskCompletionMessage)

	if err == nil {
		t.Error("invalid task should have resulted in an error.")
	}
}

func TestTaskCompletionMessageToTaskUpdateSuccessMessage(t *testing.T) {
	taskCompletionMessage := TaskCompletionMessage{
		FullTaskId: "job_config_id_A:job_run_id_A:A",
		Status:     "SUCCESS",
		LogEntry:   map[string]interface{}{"logkey1": "logval1", "logkey2": "logval2"},
		TaskParams: map[string]interface{}{"paramkey1": "paramval1", "paramkey2": "paramval2"},
	}

	taskUpdate, err := TaskCompletionMessageToTaskUpdate(&taskCompletionMessage)

	if err != nil {
		t.Errorf("error converting completion msg JSON to TaskCompletionMessage: %v", err)
	}

	want := TaskUpdate{
		Task: &Task{
			JobConfigId: "job_config_id_A",
			JobRunId:    "job_run_id_A",
			TaskId:      "A",
			Status:      Success,
		},
		LogEntry: &LogEntry{
			data: map[string]interface{}{"logkey1": "logval1", "logkey2": "logval2"},
		},
		OriginalTaskParams: TaskParams{"paramkey1": "paramval1", "paramkey2": "paramval2"},
	}

	if taskUpdate == nil {
		t.Errorf("taskUpdate is nil, but should be %v.", want)
	} else if !reflect.DeepEqual(*taskUpdate, want) {
		t.Errorf("result: %v, wanted: %v", *taskUpdate, want)
	}
}

func TestTaskCompletionMessageToTaskUpdateFailureMessage(t *testing.T) {
	taskCompletionMessage := TaskCompletionMessage{
		FullTaskId:     "job_config_id_A:job_run_id_A:A",
		Status:         "FAILED",
		FailureType:    proto.TaskFailureType_FILE_NOT_FOUND_FAILURE,
		FailureMessage: "Failure",
		LogEntry:       map[string]interface{}{"logkey1": "logval1", "logkey2": "logval2"},
		TaskParams:     map[string]interface{}{"paramkey1": "paramval1", "paramkey2": "paramval2"},
	}

	taskUpdate, err := TaskCompletionMessageToTaskUpdate(&taskCompletionMessage)

	if err != nil {
		t.Errorf("error converting completion msg JSON to TaskCompletionMessage: %v", err)
	}

	want := TaskUpdate{
		Task: &Task{
			JobConfigId:    "job_config_id_A",
			JobRunId:       "job_run_id_A",
			TaskId:         "A",
			Status:         Failed,
			FailureType:    proto.TaskFailureType_FILE_NOT_FOUND_FAILURE,
			FailureMessage: "Failure",
		},
		LogEntry: &LogEntry{
			data: map[string]interface{}{"logkey1": "logval1", "logkey2": "logval2"},
		},
		OriginalTaskParams: TaskParams{"paramkey1": "paramval1", "paramkey2": "paramval2"},
	}

	if taskUpdate == nil {
		t.Errorf("taskUpdate is nil, but should be %v.", want)
	} else if !reflect.DeepEqual(*taskUpdate, want) {
		t.Errorf("result: %v, wanted: %v", *taskUpdate, want)
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
		t.Errorf("wanted no error, but got %v", err)
	}

	// Unmarshall it into an arbitrary map and compare
	result := make(map[string]interface{})
	err = json.Unmarshal(msg, &result)
	if err != nil {
		t.Errorf("wanted no error while unmarshalling expected result, but got %v", err)
	}

	if !reflect.DeepEqual(want, result) {
		t.Errorf("wanted %v but got %v", want, result)
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
		t.Error("expecting error, got no error.")
	}
}
