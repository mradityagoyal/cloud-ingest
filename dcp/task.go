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
)

const (
	Unqueued int64 = 0
	Queued   int64 = 1
	Failed   int64 = 2
	Success  int64 = 3

	listTaskPrefix      string = "list"
	uploadGCSTaskPrefix string = "uploadGCS:"
	loadBQTaskPrefix    string = "loadBigQuery:"

	// TODO(b/63017649): Remove this hard-coded job config and run ids.
	jobConfigId string = "ingest-job-00"
	jobRunId    string = "job-run-00"
)

type JobSpec struct {
	OnpremSrceDirectory string `json:"on_prem_src_directory"`
	GCSBucket           string `json:"gcs_bucket"`
	GCSDirectory        string `json:"gcs_directory"`
	BQDataset           string `json:"bigquery_dataset"`
	BQTable             string `json:"bigquery_table"`
}

type ListTaskSpec struct {
	TaskId              string `json:"task_id"`
	DstListResultBucket string `json:"dst_list_result_bucket"`
	DstListResultObject string `json:"dst_list_result_object"`
	SrcDirectory        string `json:"src_directory"`
}

type UploadGCSTaskSpec struct {
	TaskId    string `json:"task_id"`
	SrcFile   string `json:"src_file"`
	DstBucket string `json:"dst_bucket"`
	DstObject string `json:"dst_object"`
}

type LoadBQTaskSpec struct {
	TaskId       string `json:"task_id"`
	SrcGCSBucket string `json:"src_gcs_bucket"`
	SrcGCSObject string `json:"src_gcs_object"`
	DstBQDataset string `json:"dst_bq_dataset"`
	DstBQTable   string `json:"dst_bq_table"`
}

type Task struct {
	JobConfigId    string
	JobRunId       string
	TaskId         string
	TaskSpec       string
	Status         int64
	FailureMessage string
}

// getTaskFullId gets a unique task id  based on task (JobConfigId, JobRunId
// and TaskId).
func (t Task) getTaskFullId() string {
	return getTaskFullId(t.JobConfigId, t.JobRunId, t.TaskId)
}

// getTaskFullId is a helper method that generates a unique task id based
// on (JobConfigId, JobRunId, TaskId).
func getTaskFullId(jobConfigId string, jobRunId string, taskId string) string {
	return fmt.Sprintf("%s:%s:%s", jobConfigId, jobRunId, taskId)
}

// canChangeTaskStatus checks weather a task can be moved from a fromStatus to
// a toStatus.
func canChangeTaskStatus(fromStatus int64, toStatus int64) bool {
	// Currently, the Status has to change from Unqueued -> Queued -> Fail -> Success.
	// Later we may need to change this logic for supporting retrying of failed tasks
	// or when we add an In-Progress status.
	return toStatus > fromStatus
}

func TaskCompletionMessageJsonToTask(msg []byte) (*Task, error) {
	taskCompletionMsgMap := make(map[string]interface{})
	if err := json.Unmarshal(msg, &taskCompletionMsgMap); err != nil {
		return nil, err
	}

	taskId, ok := taskCompletionMsgMap["task_id"]
	if !ok {
		return nil, errors.New(fmt.Sprintf(
			"error reading task id from completion message: %s.", string(msg)))
	}

	// TODO(b/63017649): Remove hard-coded jobConfigId and jobRunId.
	task := &Task{
		JobConfigId: jobConfigId,
		JobRunId:    jobRunId,
		TaskId:      taskId.(string),
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
