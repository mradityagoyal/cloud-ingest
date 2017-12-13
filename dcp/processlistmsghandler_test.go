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

package dcp

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"strconv"
	"strings"
	"testing"

	"github.com/golang/mock/gomock"
)

func processListCompletionMessage() *TaskCompletionMessage {
	return &TaskCompletionMessage{
		TaskFullIDStr: testTaskFullIDStr,
		Status:        "SUCCESS",
		TaskParams: map[string]interface{}{
			"dst_list_result_bucket": "bucket1",
			"dst_list_result_object": "object",
			"src_directory":          "dir",
			"byte_offset":            0,
		},
		LogEntry: map[string]interface{}{"logkey": "logval"},
	}
}

func TestProcessListMessageHandlerInvalidCompletionMessage(t *testing.T) {
	handler := ProcessListMessageHandler{}

	taskCompletionMessage := processListCompletionMessage()
	taskCompletionMessage.TaskFullIDStr = "garbage"
	log.SetOutput(ioutil.Discard) // Suppress the log spam.
	_, err := handler.HandleMessage(nil, taskCompletionMessage)
	defer log.SetOutput(os.Stdout) // Reenable logging.
	if err == nil {
		t.Error("error is nil, expected error: can not parse full task id...")
	} else if !strings.Contains(err.Error(), "cannot parse") {
		t.Errorf("expected error: %s, found: %s.", "can not parse full task id",
			err.Error())
	}
}

func TestProcessListMessageHandlerFailReadingListResult(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	mockListReader := NewMockListingResultReader(mockCtrl)
	errorMsg := "Failed reading listing result."
	mockListReader.EXPECT().ReadLines(
		context.Background(), "bucket1", "object", int64(0), maxLinesToProcess).
		Return(nil, int64(0), errors.New(errorMsg))

	handler := ProcessListMessageHandler{
		ListingResultReader: mockListReader,
	}

	log.SetOutput(ioutil.Discard) // Suppress the log spam.
	_, err := handler.HandleMessage(nil, processListCompletionMessage())
	defer log.SetOutput(os.Stdout) // Reenable logging.
	if err == nil {
		t.Errorf("error is nil, expected error: %s.", errorMsg)
	} else if err.Error() != errorMsg {
		t.Errorf("expected error: %s, found: %s.", errorMsg, err.Error())
	}
}

func TestProcessListMessageHandlerSuccess(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	// Setup the ListingResultReader.
	mockListReader := NewMockListingResultReader(mockCtrl)
	var newBytesProcessed int64 = 123
	lines := []string{"dir/file0", "dir/file1"}
	mockListReader.EXPECT().ReadLines(
		context.Background(), "bucket1", "object", int64(0), maxLinesToProcess).
		Return(lines, newBytesProcessed, io.EOF)

	taskFullID := NewTaskFullID(testProjectID, testJobConfigID, testJobRunID, testTaskID)
	processListTask := &Task{
		TaskFullID: *taskFullID,
		TaskType:   processListTaskType,
		TaskSpec: `{
			"dst_list_result_bucket": "bucket1",
			"dst_list_result_object": "object",
			"src_directory": "dir",
			"byte_offset": 0
		}`,
		Status: Success,
	}

	handler := ProcessListMessageHandler{
		ListingResultReader: mockListReader,
	}

	jobSpec := &JobSpec{
		GCSBucket: "bucket2",
	}

	taskUpdate, err := handler.HandleMessage(jobSpec, processListCompletionMessage())
	if err != nil {
		t.Errorf("expecting success, found error: %v.", err)
	}

	expectedTaskUpdate := &TaskUpdate{
		Task: processListTask,
		LogEntry: LogEntry{
			"endingOffset":   newBytesProcessed,
			"linesProcessed": int64(2),
			"startingOffset": int64(0),
		},
		OriginalTaskParams: TaskParams{
			"byte_offset":            0,
			"dst_list_result_bucket": "bucket1",
			"dst_list_result_object": "object",
			"src_directory":          "dir",
		},
		TransactionalSemantics: ProcessListingFileSemantics{
			ExpectedByteOffset:         int64(0),
			ByteOffsetForNextIteration: newBytesProcessed,
		},
	}

	// No task spec on TaskUpdate.
	processListTask.TaskSpec = ""

	if len(taskUpdate.NewTasks) != 2 {
		t.Errorf("expecting 2 tasks, found %d.", len(taskUpdate.NewTasks))
	}

	for i := 0; i < 2; i++ {
		// Handle the task spec JSON separately, since it doesn't play well with equality checks.
		expectedNewTaskSpec := fmt.Sprintf(`{
				"dst_bucket": "bucket2",
				"dst_object": "file%d",
				"expected_generation_num": 0,
				"src_file": "dir/file%d"
			}`, i, i)

		if !AreEqualJSON(expectedNewTaskSpec, taskUpdate.NewTasks[i].TaskSpec) {
			t.Errorf("expected task spec: %s, found: %s", expectedNewTaskSpec, taskUpdate.NewTasks[i].TaskSpec)
		}

		// Blow it away.
		taskUpdate.NewTasks[i].TaskSpec = ""

		// Add task (sans spec) to our expected update.
		expectedNewTask := &Task{
			TaskFullID: TaskFullID{
				JobRunFullID: taskFullID.JobRunFullID,
				TaskID:       GetUploadGCSTaskID("dir/file" + strconv.Itoa(i)),
			},
			TaskType: uploadGCSTaskType,
		}

		expectedTaskUpdate.NewTasks = append(expectedTaskUpdate.NewTasks, expectedNewTask)
	}

	DeepEqualCompare("TaskUpdate", expectedTaskUpdate, taskUpdate, t)

	// Check pieces one at a time, for convenient visualization.
	DeepEqualCompare("TaskUpdate.Task", expectedTaskUpdate.Task, taskUpdate.Task, t)
	DeepEqualCompare("TaskUpdate.LogEntry", expectedTaskUpdate.LogEntry, taskUpdate.LogEntry, t)
	DeepEqualCompare("TaskUpdate.OriginalTaskParams",
		expectedTaskUpdate.OriginalTaskParams, taskUpdate.OriginalTaskParams, t)
	for i := 0; i < 2; i++ {
		DeepEqualCompare("NewTasks", expectedTaskUpdate.NewTasks[i], taskUpdate.NewTasks[i], t)
	}
}
