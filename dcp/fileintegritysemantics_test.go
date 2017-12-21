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
	"reflect"
	"strconv"
	"testing"

	"github.com/GoogleCloudPlatform/cloud-ingest/dcp/proto"
	"github.com/GoogleCloudPlatform/cloud-ingest/helpers"
)

const noGenNumTaskSpec string = `{"irrelevant": "foobar"}`

func getTestingTaskSpec(expectedGenerationNum int) string {
	return fmt.Sprintf(`{"irrelevant": "foobar", "expected_generation_num": %d}`,
		expectedGenerationNum)
}

func getTestingTaskUpdate(expectedGenerationNum int, status int64, failureType proto.TaskFailureType_Type, taskSpec string) *TaskUpdate {
	taskUpdate := &TaskUpdate{
		Task: &Task{
			TaskRRStruct: *NewTaskRRStruct("prjct_id", "job_cfg_id", "job_run_id", "task_id"),
			Status:       status,
			TaskSpec:     taskSpec,
		},

		// This is just to make sure it doesn't get altered.
		LogEntry: LogEntry{"foo": "bar"},

		OriginalTaskParams: TaskParams{},

		// We'll always have something here, and then we can check if it stuck around.
		NewTasks: []*Task{{}, {}},
	}

	if expectedGenerationNum >= 0 {
		taskUpdate.OriginalTaskParams["expected_generation_num"] = json.Number(strconv.Itoa(expectedGenerationNum))
	}

	if status == Failed {
		taskUpdate.Task.FailureType = failureType
		taskUpdate.Task.FailureMessage = "task failed"
	}

	return taskUpdate
}

func DeepEqualCompare(msgPrefix string, want, got interface{}, t *testing.T) {
	if !reflect.DeepEqual(want, got) {
		t.Errorf("%s: Wanted %+v; got %+v", msgPrefix, want, got)
	}
}

func compareTaskUpdates(want, got *TaskUpdate, t *testing.T) {
	// Deal with json first
	if !helpers.AreEqualJSON(want.Task.TaskSpec, got.Task.TaskSpec) {
		t.Errorf("taskSpec: wanted %v but got %v", want.Task.TaskSpec, got.Task.TaskSpec)
	}
	want.Task.TaskSpec = ""
	got.Task.TaskSpec = ""
	DeepEqualCompare("taskUpdate", want, got, t)
	DeepEqualCompare("task", want.Task, got.Task, t)
	DeepEqualCompare("original params", want.OriginalTaskParams, got.OriginalTaskParams, t)
	DeepEqualCompare("new tasks", want.NewTasks, got.NewTasks, t)
	DeepEqualCompare("log entry", want.LogEntry, got.LogEntry, t)
}

func TestFileIntegritySemantics_FailedReissue(t *testing.T) {
	// Task failed in a way that requires reissuing (e.g. MD5)
	taskUpdate := getTestingTaskUpdate(
		0, Failed, proto.TaskFailureType_MD5_MISMATCH_FAILURE, getTestingTaskSpec(0))

	var semantics TaskTransactionalSemantics = &FileIntegritySemantics{1000}
	err := semantics.Apply(taskUpdate)

	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}

	expectedTaskUpdate := getTestingTaskUpdate(0, Unqueued, 0, getTestingTaskSpec(1000))
	expectedTaskUpdate.NewTasks = nil

	compareTaskUpdates(expectedTaskUpdate, taskUpdate, t)
}

func TestFileIntegritySemantics_FailedNoReissue(t *testing.T) {
	// Task failed and we don't want to reissue (e.g. permission denied)
	taskUpdate := getTestingTaskUpdate(
		0, Failed, proto.TaskFailureType_PERMISSION_FAILURE, getTestingTaskSpec(123))

	var semantics TaskTransactionalSemantics = &FileIntegritySemantics{1000}
	err := semantics.Apply(taskUpdate)

	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}

	expectedTaskUpdate := getTestingTaskUpdate(
		0, Failed, proto.TaskFailureType_PERMISSION_FAILURE, getTestingTaskSpec(123))
	expectedTaskUpdate.NewTasks = nil

	compareTaskUpdates(expectedTaskUpdate, taskUpdate, t)
}

func TestFileIntegritySemantics_SuccessBadGenerationNum(t *testing.T) {
	// Task succeeded, but the generation number it was called with does not match that of the
	// task in spanner, so we have to update and re-issue.
	taskUpdate := getTestingTaskUpdate(0, Success, 0, getTestingTaskSpec(123))

	var semantics TaskTransactionalSemantics = &FileIntegritySemantics{1000}
	err := semantics.Apply(taskUpdate)

	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}

	// Unqueued, with updated generation number.
	expectedTaskUpdate := getTestingTaskUpdate(0, Unqueued, 0, getTestingTaskSpec(1000))
	expectedTaskUpdate.NewTasks = nil

	compareTaskUpdates(expectedTaskUpdate, taskUpdate, t)
}

