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
	"bytes"
	"context"
	"fmt"
	"os"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/GoogleCloudPlatform/cloud-ingest/gcloud"
	"github.com/GoogleCloudPlatform/cloud-ingest/helpers"
	"github.com/golang/mock/gomock"
	"github.com/golang/protobuf/proto"
	"google.golang.org/api/googleapi"

	taskpb "github.com/GoogleCloudPlatform/cloud-ingest/proto/task_go_proto"
)

func testListSpec(srcDir string) *taskpb.Spec {
	return &taskpb.Spec{
		Spec: &taskpb.Spec_ListSpec{
			ListSpec: &taskpb.ListSpec{
				DstListResultBucket:   "bucket",
				DstListResultObject:   "object",
				SrcDirectories:        []string{srcDir},
				ExpectedGenerationNum: 0,
			},
		},
	}
}

func testListTaskReqMsg(taskRelRsrcName, srcDir string) *taskpb.TaskReqMsg {
	return &taskpb.TaskReqMsg{
		TaskRelRsrcName: taskRelRsrcName,
		Spec:            testListSpec(srcDir),
	}
}

func TestDirNotFound(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	writer := helpers.NewStringWriteCloser(nil)
	mockGCS := gcloud.NewMockGCS(mockCtrl)
	mockGCS.EXPECT().NewWriterWithCondition(
		context.Background(), "bucket", "object", gomock.Any()).Return(writer)

	h := ListHandler{gcs: mockGCS}
	taskReqParams := testListTaskReqMsg("task", "dir does not exist")
	taskRespMsg := h.Do(context.Background(), taskReqParams)
	checkFailureWithType("task", taskpb.FailureType_FILE_NOT_FOUND_FAILURE, taskRespMsg, t)
	if writer.WrittenString() != "" {
		t.Errorf("expected nothing written but found: %s", writer.WrittenString())
	}
}

func TestListSuccessEmptyDir(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	writer := &helpers.StringWriteCloser{}

	taskRelRsrcName := "projects/project_A/jobConfigs/config_B/jobRuns/run_C/tasks/task_D"
	var expectedListResult bytes.Buffer
	expectedListResult.WriteString(fmt.Sprintln(taskRelRsrcName))

	tmpDir := helpers.CreateTmpDir("", "test-list-agent-")
	defer os.RemoveAll(tmpDir)

	mockGCS := gcloud.NewMockGCS(mockCtrl)
	mockGCS.EXPECT().NewWriterWithCondition(
		context.Background(), "bucket", "object", gomock.Any()).Return(writer)

	h := ListHandler{gcs: mockGCS}
	taskReqParams := testListTaskReqMsg(taskRelRsrcName, tmpDir)
	taskRespMsg := h.Do(context.Background(), taskReqParams)
	checkSuccessMsg(taskRelRsrcName, taskRespMsg, t)
	if writer.WrittenString() != expectedListResult.String() {
		t.Errorf("expected to write \"%s\", found: \"%s\"",
			expectedListResult.String(), writer.WrittenString())
	}

	wantLog := &taskpb.Log{
		Log: &taskpb.Log_ListLog{
			ListLog: &taskpb.ListLog{},
		},
	}
	if !proto.Equal(taskRespMsg.Log, wantLog) {
		t.Errorf("log = %+v, want: %+v", taskRespMsg.Log, wantLog)
	}
}

func TestListSuccessFlatDir(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	writer := &helpers.StringWriteCloser{}

	taskRelRsrcName := "projects/project_A/jobConfigs/config_B/jobRuns/run_C/tasks/task_D"
	var expectedListResult bytes.Buffer
	expectedListResult.WriteString(fmt.Sprintln(taskRelRsrcName))

	tmpDir := helpers.CreateTmpDir("", "test-list-agent-")
	defer os.RemoveAll(tmpDir)

	fileContent := "0123456789"
	filePaths := make([]string, 10)
	for i := 0; i < 10; i++ {
		filePaths[i] = helpers.CreateTmpFile(tmpDir, "test-file-", fileContent)
	}
	// The results of the list are sorted.
	sort.Strings(filePaths)
	for _, path := range filePaths {
		expectedListResult.WriteString(fmt.Sprintln(ListFileEntry{false, path}))
	}

	mockGCS := gcloud.NewMockGCS(mockCtrl)
	mockGCS.EXPECT().NewWriterWithCondition(
		context.Background(), "bucket", "object", gomock.Any()).Return(writer)

	h := ListHandler{gcs: mockGCS}
	taskReqParams := testListTaskReqMsg(taskRelRsrcName, tmpDir)
	taskRespMsg := h.Do(context.Background(), taskReqParams)
	checkSuccessMsg(taskRelRsrcName, taskRespMsg, t)
	if writer.WrittenString() != expectedListResult.String() {
		t.Errorf("expected to write \"%s\", found: \"%s\"",
			expectedListResult.String(), writer.WrittenString())
	}

	wantLog := &taskpb.Log{
		Log: &taskpb.Log_ListLog{
			ListLog: &taskpb.ListLog{
				FilesFound: 10,
				BytesFound: 100,
			},
		},
	}
	if !proto.Equal(taskRespMsg.Log, wantLog) {
		t.Errorf("log = %+v, want: %+v", taskRespMsg.Log, wantLog)
	}
}

