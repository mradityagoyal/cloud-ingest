/*
Copyright 2018 Google Inc. All Rights Reserved.
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
	"os"
	"sort"
	"testing"

	"github.com/GoogleCloudPlatform/cloud-ingest/gcloud"
	"github.com/GoogleCloudPlatform/cloud-ingest/helpers"
	"github.com/golang/mock/gomock"
	"github.com/golang/protobuf/proto"

	listpb "github.com/GoogleCloudPlatform/cloud-ingest/proto/listfile_go_proto"
	taskpb "github.com/GoogleCloudPlatform/cloud-ingest/proto/task_go_proto"
)

func testDepthFirstListTaskReqMsg(taskRelRsrcName string, srcDirs []string) *taskpb.TaskReqMsg {
	return &taskpb.TaskReqMsg{
		TaskRelRsrcName: taskRelRsrcName,
		Spec: &taskpb.Spec{
			Spec: &taskpb.Spec_ListSpec{
				ListSpec: &taskpb.ListSpec{
					DstListResultBucket:   "bucket",
					DstListResultObject:   "object",
					ExpectedGenerationNum: 0,
					SrcDirectories:        srcDirs,
				},
			},
		},
	}
}

func TestDepthFirstListDirNotFound(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	writer := helpers.NewStringWriteCloser(nil)
	mockGCS := gcloud.NewMockGCS(mockCtrl)
	mockGCS.EXPECT().NewWriterWithCondition(
		context.Background(), "bucket", "object", gomock.Any()).Return(writer)

	h := DepthFirstListHandler{gcs: mockGCS, listFileSizeThreshold: 10000, allowedDirBytes: 5 * 1024 * 1024}
	taskReqParams := testDepthFirstListTaskReqMsg("task", []string{"dir does not exist"})
	taskRespMsg := h.Do(context.Background(), taskReqParams)
	checkFailureWithType("task", taskpb.FailureType_FILE_NOT_FOUND_FAILURE, taskRespMsg, t)
	if writer.WrittenString() != "" {
		t.Errorf("expected nothing written but found: %s", writer.WrittenString())
	}
}

func TestDepthFirstListSuccessEmptyDir(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	writer := &helpers.StringWriteCloser{}

	taskRelRsrcName := "projects/project_A/jobConfigs/config_B/jobRuns/run_C/tasks/task_D"
	var expectedListResult bytes.Buffer

	tmpDir := helpers.CreateTmpDir("", "test-list-agent-")
	defer os.RemoveAll(tmpDir)

	mockGCS := gcloud.NewMockGCS(mockCtrl)
	mockGCS.EXPECT().NewWriterWithCondition(
		context.Background(), "bucket", "object", gomock.Any()).Return(writer)

	h := DepthFirstListHandler{gcs: mockGCS, listFileSizeThreshold: 10000, allowedDirBytes: 5 * 1024 * 1024}
	taskReqParams := testDepthFirstListTaskReqMsg(taskRelRsrcName, []string{tmpDir})
	taskRespMsg := h.Do(context.Background(), taskReqParams)
	checkSuccessMsg(taskRelRsrcName, taskRespMsg, t)
	if writer.WrittenString() != expectedListResult.String() {
		t.Errorf("expected to write \"%s\", found: \"%s\"",
			expectedListResult.String(), writer.WrittenString())
	}

	wantLog := &taskpb.Log{
		Log: &taskpb.Log_ListLog{
			ListLog: &taskpb.ListLog{
				DirsListed: 1,
			},
		},
	}
	if !proto.Equal(taskRespMsg.Log, wantLog) {
		t.Errorf("log = %+v, want: %+v", taskRespMsg.Log, wantLog)
	}
}

func TestDepthFirstListSuccessFlatDir(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	writer := &helpers.StringWriteCloser{}

	taskRelRsrcName := "projects/project_A/jobConfigs/config_B/jobRuns/run_C/tasks/task_D"
	var expectedListResult bytes.Buffer

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
		fileInfo, err := os.Stat(path)
		if err != nil {
			t.Fatalf("got error: %v", err)
		}
		entry := listpb.ListFileEntry{Entry: &listpb.ListFileEntry_FileInfo{FileInfo: &listpb.FileInfo{Path: path, LastModifiedTime: fileInfo.ModTime().Unix(), Size: fileInfo.Size()}}}
		writeProtobuf(&expectedListResult, &entry)
	}

	mockGCS := gcloud.NewMockGCS(mockCtrl)
	mockGCS.EXPECT().NewWriterWithCondition(
		context.Background(), "bucket", "object", gomock.Any()).Return(writer)

	h := DepthFirstListHandler{gcs: mockGCS, listFileSizeThreshold: 10000, allowedDirBytes: 5 * 1024 * 1024}
	taskReqParams := testDepthFirstListTaskReqMsg(taskRelRsrcName, []string{tmpDir})
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
				DirsListed: 1,
			},
		},
	}
	if !proto.Equal(taskRespMsg.Log, wantLog) {
		t.Errorf("log = %+v, want: %+v", taskRespMsg.Log, wantLog)
	}
}

func TestDepthFirstListFailsFileWithNewline(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	writer := &helpers.StringWriteCloser{}

	taskRelRsrcName := "projects/project_A/jobConfigs/config_B/jobRuns/run_C/tasks/task_D"
	var expectedListResult bytes.Buffer

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
		fileInfo, err := os.Stat(path)
		if err != nil {
			t.Fatalf("got error: %v", err)
		}
		entry := listpb.ListFileEntry{Entry: &listpb.ListFileEntry_FileInfo{FileInfo: &listpb.FileInfo{Path: path, LastModifiedTime: fileInfo.ModTime().Unix(), Size: fileInfo.Size()}}}
		writeProtobuf(&expectedListResult, &entry)
	}

	mockGCS := gcloud.NewMockGCS(mockCtrl)
	mockGCS.EXPECT().NewWriterWithCondition(
		context.Background(), "bucket", "object", gomock.Any()).Return(writer)

	h := DepthFirstListHandler{gcs: mockGCS, listFileSizeThreshold: 10000, allowedDirBytes: 5 * 1024 * 1024}
	taskReqParams := testDepthFirstListTaskReqMsg(taskRelRsrcName, []string{tmpDir})
	taskRespMsg := h.Do(context.Background(), taskReqParams)
	// TODO(b/111502687): Failing with UNKNOWN_FAILURE is temporary. In the long
	// term, we will escape file with newlines.
	checkFailureWithType(taskRelRsrcName, taskpb.FailureType_UNKNOWN_FAILURE, taskRespMsg, t)
	if writer.WrittenString() != "" {
		t.Errorf("expected nothing written but found: %s", writer.WrittenString())
	}
}

func TestDepthFirstListSuccessNestedDirSmallListFile(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	writer := &helpers.StringWriteCloser{}

	taskRelRsrcName := "projects/project_A/jobConfigs/config_B/jobRuns/run_C/tasks/task_D"
	var expectedListResult bytes.Buffer

	tmpDir := helpers.CreateTmpDir("", "test-list-agent-")
	nestedTmpDir := helpers.CreateTmpDir(tmpDir, "sub-dir-")
	emptyDir := helpers.CreateTmpDir(tmpDir, "empty-dir-")
	defer os.RemoveAll(tmpDir)

	fileContent := "0123456789"
	filePaths := make([]string, 0)

	for i := 0; i < 10; i++ {
		filePaths = append(filePaths, helpers.CreateTmpFile(tmpDir, "test-file-", fileContent))
	}
	// The results of the list are in sorted order.
	sort.Strings(filePaths)
	for _, path := range filePaths {
		fileInfo, err := os.Stat(path)
		if err != nil {
			t.Fatalf("got error: %v", err)
		}
		entry := listpb.ListFileEntry{Entry: &listpb.ListFileEntry_FileInfo{FileInfo: &listpb.FileInfo{Path: path, LastModifiedTime: fileInfo.ModTime().Unix(), Size: fileInfo.Size()}}}
		writeProtobuf(&expectedListResult, &entry)
	}
	// Create some files in the sub-dir. These should not be in the list output.
	for i := 0; i < 10; i++ {
		filePaths = append(filePaths, helpers.CreateTmpFile(nestedTmpDir, "test-file-", fileContent))
	}

	// Add unexplored dirs to list file
	entry := listpb.ListFileEntry{Entry: &listpb.ListFileEntry_DirectoryInfo{DirectoryInfo: &listpb.DirectoryInfo{Path: emptyDir}}}
	writeProtobuf(&expectedListResult, &entry)
	entry = listpb.ListFileEntry{Entry: &listpb.ListFileEntry_DirectoryInfo{DirectoryInfo: &listpb.DirectoryInfo{Path: nestedTmpDir}}}
	writeProtobuf(&expectedListResult, &entry)

	mockGCS := gcloud.NewMockGCS(mockCtrl)
	mockGCS.EXPECT().NewWriterWithCondition(
		context.Background(), "bucket", "object", gomock.Any()).Return(writer)

	h := DepthFirstListHandler{gcs: mockGCS, listFileSizeThreshold: 1, allowedDirBytes: 5 * 1024 * 1024}
	taskReqParams := testDepthFirstListTaskReqMsg(taskRelRsrcName, []string{tmpDir})
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
				DirsListed: 1,
			},
		},
	}
	if !proto.Equal(taskRespMsg.Log, wantLog) {
		t.Errorf("log = %+v, want: %+v", taskRespMsg.Log, wantLog)
	}
}

func TestDepthFirstListSuccessNestedDirLargeListFile(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	writer := &helpers.StringWriteCloser{}

	taskRelRsrcName := "projects/project_A/jobConfigs/config_B/jobRuns/run_C/tasks/task_D"
	var expectedListResult bytes.Buffer

	tmpDir := helpers.CreateTmpDir("", "test-list-agent-")
	nestedTmpDir := helpers.CreateTmpDir(tmpDir, "sub-dir-")
	helpers.CreateTmpDir(tmpDir, "empty-dir-")
	defer os.RemoveAll(tmpDir)

	fileContent := "0123456789"
	filePaths := make([]string, 0)

	for i := 0; i < 10; i++ {
		filePaths = append(filePaths, helpers.CreateTmpFile(tmpDir, "test-file-", fileContent))
	}
	// The results of the list are in sorted order.
	sort.Strings(filePaths)
	for _, path := range filePaths {
		fileInfo, err := os.Stat(path)
		if err != nil {
			t.Fatalf("got error: %v", err)
		}
		entry := listpb.ListFileEntry{Entry: &listpb.ListFileEntry_FileInfo{FileInfo: &listpb.FileInfo{Path: path, LastModifiedTime: fileInfo.ModTime().Unix(), Size: fileInfo.Size()}}}
		writeProtobuf(&expectedListResult, &entry)
	}

	// Create some files in the sub-dir.
	filePaths = make([]string, 0)
	for i := 0; i < 10; i++ {
		filePaths = append(filePaths, helpers.CreateTmpFile(nestedTmpDir, "test-file-", fileContent))
	}
	// The results of the list are in sorted order.
	sort.Strings(filePaths)
	for _, path := range filePaths {
		fileInfo, err := os.Stat(path)
		if err != nil {
			t.Fatalf("got error: %v", err)
		}
		entry := listpb.ListFileEntry{Entry: &listpb.ListFileEntry_FileInfo{FileInfo: &listpb.FileInfo{Path: path, LastModifiedTime: fileInfo.ModTime().Unix(), Size: fileInfo.Size()}}}
		writeProtobuf(&expectedListResult, &entry)
	}

	mockGCS := gcloud.NewMockGCS(mockCtrl)
	mockGCS.EXPECT().NewWriterWithCondition(
		context.Background(), "bucket", "object", gomock.Any()).Return(writer)

	h := DepthFirstListHandler{gcs: mockGCS, listFileSizeThreshold: 1000, allowedDirBytes: 5 * 1024 * 1024}
	taskReqParams := testDepthFirstListTaskReqMsg(taskRelRsrcName, []string{tmpDir})
	taskRespMsg := h.Do(context.Background(), taskReqParams)
	checkSuccessMsg(taskRelRsrcName, taskRespMsg, t)
	if writer.WrittenString() != expectedListResult.String() {
		t.Errorf("expected to write \"%s\", found: \"%s\"",
			expectedListResult.String(), writer.WrittenString())
	}

	wantLog := &taskpb.Log{
		Log: &taskpb.Log_ListLog{
			ListLog: &taskpb.ListLog{
				FilesFound: 20,
				BytesFound: 200,
				DirsFound:  2,
				DirsListed: 3,
			},
		},
	}
	if !proto.Equal(taskRespMsg.Log, wantLog) {
		t.Errorf("log = %+v, want: %+v", taskRespMsg.Log, wantLog)
	}
}

func TestDepthFirstListMakesProgressWhenSrcDirsExceedsMemDirLimit(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	writer := &helpers.StringWriteCloser{}

	taskRelRsrcName := "projects/project_A/jobConfigs/config_B/jobRuns/run_C/tasks/task_D"
	var expectedListResult bytes.Buffer

	tmpDir := helpers.CreateTmpDir("", "test-list-agent-")
	defer os.RemoveAll(tmpDir)

	fileContent := "0123456789"
	filePaths := make([]string, 0)

	for i := 0; i < 10; i++ {
		filePaths = append(filePaths, helpers.CreateTmpFile(tmpDir, "test-file-", fileContent))
	}
	// The results of the list are in sorted order.
	sort.Strings(filePaths)
	for _, path := range filePaths {
		fileInfo, err := os.Stat(path)
		if err != nil {
			t.Fatalf("got error: %v", err)
		}
		entry := listpb.ListFileEntry{Entry: &listpb.ListFileEntry_FileInfo{FileInfo: &listpb.FileInfo{Path: path, LastModifiedTime: fileInfo.ModTime().Unix(), Size: fileInfo.Size()}}}
		writeProtobuf(&expectedListResult, &entry)
	}

	mockGCS := gcloud.NewMockGCS(mockCtrl)
	mockGCS.EXPECT().NewWriterWithCondition(
		context.Background(), "bucket", "object", gomock.Any()).Return(writer)

	h := DepthFirstListHandler{gcs: mockGCS, listFileSizeThreshold: 10000, allowedDirBytes: 1}
	taskReqParams := testDepthFirstListTaskReqMsg(taskRelRsrcName, []string{tmpDir})
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
				DirsFound:  0,
				DirsListed: 1,
			},
		},
	}
	if !proto.Equal(taskRespMsg.Log, wantLog) {
		t.Errorf("log = %+v, want: %+v", taskRespMsg.Log, wantLog)
	}
}

func TestDepthFirstListSuccessNestedDirSmallMemoryLimitListFile(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	writer := &helpers.StringWriteCloser{}

	taskRelRsrcName := "projects/project_A/jobConfigs/config_B/jobRuns/run_C/tasks/task_D"
	var expectedListResult bytes.Buffer

	tmpDir := helpers.CreateTmpDir("", "test-list-agent-")
	nestedTmpDir := helpers.CreateTmpDir(tmpDir, "sub-dir-")
	childOfNestedTmpDir := helpers.CreateTmpDir(nestedTmpDir, "sub-dir2-")
	child2OfNestedTmpDir := helpers.CreateTmpDir(nestedTmpDir, "sub-dir3-")
	defer os.RemoveAll(tmpDir)

	fileContent := "0123456789"
	filePaths := make([]string, 0)

	for i := 0; i < 10; i++ {
		filePaths = append(filePaths, helpers.CreateTmpFile(tmpDir, "test-file-", fileContent))
	}
	// The results of the list are in sorted order.
	sort.Strings(filePaths)
	for _, path := range filePaths {
		fileInfo, err := os.Stat(path)
		if err != nil {
			t.Fatalf("got error: %v", err)
		}
		entry := listpb.ListFileEntry{Entry: &listpb.ListFileEntry_FileInfo{FileInfo: &listpb.FileInfo{Path: path, LastModifiedTime: fileInfo.ModTime().Unix(), Size: fileInfo.Size()}}}
		writeProtobuf(&expectedListResult, &entry)
	}

	// Create some files in the sub-dir and add them to the expected list file.
	filePaths = make([]string, 0)
	for i := 0; i < 10; i++ {
		filePaths = append(filePaths, helpers.CreateTmpFile(nestedTmpDir, "test-file-", fileContent))
	}
	// The results of the list are in sorted order.
	sort.Strings(filePaths)
	for _, path := range filePaths {
		fileInfo, err := os.Stat(path)
		if err != nil {
			t.Fatalf("got error: %v", err)
		}
		entry := listpb.ListFileEntry{Entry: &listpb.ListFileEntry_FileInfo{FileInfo: &listpb.FileInfo{Path: path, LastModifiedTime: fileInfo.ModTime().Unix(), Size: fileInfo.Size()}}}
		writeProtobuf(&expectedListResult, &entry)
	}

	// Create some files in the sub-dir's child dir. These should not be in the list output.
	for i := 0; i < 10; i++ {
		helpers.CreateTmpFile(childOfNestedTmpDir, "test-file-", fileContent)
	}

	// Add unexplored dirs to list file
	entry := listpb.ListFileEntry{Entry: &listpb.ListFileEntry_DirectoryInfo{DirectoryInfo: &listpb.DirectoryInfo{Path: childOfNestedTmpDir}}}
	writeProtobuf(&expectedListResult, &entry)
	entry = listpb.ListFileEntry{Entry: &listpb.ListFileEntry_DirectoryInfo{DirectoryInfo: &listpb.DirectoryInfo{Path: child2OfNestedTmpDir}}}
	writeProtobuf(&expectedListResult, &entry)

	mockGCS := gcloud.NewMockGCS(mockCtrl)
	mockGCS.EXPECT().NewWriterWithCondition(
		context.Background(), "bucket", "object", gomock.Any()).Return(writer)

	h := DepthFirstListHandler{gcs: mockGCS, listFileSizeThreshold: 10000, allowedDirBytes: directoryInfoProtoOverhead*2 + len(childOfNestedTmpDir) + len(child2OfNestedTmpDir)}
	taskReqParams := testDepthFirstListTaskReqMsg(taskRelRsrcName, []string{tmpDir})
	taskRespMsg := h.Do(context.Background(), taskReqParams)
	checkSuccessMsg(taskRelRsrcName, taskRespMsg, t)
	if writer.WrittenString() != expectedListResult.String() {
		t.Errorf("expected to write \"%s\", found: \"%s\"",
			expectedListResult.String(), writer.WrittenString())
	}

	wantLog := &taskpb.Log{
		Log: &taskpb.Log_ListLog{
			ListLog: &taskpb.ListLog{
				FilesFound: 20,
				BytesFound: 200,
				DirsFound:  3,
				DirsListed: 2,
			},
		},
	}
	if !proto.Equal(taskRespMsg.Log, wantLog) {
		t.Errorf("log = %+v, want: %+v", taskRespMsg.Log, wantLog)
	}
}