func TestFileIntegritySemantics_Success(t *testing.T) {
	// Task succeeded, and the generation number it was called with matches that which is in
	// spanner. Retain the SUCCESS!
	taskUpdate := getTestingTaskUpdate(123, Success, 0, getTestingTaskSpec(123))

	var semantics TaskTransactionalSemantics = &FileIntegritySemantics{1000}
	err := semantics.Apply(taskUpdate)

	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}

	// Unqueued, with updated generation number.
	expectedTaskUpdate := getTestingTaskUpdate(123, Success, 0, getTestingTaskSpec(123))
	compareTaskUpdates(expectedTaskUpdate, taskUpdate, t)
}

func TestFileIntegritySemantics_InvalidTaskSpec(t *testing.T) {
	// Verifies we fail in the event of an invalid task spec.
	taskUpdate := getTestingTaskUpdate(123, Success, 0, "not a legit task spec")

	var semantics TaskTransactionalSemantics = &FileIntegritySemantics{1000}
	err := semantics.Apply(taskUpdate)

	if err == nil {
		t.Error("expected json parsing error")
	}
}

func TestFileIntegritySemantics_MissingGenNumSpec(t *testing.T) {
	// Verifies that we reissue when task spec is missing generation number.
	taskUpdate := getTestingTaskUpdate(123, Success, 0, noGenNumTaskSpec)

	var semantics TaskTransactionalSemantics = &FileIntegritySemantics{1000}
	err := semantics.Apply(taskUpdate)

	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}

	// Unqueued, with updated generation number.
	expectedTaskUpdate := getTestingTaskUpdate(123, Unqueued, 0, getTestingTaskSpec(1000))
	expectedTaskUpdate.NewTasks = nil

	compareTaskUpdates(expectedTaskUpdate, taskUpdate, t)
}

func TestFileIntegritySemantics_MissingGenNumParams(t *testing.T) {
	// Verifies that we reissue when params are missing generation number.
	taskUpdate := getTestingTaskUpdate(0, Success, 0, getTestingTaskSpec(123))
	taskUpdate.OriginalTaskParams = TaskParams{}

	var semantics TaskTransactionalSemantics = &FileIntegritySemantics{1000}
	err := semantics.Apply(taskUpdate)

	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}

	// Unqueued, with updated generation number.
	expectedTaskUpdate := getTestingTaskUpdate(0, Unqueued, 0, getTestingTaskSpec(1000))
	expectedTaskUpdate.OriginalTaskParams = TaskParams{}
	expectedTaskUpdate.NewTasks = nil

	compareTaskUpdates(expectedTaskUpdate, taskUpdate, t)
}

func TestFileIntegritySemantics_MissingGenNumSpecAndParams(t *testing.T) {
	// Verifies that we reissue when neither params nor spec have generation number.
	taskUpdate := getTestingTaskUpdate(0, Success, 0, noGenNumTaskSpec)
	taskUpdate.OriginalTaskParams = TaskParams{}

	var semantics TaskTransactionalSemantics = &FileIntegritySemantics{1000}
	err := semantics.Apply(taskUpdate)

	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}

	// Unqueued, with updated generation number.
	expectedTaskUpdate := getTestingTaskUpdate(0, Unqueued, 0, getTestingTaskSpec(1000))
	expectedTaskUpdate.OriginalTaskParams = TaskParams{}
	expectedTaskUpdate.NewTasks = nil

	compareTaskUpdates(expectedTaskUpdate, taskUpdate, t)
}

func TestNeedGenerationNumCheck(t *testing.T) {
	var tests = []struct {
		status      int64
		failureType proto.TaskFailureType_Type
		needGenNum  bool
	}{
		{Success, proto.TaskFailureType_UNUSED, true},
		{Failed, proto.TaskFailureType_UNUSED, false},
		{Failed, proto.TaskFailureType_UNKNOWN, false},
		{Failed, proto.TaskFailureType_FILE_MODIFIED_FAILURE, true},
		{Failed, proto.TaskFailureType_MD5_MISMATCH_FAILURE, true},
		{Failed, proto.TaskFailureType_PRECONDITION_FAILURE, true},
		{Failed, proto.TaskFailureType_FILE_NOT_FOUND_FAILURE, false},
		{Failed, proto.TaskFailureType_PERMISSION_FAILURE, false},
	}

	for _, tc := range tests {
		if tc.needGenNum != NeedGenerationNumCheck(&Task{
			Status:      tc.status,
			FailureType: tc.failureType,
		}) {
			t.Errorf("expected needGenNum = %v for Status %v, FailureType %v",
				tc.needGenNum, tc.status, tc.failureType)
		}
	}
}
