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
	errTaskNotFound   = errors.New("task not found")
	errInsertNewTasks = errors.New("inserting new tasks")
)

// FakeStore is a fake implementation of Store interface that is used for test
// purposes.
type FakeStore struct {
	tasks map[string]*Task
}

func (s *FakeStore) GetJobSpec(jobConfigId string) (*JobSpec, error) {
	return nil, errors.New("GetJobSpec: Not implemented.")
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

func (s *FakeStore) UpdateTasks(tasks []*Task) error {
	for _, task := range tasks {
		s.tasks[task.getTaskFullId()] = task
	}
	return nil
}

func (s *FakeStore) UpdateAndInsertTasks(taskMap map[*Task][]*Task) error {
	for updateTask, insertList := range taskMap {
		s.tasks[updateTask.getTaskFullId()] = updateTask
		for _, task := range insertList {
			s.tasks[task.getTaskFullId()] = task
		}
	}
	return nil
}

func (s *FakeStore) QueueTasks(n int, listTopic *pubsub.Topic, copyTopic *pubsub.Topic,
	loadBigQueryTopic *pubsub.Topic) error {
	return errors.New("QueueTasks: Not implemented.")
}
