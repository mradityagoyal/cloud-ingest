package list

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
