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
	"github.com/GoogleCloudPlatform/cloud-ingest/dcp/proto"
	"strings"
)

const (
	Unqueued int64 = 0
	Queued   int64 = 1
	Failed   int64 = 2
	Success  int64 = 3

	taskIdSeparator string = ":"

	listTaskPrefix      string = "list"
	uploadGCSTaskPrefix string = "uploadGCS"
	loadBQTaskPrefix    string = "loadBigQuery"

	listTaskType      int64 = 1
	uploadGCSTaskType int64 = 2
	loadBQTaskType    int64 = 3
)

type JobSpec struct {
	OnpremSrcDirectory string `json:"onPremSrcDirectory"`
	GCSBucket          string `json:"gcsBucket"`
	GCSDirectory       string `json:"gcsDirectory"`
	BQDataset          string `json:"bigqueryDataset"`
	BQTable            string `json:"bigqueryTable"`
}

type ListTaskSpec struct {
	DstListResultBucket   string `json:"dst_list_result_bucket"`
	DstListResultObject   string `json:"dst_list_result_object"`
	SrcDirectory          string `json:"src_directory"`
	ExpectedGenerationNum int64  `json:"expected_generation_num"`
}

type UploadGCSTaskSpec struct {
	SrcFile               string `json:"src_file"`
	DstBucket             string `json:"dst_bucket"`
	DstObject             string `json:"dst_object"`
	ExpectedGenerationNum int64  `json:"expected_generation_num"`
}

type LoadBQTaskSpec struct {
	SrcGCSBucket string `json:"src_gcs_bucket"`
	SrcGCSObject string `json:"src_gcs_object"`
	DstBQDataset string `json:"dst_bq_dataset"`
	DstBQTable   string `json:"dst_bq_table"`
}

type Task struct {
	JobConfigId          string
	JobRunId             string
	TaskId               string
	TaskSpec             string
	TaskType             int64
	Status               int64
	CreationTime         int64
	LastModificationTime int64
	FailureMessage       string
	FailureType          proto.TaskFailureType_Type
}

// TaskUpdate represents a task to be updated, with it's log entry and new tasks
// generated from the update.
type TaskUpdate struct {
	// Task to be updated.
	Task *Task

	// LogEntry that is associated with task update.
	LogEntry *LogEntry

	// NewTasks that are generated by this task update.
	NewTasks []*Task
}

// TaskUpdateCollection is a collection of task updates to be committed to the
// store.
type TaskUpdateCollection struct {
	// tasks is a map from full task id string to task update details.
	tasks map[string]*TaskUpdate
}

// AddTaskUpdate adds a taskUpdate into the collection. If there is a task in
// the collection that has the same full task id as taskUpdate but with
// different status, the statuses will be compared and only the task with the
// higher status (as defined by canChangeTaskStatus) will be added to the
// collection.
//
// For example, consider the following task updates:
//
//	taskUpdateA := &TaskUpdate{
//		Task: &Task{
//			JobConfigId: "a",
//			JobRunId:    "a",
//			TaskId:      "list",
//			Status:      Failed,
//		},
//	}
//
// 	taskUpdateB := &TaskUpdate{
//		Task: &Task{
//			JobConfigId: "a",
//			JobRunId:    "a",
//			TaskId:      "list",
//			Status:      Success,
//		},
//		NewTasks: []*Task{
//			&Task{
//				JobConfigId: "a",
//				JobRunId:    "a",
//				TaskId:      "upload",
//				Status:      Unqueued,
//			},
//		},
//	}
//
// Only one of taskUpdateA and taskUpdateB can exist in the collection since they
// have the same full task IDs.  If taskUpdateA is already in the collection and
// the caller tries to add taskUpdateB, taskUpdateA will be replaced in the
// collection by taskUpdateB because canChangeTaskStatus(taskUpdateA.Task.Status,
// taskUpdateB.Task.Status) is true (Failed -> Success).
func (tc *TaskUpdateCollection) AddTaskUpdate(taskUpdate *TaskUpdate) {
	if tc.tasks == nil {
		tc.tasks = make(map[string]*TaskUpdate)
	}
	fullId := taskUpdate.Task.getTaskFullId()

	otherTaskUpdate, exists := tc.tasks[fullId]
	if !exists || canChangeTaskStatus(otherTaskUpdate.Task.Status, taskUpdate.Task.Status) {
		// This is the only task so far with this full id or it is
		// more recent than any other tasks seen so far with the same full id.
		tc.tasks[fullId] = taskUpdate
	}
}

func (tc TaskUpdateCollection) Size() int {
	return len(tc.tasks)
}

func (tc TaskUpdateCollection) GetTaskUpdate(fullTaskId string) *TaskUpdate {
	return tc.tasks[fullTaskId]
}

func (tc TaskUpdateCollection) GetTaskUpdates() <-chan *TaskUpdate {
	c := make(chan *TaskUpdate)
	go func() {
		defer close(c)
		for _, taskUpdate := range tc.tasks {
			c <- taskUpdate
		}
	}()
	return c
}

