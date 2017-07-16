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
	"errors"
	"fmt"
	"strings"

	"golang.org/x/net/context"

	"cloud.google.com/go/pubsub"
	"cloud.google.com/go/spanner"
	"google.golang.org/api/iterator"
)

// Store provides an interface for the backing store that is used by the dcp.
type Store interface {
	// GetJobSpec retrieves the JobSpec from the store.
	GetJobSpec(jobConfigId string) (*JobSpec, error)

	// GetTaskSpec returns a task from the store that matches the id
	// (jobConfigId, jobRunId, taskId).
	GetTaskSpec(jobConfigId string, jobRunId string, taskId string) (*Task, error)

	// QueueTasks retrieves at most n tasks from the unqueued tasks, sends PubSub
	// messages to the corresponding topic, and updates the status of the task to
	// queued.
	// TODO(b/63015068): This method should be generic and should get arbitrary
	// number of topics to publish to.
	QueueTasks(n int, listTopic *pubsub.Topic, copyTopic *pubsub.Topic,
		loadBigQueryTopic *pubsub.Topic) error

	// InsertNewTasks adds the passed tasks to the store.
	// TODO(b/63017414): Optimize insert new tasks and update tasks to be done in
	// one transaction.
	InsertNewTasks(tasks []*Task) error

	// UpdateTasks updates the store with the passed tasks. Each passed task must
	// contain an existing (JobConfigId, JobRunId, TaskId), otherwise, error will
	// be returned.
	UpdateTasks(tasks []*Task) error
}

// TODO(b/63749083): Replace empty context (context.Background) when interacting
// with spanner. If the spanner transaction is stuck for any reason, there are
// no way to recover. Suggest to use WithTimeOut context.
// SpannerStore is a Google Cloud Spanner implementation of the Store interface.
type SpannerStore struct {
	Client *spanner.Client
}

func (s *SpannerStore) GetJobSpec(jobConfigId string) (*JobSpec, error) {
	jobConfigRow, err := s.Client.Single().ReadRow(
		context.Background(),
		"JobConfigs",
		spanner.Key{jobConfigId},
		[]string{"JobSpec"})
	if err != nil {
		return nil, err
	}

	jobSpec := new(JobSpec)
	var jobSpecJson string
	jobConfigRow.Column(0, &jobSpecJson)
	if err = json.Unmarshal([]byte(jobSpecJson), jobSpec); err != nil {
		return nil, err
	}
	return jobSpec, nil
}

func (s *SpannerStore) GetTaskSpec(
	jobConfigId string, jobRunId string, taskId string) (*Task, error) {

	taskRow, err := s.Client.Single().ReadRow(
		context.Background(),
		"Tasks",
		spanner.Key{jobConfigId, jobRunId, taskId},
		[]string{"TaskSpec"})
	if err != nil {
		return nil, err
	}

	task := &Task{
		JobConfigId: jobConfigId,
		JobRunId:    jobRunId,
		TaskId:      taskId,
	}

	taskRow.Column(0, &task.TaskSpec)
	return task, nil
}

func (s *SpannerStore) InsertNewTasks(tasks []*Task) error {
	if len(tasks) == 0 {
		return nil
	}
	// TODO(b/63100514): Define constants for spanner table names that can be
	// shared across store and potentially infrastructure setup implementation.
	columns := []string{
		"JobConfigId",
		"JobRunId",
		"TaskId",
		"TaskSpec",
		"Status",
	}
	mutation := make([]*spanner.Mutation, len(tasks))

	for i, task := range tasks {
		mutation[i] = spanner.InsertOrUpdate("Tasks", columns, []interface{}{
			task.JobConfigId,
			task.JobRunId,
			task.TaskId,
			task.TaskSpec,
			Unqueued,
		})
	}
	_, err := s.Client.Apply(context.Background(), mutation)
	return err
}

func (s *SpannerStore) UpdateTasks(tasks []*Task) error {
	if len(tasks) == 0 {
		return nil
	}

	columns := []string{
		"JobConfigId",
		"JobRunId",
		"TaskId",
		"Status",
		"FailureMessage",
	}
	mutation := make([]*spanner.Mutation, len(tasks))

	for i, task := range tasks {
		mutation[i] = spanner.Update("Tasks", columns, []interface{}{
			task.JobConfigId,
			task.JobRunId,
			task.TaskId,
			task.Status,
			task.FailureMessage,
		})
	}
	_, err := s.Client.Apply(context.Background(), mutation)
	return err
}

func (s *SpannerStore) QueueTasks(n int, listTopic *pubsub.Topic, copyTopic *pubsub.Topic,
	loadBigQueryTopic *pubsub.Topic) error {
	tasks, err := s.getUnqueuedTasks(n)
	if err != nil {
		return err
	}
	var publishResults []*pubsub.PublishResult
	messagesPublished := true
	for i, task := range tasks {
		var topic *pubsub.Topic
		// TODO(b/63104595): Separating out the tasks type in the task spec instead
		// parsing out the task id to get the type.
		if strings.HasPrefix(task.TaskId, listTaskPrefix) {
			topic = listTopic
		} else if strings.HasPrefix(task.TaskId, uploadGCSTaskPrefix) {
			topic = copyTopic
		} else if strings.HasPrefix(task.TaskId, loadBQTaskPrefix) {
			topic = loadBigQueryTopic
		} else {
			return errors.New(fmt.Sprintf("Unknown Task, task id: %s.", task.TaskId))
		}

		// Publish the messages.
		// TODO(b/63018625): Adjust the PubSub publish settings to control batching
		// the messages and the timeout to publish any set of messages.
		publishResults = append(publishResults, topic.Publish(
			context.Background(), &pubsub.Message{Data: []byte(task.TaskSpec)}))
		// Mark the tasks as queued.
		tasks[i].Status = Queued
	}
	for _, publishResult := range publishResults {
		if _, err := publishResult.Get(context.Background()); err != nil {
			messagesPublished = false
			break
		}
	}
	if messagesPublished {
		// Only update the tasks in the store if new messages published successfully.
		return s.UpdateTasks(tasks)
	}
	return nil
}

// getUnqueuedTasks retrieves at most n unqueued tasks from the store.
func (s *SpannerStore) getUnqueuedTasks(n int) ([]*Task, error) {
	var tasks []*Task
	stmt := spanner.Statement{
		SQL: `SELECT JobConfigId, JobRunId, TaskId, TaskSpec
          FROM Tasks@{FORCE_INDEX=TasksByStatus}
          WHERE Status = @status LIMIT @maxtasks`,
		Params: map[string]interface{}{
			"status":   Unqueued,
			"maxtasks": n,
		},
	}
	iter := s.Client.Single().Query(
		context.Background(), stmt)
	defer iter.Stop()
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}
		task := new(Task)
		if err := row.ColumnByName("JobConfigId", &task.JobConfigId); err != nil {
			return nil, err
		}
		if err := row.ColumnByName("JobRunId", &task.JobRunId); err != nil {
			return nil, err
		}
		if err := row.ColumnByName("TaskId", &task.TaskId); err != nil {
			return nil, err
		}
		if err := row.ColumnByName("TaskSpec", &task.TaskSpec); err != nil {
			return nil, err
		}
		tasks = append(tasks, task)
	}

	return tasks, nil
}
