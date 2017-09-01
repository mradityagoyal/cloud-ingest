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
	DstListResultBucket string `json:"dst_list_result_bucket"`
	DstListResultObject string `json:"dst_list_result_object"`
	SrcDirectory        string `json:"src_directory"`
}

type UploadGCSTaskSpec struct {
	SrcFile   string `json:"src_file"`
	DstBucket string `json:"dst_bucket"`
	DstObject string `json:"dst_object"`
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
}

// getTaskFullId gets a unique task id  based on task (JobConfigId, JobRunId
// and TaskId).
func (t Task) getTaskFullId() string {
	return getTaskFullId(t.JobConfigId, t.JobRunId, t.TaskId)
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

// canChangeTaskStatus checks weather a task can be moved from a fromStatus to
// a toStatus.
func canChangeTaskStatus(fromStatus int64, toStatus int64) bool {
	// Currently, the Status has to change from Unqueued -> Queued -> Fail -> Success.
	// Later we may need to change this logic for supporting retrying of failed tasks
	// or when we add an In-Progress status.
	return toStatus > fromStatus
}

// constructPubSubTaskMsg constructs the pubsub message for the passed task to
// send to the worker agents.
func constructPubSubTaskMsg(task *Task) ([]byte, error) {
	taskMsg := make(map[string]interface{})
	if err := json.Unmarshal([]byte(task.TaskSpec), &taskMsg); err != nil {
		return nil, errors.New(fmt.Sprintf(
			"error decoding JSON task spec string %s for task %v.",
			task.TaskSpec, task))
	}

	taskMsg["task_id"] = task.getTaskFullId()
	return json.Marshal(taskMsg)
}

func TaskCompletionMessageJsonToTask(msg []byte) (*Task, error) {
	taskCompletionMsgMap := make(map[string]interface{})
	if err := json.Unmarshal(msg, &taskCompletionMsgMap); err != nil {
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
		}
	} else if taskCompletionMsgMap["status"] == "SUCCESS" {
		task.Status = Success
	} else {
		return nil, errors.New(fmt.Sprintf(
			"undefined status from the completion message: %s", string(msg)))
	}

	return task, nil
}
