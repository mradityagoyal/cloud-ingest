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
	"context"
	"strings"
	"testing"

	"github.com/GoogleCloudPlatform/cloud-ingest/dcp/proto"
)

func checkFailureWithType(taskRRName string, failureType proto.TaskFailureType_Type, msg taskProgressMsg, t *testing.T) {
	if msg.TaskRRName != taskRRName {
		t.Errorf("want task id \"%s\", got \"%s\"", taskRRName, msg.TaskRRName)
	}
	if msg.Status != "FAILURE" {
		t.Errorf("want task fail, found: %s", msg.Status)
	}
	if msg.FailureType != failureType {
		t.Errorf("want task to fail with %s type, got: %s",
			proto.TaskFailureType_Type_name[int32(failureType)],
			proto.TaskFailureType_Type_name[int32(msg.FailureType)])
	}
}

func checkForInvalidTaskReqParamsArguments(taskRRName string, msg taskProgressMsg, t *testing.T) {
	checkFailureWithType(taskRRName, proto.TaskFailureType_UNKNOWN, msg, t)
	if !strings.Contains(msg.FailureMessage, "Invalid taskReqParams arguments") {
		t.Errorf("failure message want \"Invalid taskReqParams arguments\", got: %s",
			msg.FailureMessage)
	}
}

func checkSuccessMsg(taskRRName string, msg taskProgressMsg, t *testing.T) {
	if msg.TaskRRName != taskRRName {
		t.Errorf("want task id \"%s\", got \"%s\"", taskRRName, msg.TaskRRName)
	}
	if msg.Status != "SUCCESS" {
		t.Errorf("want message success, got: %s", msg.Status)
	}
}

func testMissingOneTaskReqParams(h WorkHandler, taskReqParams taskReqParams, t *testing.T) {
	for param := range taskReqParams {
		paramVal := taskReqParams[param]
		delete(taskReqParams, param)
		msg := h.Do(context.Background(), "task", taskReqParams)
		checkForInvalidTaskReqParamsArguments("task", msg, t)
		taskReqParams[param] = paramVal
	}
}