func (tc *TaskUpdateCollection) Clear() {
	tc.tasks = make(map[string]*TaskUpdate)
}

// getTaskFullId gets a unique task id  based on task (JobConfigId, JobRunId
// and TaskId).
func (t Task) getTaskFullId() string {
	return getTaskFullId(t.JobConfigId, t.JobRunId, t.TaskId)
}

// getJobRunFullId gets a JobRunFullId for the job run of this task
func (t Task) getJobRunFullId() JobRunFullId {
	return JobRunFullId{t.JobConfigId, t.JobRunId}
}

// GetUploadGCSTaskId returns the task id of an uploadGCS type task for
// the given file.
func GetUploadGCSTaskId(filePath string) string {
	// TODO(b/64038794): The task ids should be a hash of the filePath, the
	// filePath might be too long and already duplicated in the task spec.
	return uploadGCSTaskPrefix + taskIdSeparator + filePath
}

// GetLoadBQTaskId returns the task id of a loadBiqQuery type task for
// the given file.
func GetLoadBQTaskId(srcGCSObject string) string {
	// TODO(b/64038794): The task ids should be a hash of the SrcGCSObject, the
	// SrcGCSObject might be too long and already duplicated in the task spec.
	return loadBQTaskPrefix + taskIdSeparator + srcGCSObject
}

// getTaskFullId is a helper method that generates a unique task id based
// on (JobConfigId, JobRunId, TaskId).
func getTaskFullId(jobConfigId string, jobRunId string, taskId string) string {
	return fmt.Sprintf(
		"%s%s%s%s%s", jobConfigId, taskIdSeparator, jobRunId, taskIdSeparator, taskId)
}

// constructTaskFromFullTaskId constructs a task from a colon-separated full
// task id "job_config_id:job_run_id:task_id"
func constructTaskFromFullTaskId(fullTaskId string) (*Task, error) {
	task_id_components := strings.SplitN(fullTaskId, taskIdSeparator, 3)
	if len(task_id_components) != 3 {
		return nil, errors.New(fmt.Sprintf(
			"can not parse full task id: %s, expecting 3 strings separated by ':'", fullTaskId))
	}
	return &Task{
		JobConfigId: task_id_components[0],
		JobRunId:    task_id_components[1],
		TaskId:      task_id_components[2],
	}, nil
}

// canChangeTaskStatus checks whether a task can be moved from a fromStatus to
// a toStatus.
func canChangeTaskStatus(fromStatus int64, toStatus int64) bool {
	// Currently, the Status has to change from Unqueued -> Queued -> Fail -> Success.
	// Later we may need to change this logic for supporting retrying of failed tasks
	// or when we add an In-Progress status.
	return toStatus > fromStatus
}

// constructPubSubTaskMsg constructs the Pub/Sub message for the passed task to
// send to the worker agents.
func constructPubSubTaskMsg(task *Task) ([]byte, error) {
	taskParams := make(map[string]interface{})
	if err := json.Unmarshal([]byte(task.TaskSpec), &taskParams); err != nil {
		return nil, errors.New(fmt.Sprintf(
			"error decoding JSON task spec string %s for task %v.",
			task.TaskSpec, task))
	}

	taskMsg := make(map[string]interface{})
	taskMsg["task_id"] = task.getTaskFullId()
	taskMsg["task_params"] = taskParams
	return json.Marshal(taskMsg)
}

func TaskCompletionMessageJsonToTaskUpdate(msg []byte) (*TaskUpdate, error) {
	taskCompletionMsgMap := make(map[string]interface{})
	d := json.NewDecoder(strings.NewReader(string(msg)))
	d.UseNumber()
	if err := d.Decode(&taskCompletionMsgMap); err != nil {
		return nil, err
	}

	fullTaskId, ok := taskCompletionMsgMap["task_id"].(string)
	if !ok {
		return nil, errors.New(fmt.Sprintf(
			"error reading task id from completion message: %s.", string(msg)))
	}

	task, err := constructTaskFromFullTaskId(fullTaskId)
	if err != nil {
		return nil, err
	}

	if taskCompletionMsgMap["status"] == "FAILED" {
		task.Status = Failed
		if failureMsg, ok := taskCompletionMsgMap["failure_message"]; ok {
			task.FailureMessage = failureMsg.(string)
			// TODO(b/68710612): Use a meaningful failure type.
			task.FailureType = proto.TaskFailureType_UNUSED
		}
	} else if taskCompletionMsgMap["status"] == "SUCCESS" {
		task.Status = Success
	} else {
		return nil, errors.New(fmt.Sprintf(
			"undefined status from the completion message: %s", string(msg)))
	}

	// TODO(Step 4 in b/65462509) A "log_entry" should be mandatory with the update message.
	var logEntry *LogEntry
	if taskCompletionMsgMap["log_entry"] != nil {
		logEntry = NewLogEntry(taskCompletionMsgMap["log_entry"].(map[string]interface{}))
	}

	return &TaskUpdate{
		Task:     task,
		LogEntry: logEntry,
		NewTasks: []*Task{},
	}, nil
}
