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
	"io"
	"path/filepath"
	"strings"

	"github.com/GoogleCloudPlatform/cloud-ingest/dcp/proto"
	"github.com/GoogleCloudPlatform/cloud-ingest/helpers"
	"github.com/golang/glog"
)

type ProcessListMessageHandler struct {
	ListingResultReader ListingResultReader
}

const (
	maxEntriesToProcess   = 1000
	expectedByteOffsetKey = "byte_offset"
	// TODO(b/71637535): Move the audit logs to the cloud-ingest working space.
	// This is the root directory in the destination GCS bucket which contains files
	// created as part of the ingest process, namely list files and audit logs.
	cloudIngestWorkingSpace = "cloud-ingest"
)

// ProcessListingFileSemantics implements the TaskTransactionalSemantics interface (see
// task.go) to ensure two things:
// 1) ExpectedByteOffset defines what the ByteOffset must be in the Spanner database's
//    ProcessListTaskSpec for this process list task. If the ByteOffset does not match,
//    then something else has done some work and this task transaction will fail.
// 2) ByteOffsetForNextIteration sets the next ByteOffset in the ProcessListTaskSpec when
//    this transaction is committed. This allows subsequent work on this listing file to
//    resume where this task left off.
type ProcessListingFileSemantics struct {
	ExpectedByteOffset         int64
	ByteOffsetForNextIteration int64
}

func (plfs ProcessListingFileSemantics) Apply(taskUpdate *TaskUpdate) (bool, error) {
	// Parse the task spec.
	var ts map[string]interface{}
	decoder := json.NewDecoder(strings.NewReader(taskUpdate.Task.TaskSpec))
	decoder.UseNumber()
	if err := decoder.Decode(&ts); err != nil {
		return false, err
	}
	spannerByteOffsetJSONNumber, ok := ts[expectedByteOffsetKey]
	if !ok {
		return false, errors.New("byte_offset missing from spanner Task Spec")
	}
	spannerByteOffset, err := helpers.ToInt64(spannerByteOffsetJSONNumber)
	if err != nil {
		return false, err
	}
	if spannerByteOffset != plfs.ExpectedByteOffset {
		glog.Warningf(
			"ByteOffset doesn't match expectation, spannerByteOffset:%v, "+
				"paramByteOffset:%v. Will skip update task %s",
			spannerByteOffset, plfs.ExpectedByteOffset, taskUpdate.Task.TaskRRStruct)
		return false, nil
	}

	// Update the TaskSpec's ByteOffset field.
	ts[expectedByteOffsetKey] = plfs.ByteOffsetForNextIteration
	newTaskSpec, err := json.Marshal(ts)
	if err != nil {
		return false, err
	}
	task := taskUpdate.Task
	task.TaskSpec = string(newTaskSpec)
	task.FailureType = proto.TaskFailureType_UNUSED
	task.FailureMessage = ""
	return true, nil
}

func (h *ProcessListMessageHandler) HandleMessage(
	jobSpec *JobSpec, taskCompletionMessage *TaskCompletionMessage) (*TaskUpdate, error) {

	ctx := context.Background()
	taskUpdate, err := TaskCompletionMessageToTaskUpdate(taskCompletionMessage)
	if err != nil {
		glog.Errorf(
			"Error extracting taskCompletionMessage %v: %v",
			taskCompletionMessage, err)
		return nil, err
	}

	taskUpdate.Task.TaskType = processListTaskType
	task := taskUpdate.Task
	spec, err := NewProcessListTaskSpecFromMap(taskUpdate.OriginalTaskParams)
	if err != nil {
		return nil, err
	}

	listFileEntries, offset, err := h.ListingResultReader.ReadEntries(
		ctx, spec.DstListResultBucket, spec.DstListResultObject,
		spec.ByteOffset, maxEntriesToProcess)
	if err == io.EOF {
		task.Status = Success
	} else if err != nil {
		glog.Errorf(
			"Error reading the listing file, bucket/object: %v/%v, with error: %v.",
			spec.DstListResultBucket, spec.DstListResultObject, err)
		return nil, err
	} else {
		task.Status = Unqueued
	}

	taskUpdate.TransactionalSemantics = ProcessListingFileSemantics{
		ExpectedByteOffset:         spec.ByteOffset,
		ByteOffsetForNextIteration: offset,
	}

	var newTasks []*Task
	for _, lfe := range listFileEntries {
		newTasks, err = processListFileEntry(lfe, task, jobSpec, newTasks)
		if err != nil {
			glog.Errorf("Error processing listFileEntry %v, err: %v", lfe, err)
			return nil, err
		}
	}
	taskUpdate.NewTasks = newTasks

	logEntry := make(map[string]interface{})
	logEntry["entriesProcessed"] = int64(len(listFileEntries))
	logEntry["startingOffset"] = spec.ByteOffset
	logEntry["endingOffset"] = offset
	taskUpdate.LogEntry = logEntry

	return taskUpdate, nil
}

func processListFileEntry(lfe ListFileEntry, task *Task, jobSpec *JobSpec, newTasks []*Task) ([]*Task, error) {
	var taskType int64
	var taskID string
	var taskSpec interface{}
	filePathRelToOnPremSrcDir := helpers.GetRelPathOsAgnostic(jobSpec.OnpremSrcDirectory, lfe.FilePath)
	if lfe.IsDir {
		taskType = listTaskType
		taskID = GetListTaskID(lfe.FilePath)
		dstObject := filepath.Join(
			cloudIngestWorkingSpace, "listfiles", task.TaskRRStruct.JobConfigID,
			task.TaskRRStruct.JobRunID, filePathRelToOnPremSrcDir, "list")
		taskSpec = ListTaskSpec{
			DstListResultBucket:   jobSpec.GCSBucket,
			DstListResultObject:   dstObject,
			SrcDirectory:          lfe.FilePath,
			ExpectedGenerationNum: 0,
		}
	} else {
		taskType = copyTaskType
		taskID = GetCopyTaskID(lfe.FilePath)
		// TODO(b/69319257): Amend this logic when we implement synchronization.
		taskSpec = CopyTaskSpec{
			DstBucket:             jobSpec.GCSBucket,
			DstObject:             filePathRelToOnPremSrcDir,
			SrcFile:               lfe.FilePath,
			ExpectedGenerationNum: 0,
		}
	}
	taskSpecJSON, err := json.Marshal(taskSpec)
	if err != nil {
		glog.Errorf("Error JSON string encoding task spec: %v, err: %v.", taskSpec, err)
		return nil, err
	}
	return append(newTasks, &Task{
		TaskRRStruct: TaskRRStruct{task.TaskRRStruct.JobRunRRStruct, taskID},
		TaskType:     taskType,
		TaskSpec:     string(taskSpecJSON),
	}), nil
}
