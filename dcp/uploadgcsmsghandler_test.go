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
	"reflect"
	"testing"
)

func TestUploadGCSProgressMessageHandlerFailedTask(t *testing.T) {
	store := FakeStore{
		tasks: make(map[string]*Task),
	}
	handler := UploadGCSProgressMessageHandler{
		Store: &store,
	}
	task := &Task{Status: Failed, TaskId: "A"}
	if err := handler.HandleMessage(nil /* jobSpec */, task); err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if store.tasks == nil {
		t.Errorf("expected store to contain a single task, found 0 tasks in the store.")
	}

	storeTask, exists := store.tasks[task.getTaskFullId()]
	if !exists {
		t.Errorf("task %v should exist in the store", task)
	}
	if storeTask.Status != Failed {
		t.Errorf("expected store task to have status Failed(2), found %d",
			storeTask.Status)
	}
}

func TestUploadGCSProgressMessageHandlerTaskDoesNotExist(t *testing.T) {
	store := FakeStore{
		tasks: make(map[string]*Task),
	}
	handler := UploadGCSProgressMessageHandler{
		Store: &store,
	}

	task := &Task{Status: Success}
	err := handler.HandleMessage(nil /* jobSpec */, task)
	if err == nil {
		t.Errorf("error is nil, expected error: %v.", errTaskNotFound)
	}
	if err != errTaskNotFound {
		t.Errorf("expected error: %v, found: %v.", errTaskNotFound, err)
	}
}

func TestUploadGCSProgressMessageHandlerInvalidTaskSpec(t *testing.T) {
	uploadGCSTask := &Task{
		JobConfigId: jobConfigId,
		JobRunId:    jobRunId,
		TaskId:      "A",
		TaskType:    uploadGCSTaskType,
		TaskSpec:    "Invalid JSON Task Spec",
		Status:      Success,
	}
	store := FakeStore{
		tasks: map[string]*Task{
			uploadGCSTask.getTaskFullId(): uploadGCSTask,
		},
	}
	handler := UploadGCSProgressMessageHandler{
		Store: &store,
	}

	// Reset the task spec
	uploadGCSTask.TaskSpec = ""
	err := handler.HandleMessage(nil /* jobSpec */, uploadGCSTask)
	if err == nil {
		t.Errorf("error is nil, expected JSON decode error.")
	}
}

func TestUploadGCSProgressMessageHandlerSuccess(t *testing.T) {
	uploadGCSTask := &Task{
		JobConfigId: jobConfigId,
		JobRunId:    jobRunId,
		TaskType:    uploadGCSTaskType,
		TaskId:      "A",
		TaskSpec: `{
			"task_id": "A",
			"src_file": "file",
			"dst_bucket": "bucket",
			"dst_object": "object"
		}`,
		Status: Success,
	}
	store := FakeStore{
		tasks: map[string]*Task{
			uploadGCSTask.getTaskFullId(): uploadGCSTask,
		},
	}
	handler := UploadGCSProgressMessageHandler{
		Store: &store,
	}

	jobSpec := &JobSpec{
		BQDataset: "dataset",
		BQTable:   "table",
	}
	if err := handler.HandleMessage(jobSpec, uploadGCSTask); err != nil {
		t.Errorf("expecting success, found error: %v.", err)
	}

	if len(store.tasks) != 2 {
		t.Errorf("expecting 2 tasks in the the store, found %d.", len(store.tasks))
	}
	expectedNewTask := Task{
		JobConfigId: jobConfigId,
		JobRunId:    jobRunId,
		TaskType:    loadBQTaskType,
		TaskId:      GetLoadBQTaskId("object"),
	}
	expectedNewTaskSpec :=
		`{
			"src_gcs_bucket": "bucket",
			"src_gcs_object": "object",
			"dst_bq_dataset": "dataset",
			"dst_bq_table": "table"
		}`
	insertedTask, ok := store.tasks[expectedNewTask.getTaskFullId()]
	if !ok {
		t.Errorf("task %v should exist in the store", expectedNewTask)
	}
	if !AreEqualJSON(expectedNewTaskSpec, insertedTask.TaskSpec) {
		t.Errorf("expected task spec: %s, found: %s", expectedNewTask, insertedTask.TaskSpec)
	}
	// Clear the task spec to compare the remaining of the struct.
	insertedTask.TaskSpec = ""
	if !reflect.DeepEqual(expectedNewTask, *insertedTask) {
		t.Errorf("expected task: %v, found: %v.", expectedNewTask, *insertedTask)
	}
}
