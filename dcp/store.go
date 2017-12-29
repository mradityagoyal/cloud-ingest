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
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"cloud.google.com/go/pubsub"
	"cloud.google.com/go/spanner"
	"github.com/GoogleCloudPlatform/cloud-ingest/gcloud"
	"github.com/golang/glog"
	"google.golang.org/api/iterator"
)

// Note that these constants should be kept in sync with changes to the Spanner schema.
// Changing the number of columns in a table, or creating or removing indexes could
// change these limits.
const (
	MarkLogsAsProcessedBatchSize int = 2000
	// TODO(b/69808978): Determine and implement other transaction limits.

	fallbackListTopicID string = "cloud-ingest-list"
	fallbackCopyTopicID string = "cloud-ingest-copy"
)

// Store provides an interface for the backing store that is used by the DCP.
type Store interface {
	// GetJobSpec retrieves the JobSpec from the store.
	GetJobSpec(jobConfigRRStruct JobConfigRRStruct) (*JobSpec, error)

	// GetTaskSpec returns the task spec string for a Task.
	GetTaskSpec(taskRRStruct TaskRRStruct) (string, error)

	// RoundRobinQueueTasks iterates over all projects, getting at most n tasks
	// from the unqueued tasks for each project. It sends PubSub messages to the
	// corresponding topic, and updates the status of the task to
	// queued.
	// If there is any error in retrieving topics for projects, or no projects are
	// populated in Spanner, this function falls back to using default topic
	// names in the fallback project ID.
	// TODO (b/70989356): Remove fallback logic once project IDs and topics are
	// populated by webconsole.
	RoundRobinQueueTasks(n int, processListTopic gcloud.PSTopic, fallbackProjectID string) error

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
	Spanner gcloud.Spanner
	PubSub  gcloud.PS
}

// Helper struct for passing around topics associated with a project.
type PubSubTopics struct {
	ListTopicID string
	CopyTopicID string
}

