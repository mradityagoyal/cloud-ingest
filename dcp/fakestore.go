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

	"cloud.google.com/go/pubsub"
)

var (
	errTaskNotFound             = errors.New("task not found")
	errInsertNewTasks           = errors.New("inserting new tasks")
	errInvalidCompletionMessage = errors.New("invalid task completion message")
)

// FakeStore is a fake implementation of Store interface that is used for test
// purposes.
type FakeStore struct {
	jobSpec      *JobSpec
	tasks        map[string]*Task
	logEntryRows []*LogEntryRow
}

func (s *FakeStore) GetJobSpec(jobConfigId string) (*JobSpec, error) {
	return s.jobSpec, nil
}

func (s *FakeStore) GetTaskSpec(
	jobConfigId string, jobRunId string, taskId string) (string, error) {

	task, ok := s.tasks[getTaskFullId(jobConfigId, jobRunId, taskId)]
	if !ok {
		return "", errTaskNotFound
	}
	return task.TaskSpec, nil
}

func (s *FakeStore) InsertNewTasks(tasks []*Task) error {
	if s.tasks == nil {
		return errInsertNewTasks
	}
	for _, task := range tasks {
		s.tasks[task.getTaskFullId()] = task
	}
	return nil
}

func (s *FakeStore) UpdateAndInsertTasks(taskUpdates *TaskUpdateCollection) error {
	for taskUpdate := range taskUpdates.GetTaskUpdates() {
		s.tasks[taskUpdate.Task.getTaskFullId()] = taskUpdate.Task
		for _, task := range taskUpdate.NewTasks {
			s.tasks[task.getTaskFullId()] = task
		}
	}
	return nil
}

func (s *FakeStore) QueueTasks(n int, listTopic *pubsub.Topic, copyTopic *pubsub.Topic) error {
	return errors.New("QueueTasks: Not implemented.")
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
			if l.JobConfigId == sl.JobConfigId && l.JobRunId == sl.JobRunId &&
				l.TaskId == sl.TaskId && l.LogEntryId == sl.LogEntryId {
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
