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
	"strings"
)

const (
	idSeparator string = ":"
)

type JobConfigFullID struct {
	ProjectID   string
	JobConfigID string
}

func (j JobConfigFullID) String() string {
	return fmt.Sprint(j.ProjectID, idSeparator, j.JobConfigID)
}

type JobRunFullID struct {
	JobConfigFullID
	JobRunID string
}

func NewJobRunFullID(projectID, configID, runID string) *JobRunFullID {
	return &JobRunFullID{
		JobConfigFullID: JobConfigFullID{projectID, configID},
		JobRunID:        runID,
	}
}

func (j JobRunFullID) String() string {
	return fmt.Sprint(j.JobConfigFullID, idSeparator, j.JobRunID)
}

type TaskFullID struct {
	JobRunFullID
	TaskID string
}

func NewTaskFullID(projectID, configID, runID, taskID string) *TaskFullID {
	return &TaskFullID{
		JobRunFullID: *NewJobRunFullID(projectID, configID, runID),
		TaskID:       taskID,
	}
}

func TaskFullIDFromStr(s string) (*TaskFullID, error) {
	components := strings.SplitN(s, idSeparator, 4)
	if len(components) != 4 {
		return nil, errors.New(fmt.Sprintf(
			"cannot parse task id: %s, expecting 4 strings separated by '%s'",
			s, idSeparator))
	}
	return NewTaskFullID(
		components[0], components[1], components[2], components[3]), nil
}

func (t TaskFullID) String() string {
	return fmt.Sprint(t.JobRunFullID, idSeparator, t.TaskID)
}
