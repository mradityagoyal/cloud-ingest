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

	"github.com/GoogleCloudPlatform/cloud-ingest/dcp"
	"github.com/GoogleCloudPlatform/cloud-ingest/dcp/proto"
	"github.com/golang/glog"
	"google.golang.org/api/googleapi"
)

func getFailureTypeFromError(err error) proto.TaskFailureType_Type {
	if os.IsNotExist(err) {
		return proto.TaskFailureType_FILE_NOT_FOUND_FAILURE
	}
	if os.IsPermission(err) {
		return proto.TaskFailureType_PERMISSION_FAILURE
	}
	if t, ok := err.(*googleapi.Error); ok {
		switch t.Code {
		case http.StatusPreconditionFailed:
			return proto.TaskFailureType_PRECONDITION_FAILURE
		case http.StatusForbidden:
			return proto.TaskFailureType_PERMISSION_FAILURE
		case http.StatusUnauthorized:
			return proto.TaskFailureType_PERMISSION_FAILURE
		}
	}
	if t, ok := err.(AgentError); ok {
		return t.FailureType
	}
	return proto.TaskFailureType_UNKNOWN
}

// buildTaskDoneMsg constructs and returns a taskDoneMsg from the params;
//   taskRRName is the task's relative resource name
//   tp are the taskParams that the task was originally called with
//   tpUpdates are modifications to the taskParams that the DCP will consume
//   le is the logEntry for this task
//   err defines whether the taskDoneMsg's Status is SUCCESS or FAILURE
func buildTaskDoneMsg(taskRRName string, tp, tpUpdates taskParams, le dcp.LogEntry, err error) taskDoneMsg {
	msg := taskDoneMsg{
		TaskRRName:       taskRRName,
		TaskParams:       tp,
		TaskParamUpdates: tpUpdates,
		LogEntry:         le,
	}
	if err != nil {
		msg.Status = "FAILURE"
		msg.FailureType = getFailureTypeFromError(err)
		msg.FailureMessage = fmt.Sprint(err)
		glog.Warningf("Encountered error in processing msg: %+v.", msg)
	} else {
		msg.Status = "SUCCESS"
	}
	return msg
}
