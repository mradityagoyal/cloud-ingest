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
	jobSpec := &JobSpec{BQDataset: "dummy", BQTable: "dummy"}
	taskUpdate := &TaskUpdate{
		Task: &Task{Status: Failed, TaskId: "A"},
	}

	err := handler.HandleMessage(jobSpec, taskUpdate)

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if taskUpdate.NewTasks != nil && len(taskUpdate.NewTasks) != 0 {
		t.Errorf("new tasks should be an empty array, new tasks: %v", taskUpdate.NewTasks)
	}
}

func TestUploadGCSProgressMessageHandlerTaskDoesNotExist(t *testing.T) {
	store := FakeStore{
		tasks: make(map[string]*Task),
	}
	handler := UploadGCSProgressMessageHandler{
		Store: &store,
	}

	taskUpdate := &TaskUpdate{
		Task: &Task{Status: Success},
	}
	jobSpec := &JobSpec{BQDataset: "dummy", BQTable: "dummy"}
	err := handler.HandleMessage(jobSpec, taskUpdate)
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

	jobSpec := &JobSpec{BQDataset: "dummy", BQTable: "dummy"}
	taskUpdate := &TaskUpdate{
		Task: uploadGCSTask,
	}
	err := handler.HandleMessage(jobSpec, taskUpdate)
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
	taskUpdate := &TaskUpdate{
		Task: uploadGCSTask,
	}
	err := handler.HandleMessage(jobSpec, taskUpdate)
	if err != nil {
		t.Errorf("expecting success, found error: %v.", err)
	}
	newTasks := taskUpdate.NewTasks

	if len(newTasks) != 1 {
		t.Errorf("expecting 1 task when handling an upload GCS message, found %d.",
			len(newTasks))
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

	if !AreEqualJSON(expectedNewTaskSpec, newTasks[0].TaskSpec) {
		t.Errorf("expected task spec: %s, found: %s", expectedNewTask, newTasks[0].TaskSpec)
	}

	// Clear the task spec to compare the remaining of the struct.
	newTasks[0].TaskSpec = ""
	if !reflect.DeepEqual(expectedNewTask, *newTasks[0]) {
		t.Errorf("expected task: %v, found: %v.", expectedNewTask, *newTasks[0])
	}
}

func TestUploadGCSProgressMessageHandlerNoBQTask(t *testing.T) {
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
		Status: Failed,
	}
	store := FakeStore{
		tasks: map[string]*Task{
			uploadGCSTask.getTaskFullId(): uploadGCSTask,
		},
	}
	handler := UploadGCSProgressMessageHandler{
		Store: &store,
	}

	jobSpec := &JobSpec{}
	// Turn the task to success
	uploadGCSTask.Status = Success

	taskUpdate := &TaskUpdate{
		Task: uploadGCSTask,
	}
	err := handler.HandleMessage(jobSpec, taskUpdate)

	if err != nil {
		t.Errorf("expecting success, found error: %v.", err)
	}

	if len(store.tasks) != 1 {
		t.Errorf("expecting 1 tasks in the the store, found %d.", len(store.tasks))
	}

	if taskUpdate.NewTasks != nil && len(taskUpdate.NewTasks) != 0 {
		t.Errorf("new tasks should be an empty array, new tasks: %v", taskUpdate.NewTasks)
	}
}
