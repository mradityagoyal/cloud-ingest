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

	"cloud.google.com/go/spanner"
)

func createDummyTask() *Task {
	return &Task{
		JobConfigId:          "config_A",
		JobRunId:             "run_A",
		TaskId:               "taskid_A",
		TaskSpec:             "taskspec_A",
		Status:               Queued,
		CreationTime:         123,
		LastModificationTime: 234,
		FailureMessage:       "failure message A",
	}
}

func TestInsertLogEntryMutation(t *testing.T) {
	listTask := createDummyTask()
	previousStatus := Unqueued
	logEntry := "dummy log entry"
	timestamp := int64(1111)

	mutations := []*spanner.Mutation{}
	insertLogEntryMutation(&mutations, listTask, previousStatus, logEntry, timestamp)
	if len(mutations) != 1 {
		t.Errorf("Expected a single mutation, found %v.", len(mutations))

	}

	test_mutation := spanner.Insert("LogEntries", getLogEntryInsertColumns(),
		[]interface{}{
			"config_A",
			"run_A",
			"taskid_A",
			int64(515984276571567629),
			timestamp,
			Queued,
			Unqueued,
			"failure message A",
			"dummy log entry",
		})

	if !reflect.DeepEqual(mutations[0], test_mutation) {
		t.Errorf("Generated mutation doesn't match test mutation,\n%s\nvs\n%s\n",
			mutations[0], test_mutation)
	}
}
