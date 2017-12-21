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
	"strings"

	"github.com/GoogleCloudPlatform/cloud-ingest/dcp/proto"
	"github.com/golang/glog"
)

const expectedGenerationNumKey string = "expected_generation_num"

var reissuableFailureTypes = []proto.TaskFailureType_Type{
	proto.TaskFailureType_FILE_MODIFIED_FAILURE,
	proto.TaskFailureType_MD5_MISMATCH_FAILURE,
	proto.TaskFailureType_PRECONDITION_FAILURE,
}

type FileIntegritySemantics struct {
	ExpectedGenerationNum int64
}

func isReissuableFailure(failureType proto.TaskFailureType_Type) bool {
	for _, ft := range reissuableFailureTypes {
		if failureType == ft {
			return true
		}
	}
	return false
}

func NeedGenerationNumCheck(task *Task) bool {
	return task.Status == Success || task.Status == Failed && isReissuableFailure(task.FailureType)
}

func stageTaskForReissue(task *Task, taskSpec map[string]interface{}, expectedGenerationNum int64) error {
	// Here, we take in a partially-filled task, and update/populate its
	// fields so it can get reissued.
	taskSpec[expectedGenerationNumKey] = expectedGenerationNum
	newTaskSpec, err := json.Marshal(taskSpec)
	if err != nil {
		return err
	}

	task.TaskSpec = string(newTaskSpec)
	task.Status = Unqueued
	task.FailureType = proto.TaskFailureType_UNUSED
	task.FailureMessage = ""
	return nil
}

func (fis *FileIntegritySemantics) Apply(taskUpdate *TaskUpdate) error {

	if NeedGenerationNumCheck(taskUpdate.Task) {

		// Parse task spec
		var ts map[string]interface{}

		decoder := json.NewDecoder(strings.NewReader(taskUpdate.Task.TaskSpec))
		decoder.UseNumber()
		if err := decoder.Decode(&ts); err != nil {
			return err
		}

		if taskUpdate.Task.Status == Success {
			// Compare current task spec generation number with that passed into the task params.
			spannerGenNum, ok1 := ts[expectedGenerationNumKey]
			paramGenNum, ok2 := taskUpdate.OriginalTaskParams[expectedGenerationNumKey]

			if !ok1 || !ok2 || spannerGenNum != paramGenNum {
				// The semantics struct will be carrying with it the latest and greatest, computed
				// before the transaction.
				glog.Warningf("Re-issuing task %s: spanner taskSpec generation number (%d) differs from "+
					"original task param generation number (%d).", taskUpdate.Task.TaskRRStruct,
					spannerGenNum, paramGenNum)
				err := stageTaskForReissue(taskUpdate.Task, ts, fis.ExpectedGenerationNum)
				if err != nil {
					return err
				}
			}
		} else {
			glog.Warningf("Re-issuing task %s: %v", taskUpdate.Task.TaskRRStruct, taskUpdate.Task.FailureType)
			err := stageTaskForReissue(taskUpdate.Task, ts, fis.ExpectedGenerationNum)
			if err != nil {
				return err
			}
		}
	}

	// Nuke all new tasks in all non-success cases.
	if taskUpdate.Task.Status != Success {
		taskUpdate.NewTasks = nil
	}

	return nil
}
