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
	"fmt"
	"reflect"
	"testing"
)

func TestTaskCompletionMessageJsonToTask(t *testing.T) {
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

		task, err := TaskCompletionMessageJsonToTask(msg)
		if err != nil {
			t.Errorf("error converting task msg JSON to Task, error: %v.", err)
		}
		if !reflect.DeepEqual(*task, want) {
			t.Errorf("result: %v, wanted: %v", *task, want)
		}
	}
}

func TestTaskCompletionMessageJsonToTaskFailureMsg(t *testing.T) {
	want := Task{
		JobConfigId:    jobConfigId,
		JobRunId:       jobRunId,
		TaskId:         "A",
		Status:         Failed,
		FailureMessage: "Failure",
	}
	msg := []byte(`{
		"task_id": "job_config_id_A:job_run_id_A:A",
		"status": "FAILED",
		"failure_message": "Failure"
	}`)
	task, err := TaskCompletionMessageJsonToTask(msg)
	if err != nil {
		t.Errorf("error converting task msg JSON to Task, error: %v.", err)
	}
	if !reflect.DeepEqual(*task, want) {
		t.Errorf("result: %v, wanted: %v", *task, want)
	}
}

func TestTaskCompletionMessageJsonToTaskFailureMsgFailParse(t *testing.T) {
	msg := []byte("Invalid JSON.")
	if _, err := TaskCompletionMessageJsonToTask(msg); err == nil {
		t.Errorf("invalid JSON: %s. Method should not be able to parse", string(msg))
	}
}

func TestTaskCompletionMessageJsonToTaskMissingId(t *testing.T) {
	msg := []byte(`{
		"status": "FAILED",
		"failure_message": "Failure"
	}`)
	if _, err := TaskCompletionMessageJsonToTask(msg); err == nil {
		t.Errorf(
			"task id does not exist in the message: %s, expected error.", string(msg))
	}
}

func TestTaskCompletionMessageJsonToTaskMissingStatus(t *testing.T) {
	msg := []byte(`{
		"task_id": "A"
	}`)
	if _, err := TaskCompletionMessageJsonToTask(msg); err == nil {
		t.Errorf(
			"status does not exist in the message: %s, expected error.", string(msg))
	}
}

func TestTaskCompletionMessageJsonToTaskIncorrectStatus(t *testing.T) {
	msg := []byte(`{
		"task_id": "A",
		"status": "Incorrect status"
	}`)
	if _, err := TaskCompletionMessageJsonToTask(msg); err == nil {
		t.Errorf(
			"incorrect status in the message: %s, expected error.", string(msg))
	}
}
