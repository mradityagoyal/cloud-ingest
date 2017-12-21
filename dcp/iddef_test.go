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

func TestJobConfigRRName(t *testing.T) {
	j := JobConfigRRStruct{
		ProjectID:   "project",
		JobConfigID: "config",
	}
	if j.String() != "projects/project/jobConfigs/config" {
		t.Errorf("expected job config string id to be: (%s), but found: (%s)",
			"projects/project/jobConfigs/config", j)
	}
}

func TestJobRunRRStruct(t *testing.T) {
	j := NewJobRunRRStruct("project", "config", "run")
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

func TestJobRunRRName(t *testing.T) {
	j := NewJobRunRRStruct("project", "config", "run")
	if j.String() != "projects/project/jobConfigs/config/jobRuns/run" {
		t.Errorf("expected job run string id to be: (%s), but found: (%s)",
			"projects/project/jobConfigs/config/jobRuns/run", j)
	}
}

func TestTaskRRStruct(t *testing.T) {
	taskRRStruct := NewTaskRRStruct("project", "config", "run", "task")
	if taskRRStruct.ProjectID != "project" {
		t.Errorf("expected task project id to be: (%s), but found: (%s)",
			"project", taskRRStruct.ProjectID)
	}
	if taskRRStruct.JobConfigID != "config" {
		t.Errorf("expected task config id to be: (%s), but found: (%s)",
			"config", taskRRStruct.JobConfigID)
	}
	if taskRRStruct.JobRunID != "run" {
		t.Errorf("expected task run id to be: (%s), but found: (%s)",
			"run", taskRRStruct.JobRunID)
	}
	if taskRRStruct.TaskID != "task" {
		t.Errorf("expected task ID to be: (%s), but found: (%s)",
			"task", taskRRStruct.TaskID)
	}
}

func TaskRRStructFromTaskRRNameSuccess(t *testing.T) {
	taskRRName := "projects/project/jobConfigs/config/jobRuns/run/tasks/task"
	taskRRStruct, err := TaskRRStructFromTaskRRName(taskRRName)
	if err != nil {
		t.Errorf("expected no error in parsing task %s but found err: %v", taskRRName, err)
	}
	if taskRRStruct.ProjectID != "project" {
		t.Errorf("expected task project id to be: (%s), but found: (%s)",
			"project", taskRRStruct.ProjectID)
	}
	if taskRRStruct.JobConfigID != "config" {
		t.Errorf("expected task config id to be: (%s), but found: (%s)",
			"config", taskRRStruct.JobConfigID)
	}
	if taskRRStruct.JobRunID != "run" {
		t.Errorf("expected task run id to be: (%s), but found: (%s)",
			"run", taskRRStruct.JobRunID)
	}
	if taskRRStruct.TaskID != "task" {
		t.Errorf("expected task ID to be: (%s), but found: (%s)",
			"task", taskRRStruct.TaskID)
	}
	if taskRRStruct.String() != taskRRName {
		t.Errorf("expected task ID string id to be: (%s), but found: (%s)",
			taskRRName, *taskRRStruct)
	}
}

func TaskRRStructFromTaskRRNameHasSeparator(t *testing.T) {
	taskRRName := "projects/project/jobConfigs/config/jobRuns/run/tasks/list/file1"
	taskRRStruct, err := TaskRRStructFromTaskRRName(taskRRName)
	if err != nil {
		t.Errorf("expected no error in parsing task %s but found err: %v", taskRRName, err)
	}
	if taskRRStruct.ProjectID != "project" {
		t.Errorf("expected task project id to be: (%s), but found: (%s)",
			"project", taskRRStruct.ProjectID)
	}
	if taskRRStruct.JobConfigID != "config" {
		t.Errorf("expected task config id to be: (%s), but found: (%s)",
			"config", taskRRStruct.JobConfigID)
	}
	if taskRRStruct.JobRunID != "run" {
		t.Errorf("expected task run id to be: (%s), but found: (%s)",
			"run", taskRRStruct.JobRunID)
	}
	if taskRRStruct.TaskID != "list/file1" {
		t.Errorf("expected task ID to be: (%s), but found: (%s)",
			"list:file1", taskRRStruct.TaskID)
	}
	if taskRRStruct.String() != taskRRName {
		t.Errorf("expected task ID string id to be: (%s), but found: (%s)",
			taskRRName, *taskRRStruct)
	}
}

func TaskRRStructFromTaskRRNameFail(t *testing.T) {
	// Missing taskID.
	taskRRName := "projects/project/jobConfigs/config/jobRuns/run"
	if _, err := TaskRRStructFromTaskRRName(taskRRName); err == nil {
		t.Errorf("expected error in parsing task ID %s but found err is nil", taskRRName)
	}

	// Garbage taskRRName.
	taskRRName = "notID"
	if _, err := TaskRRStructFromTaskRRName(taskRRName); err == nil {
		t.Errorf("expected error in parsing task ID %s but found err is nil", taskRRName)
	}

	// Malformed 'projects' collection ID.
	taskRRName = "schmojects/project/jobConfigs/config/jobRuns/run/tasks/list/file1"
	if _, err := TaskRRStructFromTaskRRName(taskRRName); err == nil {
		t.Errorf("expected error in parsing task ID %s but found err is nil", taskRRName)
	}

	// Malformed 'jobConfigs' collection ID.
	taskRRName = "projects/project/jobKlonfigs/config/jobRuns/run/tasks/list/file1"
	if _, err := TaskRRStructFromTaskRRName(taskRRName); err == nil {
		t.Errorf("expected error in parsing task ID %s but found err is nil", taskRRName)
	}

	// Malformed 'jobRuns' collection ID.
	taskRRName = "projects/project/jobConfigs/config/jerrrbuns/run/tasks/list/file1"
	if _, err := TaskRRStructFromTaskRRName(taskRRName); err == nil {
		t.Errorf("expected error in parsing task ID %s but found err is nil", taskRRName)
	}

	// Malformed 'tasks' collection ID.
	taskRRName = "projects/project/jobConfigs/config/jobRuns/run/tuttuts/list/file1"
	if _, err := TaskRRStructFromTaskRRName(taskRRName); err == nil {
		t.Errorf("expected error in parsing task ID %s but found err is nil", taskRRName)
	}
}

func TestTaskRRStructString(t *testing.T) {
	taskRRStruct := NewTaskRRStruct("project", "config", "run", "task")
	if taskRRStruct.String() != "projects/project/jobConfigs/config/jobRuns/run/tasks/task" {
		t.Errorf("expected task ID string id to be: (%s), but found: (%s)",
			"projects/project/jobConfigs/config/jobRuns/run/tasks/task", *taskRRStruct)
	}
}