// getTaskInsertColumns returns an array of the columns necessary for task
// insertion
func getTaskInsertColumns() []string {
	// TODO(b/63100514): Define constants for spanner table names that can be
	// shared across store and potentially infrastructure setup implementation.
	return []string{
		"ProjectId",
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

func (s *SpannerStore) GetJobSpec(jobConfigRRStruct JobConfigRRStruct) (*JobSpec, error) {
	jobConfigRow, err := s.Spanner.Single().ReadRow(
		context.Background(),
		"JobConfigs",
		spanner.Key{jobConfigRRStruct.ProjectID, jobConfigRRStruct.JobConfigID},
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

func (s *SpannerStore) GetTaskSpec(taskRRStruct TaskRRStruct) (string, error) {
	taskRow, err := s.Spanner.Single().ReadRow(
		context.Background(),
		"Tasks",
		spanner.Key{
			taskRRStruct.ProjectID,
			taskRRStruct.JobConfigID,
			taskRRStruct.JobRunID,
			taskRRStruct.TaskID},
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

// getJobRunRRStructFromJobRow returns a JobRunRRStruct created from the given row.
func getJobRunRRStructFromJobRow(row *spanner.Row) (JobRunRRStruct, error) {
	var jobRunRRStruct JobRunRRStruct
	if err := row.ColumnByName(
		"ProjectId", &jobRunRRStruct.ProjectID); err != nil {
		return jobRunRRStruct, err
	}
	if err := row.ColumnByName(
		"JobConfigId", &jobRunRRStruct.JobConfigID); err != nil {
		return jobRunRRStruct, err
	}
	if err := row.ColumnByName("JobRunId", &jobRunRRStruct.JobRunID); err != nil {
		return jobRunRRStruct, err
	}
	return jobRunRRStruct, nil
}

// writeJobCountersObjectUpdatesToBuffer uses the deltas stored in the given
// map to create and add Spanner mutations that save the modified
// JobCountersSpecs to the buffer of writes to be executed when the transaction
// is committed (uses BufferWrite).
// In order to create the update mutations, the method batch reads the existing
// job counters objects from Spanner.
func writeJobCountersObjectUpdatesToBuffer(ctx context.Context,
	txn gcloud.ReadWriteTransaction,
	counters JobCountersCollection) error {

	// Batch read the job counters strings to be updated
	jobColumns := []string{
		"ProjectId",
		"JobConfigId",
		"JobRunId",
		"Counters",
		"Status",
	}

	keys := spanner.KeySets()
	for jobRunRRStruct, _ := range counters.deltas {
		keys = spanner.KeySets(
			keys, spanner.Key{
				jobRunRRStruct.ProjectID,
				jobRunRRStruct.JobConfigID,
				jobRunRRStruct.JobRunID,
			})
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
		jobRunRRStruct, err := getJobRunRRStructFromJobRow(row)
		if err != nil {
			return err
		}
		var oldStatus int64
		err = row.ColumnByName("Status", &oldStatus)
		if err != nil {
			return err
		}

		// Update totalTasks and create mutation.
		deltaObj := counters.deltas[jobRunRRStruct]
		countersObj.ApplyDelta(deltaObj)
		countersBytes, err := countersObj.Marshal()
		if err != nil {
			return err
		}

		jobInsertColumns := []string{
			"ProjectId",
			"JobConfigId",
			"JobRunId",
			"Counters",
		}

		jobInsertValues := []interface{}{
			jobRunRRStruct.ProjectID,
			jobRunRRStruct.JobConfigID,
			jobRunRRStruct.JobRunID,
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

// getTaskRRStructFromRow returns the task id constructed from the ProjectID,
// JobConfigID, JobRunID, and TaskID values stored in the row.
func getTaskRRStructFromRow(row *spanner.Row) (TaskRRStruct, error) {
	var taskRRStruct TaskRRStruct
	jobRunRRStruct, err := getJobRunRRStructFromJobRow(row)
	if err != nil {
		return taskRRStruct, err
	}
	taskRRStruct.JobRunRRStruct = jobRunRRStruct

	err = row.ColumnByName("TaskId", &taskRRStruct.TaskID)
	if err != nil {
		return taskRRStruct, err
	}

	return taskRRStruct, nil
}

// readTasksFromSpanner takes in a map from task full id to Task and
// batch reads the tasks rows with the given full ids. Only ProjectID, JobConfigID,
// JobRunID, TaskID, and Status columns are read. Returns a spanner.RowIterator
// that can be used to iterate over the read rows. (Does not modify idToTask.)
func readTasksFromSpanner(ctx context.Context,
	txn gcloud.ReadWriteTransaction,
	taskUpdateCollection *TaskUpdateCollection) gcloud.RowIterator {
	var keys = spanner.KeySets()

	// Create a KeySet for all the tasks to be updated
	for taskUpdate := range taskUpdateCollection.GetTaskUpdates() {
		taskRRStruct := taskUpdate.Task.TaskRRStruct
		keys = spanner.KeySets(
			keys, spanner.Key{
				taskRRStruct.ProjectID,
				taskRRStruct.JobConfigID,
				taskRRStruct.JobRunID,
				taskRRStruct.TaskID})
	}

	// Read the previous state of the tasks to be updated
	return txn.Read(ctx, "Tasks", keys, []string{
		"ProjectId",
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
	txn gcloud.ReadWriteTransaction, updateTask *TaskUpdate,
	oldStatus int64) []*spanner.Mutation {

	timestamp := time.Now().UnixNano()
	insertTasks := updateTask.NewTasks
	mutations := make([]*spanner.Mutation, len(insertTasks))

	// Insert the tasks associated with the update task.
	for i, insertTask := range insertTasks {
		taskRRStruct := insertTask.TaskRRStruct
		mutations[i] = spanner.Insert("Tasks", getTaskInsertColumns(),
			[]interface{}{
				taskRRStruct.ProjectID,
				taskRRStruct.JobConfigID,
				taskRRStruct.JobRunID,
				taskRRStruct.TaskID,
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
	taskRRStruct := task.TaskRRStruct
	var updateInputMap = map[string]interface{}{
		"ProjectId":            taskRRStruct.ProjectID,
		"JobConfigId":          taskRRStruct.JobConfigID,
		"JobRunId":             taskRRStruct.JobRunID,
		"TaskId":               taskRRStruct.TaskID,
		"Status":               task.Status,
		"TaskSpec":             task.TaskSpec,
		"LastModificationTime": timestamp,
	}
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

	// Fail the transaction if it changes it from Unqueued to Unqueued.
	// TODO(b/71503268): Have task to ignore individual task updates instead of
	// aborting all the task updates in the transaction. Additionally, there should
	// be a clear semantic about the task update and Pub/Sub message associated to
	// it. i.e. Update and ack' the message, skip update but ack' the message, or
	// skip update and nack' the message.
	if status == Unqueued && updateTask.Status == Unqueued {
		return false, 0, fmt.Errorf(
			"task %s has to be queued first before it's processed.", updateTask.TaskRRStruct)
	}
	return canChangeTaskStatus(status, updateTask.Status), status, nil
}

func (s *SpannerStore) UpdateAndInsertTasks(tasks *TaskUpdateCollection) error {
	if tasks.Size() == 0 {
		return nil
	}

	_, err := s.Spanner.ReadWriteTransaction(
		context.Background(),
		func(ctx context.Context, txn gcloud.ReadWriteTransaction) error {
			iter := readTasksFromSpanner(ctx, txn, tasks)
			var counters JobCountersCollection

			// Iterate over all of the tasks to be updated, checking if they
			// can be updated. If they can be updated, update the task and insert
			// the associated tasks.
			err := iter.Do(func(row *spanner.Row) error {
				taskID, err := getTaskRRStructFromRow(row)

				// Grab the task spec from the row as well. It contains information
				// used by task-specific semantics, and is critical for correctly updating status.
				var taskSpec string
				err = row.ColumnByName("TaskSpec", &taskSpec)
				if err != nil {
					return err
				}
				taskUpdate := tasks.GetTaskUpdate(taskID)
				taskUpdate.Task.TaskSpec = taskSpec

				// If there are any task-specific semantics that need to be part of the transaction, do
				// them here.
				if taskUpdate.TransactionalSemantics != nil {
					if proceed, err := taskUpdate.TransactionalSemantics.Apply(
						taskUpdate); err != nil || !proceed {
						return err
					}
				}

				validUpdate, oldStatus, err := isValidUpdate(row, taskUpdate.Task)
				if err != nil {
					return err
				}
				if !validUpdate {
					glog.Infof("Ignore updating task %s from status %d to status %d.",
						taskID, oldStatus, taskUpdate.Task.Status)
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

func (s *SpannerStore) RoundRobinQueueTasks(n int, processListTopic gcloud.PSTopic, fallbackProjectID string) error {
	m, err := s.getProjectTopicsMap()
	if err != nil || len(m) == 0 {
		// Fallback to default topics and project.
		m = map[string]PubSubTopics{
			fallbackProjectID: PubSubTopics{fallbackListTopicID, fallbackCopyTopicID},
		}
	}
	for projectID, pst := range m {
		// TODO (b/70989550): Maintain references to Topics in a map to avoid repeated
		// setup/teardown of Topics.
		listTopic := s.PubSub.TopicInProject(pst.ListTopicID, projectID)
		defer listTopic.Stop()
		copyTopic := s.PubSub.TopicInProject(pst.CopyTopicID, projectID)
		defer copyTopic.Stop()
		if err := s.queueTasks(n, projectID, listTopic, processListTopic, copyTopic); err != nil {
			return err
		}
	}
	return nil
}

func (s *SpannerStore) getProjectTopicsMap() (map[string]PubSubTopics, error) {
	stmt := spanner.NewStatement("SQL: `SELECT ProjectId, ListTopicId, CopyTopicId FROM Projects")
	iter := s.Spanner.Single().Query(context.Background(), stmt)
	defer iter.Stop()
	m := make(map[string]PubSubTopics)
	err := iter.Do(func(row *spanner.Row) error {
		var projectID string
		if err := row.ColumnByName("ProjectId", &projectID); err != nil {
			return err
		}
		var pst PubSubTopics
		if err := row.ColumnByName("ListTopicId", &pst.ListTopicID); err != nil {
			return err
		}
		if err := row.ColumnByName("CopyTopicId", &pst.CopyTopicID); err != nil {
			return err
		}
		m[projectID] = pst
		return nil
	})
	if err != nil {
		return nil, err
	}
	return m, nil
}

func (s *SpannerStore) queueTasks(n int, projectID string, listTopic, processListTopic, copyTopic gcloud.PSTopic) error {
	tasks, err := s.getUnqueuedTasks(n, projectID)
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
		var topic gcloud.PSTopic
		switch task.TaskType {
		case listTaskType:
			topic = listTopic
		case processListTaskType:
			topic = processListTopic
		case uploadGCSTaskType:
			topic = copyTopic
		default:
			return errors.New(fmt.Sprintf("unknown Task, task id: %v.", task.TaskRRStruct))
		}

		// Publish the messages.
		// TODO(b/63018625): Adjust the PubSub publish settings to control batching
		// the messages and the timeout to publish any set of messages.
		taskMsgJSON, err := constructPubSubTaskMsg(task)

		if err != nil {
			glog.Errorf("Unable to form task msg from task: %v with error: %v.",
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
			glog.Errorf("PubSub publish error:", err)
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
		taskRRStruct, err := getTaskRRStructFromRow(row)
		if err != nil {
			return nil, err
		}

		if err := row.ColumnByName("LogEntryId", &ler.LogEntryID); err != nil {
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
		ler.TaskRRStruct = taskRRStruct
		logEntryRows = append(logEntryRows, ler)
	}
	return logEntryRows, nil
}

func (s *SpannerStore) MarkLogsAsProcessed(logEntryRows []*LogEntryRow) error {
	if len(logEntryRows) == 0 {
		return nil
	}
	// Break the logEntryRows into multiple transactions to get under the 20000
	// Spanner mutation transaction limit. These entries don't all have to be updated
	// in a single transaction, since the LogEntryProcessor guarantees at-least-once
	// log writing to GCS.
	txnSize := MarkLogsAsProcessedBatchSize
	for i := 0; i < len(logEntryRows); i += txnSize {
		rangeEnd := i + txnSize
		if rangeEnd > len(logEntryRows) {
			rangeEnd = len(logEntryRows)
		}
		txnRows := logEntryRows[i:rangeEnd]
		_, err := s.Spanner.ReadWriteTransaction(
			context.Background(),
			func(ctx context.Context, txn gcloud.ReadWriteTransaction) error {
				mutations := make([]*spanner.Mutation, len(txnRows))
				for i, l := range txnRows {
					taskRRStruct := l.TaskRRStruct
					mutations[i] = spanner.Update(
						"LogEntries",
						[]string{
							"ProjectId",
							"JobConfigId",
							"JobRunId",
							"TaskId",
							"LogEntryId",
							"Processed",
						},
						[]interface{}{
							taskRRStruct.ProjectID,
							taskRRStruct.JobConfigID,
							taskRRStruct.JobRunID,
							taskRRStruct.TaskID,
							l.LogEntryID,
							true, /* Processed.*/
						})
				}
				return txn.BufferWrite(mutations)
			})
		if err != nil {
			return err
		}
	}
	return nil
}

// getUnqueuedTasks retrieves at most n unqueued tasks for projectID from the store.
func (s *SpannerStore) getUnqueuedTasks(n int, projectID string) ([]*Task, error) {
	var tasks []*Task
	stmt := spanner.Statement{
		SQL: `SELECT ProjectId, JobConfigId, JobRunId, TaskId, TaskType, TaskSpec
          FROM Tasks@{FORCE_INDEX=TasksByStatus}
          WHERE Status = @status AND ProjectId = @pid LIMIT @maxtasks`,
		Params: map[string]interface{}{
			"status":   Unqueued,
			"pid":      projectID,
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
		taskRRStruct, err := getTaskRRStructFromRow(row)
		if err != nil {
			return nil, err
		}
		task := &Task{TaskRRStruct: taskRRStruct}
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
