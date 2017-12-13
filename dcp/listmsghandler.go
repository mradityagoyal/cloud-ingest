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
	"log"

	"cloud.google.com/go/storage"
)

type ListProgressMessageHandler struct {
	ObjectMetadataReader ObjectMetadataReader
}

func (h *ListProgressMessageHandler) retrieveGenerationNumber(ctx context.Context, bucketName string, objectName string) (int64, error) {
	// TODO (b/69319257) Move this portion of the work to the queueing worker.
	metadata, err := h.ObjectMetadataReader.GetMetadata(ctx, bucketName, objectName)
	if err == nil {
		return metadata.GenerationNumber, nil
	} else if err == storage.ErrObjectNotExist {
		return 0, nil
	} else {
		return 0, err
	}
}

func (h *ListProgressMessageHandler) HandleMessage(
	jobSpec *JobSpec, taskCompletionMessage *TaskCompletionMessage) (*TaskUpdate, error) {

	ctx := context.Background()
	taskUpdate, err := TaskCompletionMessageToTaskUpdate(taskCompletionMessage)
	if err != nil {
		log.Printf("Error extracting taskCompletionMessage %v: %v", taskCompletionMessage, err)
		return nil, err
	}

	task := taskUpdate.Task
	task.TaskType = listTaskType

	listTaskSpec, err := NewListTaskSpecFromMap(taskUpdate.OriginalTaskParams)
	if err != nil {
		return nil, err
	}

	// If there's a chance that we need to reissue the task, we should
	// look up the expected generation number from GCS. We do this here,
	// so we don't make a blocking call within the read-write transaction.
	if NeedGenerationNumCheck(taskUpdate.Task) {
		generationNumber, err := h.retrieveGenerationNumber(
			ctx, listTaskSpec.DstListResultBucket, listTaskSpec.DstListResultObject)
		if err != nil {
			return nil, err
		}
		taskUpdate.TransactionalSemantics = &FileIntegritySemantics{
			ExpectedGenerationNum: generationNumber,
		}
	}

	if taskUpdate.Task.Status != Success {
		return taskUpdate, nil
	}

	// Create the "process list" task.
	processListTaskSpec := ProcessListTaskSpec{
		DstListResultBucket: listTaskSpec.DstListResultBucket,
		DstListResultObject: listTaskSpec.DstListResultObject,
		SrcDirectory: listTaskSpec.SrcDirectory,
		ByteOffset: 0,
	}
	processListTaskSpecJson, err := json.Marshal(processListTaskSpec);
	if err != nil {
		log.Printf("Error encoding task spec to JSON string, task spec: %v, err: %v.",
			processListTaskSpec, err)
		return nil, err
	}
	processListTaskId := GetProcessListTaskID(
		listTaskSpec.DstListResultBucket, listTaskSpec.DstListResultObject)
	newTasks := []*Task{
		&Task{
			TaskFullID: TaskFullID{task.TaskFullID.JobRunFullID, processListTaskId},
			TaskSpec:   string(processListTaskSpecJson),
			TaskType:   processListTaskType,
		},
	}
	taskUpdate.NewTasks = newTasks

	return taskUpdate, nil
}
