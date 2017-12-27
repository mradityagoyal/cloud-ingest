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

	"github.com/GoogleCloudPlatform/cloud-ingest/gcloud"
)

var (
	errTaskNotFound             = errors.New("task not found")
	errInvalidCompletionMessage = errors.New("invalid task completion message")
)

// FakeStore is a fake implementation of Store interface that is used for test
// purposes.
type FakeStore struct {
	jobSpec      *JobSpec
	tasks        map[TaskRRStruct]*Task
	logEntryRows []*LogEntryRow
}

func (s *FakeStore) GetJobSpec(jobConfigRRStruct JobConfigRRStruct) (*JobSpec, error) {
	return s.jobSpec, nil
}

func (s *FakeStore) GetTaskSpec(taskRRStruct TaskRRStruct) (string, error) {

	task, ok := s.tasks[taskRRStruct]
	if !ok {
		return "", errTaskNotFound
	}
	return task.TaskSpec, nil
}

func (s *FakeStore) UpdateAndInsertTasks(taskUpdates *TaskUpdateCollection) error {
	for taskUpdate := range taskUpdates.GetTaskUpdates() {
		s.tasks[taskUpdate.Task.TaskRRStruct] = taskUpdate.Task
		for _, task := range taskUpdate.NewTasks {
			s.tasks[task.TaskRRStruct] = task
		}
	}
	return nil
}

func (s *FakeStore) RoundRobinQueueTasks(n int, processListTopic gcloud.PSTopic, fallbackProjectID string) error {
	return errors.New("RoundRobinQueueTasks: Not implemented.")
}

func (s *FakeStore) GetNumUnprocessedLogs() (int64, error) {
	numUnprocessedLogs := int64(0)
	for _, l := range s.logEntryRows {
		if l.Processed == false {
			numUnprocessedLogs++
		}
	}
	return numUnprocessedLogs, nil
}

func (s *FakeStore) GetUnprocessedLogs(n int64) ([]*LogEntryRow, error) {
	var logEntryRows []*LogEntryRow
	for _, l := range s.logEntryRows {
		if l.Processed == false {
			logEntryRows = append(logEntryRows, l)
			if int64(len(logEntryRows)) >= n {
				break
			}
		}
	}
	return logEntryRows, nil
}

func (s *FakeStore) MarkLogsAsProcessed(logEntryRows []*LogEntryRow) error {
	for _, l := range logEntryRows {
		foundEntry := false
		for _, sl := range s.logEntryRows {
			if l.TaskRRStruct == sl.TaskRRStruct && l.LogEntryID == sl.LogEntryID {
				sl.Processed = true
				foundEntry = true
				break
			}
		}
		if !foundEntry {
			return errors.New("LogEntryRow to mark as processed not found.")
		}
	}
	return nil
}
