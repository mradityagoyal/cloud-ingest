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
	"testing"
)

func TestJobConfigFullIDString(t *testing.T) {
	j := JobConfigFullID{
		ProjectID:   "project",
		JobConfigID: "config",
	}
	if j.String() != "project:config" {
		t.Errorf("expected job config string id to be: (%s), but found: (%s)",
			"project:config", j)
	}
}

func TestJobRunFullID(t *testing.T) {
	j := NewJobRunFullID("project", "config", "run")
	if j.ProjectID != "project" {
		t.Errorf("expected job run project id to be: (%s), but found: (%s)",
			"project", j.ProjectID)
	}
	if j.JobConfigID != "config" {
		t.Errorf("expected job run config id to be: (%s), but found: (%s)",
			"config", j.JobConfigID)
	}
	if j.JobRunID != "run" {
		t.Errorf("expected job run id to be: (%s), but found: (%s)",
			"run", j.JobRunID)
	}
}

func TestJobRunFullIDString(t *testing.T) {
	j := NewJobRunFullID("project", "config", "run")
	if j.String() != "project:config:run" {
		t.Errorf("expected job run string id to be: (%s), but found: (%s)",
			"project:config:run", j)
	}
}

func TestTaskFullID(t *testing.T) {
	taskFullID := NewTaskFullID("project", "config", "run", "task")
	if taskFullID.ProjectID != "project" {
		t.Errorf("expected task project id to be: (%s), but found: (%s)",
			"project", taskFullID.ProjectID)
	}
	if taskFullID.JobConfigID != "config" {
		t.Errorf("expected task config id to be: (%s), but found: (%s)",
			"config", taskFullID.JobConfigID)
	}
	if taskFullID.JobRunID != "run" {
		t.Errorf("expected task run id to be: (%s), but found: (%s)",
			"run", taskFullID.JobRunID)
	}
	if taskFullID.TaskID != "task" {
		t.Errorf("expected task id to be: (%s), but found: (%s)",
			"task", taskFullID.TaskID)
	}
}

func TestTaskIDFromStrSuccess(t *testing.T) {
	taskStr := "project:config:run:task"
	taskFullID, err := TaskFullIDFromStr(taskStr)
	if err != nil {
		t.Errorf("expected no error is parsing task %s but found err: %v", taskStr, err)
	}
	if taskFullID.ProjectID != "project" {
		t.Errorf("expected task project id to be: (%s), but found: (%s)",
			"project", taskFullID.ProjectID)
	}
	if taskFullID.JobConfigID != "config" {
		t.Errorf("expected task config id to be: (%s), but found: (%s)",
			"config", taskFullID.JobConfigID)
	}
	if taskFullID.JobRunID != "run" {
		t.Errorf("expected task run id to be: (%s), but found: (%s)",
			"run", taskFullID.JobRunID)
	}
	if taskFullID.TaskID != "task" {
		t.Errorf("expected task id to be: (%s), but found: (%s)",
			"task", taskFullID.TaskID)
	}
	if taskFullID.String() != taskStr {
		t.Errorf("expected task id string id to be: (%s), but found: (%s)",
			taskStr, *taskFullID)
	}
}

func TestTaskIDFromStrIDHasSeparator(t *testing.T) {
	taskStr := "project:config:run:list:file1"
	taskFullID, err := TaskFullIDFromStr(taskStr)
	if err != nil {
		t.Errorf("expected no error is parsing task %s but found err: %v", taskStr, err)
	}
	if taskFullID.ProjectID != "project" {
		t.Errorf("expected task project id to be: (%s), but found: (%s)",
			"project", taskFullID.ProjectID)
	}
	if taskFullID.JobConfigID != "config" {
		t.Errorf("expected task config id to be: (%s), but found: (%s)",
			"config", taskFullID.JobConfigID)
	}
	if taskFullID.JobRunID != "run" {
		t.Errorf("expected task run id to be: (%s), but found: (%s)",
			"run", taskFullID.JobRunID)
	}
	if taskFullID.TaskID != "list:file1" {
		t.Errorf("expected task id to be: (%s), but found: (%s)",
			"list:file1", taskFullID.TaskID)
	}
	if taskFullID.String() != taskStr {
		t.Errorf("expected task id string id to be: (%s), but found: (%s)",
			taskStr, *taskFullID)
	}
}

func TestTaskIDFromStrFail(t *testing.T) {
	taskStr := "project:config:run"
	if _, err := TaskFullIDFromStr(taskStr); err == nil {
		t.Errorf("expected error in parsing task id %s but found err is nil", taskStr)
	}

	taskStr = "notID"
	if _, err := TaskFullIDFromStr(taskStr); err == nil {
		t.Errorf("expected error in parsing task id %s but found err is nil", taskStr)
	}
}

func TestTaskIDString(t *testing.T) {
	taskFullID := NewTaskFullID("project", "config", "run", "task")
	if taskFullID.String() != "project:config:run:task" {
		t.Errorf("expected task id string id to be: (%s), but found: (%s)",
			"project:config:run:task", *taskFullID)
	}
}
