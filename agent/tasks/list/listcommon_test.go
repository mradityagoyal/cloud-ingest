package list

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/GoogleCloudPlatform/cloud-ingest/agent/tasks/common"

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

func TestDoesSymlinkPointToDir(t *testing.T) {
	// Create temp dirs and file.
	tmpDir := common.CreateTmpDir("", "dir")
	defer os.RemoveAll(tmpDir)
	tmpNestedDir := common.CreateTmpDir(tmpDir, "nestedDir")
	tmpFile := common.CreateTmpFile(tmpDir, "tmpfile", "dummyFileContent")

	// Create up the symlinks.
	dirSymlink := filepath.Join(tmpDir, "dirSymlink")
	err := os.Symlink(tmpNestedDir, dirSymlink)
	if err != nil {
		t.Fatalf("os.Symlink(%q, %q) got err: %v", tmpNestedDir, dirSymlink, err)
	}
	fileSymlink := filepath.Join(tmpDir, "fileSymlink")
	err = os.Symlink(tmpFile, fileSymlink)
	if err != nil {
		t.Fatalf("os.Symlink(%q, %q) got err: %v", tmpFile, fileSymlink, err)
	}

	// Test doesSymlinkPointToDir.
	gotDirSymlink, err := doesSymlinkPointToDir("", dirSymlink)
	if err != nil {
		t.Errorf("doesSymlinkPointToDir(\"\", %q) got err: %v", dirSymlink, err)
	}
	if gotDirSymlink == false {
		t.Errorf("doesSymlinkPointToDir(\"\", %q) = false, want true", dirSymlink)
	}

	gotFileSymlink, err := doesSymlinkPointToDir("", fileSymlink)
	if err != nil {
		t.Errorf("doesSymlinkPointToDir(\"\", %q) got err: %v", fileSymlink, err)
	}
	if gotFileSymlink == true {
		t.Errorf("doesSymlinkPointToDir(\"\", %q) = true, want false", fileSymlink)
	}
}
