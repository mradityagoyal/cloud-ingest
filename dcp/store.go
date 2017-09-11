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
	"time"

	"golang.org/x/net/context"

	"cloud.google.com/go/pubsub"
	"cloud.google.com/go/spanner"
	"google.golang.org/api/iterator"
)

// Store provides an interface for the backing store that is used by the dcp.
type Store interface {
	// GetJobSpec retrieves the JobSpec from the store.
	GetJobSpec(jobConfigId string) (*JobSpec, error)

	// GetTaskSpec returns the task spec string for the task with the given
	// (jobConfigId, jobRunId, taskId).
	GetTaskSpec(jobConfigId string, jobRunId string, taskId string) (string, error)

	// QueueTasks retrieves at most n tasks from the unqueued tasks, sends PubSub
	// messages to the corresponding topic, and updates the status of the task to
	// queued.
	// TODO(b/63015068): This method should be generic and should get arbitrary
	// number of topics to publish to.
	QueueTasks(n int, listTopic *pubsub.Topic, copyTopic *pubsub.Topic,
		loadBigQueryTopic *pubsub.Topic) error

	// InsertNewTasks should only be used for tasks that you are certain
	// do not already exist in the store. Calling this method with tasks already
	// in the store WILL result in an error being returned. If you are inserting
	// tasks as a result of receiving a PubSub message, use UpdateAndInsertTasks
	// instead.
	// InsertNewTasks adds the passed tasks to the store. Also updates the
	// totalTasks field in the relevant job run progress string.
	// TODO(b/63017414): Optimize insert new tasks and update tasks to be done in
	// one transaction.
	InsertNewTasks(tasks []*Task) error

	// UpdateTasks updates the store with the passed tasks. Each passed task must
	// contain an existing (JobConfigId, JobRunId, TaskId), otherwise, error will
	// be returned.
	UpdateTasks(tasks []*Task) error

	// UpdateAndInsertTasks takes in a map that maps from task to be updated to
	// a list of tasks to be inserted if the update task can be updated.
	// If there are two update tasks (keys in the taskMap) with the same full id
	// but different statuses, the statuses will be compared and only the task
	// with the higher status (as defined by can canChangeTaskStatus) will be
	// processed.
	//
	// For example, consider two update tasks that only differ by status:
	// let taskA = &Task{JobConfigId: "a", JobRunId: "a", TaskId: "list",
	//                   Status: Fail}
	// let taskAList = []
	// let taskB = &Task{JobConfigId: "a", JobRunId: "a", TaskId: "list",
	//                   Status: Success}
	// let taskC = &Task{JobConfigId: "a", JobRunId: "a", TaskId: "uploadGCS:file",
	//                   Status: Unqueued}
	// let taskBList = [taskC]
	// let taskMap = {taskA: taskAList, taskB: taskBList}
	//
	// Since taskA and taskB both have the same full id,
	// (JobConfigId, JobRunId, TaskId), only one of them will be processed.
	// Since canChangeTaskStatus(taskA.Status, taskB.Status) is true
	// (fail -> success), taskB will be used to update the task in the
	// database with the same full id and the tasks in taskBList (taskC)
	// will be inserted.
	UpdateAndInsertTasks(taskMap map[*Task][]*Task) error
}

// TODO(b/63749083): Replace empty context (context.Background) when interacting
// with spanner. If the spanner transaction is stuck for any reason, there are
// no way to recover. Suggest to use WithTimeOut context.
// TODO(b/65497968): Write tests for Store class
// SpannerStore is a Google Cloud Spanner implementation of the Store interface.
type SpannerStore struct {
	Client *spanner.Client
}

// getTaskInsertColumns returns an array of the columns necessary for task
// insertion
func getTaskInsertColumns() []string {
	// TODO(b/63100514): Define constants for spanner table names that can be
	// shared across store and potentially infrastructure setup implementation.
	return []string{
		"JobConfigId",
		"JobRunId",
		"TaskId",
		"TaskType",
		"TaskSpec",
		"Status",
		"CreationTime",
		"LastModificationTime",
	}
}

// getTaskUpdateColumns returns an array of the columns necessary for task
// updates
func getTaskUpdateColumns() []string {
	// TODO(b/63100514): Define constants for spanner table names that can be
	// shared across store and potentially infrastructure setup implementation.
	return []string{
		"JobConfigId",
		"JobRunId",
		"TaskId",
		"Status",
		"FailureMessage",
		"LastModificationTime",
	}
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
	jobConfigId string, jobRunId string, taskId string) (string, error) {

	taskRow, err := s.Client.Single().ReadRow(
		context.Background(),
		"Tasks",
		spanner.Key{jobConfigId, jobRunId, taskId},
		[]string{"TaskSpec"})
	if err != nil {
		return "", err
	}

	var taskSpec string

	taskRow.Column(0, &taskSpec)
	return taskSpec, nil
}

