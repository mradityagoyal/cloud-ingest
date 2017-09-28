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
	"path/filepath"
)

const (
	noTaskIdInListOutput string = ("expected task ID %s in first line of list task output file, but got %s")
)

type ListProgressMessageHandler struct {
	Store               Store
	ListingResultReader ListingResultReader
}

func (h *ListProgressMessageHandler) HandleMessage(jobSpec *JobSpec, taskWithLog TaskWithLog) error {
	if taskWithLog.Task.Status != Success {
		return h.Store.UpdateTasks([]TaskWithLog{taskWithLog})
	}
	task := taskWithLog.Task

	// TODO(b/63014658): denormalize the task spec into the progress message, so
	// you do not have to query the database to get the task spec.
	taskSpec, err := h.Store.GetTaskSpec(task.JobConfigId, task.JobRunId, task.TaskId)
	if err != nil {
		fmt.Printf("Error getting task spec of task: %v, with error: %v.\n",
			task, err)
		return err
	}

	var listTaskSpec ListTaskSpec
	if err := json.Unmarshal([]byte(taskSpec), &listTaskSpec); err != nil {
		fmt.Printf(
			"Error decoding task spec: %s, with error: %v.\n", taskSpec, err)
		return err
	}

	filePaths, err := h.ListingResultReader.ReadListResult(
		listTaskSpec.DstListResultBucket, listTaskSpec.DstListResultObject)
	if err != nil {
		fmt.Printf(
			"Error reading the list task result, list task spec: %v, with error: %v.\n",
			listTaskSpec, err)
		return err
	}
	taskIdFromFile := <-filePaths

	if taskIdFromFile != task.getTaskFullId() {
		return errors.New(fmt.Sprintf(noTaskIdInListOutput, task.getTaskFullId(), taskIdFromFile))
	}
	var newTasks []*Task
	for filePath := range filePaths {
		uploadGCSTaskId := GetUploadGCSTaskId(filePath)
		dstObject, _ := filepath.Rel(listTaskSpec.SrcDirectory, filePath)
		uploadGCSTaskSpec := UploadGCSTaskSpec{
			SrcFile:   filePath,
			DstBucket: jobSpec.GCSBucket,
			DstObject: dstObject,
		}
		uploadGCSTaskSpecJson, err := json.Marshal(uploadGCSTaskSpec)
		if err != nil {
			fmt.Printf(
				"Error encoding task spec to JSON string, task spec: %v, err: %v.\n",
				uploadGCSTaskSpec, err)
			return err
		}
		newTasks = append(newTasks, &Task{
			JobConfigId: task.JobConfigId,
			JobRunId:    task.JobRunId,
			TaskId:      uploadGCSTaskId,
			TaskType:    uploadGCSTaskType,
			TaskSpec:    string(uploadGCSTaskSpecJson),
		})
	}

	taskWithLogMap := make(map[TaskWithLog][]*Task)
	taskWithLogMap[taskWithLog] = newTasks
	if err := h.Store.UpdateAndInsertTasks(taskWithLogMap); err != nil {
		fmt.Printf("Error adding new tasks to store with err: %v.\n", err)
		return err
	}
	return nil
}
