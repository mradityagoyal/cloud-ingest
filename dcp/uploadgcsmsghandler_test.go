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
	"io/ioutil"
	"log"
	"os"
	"testing"
)

func copySuccessCompletionMessage() *TaskCompletionMessage {
	return &TaskCompletionMessage{
		FullTaskId: jobConfigId + ":" + jobRunId + ":" + taskId,
		Status:     "SUCCESS",
		TaskParams: map[string]interface{}{
			"src_file":   "file",
			"dst_bucket": "bucket",
			"dst_object": "object",
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

func TestUploadGCSProgressMessageHandlerSuccess(t *testing.T) {
	uploadGCSTask := &Task{
		JobConfigId: jobConfigId,
		JobRunId:    jobRunId,
		TaskType:    uploadGCSTaskType,
		TaskId:      taskId,
		TaskSpec: `{
			"task_id": task_id_A,
			"src_file": "file",
			"dst_bucket": "bucket",
			"dst_object": "object"
		}`,
		Status: Failed,
	}
	handler := UploadGCSProgressMessageHandler{}

	jobSpec := &JobSpec{}
	taskUpdate, err := handler.HandleMessage(jobSpec, copySuccessCompletionMessage())

	if err != nil {
		t.Errorf("expecting success, found error: %v.", err)
	}

	if taskUpdate.NewTasks != nil && len(taskUpdate.NewTasks) != 0 {
		t.Errorf("new tasks should be an empty array, new tasks: %v", taskUpdate.NewTasks)
	}

	// Update task to re-use in comparison.
	uploadGCSTask.TaskSpec = ""
	uploadGCSTask.Status = Success

	expectedTaskUpdate := &TaskUpdate{
		Task:     uploadGCSTask,
		LogEntry: NewLogEntry(map[string]interface{}{"logkey": "logval"}),
		OriginalTaskParams: (*TaskParams)(&map[string]interface{}{
			"src_file":   "file",
			"dst_bucket": "bucket",
			"dst_object": "object",
		}),
	}

	DeepEqualCompare("TaskUpdate", expectedTaskUpdate, taskUpdate, t)

	// Check pieces one at a time, for convenient visualization.
	DeepEqualCompare("TaskUpdate.Task", expectedTaskUpdate.Task, taskUpdate.Task, t)
	DeepEqualCompare("TaskUpdate.LogEntry", expectedTaskUpdate.LogEntry, taskUpdate.LogEntry, t)
	DeepEqualCompare("TaskUpdate.OriginalTaskParams",
		expectedTaskUpdate.OriginalTaskParams, taskUpdate.OriginalTaskParams, t)
}