// getProgressObjFromRow returns a JobProgressSpec created from the progress
// string stored in the given row
func getProgressObjFromRow(row *spanner.Row) (*JobProgressSpec, error) {
	var progressString string
	if err := row.ColumnByName("Progress", &progressString); err != nil {
		return nil, err
	}

	progress := new(JobProgressSpec)
	if err := json.Unmarshal([]byte(progressString), progress); err != nil {
		return nil, err
	}
	return progress, nil
}

// getFullIdFromJobRow returns a JobFullRunId created from the given row.
func getFullIdFromJobRow(row *spanner.Row) (JobRunFullId, error) {
	var fullId JobRunFullId
	if err := row.ColumnByName("JobConfigId", &fullId.JobConfigId); err != nil {
		return fullId, err
	}
	if err := row.ColumnByName("JobRunId", &fullId.JobRunId); err != nil {
		return fullId, err
	}
	return fullId, nil
}

// writeJobProgressObjectUpdatesToBuffer takes in a map from JobRunFullId to
// delta counts (amount by which to increase TotalTasks) and creates and adds
// the Spanner mutations that save the modified JobProgressSpecs to the buffer
// of writes to be executed when the transaction is committed (uses BufferWrite).
// In order to create the update mutations, the method batch reads the existing
// job progress objects from Spanner.
func writeJobProgressObjectUpdatesToBuffer(ctx context.Context,
	txn *spanner.ReadWriteTransaction,
	taskDeltas map[JobRunFullId]int64) error {

	// Batch read the job progress strings to be updated
	jobColumns := []string{
		"JobConfigId",
		"JobRunId",
		"Progress",
	}

	keys := spanner.KeySets()
	for fullRunId, _ := range(taskDeltas) {
		keys = spanner.KeySets(
			keys, spanner.Key{fullRunId.JobConfigId, fullRunId.JobRunId})
	}
	iter := txn.Read(ctx, "JobRuns", keys, jobColumns)

	// Create update mutations for each job progress string
	// and write them to the transaction write buffer using
	// BufferWrite
	return iter.Do(func(row *spanner.Row) error {
		progressObj, err := getProgressObjFromRow(row)
		if err != nil {
			return err
		}
		fullJobId, err := getFullIdFromJobRow(row)
		if err != nil {
			return err
		}

		// Update totalTasks and create mutation
		progressObj.TotalTasks += taskDeltas[fullJobId]
		progressBytes, err := json.Marshal(progressObj)
		if err != nil {
			return err
		}
		return txn.BufferWrite([]*spanner.Mutation{spanner.Update(
				"JobRuns",
				jobColumns,
				[]interface{}{
					fullJobId.JobConfigId,
					fullJobId.JobRunId,
					string(progressBytes),
				},
		)})
	})
}

// getNumOfNewTasksPerJob takes in a list of tasks and returns a map
// mapping from JobRunFullId to the number of of new tasks for that JobRun.
func getNumOfNewTasksPerJob(tasks []*Task) map[JobRunFullId]int64 {
	taskDeltas := make(map[JobRunFullId]int64)
	for _, task := range(tasks) {
		fullId := task.getJobRunFullId()
		delta, exists := taskDeltas[fullId]
		if !exists {
			delta = 0
		}
		taskDeltas[fullId] = delta + 1
	}
	return taskDeltas
}

// CAUTION: Call only with tasks that do not already exist in the store.
// Calling this method with tasks that already exist will result in an error
// being returned. If inserting tasks as a result of receiving a PubSub message,
// use UpdateAndInsertTasks instead.
func (s *SpannerStore) InsertNewTasks(tasks []*Task) error {
	// TODO(b/65546216): Better error handling, especially for duplicate inserts
	if len(tasks) == 0 {
		return nil
	}

	taskDeltas := getNumOfNewTasksPerJob(tasks)

	_, err := s.Client.ReadWriteTransaction(
		context.Background(),
		func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
			// Insert the new tasks
			// TODO(b/63100514): Define constants for spanner table names that can be
			// shared across store and potentially infrastructure setup implementation.
			taskColumns := []string{
				"JobConfigId",
				"JobRunId",
				"TaskId",
				"TaskType",
				"TaskSpec",
				"Status",
				"CreationTime",
				"LastModificationTime",
			}

			mutation := make([]*spanner.Mutation, len(tasks))
			timestamp := time.Now().UnixNano()

			for i, task := range tasks {
				// Create a mutation to insert the task
				mutation[i] = spanner.Insert("Tasks",
					taskColumns,
					[]interface{}{
						task.JobConfigId,
						task.JobRunId,
						task.TaskId,
						task.TaskType,
						task.TaskSpec,
						Unqueued,
						timestamp,
						timestamp,
					})
			}

			// Store the task insertion mutations in the transaction write buffer
			err := txn.BufferWrite(mutation)
			if err != nil {
				return err
			}

			// Calculate and store the job progress update mutations in the
			// transaction write buffer
			return writeJobProgressObjectUpdatesToBuffer(
				ctx,
				txn,
				taskDeltas,
			)
		})
	return err
}

