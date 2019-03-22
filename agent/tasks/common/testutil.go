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

package common

import (
	"testing"

	taskpb "github.com/GoogleCloudPlatform/cloud-ingest/proto/task_go_proto"
)

func CheckFailureWithType(taskRelRsrcName string, failureType taskpb.FailureType, taskRespMsg *taskpb.TaskRespMsg, t *testing.T) {
	if taskRespMsg.TaskRelRsrcName != taskRelRsrcName {
		t.Errorf("want task id \"%s\", got \"%s\"", taskRelRsrcName, taskRespMsg.TaskRelRsrcName)
	}
	if taskRespMsg.Status != "FAILURE" {
		t.Errorf("want task fail, found: %s", taskRespMsg.Status)
	}
	if taskRespMsg.FailureType != failureType {
		t.Errorf("want task to fail with %s type, got: %s",
			taskpb.FailureType_name[int32(failureType)],
			taskpb.FailureType_name[int32(taskRespMsg.FailureType)])
	}
}

func CheckSuccessMsg(taskRelRsrcName string, taskRespMsg *taskpb.TaskRespMsg, t *testing.T) {
	if taskRespMsg.TaskRelRsrcName != taskRelRsrcName {
		t.Errorf("want task id \"%s\", got \"%s\"", taskRelRsrcName, taskRespMsg.TaskRelRsrcName)
	}
	if taskRespMsg.Status != "SUCCESS" {
		t.Errorf("want message success, got: %s", taskRespMsg.Status)
	}
}
