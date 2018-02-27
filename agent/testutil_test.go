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

func checkFailureWithType(
	taskRRName string, failureType proto.TaskFailureType_Type,
	msg taskDoneMsg, t *testing.T) {

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

func checkForInvalidTaskParamsArguments(
	taskRRName string, msg taskDoneMsg, t *testing.T) {
	checkFailureWithType(taskRRName, proto.TaskFailureType_UNKNOWN, msg, t)
	if !strings.Contains(msg.FailureMessage, "Invalid task params arguments") {
		t.Errorf("failure message want \"Invalid task params arguments\", got: %s",
			msg.FailureMessage)
	}
}

func checkSuccessMsg(taskRRName string, msg taskDoneMsg, t *testing.T) {
	if msg.TaskRRName != taskRRName {
		t.Errorf("want task id \"%s\", got \"%s\"", taskRRName, msg.TaskRRName)
	}
	if msg.Status != "SUCCESS" {
		t.Errorf("want message success, got: %s", msg.Status)
	}
}

func testMissingOneTaskParams(h WorkHandler, tp taskParams, t *testing.T) {
	for param := range tp {
		paramVal := tp[param]
		delete(tp, param)
		msg := h.Do(context.Background(), "task", tp)
		checkForInvalidTaskParamsArguments("task", msg, t)
		tp[param] = paramVal
	}
}