func TestListFailsFileWithNewline(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	writer := &helpers.StringWriteCloser{}

	taskRelRsrcName := "projects/project_A/jobConfigs/config_B/jobRuns/run_C/tasks/task_D"
	var expectedListResult bytes.Buffer
	expectedListResult.WriteString(fmt.Sprintln(taskRelRsrcName))

	tmpDir := helpers.CreateTmpDir("", "test-list-agent-")
	defer os.RemoveAll(tmpDir)

	fileContent := "0123456789"
	filePaths := make([]string, 11)
	for i := 0; i < 10; i++ {
		filePaths[i] = helpers.CreateTmpFile(tmpDir, "test-file-", fileContent)
	}
	filePaths[10] = helpers.CreateTmpFile(tmpDir, "test-file-with-\n-newline", fileContent)

	// The results of the list are sorted.
	sort.Strings(filePaths)
	for _, path := range filePaths {
		expectedListResult.WriteString(fmt.Sprintln(ListFileEntry{false, path}))
	}

	mockGCS := gcloud.NewMockGCS(mockCtrl)
	mockGCS.EXPECT().NewWriterWithCondition(
		context.Background(), "bucket", "object", gomock.Any()).Return(writer)

	h := ListHandler{gcs: mockGCS}
	taskReqParams := testListTaskReqMsg(taskRelRsrcName, tmpDir)
	taskRespMsg := h.Do(context.Background(), taskReqParams)
	// TODO(b/111502687): Failing with UNKNOWN_FAILURE is temporary. In the long
	// term, we will escape file with newlines.
	checkFailureWithType(taskRelRsrcName, taskpb.FailureType_UNKNOWN_FAILURE, taskRespMsg, t)
	if writer.WrittenString() != "" {
		t.Errorf("expected nothing written but found: %s", writer.WrittenString())
	}
}

func TestListSuccessNestedDir(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	writer := &helpers.StringWriteCloser{}

	taskRelRsrcName := "projects/project_A/jobConfigs/config_B/jobRuns/run_C/tasks/task_D"
	var expectedListResult bytes.Buffer
	expectedListResult.WriteString(fmt.Sprintln(taskRelRsrcName))

	tmpDir := helpers.CreateTmpDir("", "test-list-agent-")
	nestedTmpDir := helpers.CreateTmpDir(tmpDir, "sub-dir-")
	emptyDir := helpers.CreateTmpDir(tmpDir, "empty-dir-")
	defer os.RemoveAll(tmpDir)

	expectedListResult.WriteString(fmt.Sprintln(ListFileEntry{true, emptyDir}))
	expectedListResult.WriteString(fmt.Sprintln(ListFileEntry{true, nestedTmpDir}))

	fileContent := "0123456789"
	filePaths := make([]string, 0)

	for i := 0; i < 10; i++ {
		filePaths = append(filePaths, helpers.CreateTmpFile(tmpDir, "test-file-", fileContent))
	}
	// The results of the list are in sorted order.
	sort.Strings(filePaths)
	for _, path := range filePaths {
		expectedListResult.WriteString(fmt.Sprintln(ListFileEntry{false, path}))
	}
	// Create some files in the sub-dir. These should not be in the list output.
	for i := 0; i < 10; i++ {
		filePaths = append(filePaths, helpers.CreateTmpFile(nestedTmpDir, "test-file-", fileContent))
	}

	mockGCS := gcloud.NewMockGCS(mockCtrl)
	mockGCS.EXPECT().NewWriterWithCondition(
		context.Background(), "bucket", "object", gomock.Any()).Return(writer)

	h := ListHandler{gcs: mockGCS}
	taskReqParams := testListTaskReqMsg(taskRelRsrcName, tmpDir)
	taskRespMsg := h.Do(context.Background(), taskReqParams)
	checkSuccessMsg(taskRelRsrcName, taskRespMsg, t)
	if writer.WrittenString() != expectedListResult.String() {
		t.Errorf("expected to write \"%s\", found: \"%s\"",
			expectedListResult.String(), writer.WrittenString())
	}

	wantLog := &taskpb.Log{
		Log: &taskpb.Log_ListLog{
			ListLog: &taskpb.ListLog{
				FilesFound: 10,
				BytesFound: 100,
				DirsFound:  2,
			},
		},
	}
	if !proto.Equal(taskRespMsg.Log, wantLog) {
		t.Errorf("log = %+v, want: %+v", taskRespMsg.Log, wantLog)
	}
}

