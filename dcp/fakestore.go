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

	"cloud.google.com/go/pubsub"
)

var (
	errTaskNotFound   = errors.New("task not found")
	errInsertNewTasks = errors.New("inserting new tasks")
)

// FakeStore is a fake implementation of Store interface that is used for test
// purposes.
type FakeStore struct {
	tasks map[string]*Task
}

// getTaskFullId is a helper method that generates a fake unique task id based
// on (JobConfigId, JobRunId, TaskId).
func getTaskFullId(task *Task) string {
	return fmt.Sprintf("%s:%s:%s", task.JobConfigId, task.JobRunId, task.TaskId)
}

func (s *FakeStore) GetJobSpec(jobConfigId string) (*JobSpec, error) {
	return nil, errors.New("GetJobSpec: Not implemented.")
}

func (s *FakeStore) GetTaskSpec(
	jobConfigId string, jobRunId string, taskId string) (*Task, error) {

	task, ok := s.tasks[getTaskFullId(&Task{
		JobConfigId: jobConfigId,
		JobRunId:    jobRunId,
		TaskId:      taskId,
	})]
	if !ok {
		return nil, errTaskNotFound
	}
	return task, nil
}

func (s *FakeStore) InsertNewTasks(tasks []*Task) error {
	if s.tasks == nil {
		return errInsertNewTasks
	}
	for _, task := range tasks {
		s.tasks[getTaskFullId(task)] = task
	}
	return nil
}

func (s *FakeStore) UpdateTasks(tasks []*Task) error {
	return errors.New("UpdateTasks: Not implemented.")
}

func (s *FakeStore) QueueTasks(n int, listTopic *pubsub.Topic, copyTopic *pubsub.Topic,
	loadBigQueryTopic *pubsub.Topic) error {
	return errors.New("QueueTasks: Not implemented.")
}
