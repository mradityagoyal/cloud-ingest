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
	"fmt"
	"reflect"
	"strconv"
	"testing"

	"github.com/golang/mock/gomock"
)

var (
	jobConfigId string = "job_config_id_A"
	jobRunId    string = "job_run_id_A"
)

func TestListProgressMessageHandlerTaskDoesNotExist(t *testing.T) {
	store := FakeStore{}
	handler := ListProgressMessageHandler{
		Store: &store,
	}

	task := &Task{Status: Success}
	_, err := handler.HandleMessage(nil /* jobSpec */, TaskWithLog{task, ""})
	if err == nil {
		t.Errorf("error is nil, expected error: %v.", errTaskNotFound)
	}
	if err != errTaskNotFound {
		t.Errorf("expected error: %v, found: %v.", errTaskNotFound, err)
	}
}

func TestListProgressMessageHandlerInvalidTaskSpec(t *testing.T) {
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
	handler := ListProgressMessageHandler{
		Store: &store,
	}

	// Reset the task spec
	uploadGCSTask.TaskSpec = ""
	_, err := handler.HandleMessage(nil /* jobSpec */, TaskWithLog{uploadGCSTask, ""})
	if err == nil {
		t.Errorf("error is nil, expected JSON decode error.")
	}
}

func TestListProgressMessageHandlerFailReadingListResult(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	mockListReader := NewMockListingResultReader(mockCtrl)
	errorMsg := "Failed reading listing result."
	mockListReader.EXPECT().ReadListResult("bucket", "object").Return(nil, errors.New(errorMsg))

	listTask := &Task{
		JobConfigId: jobConfigId,
		JobRunId:    jobRunId,
		TaskId:      "A",
		TaskType:    listTaskType,
		TaskSpec: `{
			"task_id": "A",
			"dst_list_result_bucket": "bucket",
			"dst_list_result_object": "object",
			"src_directory": "dir"
		}`,
		Status: Success,
	}
	store := FakeStore{
		tasks: map[string]*Task{
			listTask.getTaskFullId(): listTask,
		},
	}
	handler := ListProgressMessageHandler{
		Store:               &store,
		ListingResultReader: mockListReader,
	}

	_, err := handler.HandleMessage(nil /* jobSpec */, TaskWithLog{listTask, ""})
	if err == nil {
		t.Errorf("error is nil, expected error: %s.", errorMsg)
	}
	if err.Error() != errorMsg {
		t.Errorf("expected error: %s, found: %s.", errorMsg, err.Error())
	}
}

