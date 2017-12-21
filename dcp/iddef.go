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
	"strings"
)

// The relative resource structs.
type JobConfigRRStruct struct {
	ProjectID   string
	JobConfigID string
}

type JobRunRRStruct struct {
	JobConfigRRStruct
	JobRunID string
}

type TaskRRStruct struct {
	JobRunRRStruct
	TaskID string
}

// Their String() functions.
func (j JobConfigRRStruct) String() string {
	return fmt.Sprint("projects/", j.ProjectID, "/jobConfigs/", j.JobConfigID)
}

func (j JobRunRRStruct) String() string {
	return fmt.Sprint(j.JobConfigRRStruct, "/jobRuns/", j.JobRunID)
}

func (t TaskRRStruct) String() string {
	return fmt.Sprint(t.JobRunRRStruct, "/tasks/", t.TaskID)
}

// Some convenience constructors.
func NewJobConfigRRStruct(projectID, configID string) *JobConfigRRStruct {
	return &JobConfigRRStruct{projectID, configID}
}

func NewJobRunRRStruct(projectID, configID, runID string) *JobRunRRStruct {
	return &JobRunRRStruct{JobConfigRRStruct{projectID, configID}, runID}
}

func NewTaskRRStruct(projectID, configID, runID, taskID string) *TaskRRStruct {
	return &TaskRRStruct{JobRunRRStruct{JobConfigRRStruct{projectID, configID}, runID}, taskID}
}

func TaskRRStructFromTaskRRName(taskRRName string) (*TaskRRStruct, error) {
	components := strings.SplitN(taskRRName, "/", 8)
	if len(components) != 8 {
		return nil, fmt.Errorf(
			"cannot parse taskRelativeResourceName: %s, expecting 8 strings separated by '/'",
			taskRRName)
	}
	if components[0] != "projects" {
		return nil, fmt.Errorf("expected collection ID of 'projects', instead got: %s for %s",
			components[0], taskRRName)
	}
	if components[2] != "jobConfigs" {
		return nil, fmt.Errorf("expected collection ID of 'jobConfigs', instead got: %s for %s",
			components[2], taskRRName)
	}
	if components[4] != "jobRuns" {
		return nil, fmt.Errorf("expected collection ID of 'jobRuns', instead got: %s for %s",
			components[4], taskRRName)
	}
	if components[6] != "tasks" {
		return nil, fmt.Errorf("expected collection ID of 'tasks', instead got: %s for %s",
			components[6], taskRRName)
	}
	return NewTaskRRStruct(components[1], components[3], components[5], components[7]), nil
}
