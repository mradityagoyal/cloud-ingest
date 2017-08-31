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
	"fmt"
)

type UploadGCSProgressMessageHandler struct {
	Store Store
}

func (h *UploadGCSProgressMessageHandler) HandleMessage(jobSpec *JobSpec, task *Task) error {
	if task.Status != Success {
		return nil
	}
	// TODO(b/63014658): de-normalize the task spec into the progress message,
	// so you do not have to query the database again.
	task, err := h.Store.GetTaskSpec(task.JobConfigId, task.JobRunId, task.TaskId)
	if err != nil {
		fmt.Printf("Error getting task spec of task: %v, with error: %v.\n",
			task, err)
		return err
	}

	var uploadGCSTaskSpec UploadGCSTaskSpec
	if err := json.Unmarshal([]byte(task.TaskSpec), &uploadGCSTaskSpec); err != nil {
		fmt.Printf(
			"Error decoding task spec: %s, with error: %v.\n", task.TaskSpec, err)
		return err
	}

	loadBQTaskId := GetLoadBQTaskId(uploadGCSTaskSpec.DstObject)
	loadBQTaskSpec := LoadBQTaskSpec{
		SrcGCSBucket: uploadGCSTaskSpec.DstBucket,
		SrcGCSObject: uploadGCSTaskSpec.DstObject,
		DstBQDataset: jobSpec.BQDataset,
		DstBQTable:   jobSpec.BQTable,
	}

	loadBigQueryTaskSpecJson, err := json.Marshal(loadBQTaskSpec)
	if err != nil {
		fmt.Printf(
			"Error encoding task spec to JSON string, task spec: %v, err: %v.\n",
			loadBQTaskSpec, err)
		return err
	}
	return h.Store.InsertNewTasks([]*Task{&Task{
		JobConfigId: task.JobConfigId,
		JobRunId:    task.JobRunId,
		TaskId:      loadBQTaskId,
		TaskType:    loadBQTaskType,
		TaskSpec:    string(loadBigQueryTaskSpecJson),
	}})
}
