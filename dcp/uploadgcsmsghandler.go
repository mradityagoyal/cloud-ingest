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
	"log"

	"cloud.google.com/go/storage"
)

type UploadGCSProgressMessageHandler struct {
	ObjectMetadataReader ObjectMetadataReader
}

func (h *UploadGCSProgressMessageHandler) extractGenerationNum(ctx context.Context, completionMsg *TaskCompletionMessage) (int64, error) {
	taskSpec, err := NewUploadGCSTaskSpecFromMap(completionMsg.TaskParams)
	if err != nil {
		return 0, err
	}

	// TODO (b/69319257) Move this portion of the work to the queueing worker.
	metadata, err := h.ObjectMetadataReader.GetMetadata(ctx, taskSpec.DstBucket, taskSpec.DstObject)
	if err == nil {
		return metadata.GenerationNumber, nil
	} else if err == storage.ErrObjectNotExist {
		return 0, nil
	} else {
		return 0, err
	}
}

func (h *UploadGCSProgressMessageHandler) HandleMessage(
	jobSpec *JobSpec, taskCompletionMessage *TaskCompletionMessage) (*TaskUpdate, error) {
	ctx := context.Background()
	taskUpdate, err := TaskCompletionMessageToTaskUpdate(taskCompletionMessage)
	if err != nil {
		log.Printf("Error extracting taskCompletionMessage %v: %v", taskCompletionMessage, err)
		return nil, err
	}
	taskUpdate.Task.TaskType = uploadGCSTaskType

	// If there's a chance that we need to reissue the task, we should
	// look up the expected generation number from GCS. We do this here,
	// so we don't make a blocking call within the read-write transaction.
	if NeedGenerationNumCheck(taskUpdate.Task) {

		generationNumber, err := h.extractGenerationNum(ctx, taskCompletionMessage)
		if err != nil {
			return nil, err
		}
		taskUpdate.TransactionalSemantics = &FileIntegritySemantics{
			ExpectedGenerationNum: generationNumber,
		}
	}

	return taskUpdate, nil
}