func (s *SpannerStore) UpdateTasks(tasks []*Task) error {
	if len(tasks) == 0 {
		return nil
	}

	_, err := s.Client.ReadWriteTransaction(
		context.Background(),
		func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
			var keys = spanner.KeySets()
			tasksmap := map[string]*Task{}

			for _, task := range tasks {
				tasksmap[task.getTaskFullId()] = task
				keys = spanner.KeySets(
					keys, spanner.Key{task.JobConfigId, task.JobRunId, task.TaskId})
			}

			iter := txn.Read(ctx, "Tasks", keys, getTaskUpdateColumns())
			timestamp := time.Now().UnixNano()

			return iter.Do(func(row *spanner.Row) error {
				var jobConfigId string
				var jobRunId string
				var taskId string
				var status int64

				row.ColumnByName("JobConfigId", &jobConfigId)
				row.ColumnByName("JobRunId", &jobRunId)
				row.ColumnByName("TaskId", &taskId)
				row.ColumnByName("Status", &status)

				task := tasksmap[getTaskFullId(jobConfigId, jobRunId, taskId)]
				if !canChangeTaskStatus(status, task.Status) {
					fmt.Printf("Ignore updating task %s from status %d to status %d.\n",
						taskId, status, task.Status)
					return nil
				}
				return txn.BufferWrite([]*spanner.Mutation{
					spanner.Update("Tasks", getTaskUpdateColumns(), []interface{}{
						task.JobConfigId,
						task.JobRunId,
						task.TaskId,
						task.Status,
						task.FailureMessage,
						timestamp,
					}),
				})
			})
		})
	return err
}

// removeDuplicatesAndCreateIdMaps removes any duplicate update tasks and
// creates two maps, one mapping from task full id to update tasks and the
// other mapping from the update task full id to the list of tasks that should
// be inserted if the specified update task can be updated.
func removeDuplicatesAndCreateIdMaps(
	taskMap map[*Task][]*Task) (map[string]*Task, map[string][]*Task) {

	updateTasks := make(map[string]*Task)
	insertTasks := make(map[string][]*Task)
	for updateTask, _ := range taskMap {
		fullId := updateTask.getTaskFullId()
		otherTask, exists := updateTasks[fullId]
		if !exists || canChangeTaskStatus(otherTask.Status, updateTask.Status) {
			// This is the only task so far with this full id or it is
			// more recent than any other tasks seen so far with the same full id
			updateTasks[fullId] = updateTask
			insertTasks[fullId] = taskMap[updateTask]
		}
	}

	return updateTasks, insertTasks
}

// getFullTaskIdFromRow returns the full task id constructed
// from the JobConfigId, JobRunId, and TaskId values stored in the row.
func getFullTaskIdFromRow(row *spanner.Row) (string, error) {
	var jobConfigId string
	var jobRunId string
	var taskId string

	err := row.ColumnByName("JobConfigId", &jobConfigId)
	if err != nil {
		return "", err
	}
	err = row.ColumnByName("JobRunId", &jobRunId)
	if err != nil {
		return "", err
	}
	err = row.ColumnByName("TaskId", &taskId)
	if err != nil {
		return "", err
	}

	return getTaskFullId(jobConfigId, jobRunId, taskId), nil
}

// readTasksFromSpanner takes in a map from task full id to Task and
// batch reads the tasks rows with the given full ids. Only JobConfigId,
// JobRunId, TaskId, and Status columns are read. Returns a spanner.RowIterator
// that can be used to iterate over the read rows. (Does not modify idToTask.)
func readTasksFromSpanner(ctx context.Context,
	txn *spanner.ReadWriteTransaction,
	idToTask map[string]*Task) *spanner.RowIterator {
	var keys = spanner.KeySets()

	// Create a KeySet for all the tasks to be updated
	for _, task := range idToTask {
		keys = spanner.KeySets(
			keys, spanner.Key{task.JobConfigId, task.JobRunId, task.TaskId})
	}

	// Read the previous state of the tasks to be updated
	return txn.Read(ctx, "Tasks", keys, []string{
		"JobConfigId",
		"JobRunId",
		"TaskId",
		"Status",
	})
}

