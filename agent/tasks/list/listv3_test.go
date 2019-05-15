/*
Copyright 2019 Google Inc. All Rights Reserved.
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

package list

import (
	"bytes"
	"context"
	"io"
	"os"
	"testing"

	"github.com/GoogleCloudPlatform/cloud-ingest/agent/gcloud"
	"github.com/GoogleCloudPlatform/cloud-ingest/agent/stats"
	"github.com/GoogleCloudPlatform/cloud-ingest/agent/tasks/common"
	"github.com/golang/mock/gomock"
	"github.com/golang/protobuf/proto"

	listfilepb "github.com/GoogleCloudPlatform/cloud-ingest/proto/listfile_go_proto"
	taskpb "github.com/GoogleCloudPlatform/cloud-ingest/proto/task_go_proto"
)

const (
	testBucket  = "bucket"
	unexplored  = "unexplored"
	testObject  = "object"
	fileContent = "0123456789"
)

func testListV3TaskReqMsg(taskRelRsrcName string, srcDirs []string, rootDir string) *taskpb.TaskReqMsg {
	return &taskpb.TaskReqMsg{
		TaskRelRsrcName: taskRelRsrcName,
		Spec: &taskpb.Spec{
			Spec: &taskpb.Spec_ListSpec{
				ListSpec: &taskpb.ListSpec{
					DstListResultBucket:     testBucket,
					DstListResultObject:     testObject,
					DstUnexploredDirsObject: unexplored,
					ExpectedGenerationNum:   0,
					SrcDirectories:          srcDirs,
					RootDirectory:           rootDir,
				},
			},
		},
	}
}

func createFile(t *testing.T, dir, prefix, content string) *listfilepb.ListFileEntry {
	t.Helper()
	file := common.CreateTmpFile(dir, prefix, content)
	fileInfo, err := os.Stat(file)
	if err != nil {
		t.Fatalf("got error: %v", err)
	}
	return fileInfoEntry(file, fileInfo.ModTime().Unix(), fileInfo.Size())
}

func sortAndWriteEntries(t *testing.T, w io.Writer, entries []*listfilepb.ListFileEntry) {
	t.Helper()
	if err := sortListFileEntries(entries); err != nil {
		t.Fatalf("got error: %v", err)
	}
	for _, entry := range entries {
		if err := writeProtobuf(w, entry); err != nil {
			t.Fatalf("writeProtobuf got error: %v", err)
		}
	}
}

func TestListV3DirAndSrcDirNotFound(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	listWriter := &common.StringWriteCloser{}
	mockGCS := gcloud.NewMockGCS(mockCtrl)
	mockGCS.EXPECT().NewWriterWithCondition(
		context.Background(), testBucket, testObject, gomock.Any()).Return(listWriter)
	ctx := context.Background()
	st := stats.NewTracker(ctx)
	h := ListHandlerV3{gcs: mockGCS, listFileSizeThreshold: 10000, allowedDirBytes: 5 * 1024 * 1024, statsTracker: st}
	taskReqMsg := testListV3TaskReqMsg("task", []string{"dir does not exist"}, "can't find me either")
	taskRespMsg := h.Do(context.Background(), taskReqMsg)
	CheckFailureWithType("task", taskpb.FailureType_SOURCE_DIR_NOT_FOUND, taskRespMsg, t)
	if listWriter.WrittenString() != "" {
		t.Errorf("expected nothing written to list file but found: %s", listWriter.WrittenString())
	}
}

func TestListV3DirInListSpecNotFound(t *testing.T) {
	// Test that the handler properly handles the case where one of the src_directories in the list
	// spec is not found.
	tmpDir := common.CreateTmpDir("", "test-list-agent-")
	notFoundDir := "dir was deleted"

	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	listWriter := &common.StringWriteCloser{}
	dirsWriter := &common.StringWriteCloser{}
	mockGCS := gcloud.NewMockGCS(mockCtrl)
	gomock.InOrder(
		mockGCS.EXPECT().NewWriterWithCondition(
			context.Background(), testBucket, testObject, gomock.Any()).Return(listWriter),
		mockGCS.EXPECT().NewWriterWithCondition(
			context.Background(), testBucket, unexplored, gomock.Any()).Return(dirsWriter),
	)
	ctx := context.Background()
	st := stats.NewTracker(ctx)
	h := ListHandlerV3{gcs: mockGCS, listFileSizeThreshold: 10000, allowedDirBytes: 5 * 1024 * 1024, statsTracker: st}
	taskRelRsrcName := "projects/project_A/jobConfigs/config_B/jobRuns/run_C/tasks/task_D"
	taskReqMsg := testListV3TaskReqMsg(taskRelRsrcName, []string{notFoundDir}, tmpDir)
	taskRespMsg := h.Do(context.Background(), taskReqMsg)
	CheckSuccessMsg(taskRelRsrcName, taskRespMsg, t)
	if listWriter.WrittenString() != "" {
		t.Errorf("got list file: \"%s\", want: \"%s\"",
			listWriter.WrittenString(), "")
	}
	if dirsWriter.WrittenString() != "" {
		t.Errorf("got unexplored dirs file: \"%s\", want: \"%s\"",
			dirsWriter.WrittenString(), "")
	}

	wantLog := &taskpb.Log{
		Log: &taskpb.Log_ListLog{
			ListLog: &taskpb.ListLog{
				DirsNotFound: []string{notFoundDir},
			},
		},
	}
	if !proto.Equal(taskRespMsg.Log, wantLog) {
		t.Errorf("log = %+v, want: %+v", taskRespMsg.Log, wantLog)
	}
}

func TestListV3DirFoundThenDeleted(t *testing.T) {
	// Test that the handler properly handles the case where:
	// - Directory is found
	// - Directory is deleted
	// - Task tries to list the directory that no longer exists
	taskRelRsrcName := "projects/project_A/jobConfigs/config_B/jobRuns/run_C/tasks/task_D"
	tmpDir := common.CreateTmpDir("", "test-list-agent-")
	taskReqMsg := testListV3TaskReqMsg(taskRelRsrcName, []string{}, tmpDir)

	listWriter := &common.StringWriteCloser{}
	dirStore := NewDirectoryInfoStore()
	err := dirStore.Add(listfilepb.DirectoryInfo{Path: "dir was deleted"})
	if err != nil {
		t.Fatalf("DirectoryInfoStore.Add() got error: %v", err)
	}
	// Have to test helper method to avoid race conditions.
	listMD, err := processDirectories(listWriter, dirStore, 10000, 500000, true, *taskReqMsg.Spec.GetListSpec())
	if err != nil {
		t.Errorf("processDirectories() got error %v", err)
	}
	if listMD.dirsDiscovered != -1 {
		t.Errorf("processDirectories() got listMD.dirsDiscovered = %v, want -1", listMD.dirsDiscovered)
	}
}

func TestListV3SuccessEmptyDir(t *testing.T) {
	tmpDir := common.CreateTmpDir("", "test-list-agent-")
	defer os.RemoveAll(tmpDir)

	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	listWriter := &common.StringWriteCloser{}
	dirsWriter := &common.StringWriteCloser{}
	mockGCS := gcloud.NewMockGCS(mockCtrl)
	gomock.InOrder(
		mockGCS.EXPECT().NewWriterWithCondition(
			context.Background(), testBucket, testObject, gomock.Any()).Return(listWriter),
		mockGCS.EXPECT().NewWriterWithCondition(
			context.Background(), testBucket, unexplored, gomock.Any()).Return(dirsWriter),
	)
	ctx := context.Background()
	st := stats.NewTracker(ctx)
	h := ListHandlerV3{gcs: mockGCS, listFileSizeThreshold: 10000, allowedDirBytes: 5 * 1024 * 1024, statsTracker: st}
	taskRelRsrcName := "projects/project_A/jobConfigs/config_B/jobRuns/run_C/tasks/task_D"
	taskReqMsg := testListV3TaskReqMsg(taskRelRsrcName, []string{tmpDir}, tmpDir)
	taskRespMsg := h.Do(context.Background(), taskReqMsg)
	CheckSuccessMsg(taskRelRsrcName, taskRespMsg, t)
	if listWriter.WrittenString() != "" {
		t.Errorf("got list file: \"%s\", want: \"%s\"",
			listWriter.WrittenString(), "")
	}
	if dirsWriter.WrittenString() != "" {
		t.Errorf("got unexplored dirs file: \"%s\", want: \"%s\"",
			dirsWriter.WrittenString(), "")
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

func TestListV3SuccessFlatDir(t *testing.T) {
	var expectedListResult bytes.Buffer

	tmpDir := common.CreateTmpDir("", "test-list-agent-")
	defer os.RemoveAll(tmpDir)

	var dirEntries []*listfilepb.ListFileEntry
	for i := 0; i < 10; i++ {
		dirEntries = append(dirEntries, createFile(t, tmpDir, "test-file-", fileContent))
	}
	sortAndWriteEntries(t, &expectedListResult, dirEntries)

	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	listWriter := &common.StringWriteCloser{}
	dirsWriter := &common.StringWriteCloser{}
	mockGCS := gcloud.NewMockGCS(mockCtrl)
	gomock.InOrder(
		mockGCS.EXPECT().NewWriterWithCondition(
			context.Background(), testBucket, testObject, gomock.Any()).Return(listWriter),
		mockGCS.EXPECT().NewWriterWithCondition(
			context.Background(), testBucket, unexplored, gomock.Any()).Return(dirsWriter),
	)
	ctx := context.Background()
	st := stats.NewTracker(ctx)
	h := ListHandlerV3{gcs: mockGCS, listFileSizeThreshold: 10000, allowedDirBytes: 5 * 1024 * 1024, statsTracker: st}
	taskRelRsrcName := "projects/project_A/jobConfigs/config_B/jobRuns/run_C/tasks/task_D"
	taskReqMsg := testListV3TaskReqMsg(taskRelRsrcName, []string{tmpDir}, tmpDir)
	taskRespMsg := h.Do(context.Background(), taskReqMsg)
	CheckSuccessMsg(taskRelRsrcName, taskRespMsg, t)
	if listWriter.WrittenString() != expectedListResult.String() {
		t.Errorf("got list file: \"%s\", want: \"%s\"",
			listWriter.WrittenString(), expectedListResult.String())
	}
	if dirsWriter.WrittenString() != "" {
		t.Errorf("got unexplored dirs file: \"%s\", want: \"%s\"",
			dirsWriter.WrittenString(), "")
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

func TestListV3FailsFileWithNewline(t *testing.T) {
	tmpDir := common.CreateTmpDir("", "test-list-agent-")
	defer os.RemoveAll(tmpDir)

	for i := 0; i < 10; i++ {
		common.CreateTmpFile(tmpDir, "test-file-", fileContent)
	}
	common.CreateTmpFile(tmpDir, "test-file-with-\n-newline", fileContent)

	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	writer := &common.StringWriteCloser{}
	mockGCS := gcloud.NewMockGCS(mockCtrl)
	mockGCS.EXPECT().NewWriterWithCondition(
		context.Background(), testBucket, testObject, gomock.Any()).Return(writer)
	ctx := context.Background()
	st := stats.NewTracker(ctx)
	h := ListHandlerV3{gcs: mockGCS, listFileSizeThreshold: 10000, allowedDirBytes: 5 * 1024 * 1024, statsTracker: st}
	taskRelRsrcName := "projects/project_A/jobConfigs/config_B/jobRuns/run_C/tasks/task_D"
	taskReqMsg := testListV3TaskReqMsg(taskRelRsrcName, []string{tmpDir}, tmpDir)
	taskRespMsg := h.Do(context.Background(), taskReqMsg)
	// TODO(b/111502687): Failing with UNKNOWN_FAILURE is temporary. In the long
	// term, we will escape file with newlines.
	CheckFailureWithType(taskRelRsrcName, taskpb.FailureType_UNKNOWN_FAILURE, taskRespMsg, t)
	if writer.WrittenString() != "" {
		t.Errorf("got list file: \"%s\", want: \"%s\"",
			writer.WrittenString(), "")
	}
}

func TestListV3SuccessNestedDirSmallListFile(t *testing.T) {
	var expectedListResult, expectedDirsResult bytes.Buffer

	tmpDir := common.CreateTmpDir("", "test-list-agent-")
	nestedTmpDir := common.CreateTmpDir(tmpDir, "sub-dir-")
	emptyDir := common.CreateTmpDir(tmpDir, "empty-dir-")
	defer os.RemoveAll(tmpDir)

	var dirEntries []*listfilepb.ListFileEntry
	dirEntries = append(dirEntries, dirInfoEntry(nestedTmpDir))
	dirEntries = append(dirEntries, dirInfoEntry(emptyDir))
	for i := 0; i < 10; i++ {
		dirEntries = append(dirEntries, createFile(t, tmpDir, "test-file-", fileContent))
	}
	sortAndWriteEntries(t, &expectedListResult, dirEntries)

	// Create some files in the sub-dir. These should not be in the list output.
	for i := 0; i < 10; i++ {
		common.CreateTmpFile(nestedTmpDir, "test-file-", fileContent)
	}

	// Add unexplored dirs to unexplored dirs file
	unexploredDirs := []*listfilepb.ListFileEntry{dirInfoEntry(nestedTmpDir), dirInfoEntry(emptyDir)}
	sortAndWriteEntries(t, &expectedDirsResult, unexploredDirs)

	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	listWriter := &common.StringWriteCloser{}
	dirsWriter := &common.StringWriteCloser{}
	mockGCS := gcloud.NewMockGCS(mockCtrl)
	gomock.InOrder(
		mockGCS.EXPECT().NewWriterWithCondition(
			context.Background(), testBucket, testObject, gomock.Any()).Return(listWriter),
		mockGCS.EXPECT().NewWriterWithCondition(
			context.Background(), testBucket, unexplored, gomock.Any()).Return(dirsWriter),
	)
	ctx := context.Background()
	st := stats.NewTracker(ctx)
	h := ListHandlerV3{gcs: mockGCS, listFileSizeThreshold: 1, allowedDirBytes: 5 * 1024 * 1024, statsTracker: st}
	taskRelRsrcName := "projects/project_A/jobConfigs/config_B/jobRuns/run_C/tasks/task_D"
	taskReqMsg := testListV3TaskReqMsg(taskRelRsrcName, []string{tmpDir}, tmpDir)
	taskRespMsg := h.Do(context.Background(), taskReqMsg)
	CheckSuccessMsg(taskRelRsrcName, taskRespMsg, t)
	if listWriter.WrittenString() != expectedListResult.String() {
		t.Errorf("got list file: \"%s\", want: \"%s\"",
			listWriter.WrittenString(), expectedListResult.String())
	}
	if dirsWriter.WrittenString() != expectedDirsResult.String() {
		t.Errorf("got unexplored dirs file: \"%s\", want: \"%s\"",
			dirsWriter.WrittenString(), expectedDirsResult.String())
	}

	wantLog := &taskpb.Log{
		Log: &taskpb.Log_ListLog{
			ListLog: &taskpb.ListLog{
				FilesFound:    10,
				BytesFound:    100,
				DirsFound:     2,
				DirsListed:    1,
				DirsNotListed: 2,
			},
		},
	}
	if !proto.Equal(taskRespMsg.Log, wantLog) {
		t.Errorf("log = %+v, want: %+v", taskRespMsg.Log, wantLog)
	}
}

func TestListV3SuccessNestedDirLargeListFile(t *testing.T) {
	listWriter := &common.StringWriteCloser{}
	dirsWriter := &common.StringWriteCloser{}

	taskRelRsrcName := "projects/project_A/jobConfigs/config_B/jobRuns/run_C/tasks/task_D"
	var expectedListResult, expectedDirsResult bytes.Buffer

	tmpDir := common.CreateTmpDir("", "test-list-agent-")
	nestedTmpDir := common.CreateTmpDir(tmpDir, "sub-dir-")
	emptyDir := common.CreateTmpDir(tmpDir, "empty-dir-")
	defer os.RemoveAll(tmpDir)

	var dirEntries []*listfilepb.ListFileEntry
	dirEntries = append(dirEntries, dirInfoEntry(nestedTmpDir))
	dirEntries = append(dirEntries, dirInfoEntry(emptyDir))
	for i := 0; i < 10; i++ {
		dirEntries = append(dirEntries, createFile(t, tmpDir, "test-file-", fileContent))
	}
	sortAndWriteEntries(t, &expectedListResult, dirEntries)

	// Create some files in the sub-dir.
	dirEntries = nil
	for i := 0; i < 10; i++ {
		dirEntries = append(dirEntries, createFile(t, nestedTmpDir, "test-file-", fileContent))
	}
	sortAndWriteEntries(t, &expectedListResult, dirEntries)

	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	mockGCS := gcloud.NewMockGCS(mockCtrl)
	gomock.InOrder(
		mockGCS.EXPECT().NewWriterWithCondition(
			context.Background(), testBucket, testObject, gomock.Any()).Return(listWriter),
		mockGCS.EXPECT().NewWriterWithCondition(
			context.Background(), testBucket, unexplored, gomock.Any()).Return(dirsWriter),
	)
	ctx := context.Background()
	st := stats.NewTracker(ctx)
	h := ListHandlerV3{gcs: mockGCS, listFileSizeThreshold: 1000, allowedDirBytes: 5 * 1024 * 1024, statsTracker: st}
	taskReqMsg := testListV3TaskReqMsg(taskRelRsrcName, []string{tmpDir}, tmpDir)
	taskRespMsg := h.Do(context.Background(), taskReqMsg)
	CheckSuccessMsg(taskRelRsrcName, taskRespMsg, t)
	if listWriter.WrittenString() != expectedListResult.String() {
		t.Errorf("got list file: \"%s\", want: \"%s\"",
			expectedListResult.String(), listWriter.WrittenString())
	}
	if dirsWriter.WrittenString() != expectedDirsResult.String() {
		t.Errorf("got unexplored dirs file: \"%s\", want: \"%s\"",
			expectedDirsResult.String(), dirsWriter.WrittenString())
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

func TestListV3MakesProgressWhenSrcDirsExceedsMemDirLimit(t *testing.T) {
	var expectedListResult, expectedDirsResult bytes.Buffer

	tmpDir := common.CreateTmpDir("", "test-list-agent-")
	defer os.RemoveAll(tmpDir)

	var dirEntries []*listfilepb.ListFileEntry
	for i := 0; i < 10; i++ {
		dirEntries = append(dirEntries, createFile(t, tmpDir, "test-file-", fileContent))
	}
	sortAndWriteEntries(t, &expectedListResult, dirEntries)

	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	listWriter := &common.StringWriteCloser{}
	dirsWriter := &common.StringWriteCloser{}
	mockGCS := gcloud.NewMockGCS(mockCtrl)
	gomock.InOrder(
		mockGCS.EXPECT().NewWriterWithCondition(
			context.Background(), testBucket, testObject, gomock.Any()).Return(listWriter),
		mockGCS.EXPECT().NewWriterWithCondition(
			context.Background(), testBucket, unexplored, gomock.Any()).Return(dirsWriter),
	)
	ctx := context.Background()
	st := stats.NewTracker(ctx)
	h := ListHandlerV3{gcs: mockGCS, listFileSizeThreshold: 10000, allowedDirBytes: 1, statsTracker: st}
	taskRelRsrcName := "projects/project_A/jobConfigs/config_B/jobRuns/run_C/tasks/task_D"
	taskReqMsg := testListV3TaskReqMsg(taskRelRsrcName, []string{tmpDir}, tmpDir)
	taskRespMsg := h.Do(context.Background(), taskReqMsg)
	CheckSuccessMsg(taskRelRsrcName, taskRespMsg, t)
	if listWriter.WrittenString() != expectedListResult.String() {
		t.Errorf("got list file: \"%s\", want: \"%s\"",
			expectedListResult.String(), listWriter.WrittenString())
	}
	if dirsWriter.WrittenString() != expectedDirsResult.String() {
		t.Errorf("got unexplored dirs file: \"%s\", want: \"%s\"",
			expectedDirsResult.String(), dirsWriter.WrittenString())
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

func TestListV3SuccessNestedDirSmallMemoryLimitListFile(t *testing.T) {
	var expectedListResult, expectedDirsResult bytes.Buffer

	tmpDir := common.CreateTmpDir("", "test-list-agent-")
	nestedTmpDir := common.CreateTmpDir(tmpDir, "sub-dir-")
	childOfNestedTmpDir := common.CreateTmpDir(nestedTmpDir, "sub-dir2-")
	child2OfNestedTmpDir := common.CreateTmpDir(nestedTmpDir, "sub-dir3-")
	defer os.RemoveAll(tmpDir)

	// Create contents of tmpDir, and add the child dir to the list of entries
	var dirEntries []*listfilepb.ListFileEntry
	dirEntries = append(dirEntries, dirInfoEntry(nestedTmpDir))
	for i := 0; i < 10; i++ {
		dirEntries = append(dirEntries, createFile(t, tmpDir, "test-file-", fileContent))
	}
	sortAndWriteEntries(t, &expectedListResult, dirEntries)

	// Create some files in the sub-dir and add them to the expected list file. Also add the child dirs.
	dirEntries = nil
	dirEntries = append(dirEntries, dirInfoEntry(childOfNestedTmpDir))
	dirEntries = append(dirEntries, dirInfoEntry(child2OfNestedTmpDir))
	for i := 0; i < 10; i++ {
		dirEntries = append(dirEntries, createFile(t, nestedTmpDir, "test-file-", fileContent))
	}
	sortAndWriteEntries(t, &expectedListResult, dirEntries)

	// Create some files in the sub-dir's child dir. These should not be in the list output.
	for i := 0; i < 10; i++ {
		common.CreateTmpFile(childOfNestedTmpDir, "test-file-", fileContent)
	}

	// Add unexplored dirs to unexplored dirs file
	unexploredDirs := []*listfilepb.ListFileEntry{dirInfoEntry(childOfNestedTmpDir), dirInfoEntry(child2OfNestedTmpDir)}
	sortAndWriteEntries(t, &expectedDirsResult, unexploredDirs)

	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	listWriter := &common.StringWriteCloser{}
	dirsWriter := &common.StringWriteCloser{}
	mockGCS := gcloud.NewMockGCS(mockCtrl)
	gomock.InOrder(
		mockGCS.EXPECT().NewWriterWithCondition(
			context.Background(), testBucket, testObject, gomock.Any()).Return(listWriter),
		mockGCS.EXPECT().NewWriterWithCondition(
			context.Background(), testBucket, unexplored, gomock.Any()).Return(dirsWriter),
	)
	ctx := context.Background()
	st := stats.NewTracker(ctx)
	h := ListHandlerV3{gcs: mockGCS, listFileSizeThreshold: 10000, allowedDirBytes: directoryInfoProtoOverhead*2 + len(childOfNestedTmpDir) + len(child2OfNestedTmpDir), statsTracker: st}
	taskRelRsrcName := "projects/project_A/jobConfigs/config_B/jobRuns/run_C/tasks/task_D"
	taskReqMsg := testListV3TaskReqMsg(taskRelRsrcName, []string{tmpDir}, tmpDir)
	taskRespMsg := h.Do(context.Background(), taskReqMsg)
	CheckSuccessMsg(taskRelRsrcName, taskRespMsg, t)
	if listWriter.WrittenString() != expectedListResult.String() {
		t.Errorf("got list file: \"%s\", want: \"%s\"",
			expectedListResult.String(), listWriter.WrittenString())
	}
	if dirsWriter.WrittenString() != expectedDirsResult.String() {
		t.Errorf("got unexplored dirs file: \"%s\", want: \"%s\"",
			expectedDirsResult.String(), dirsWriter.WrittenString())
	}

	wantLog := &taskpb.Log{
		Log: &taskpb.Log_ListLog{
			ListLog: &taskpb.ListLog{
				FilesFound:    20,
				BytesFound:    200,
				DirsFound:     3,
				DirsListed:    2,
				DirsNotListed: 2,
			},
		},
	}
	if !proto.Equal(taskRespMsg.Log, wantLog) {
		t.Errorf("log = %+v, want: %+v", taskRespMsg.Log, wantLog)
	}
}
