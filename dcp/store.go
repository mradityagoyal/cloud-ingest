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
	"log"
	"time"

	"golang.org/x/net/context"

	"cloud.google.com/go/pubsub"
	"cloud.google.com/go/spanner"
	"google.golang.org/api/iterator"
)

// Store provides an interface for the backing store that is used by the DCP.
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
	QueueTasks(n int, listTopic *pubsub.Topic, copyTopic *pubsub.Topic) error

	// GetNumUnprocessedLogs returns the number of rows in the LogEntries table
	// with the 'Processed' column set to false. Any errors result in returning
	// zero. Note that although this function returns a simple int64, the
	// underlying Spanner code scans the entire table to count the rows, so use
	// this function judiciously.
	GetNumUnprocessedLogs() (int64, error)

	// GetUnprocessedLogs retrieves up to 'n' rows from the LogEntries table with
	// the 'Processed' column set to false. These rows are ordered by the
	// CreationTime column from least to most recent.
	GetUnprocessedLogs(n int64) ([]*LogEntryRow, error)

	// MarkLogsAsProcessed updates the rows in the LogEntries table, setting the
	// 'Processed' column to true for all the rows specified by 'logEntryRows'.
	// Any error means that none of the rows will be updated.
	MarkLogsAsProcessed(logEntryRows []*LogEntryRow) error

	// TODO(b/67453832): Deprecate InsertNewTasks, this method is not used in the
	// DCP logic. It should be removed after handling large listing tasks.
	// InsertNewTasks should only be used for tasks that you are certain
	// do not already exist in the store. Calling this method with tasks already
	// in the store WILL result in an error being returned. If you are inserting
	// tasks as a result of receiving a PubSub message, use UpdateAndInsertTasks
	// instead.
	// InsertNewTasks adds the passed tasks to the store. Also updates the
	// totalTasks field in the relevant job run counters string.
	InsertNewTasks(tasks []*Task) error

	// UpdateAndInsertTasks updates and insert news tasks provided in the passed
	// TaskUpdateCollection object. It also inserts the log entries associated
	// with the task updates.
	UpdateAndInsertTasks(tasks *TaskUpdateCollection) error
}