func fakeFile(name string, isDir bool) os.FileInfo {
	mode := os.FileMode(0777)
	if isDir {
		mode |= os.ModeDir
	}
	return helpers.NewFakeFileInfo(name, 4096, mode, time.Now())
}

func TestGetListingFileChunkSize(t *testing.T) {
	tests := []struct {
		fileInfos       []os.FileInfo
		srcDir          string
		maxChunkSize    int
		computeFromData bool // When true, ignore 'want' and generate the listing output.
		wantSize        int
		wantErr         bool
	}{
		{
			fileInfos: []os.FileInfo{
				fakeFile(strings.Repeat("f", googleapi.MinUploadChunkSize), false),
				fakeFile(strings.Repeat("d", googleapi.MinUploadChunkSize), true),
			},
			srcDir:          "/length/10",
			maxChunkSize:    100 * googleapi.MinUploadChunkSize,
			computeFromData: true,
		},
		{
			fileInfos: []os.FileInfo{
				fakeFile(strings.Repeat("f", googleapi.MinUploadChunkSize)+"1", false),
				fakeFile(strings.Repeat("f", googleapi.MinUploadChunkSize)+"2", false),
				fakeFile(strings.Repeat("d", googleapi.MinUploadChunkSize), true),
			},
			srcDir:       "/length/10",
			maxChunkSize: 2 * googleapi.MinUploadChunkSize,
			wantSize:     2 * googleapi.MinUploadChunkSize,
		},
		{
			fileInfos:    []os.FileInfo{},
			srcDir:       "/length/10",
			maxChunkSize: 2 * googleapi.MinUploadChunkSize,
			wantSize:     googleapi.MinUploadChunkSize,
		},
		{
			fileInfos: []os.FileInfo{
				fakeFile("file", false),
			},
			srcDir:       "/length/10",
			maxChunkSize: 2 * googleapi.MinUploadChunkSize,
			wantSize:     googleapi.MinUploadChunkSize,
		},
		{
			fileInfos:    []os.FileInfo{},
			srcDir:       "/length/10",
			maxChunkSize: googleapi.MinUploadChunkSize - 1,
			wantErr:      true,
		},
	}

	for _, tc := range tests {
		msgPrefix := fmt.Sprintf("getListingUploadChunkSize(%v, %v, %v)",
			tc.fileInfos, tc.srcDir, tc.maxChunkSize)
		if tc.computeFromData {
			// Write the data as-is and get its length. We deliberately couple the
			// implementations in the test, to ensure a change in file format without addressing
			// the chunk size would break the tests.
			writer := &helpers.StringWriteCloser{}
			_, err := writeListingFile(tc.fileInfos, tc.srcDir, writer)
			if err != nil {
				t.Fatalf("writeListingFile(%v, %v, %v) got error %v",
					tc.fileInfos, tc.srcDir, writer, err)
			}
			writer.Close()
			tc.wantSize = len(writer.WrittenString())
		}

		// Check results.
		gotSize, gotErr := getListingUploadChunkSize(tc.fileInfos, tc.srcDir, tc.maxChunkSize)
		if tc.wantErr {
			if gotErr == nil {
				t.Errorf("%s got nil error, want non-nil error", msgPrefix)
			}
			continue
		}

		if gotErr != nil {
			t.Errorf("%s got error %v", msgPrefix, gotErr)
			continue
		}

		if gotSize != tc.wantSize {
			t.Errorf("%s got list file size %d, want %d", msgPrefix, gotSize, tc.wantSize)
		}
	}
}
