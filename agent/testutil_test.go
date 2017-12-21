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

	"github.com/GoogleCloudPlatform/cloud-ingest/dcp"
	"github.com/GoogleCloudPlatform/cloud-ingest/dcp/proto"
)

func checkFailureWithType(
	taskRRName string, failureType proto.TaskFailureType_Type,
	msg dcp.TaskCompletionMessage, t *testing.T) {

	if msg.TaskRRName != taskRRName {
		t.Errorf("expected task id to be \"%s\", found: \"%s\"", taskRRName, msg.TaskRRName)
	}
	if msg.Status != "FAILURE" {
		t.Errorf("expected task to fail, found: %s", msg.Status)
	}
	if msg.FailureType != failureType {
		t.Errorf("expected task to fail with %s type, found: %s",
			proto.TaskFailureType_Type_name[int32(failureType)],
			proto.TaskFailureType_Type_name[int32(msg.FailureType)])
	}
}

func checkForInvalidTaskParamsArguments(
	taskRRName string, msg dcp.TaskCompletionMessage, t *testing.T) {
	checkFailureWithType(taskRRName, proto.TaskFailureType_UNKNOWN, msg, t)
	if !strings.Contains(msg.FailureMessage, "Invalid task params arguments") {
		t.Errorf("expected \"Invalid task params arguments\" failure message, found: %s",
			msg.FailureMessage)
	}
}

func checkSuccessMsg(taskRRName string, msg dcp.TaskCompletionMessage, t *testing.T) {
	if msg.TaskRRName != taskRRName {
		t.Errorf("expected task id to be \"%s\", found: \"%s\"", taskRRName, msg.TaskRRName)
	}
	if msg.Status != "SUCCESS" {
		t.Errorf("expected message to success, found: %s", msg.Status)
	}
}

func testMissingOneTaskParams(h WorkHandler, taskParams dcp.TaskParams, t *testing.T) {
	for param := range taskParams {
		paramVal := taskParams[param]
		delete(taskParams, param)
		msg := h.Do(context.Background(), "task", taskParams)
		checkForInvalidTaskParamsArguments("task", msg, t)
		taskParams[param] = paramVal
	}
}