// TODO(b/63749083): Replace empty context (context.Background) when interacting
// with spanner. If the spanner transaction is stuck for any reason, there are
// no way to recover. Suggest to use WithTimeOut context.
// TODO(b/65497968): Write tests for Store class
// SpannerStore is a Google Cloud Spanner implementation of the Store interface.
type SpannerStore struct {
	Spanner Spanner
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

func (s *SpannerStore) GetJobSpec(jobConfigId string) (*JobSpec, error) {
	jobConfigRow, err := s.Spanner.Single().ReadRow(
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

	taskRow, err := s.Spanner.Single().ReadRow(
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

// getCountersObjFromRow returns a JobCounters created from the counters
// string stored in the given row
func getCountersObjFromRow(row *spanner.Row) (*JobCounters, error) {
	var countersString string
	if err := row.ColumnByName("Counters", &countersString); err != nil {
		return nil, err
	}

	counters := new(JobCounters)
	if err := counters.Unmarshal(countersString); err != nil {
		return nil, err
	}
	return counters, nil
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

// writeJobCountersObjectUpdatesToBuffer uses the deltas stored in the given
// map to create and add Spanner mutations that save the modified
// JobCountersSpecs to the buffer of writes to be executed when the transaction
// is committed (uses BufferWrite).
// In order to create the update mutations, the method batch reads the existing
// job counters objects from Spanner.
func writeJobCountersObjectUpdatesToBuffer(ctx context.Context,
	txn ReadWriteTransaction,
	counters JobCountersCollection) error {

	// Batch read the job counters strings to be updated
	jobColumns := []string{
		"JobConfigId",
		"JobRunId",
		"Counters",
		"Status",
	}

	keys := spanner.KeySets()
	for fullRunId, _ := range counters.deltas {
		keys = spanner.KeySets(
			keys, spanner.Key{fullRunId.JobConfigId, fullRunId.JobRunId})
	}
	iter := txn.Read(ctx, "JobRuns", keys, jobColumns)

	// Create update mutations for each job counters string
	// and write them to the transaction write buffer using
	// BufferWrite
	return iter.Do(func(row *spanner.Row) error {
		countersObj, err := getCountersObjFromRow(row)
		if err != nil {
			return err
		}
		fullJobId, err := getFullIdFromJobRow(row)
		if err != nil {
			return err
		}
		var oldStatus int64
		err = row.ColumnByName("Status", &oldStatus)
		if err != nil {
			return err
		}

		// Update totalTasks and create mutation.
		deltaObj := counters.deltas[fullJobId]
		countersObj.ApplyDelta(deltaObj)
		countersBytes, err := countersObj.Marshal()
		if err != nil {
			return err
		}

		jobInsertColumns := []string{
			"JobConfigId",
			"JobRunId",
			"Counters",
		}

		jobInsertValues := []interface{}{
			fullJobId.JobConfigId,
			fullJobId.JobRunId,
			string(countersBytes),
		}

		// Check if status changed.
		newStatus := countersObj.GetJobStatus()
		if newStatus != oldStatus {
			if JobRunStatusChangeNotificationChannel != nil {
				JobRunStatusChangeNotificationChannel <- 0
			}
			// Job status has changed, add the update to the mutation params.
			jobInsertColumns = append(jobInsertColumns, "Status")
			jobInsertValues = append(jobInsertValues, newStatus)
			if IsJobTerminated(newStatus) {
				jobInsertColumns = append(jobInsertColumns, "JobFinishTime")
				jobInsertValues = append(jobInsertValues, time.Now().UnixNano())
			}
		}

		return txn.BufferWrite([]*spanner.Mutation{spanner.Update(
			"JobRuns",
			jobInsertColumns,
			jobInsertValues,
		)})
	})
}

// TODO(akaiser): Deprecate this function in favor of using UpdateAndInsertTasks with
// a TaskUpdate with a nil updated-task.
//
// CAUTION: Call only with tasks that do not already exist in the store.
// Calling this method with tasks that already exist will result in an error
// being returned. If inserting tasks as a result of receiving a PubSub message,
// use UpdateAndInsertTasks instead.
func (s *SpannerStore) InsertNewTasks(tasks []*Task) error {
	// TODO(b/65546216): Better error handling, especially for duplicate inserts
	if len(tasks) == 0 {
		return nil
	}

	var counters JobCountersCollection
	err := counters.updateForTaskUpdate(&TaskUpdate{Task: nil, LogEntry: nil, NewTasks: tasks}, 0)
	if err != nil {
		return err
	}

	_, err = s.Spanner.ReadWriteTransaction(
		context.Background(),
		func(ctx context.Context, txn ReadWriteTransaction) error {
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

			// Create and store the job counters update mutations in the
			// transaction write buffer.
			return writeJobCountersObjectUpdatesToBuffer(
				ctx,
				txn,
				counters,
			)
		})
	return err
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
	txn ReadWriteTransaction,
	taskUpdateCollection *TaskUpdateCollection) RowIterator {
	var keys = spanner.KeySets()

	// Create a KeySet for all the tasks to be updated
	for taskUpdate := range taskUpdateCollection.GetTaskUpdates() {
		keys = spanner.KeySets(
			keys, spanner.Key{
				taskUpdate.Task.JobConfigId,
				taskUpdate.Task.JobRunId,
				taskUpdate.Task.TaskId})
	}

	// Read the previous state of the tasks to be updated
	return txn.Read(ctx, "Tasks", keys, []string{
		"JobConfigId",
		"JobRunId",
		"TaskId",
		"Status",
		"TaskSpec",
	})
}

// getTaskUpdateAndInsertMutations takes in a task to update and a list
// of tasks to insert and returns a list of mutations that contains both
// the mutation to update the updateTask and the mutations to insert the
// insert tasks.
func getTaskUpdateAndInsertMutations(ctx context.Context,
	txn ReadWriteTransaction, updateTask *TaskUpdate,
	oldStatus int64) []*spanner.Mutation {

	timestamp := time.Now().UnixNano()
	insertTasks := updateTask.NewTasks
	mutations := make([]*spanner.Mutation, len(insertTasks))

	// Insert the tasks associated with the update task.
	for i, insertTask := range insertTasks {
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

	// Update the task.
	task := updateTask.Task
	mutations = getTaskUpdateMutations(mutations, task, timestamp)

	// Create the log entry for the updated task.
	insertLogEntryMutation(&mutations, task, oldStatus, updateTask.LogEntry, timestamp)
	return mutations
}

// getTaskUpdateMutations checks if the task is failed or not. If the task is of type failed,
// it updates the FailureMessage and FailureType columns. Otherwise, it doesn't update these
// fields because they are not relevant.
// It returns the spanner mutations that happened as a result of the task update.
func getTaskUpdateMutations(mutations []*spanner.Mutation, task *Task, timestamp int64) []*spanner.Mutation {
	// TODO(b/63100514): Define constants for spanner table names that can be
	// shared across store and potentially infrastructure setup implementation.
	var updateInputMap = make(map[string]interface{})
	updateInputMap["JobConfigId"] = task.JobConfigId
	updateInputMap["JobRunId"] = task.JobRunId
	updateInputMap["TaskId"] = task.TaskId
	updateInputMap["Status"] = task.Status
	updateInputMap["TaskSpec"] = task.TaskSpec
	updateInputMap["LastModificationTime"] = timestamp
	if task.Status == Failed {
		updateInputMap["FailureMessage"] = task.FailureMessage
		updateInputMap["FailureType"] = int64(task.FailureType)
	}
	return append(mutations, spanner.UpdateMap("Tasks", updateInputMap))
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

func (s *SpannerStore) UpdateAndInsertTasks(tasks *TaskUpdateCollection) error {
	if tasks.Size() == 0 {
		return nil
	}

	_, err := s.Spanner.ReadWriteTransaction(
		context.Background(),
		func(ctx context.Context, txn ReadWriteTransaction) error {
			iter := readTasksFromSpanner(ctx, txn, tasks)
			var counters JobCountersCollection

			// Iterate over all of the tasks to be updated, checking if they
			// can be updated. If they can be updated, update the task and insert
			// the associated tasks.
			err := iter.Do(func(row *spanner.Row) error {
				taskId, err := getFullTaskIdFromRow(row)

				// Grab the task spec from the row as well. It contains information
				// used by task-specific semantics, and is critical for correctly updating status.
				var taskSpec string
				err = row.ColumnByName("TaskSpec", &taskSpec)
				if err != nil {
					return err
				}
				taskUpdate := tasks.GetTaskUpdate(taskId)
				taskUpdate.Task.TaskSpec = taskSpec

				// If there are any task-specific semantics that need to be part of the transaction, do
				// them here.
				if taskUpdate.TransactionalSemantics != nil {
					err := taskUpdate.TransactionalSemantics.Apply(taskUpdate)
					if err != nil {
						return err
					}
				}

				// As a safeguard, any task that is now unqueued must not have any "next" tasks set.
				if taskUpdate.Task.Status == Unqueued && len(taskUpdate.NewTasks) > 0 {
					return fmt.Errorf("unqueued task %s has 'next' tasks.", taskId)
				}

				validUpdate, oldStatus, err := isValidUpdate(row, taskUpdate.Task)
				if err != nil {
					return err
				}
				if !validUpdate {
					log.Printf("Ignore updating task %s from status %d to status %d.",
						taskId, oldStatus, taskUpdate.Task.Status)
					return nil
				}

				err = counters.updateForTaskUpdate(taskUpdate, oldStatus)
				if err != nil {
					return err
				}

				mutations := getTaskUpdateAndInsertMutations(ctx, txn, taskUpdate, oldStatus)
				return txn.BufferWrite(mutations)
			})
			if err != nil {
				return err
			}

			return writeJobCountersObjectUpdatesToBuffer(
				ctx,
				txn,
				counters,
			)

		})
	return err
}

func (s *SpannerStore) QueueTasks(n int, listTopic *pubsub.Topic, copyTopic *pubsub.Topic) error {
	tasks, err := s.getUnqueuedTasks(n)
	if err != nil {
		return err
	}
	taskUpdates := &TaskUpdateCollection{}
	for _, task := range tasks {
		taskUpdates.AddTaskUpdate(&TaskUpdate{
			Task:     task,
			LogEntry: nil,
			NewTasks: []*Task{},
		})
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
		default:
			return errors.New(fmt.Sprintf("unknown Task, task id: %s.", task.TaskId))
		}

		// Publish the messages.
		// TODO(b/63018625): Adjust the PubSub publish settings to control batching
		// the messages and the timeout to publish any set of messages.
		taskMsgJSON, err := constructPubSubTaskMsg(task)
		if err != nil {
			log.Printf("Unable to form task msg from task: %v with error: %v.",
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
	// Only update the tasks in the store if new messages published successfully.
	if messagesPublished {
		return s.UpdateAndInsertTasks(taskUpdates)
	}
	return nil
}

func (s *SpannerStore) GetNumUnprocessedLogs() (int64, error) {
	stmt := spanner.NewStatement("SELECT COUNT(*) as count FROM LogEntries WHERE Processed = false")
	iter := s.Spanner.Single().Query(context.Background(), stmt)
	defer iter.Stop()
	row, err := iter.Next()
	if err != nil {
		return 0, err
	}
	var count int64
	err = row.ColumnByName("count", &count)
	if err != nil {
		return 0, err
	}
	return count, nil
}

func (s *SpannerStore) GetUnprocessedLogs(n int64) ([]*LogEntryRow, error) {
	var stmt spanner.Statement
	stmt = spanner.NewStatement(`SELECT * FROM LogEntries WHERE Processed = false
		 ORDER BY CreationTime LIMIT @maxtasks`)
	stmt.Params["maxtasks"] = n
	iter := s.Spanner.Single().Query(context.Background(), stmt)
	defer iter.Stop()
	var logEntryRows []*LogEntryRow
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}
		ler := new(LogEntryRow)
		if err := row.ColumnByName("JobConfigId", &ler.JobConfigId); err != nil {
			return nil, err
		}
		if err := row.ColumnByName("JobRunId", &ler.JobRunId); err != nil {
			return nil, err
		}
		if err := row.ColumnByName("TaskId", &ler.TaskId); err != nil {
			return nil, err
		}
		if err := row.ColumnByName("LogEntryId", &ler.LogEntryId); err != nil {
			return nil, err
		}
		if err := row.ColumnByName("CreationTime", &ler.CreationTime); err != nil {
			return nil, err
		}
		if err := row.ColumnByName("CurrentStatus", &ler.CurrentStatus); err != nil {
			return nil, err
		}
		if err := row.ColumnByName("PreviousStatus", &ler.PreviousStatus); err != nil {
			return nil, err
		}
		if err := row.ColumnByName("FailureMessage", &ler.FailureMessage); err != nil {
			return nil, err
		}
		if err := row.ColumnByName("LogEntry", &ler.LogEntry); err != nil {
			return nil, err
		}
		if err := row.ColumnByName("Processed", &ler.Processed); err != nil {
			return nil, err
		}
		logEntryRows = append(logEntryRows, ler)
	}
	return logEntryRows, nil
}

func (s *SpannerStore) MarkLogsAsProcessed(logEntryRows []*LogEntryRow) error {
	if len(logEntryRows) == 0 {
		return nil
	}
	_, err := s.Spanner.ReadWriteTransaction(
		context.Background(),
		func(ctx context.Context, txn ReadWriteTransaction) error {
			mutations := make([]*spanner.Mutation, len(logEntryRows))
			for i, l := range logEntryRows {
				mutations[i] = spanner.Update(
					"LogEntries",
					[]string{
						"JobConfigId",
						"JobRunId",
						"TaskId",
						"LogEntryId",
						"Processed",
					},
					[]interface{}{
						l.JobConfigId,
						l.JobRunId,
						l.TaskId,
						l.LogEntryId,
						true, /* Processed.*/
					})
			}
			return txn.BufferWrite(mutations)
		})
	return err
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
	iter := s.Spanner.Single().Query(context.Background(), stmt)
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
