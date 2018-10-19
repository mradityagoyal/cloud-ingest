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

package agent

import (
	"fmt"
	"net/http"
	"os"

	"github.com/golang/glog"
	"google.golang.org/api/googleapi"

	taskpb "github.com/GoogleCloudPlatform/cloud-ingest/proto/task_go_proto"
)

func getFailureTypeFromError(err error) taskpb.FailureType {
	if os.IsNotExist(err) {
		return taskpb.FailureType_FILE_NOT_FOUND_FAILURE
	}
	if os.IsPermission(err) {
		return taskpb.FailureType_PERMISSION_FAILURE
	}
	if t, ok := err.(*googleapi.Error); ok {
		switch t.Code {
		case http.StatusPreconditionFailed:
			return taskpb.FailureType_PRECONDITION_FAILURE
		case http.StatusForbidden:
			return taskpb.FailureType_PERMISSION_FAILURE
		case http.StatusUnauthorized:
			return taskpb.FailureType_PERMISSION_FAILURE
		}
	}
	if t, ok := err.(AgentError); ok {
		return t.FailureType
	}
	return taskpb.FailureType_UNKNOWN_FAILURE
}

// buildTaskRespMsg constructs and returns a taskResMsg from the params;
//   taskReqMsg is the taskpb.TaskReqMsg that the task was originally called with
//   respSpec is the taskpb.Spec the updated task spec as a result of this task request
//   lf are the logFields for this task
//   err defines whether the taskProgressMsg's Status is SUCCESS or FAILURE
func buildTaskRespMsg(taskReqMsg *taskpb.TaskReqMsg, respSpec *taskpb.Spec, log *taskpb.Log, err error) *taskpb.TaskRespMsg {
	taskRespMsg := &taskpb.TaskRespMsg{
		TaskRelRsrcName: taskReqMsg.TaskRelRsrcName,
		ReqSpec:         taskReqMsg.Spec,
		RespSpec:        respSpec,
		Log:             log,
	}
	if err != nil {
		taskRespMsg.Status = "FAILURE"
		taskRespMsg.FailureType = getFailureTypeFromError(err)
		taskRespMsg.FailureMessage = fmt.Sprint(err)
		if taskRespMsg.FailureType != taskpb.FailureType_NOT_ACTIVE_JOBRUN {
			glog.Warningf("Encountered error in processing taskReqMsg: %+v, err: %v", taskReqMsg, err)
		}
	} else {
		taskRespMsg.Status = "SUCCESS"
	}
	return taskRespMsg
}
