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

// buildTaskProgressMsg constructs and returns a taskProgressMsg from the params;
//   taskRRName is the task's relative resource name
//   taskReqParams are the taskReqParams that the task was originally called with
//   taskResParams are the response key/values as a result of this task request
//   lf are the logFields for this task
//   err defines whether the taskProgressMsg's Status is SUCCESS or FAILURE
func buildTaskProgressMsg(taskRRName string, taskReqParams taskReqParams, taskResParams taskResParams, lf LogFields, err error) taskProgressMsg {
	msg := taskProgressMsg{
		TaskRRName:     taskRRName,
		TaskReqParams:  taskReqParams,
		TaskResParams:  taskResParams,
		AgentLogFields: lf,
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