func TestListProgressMessageHandlerEmptyChannel(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	mockListReader := NewMockListingResultReader(mockCtrl)
	c := make(chan string)
	close(c)
	mockListReader.EXPECT().ReadListResult("bucket1", "object").Return(c, nil)

	listTask := &Task{
		JobConfigId: jobConfigId,
		JobRunId:    jobRunId,
		TaskId:      "task_id_A",
		TaskType:    listTaskType,
		TaskSpec: `{
			"dst_list_result_bucket": "bucket1",
			"dst_list_result_object": "object",
			"src_directory": "dir"
		}`,
		Status: Success,
	}
	store := FakeStore{
		tasks: map[string]*Task{
			listTask.getTaskFullId(): listTask,
		},
	}
	handler := ListProgressMessageHandler{
		Store:               &store,
		ListingResultReader: mockListReader,
	}

	jobSpec := &JobSpec{
		GCSBucket: "bucket2",
	}

	_, err := handler.HandleMessage(jobSpec, TaskWithLog{listTask, ""})
	errorMsg := fmt.Sprintf(noTaskIdInListOutput, "job_config_id_A:job_run_id_A:task_id_A", "")
	if err == nil {
		t.Errorf("error is nil, expected error: %s.", errorMsg)
	}
	if err.Error() != errorMsg {
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
	mockListReader.EXPECT().ReadListResult("bucket1", "object").Return(c, nil)

	listTask := &Task{
		JobConfigId: jobConfigId,
		JobRunId:    jobRunId,
		TaskId:      "task_id_A",
		TaskType:    listTaskType,
		TaskSpec: `{
			"dst_list_result_bucket": "bucket1",
			"dst_list_result_object": "object",
			"src_directory": "dir"
		}`,
		Status: Success,
	}
	store := FakeStore{
		tasks: map[string]*Task{
			listTask.getTaskFullId(): listTask,
		},
	}
	handler := ListProgressMessageHandler{
		Store:               &store,
		ListingResultReader: mockListReader,
	}

	jobSpec := &JobSpec{
		GCSBucket: "bucket2",
	}

	_, err := handler.HandleMessage(jobSpec, TaskWithLog{listTask, ""})
	errorMsg := fmt.Sprintf(noTaskIdInListOutput, "job_config_id_A:job_run_id_A:task_id_A", "task_id_B")
	if err == nil {
		t.Errorf("error is nil, expected error: %s.", errorMsg)
	}
	if err.Error() != errorMsg {
		t.Errorf("expected error: %s, found: %s.", errorMsg, err.Error())
	}
}

func TestListProgressMessageHandlerSuccess(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	mockListReader := NewMockListingResultReader(mockCtrl)
	c := make(chan string)
	go func() {
		defer close(c)
		c <- "job_config_id_A:job_run_id_A:task_id_A"
		c <- "dir/file0"
		c <- "dir/file1"
	}()
	mockListReader.EXPECT().ReadListResult("bucket1", "object").Return(c, nil)

	listTask := &Task{
		JobConfigId: jobConfigId,
		JobRunId:    jobRunId,
		TaskId:      "task_id_A",
		TaskType:    listTaskType,
		TaskSpec: `{
			"dst_list_result_bucket": "bucket1",
			"dst_list_result_object": "object",
			"src_directory": "dir"
		}`,
		Status: Success,
	}
	store := FakeStore{
		tasks: map[string]*Task{
			listTask.getTaskFullId(): listTask,
		},
	}
	handler := ListProgressMessageHandler{
		Store:               &store,
		ListingResultReader: mockListReader,
	}

	jobSpec := &JobSpec{
		GCSBucket: "bucket2",
	}
	newTasks, err := handler.HandleMessage(jobSpec, TaskWithLog{listTask, ""})
	if err != nil {
		t.Errorf("expecting success, found error: %v.", err)
	}
	if len(newTasks) != 2 {
		t.Errorf("expecting 2 tasks when handling a list message, found %d.", len(newTasks))
	}

	for i := 0; i < 2; i++ {
		expectedNewTask := Task{
			JobConfigId: jobConfigId,
			JobRunId:    jobRunId,
			TaskType:    uploadGCSTaskType,
			TaskId:      GetUploadGCSTaskId("dir/file" + strconv.Itoa(i)),
		}
		expectedNewTaskSpec := fmt.Sprintf(`{
				"src_file": "dir/file%d",
				"dst_bucket": "bucket2",
				"dst_object": "file%d"
			}`, i, i)

		var newTask *Task
		for _, t := range newTasks {
			if expectedNewTask.getTaskFullId() == t.getTaskFullId() {
				newTask = t
			}
		}

		if newTask == nil {
			t.Errorf("task %v should exist in the returned new tasks", expectedNewTask)
		}
		if !AreEqualJSON(expectedNewTaskSpec, newTask.TaskSpec) {
			t.Errorf("expected task spec: %s, found: %s", expectedNewTaskSpec, newTask.TaskSpec)
		}
		// Clear the task spec to compare the remaining of the struct.
		newTask.TaskSpec = ""
		if !reflect.DeepEqual(expectedNewTask, *newTask) {
			t.Errorf("expected task: %v, found: %v.", expectedNewTask, *newTask)
		}
	}
}