// getTaskUpdateAndInsertMutations takes in a task to update and a list
// of tasks to insert and returns a list of mutations that contains both
// the mutation to update the updateTask and the mutations to insert the
// insert tasks.
func getTaskUpdateAndInsertMutations(ctx context.Context,
	txn *spanner.ReadWriteTransaction, updateTask *Task,
	insertTasks map[string][]*Task) []*spanner.Mutation {

	timestamp := time.Now().UnixNano()
	taskId := updateTask.getTaskFullId()
	mutations := make([]*spanner.Mutation, len(insertTasks[taskId]))

	// Insert the tasks associated with the update task
	for i, insertTask := range insertTasks[taskId] {
		mutations[i] = spanner.Insert("Tasks", getTaskInsertColumns(),
			[]interface{}{
				insertTask.JobConfigId,
				insertTask.JobRunId,
				insertTask.TaskId,
				insertTask.TaskType,
				insertTask.TaskSpec,
				Unqueued,
				timestamp,
				timestamp,
			})
	}

	// Update the task
	mutations = append(mutations, spanner.Update("Tasks", getTaskUpdateColumns(),
		[]interface{}{
			updateTask.JobConfigId,
			updateTask.JobRunId,
			updateTask.TaskId,
			updateTask.Status,
			updateTask.FailureMessage,
			timestamp,
		}))
	return mutations
}

// isValidUpdate takes in a spanner row containing the currently stored
// task and the updated version of the task, returning whether or not the
// updated task represents a valid update. The method also returns
// the currently stored task status.
func isValidUpdate(row *spanner.Row,
	updateTask *Task) (isValid bool, oldStatus int64, err error) {
	// Read the previous status from the row
	var status int64
	err = row.ColumnByName("Status", &status)
	if err != nil {
		return false, 0, err
	}

	return canChangeTaskStatus(status, updateTask.Status), status, nil
}

func (s *SpannerStore) UpdateAndInsertTasks(taskMap map[*Task][]*Task) error {
	if len(taskMap) == 0 {
		return nil
	}

	updateTasks, insertTasks := removeDuplicatesAndCreateIdMaps(taskMap)

	_, err := s.Client.ReadWriteTransaction(
		context.Background(),
		func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
			iter := readTasksFromSpanner(ctx, txn, updateTasks)

			// Iterate over all of the tasks to be updated, checking if they
			// can be updated. If they can be updated, update the task and insert
			// the associated tasks.
			return iter.Do(func(row *spanner.Row) error {
				taskId, err := getFullTaskIdFromRow(row)
				if err != nil {
					return err
				}
				updateTask := updateTasks[taskId]

				validUpdate, oldStatus, err := isValidUpdate(row, updateTask)
				if err != nil {
					return err
				}
				if !validUpdate {
					fmt.Printf("Ignore updating task %s from status %d to status %d.\n",
						taskId, oldStatus, updateTask.Status)
					return nil
				}

				mutations := getTaskUpdateAndInsertMutations(ctx, txn, updateTask,
					insertTasks)
				return txn.BufferWrite(mutations)
			})
		})
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
		switch task.TaskType {
		case listTaskType:
			topic = listTopic
		case uploadGCSTaskType:
			topic = copyTopic
		case loadBQTaskType:
			topic = loadBigQueryTopic
		default:
			return errors.New(fmt.Sprintf("unknown Task, task id: %s.", task.TaskId))
		}

		// Publish the messages.
		// TODO(b/63018625): Adjust the PubSub publish settings to control batching
		// the messages and the timeout to publish any set of messages.
		taskMsgJSON, err := constructPubSubTaskMsg(task)
		if err != nil {
			fmt.Printf("Unable to form task msg from task: %v with error: %v.\n",
				task, err)
			return err
		}
		publishResults = append(publishResults, topic.Publish(
			context.Background(), &pubsub.Message{Data: taskMsgJSON}))
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
		SQL: `SELECT JobConfigId, JobRunId, TaskId, TaskType, TaskSpec
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
		if err := row.ColumnByName("TaskType", &task.TaskType); err != nil {
			return nil, err
		}
		if err := row.ColumnByName("TaskSpec", &task.TaskSpec); err != nil {
			return nil, err
		}
		tasks = append(tasks, task)
	}

	return tasks, nil
}
