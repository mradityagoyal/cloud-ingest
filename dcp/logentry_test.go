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
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/GoogleCloudPlatform/cloud-ingest/gcloud"
	"github.com/GoogleCloudPlatform/cloud-ingest/helpers"
	"github.com/golang/mock/gomock"

	"cloud.google.com/go/spanner"
)

func createDummyTask() *Task {
	return &Task{
		TaskRRStruct: *NewTaskRRStruct(
			testProjectID, testJobConfigID, testJobRunID, testTaskID),
		Status:               Queued,
		CreationTime:         123,
		LastModificationTime: 234,
		FailureMessage:       "failure message A",
	}
}

func TestInsertLogEntryMutation(t *testing.T) {
	listTask := createDummyTask()
	previousStatus := Unqueued
	logEntry := make(LogEntry)
	logEntry["dummyKey"] = "dummyValue"
	timestamp := int64(1111)

	mutations := []*spanner.Mutation{}
	insertLogEntryMutation(&mutations, listTask, previousStatus, logEntry, timestamp)
	if len(mutations) != 1 {
		t.Errorf("Expected a single mutation, found %v.", len(mutations))

	}

	test_mutation := spanner.Insert("LogEntries", getLogEntryInsertColumns(),
		[]interface{}{
			testProjectID,
			testJobConfigID,
			testJobRunID,
			testTaskID,
			int64(9206468313283562545),
			timestamp,
			Queued,
			Unqueued,
			"failure message A",
			"dummyKey:dummyValue",
			false,
		})

	if !reflect.DeepEqual(mutations[0], test_mutation) {
		t.Errorf("Generated mutation doesn't match test mutation,\n%s\nvs\n%s\n",
			mutations[0], test_mutation)
	}
}

func getTestingFakeStoreAndLogPath(n int64) (*FakeStore, string) {
	fakestore := new(FakeStore)
	*fakestore = FakeStore{
		jobSpec: &JobSpec{
			GCSBucket: "dummy_bucket",
		},
	}
	// Create the bogus logEntryRows.
	for i := int64(0); i < n; i++ {
		taskRRStruct := NewTaskRRStruct(testProjectID, testJobConfigID,
			fmt.Sprint(testJobRunID, i), fmt.Sprint(testTaskID, i))
		fakestore.logEntryRows = append(fakestore.logEntryRows,
			&LogEntryRow{
				TaskRRStruct: *taskRRStruct,
				LogEntryID:   i,
				// This time corresponds to the time
				// 2009-11-10T15:00:00.000000000-08:00.
				CreationTime: 1257894000000000000 + (i * 150),
				Processed:    false,
			})
	}
	logPath := fmt.Sprintf("%s/logs/%s/2009-11-10T15:00:00.000000000-08:00.log",
		cloudIngestWorkingSpace, testJobConfigID)
	return fakestore, logPath
}

func TestContinuouslyProcessLogsTicker(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	store, logPath := getTestingFakeStoreAndLogPath(numLogsToFetchPerRun)
	var writer helpers.StringWriteCloser
	mockGcs := gcloud.NewMockGCS(mockCtrl)
	lep := LogEntryProcessor{mockGcs, store}

	// Verify starting conditions.
	if c, _ := store.GetNumUnprocessedLogs(); c != numLogsToFetchPerRun {
		t.Errorf("Expected %v unprocessed logs, found %d",
			numLogsToFetchPerRun, c)
	}
	if writer.NumberLines() != 0 {
		t.Errorf("Expected 0 written lines, found %d", writer.NumberLines())
	}

	mockGcs.EXPECT().NewWriter(context.Background(), "dummy_bucket",
		logPath).Return(&writer)

	mockTicker := helpers.NewMockTicker()
	testChannel := make(chan int)
	go lep.continuouslyProcessLogs(context.Background(), mockTicker, nil, testChannel)
	mockTicker.Tick()
	<-testChannel // Block on the completion of the periodicCheck.

	// Verify the log entries have been processed.
	if c, _ := store.GetNumUnprocessedLogs(); c != 0 {
		t.Errorf("Expected 0 unprocessed logs, found %d", c)
	}
	if writer.NumberLines() != numLogsToFetchPerRun {
		t.Errorf("Expected %v written lines, found %d",
			numLogsToFetchPerRun, writer.NumberLines())
	}
}

func TestContinuouslyProcessLogsNoProgress(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	store, logPath := getTestingFakeStoreAndLogPath(3)
	var writer helpers.StringWriteCloser
	mockGcs := gcloud.NewMockGCS(mockCtrl)
	lep := LogEntryProcessor{mockGcs, store}

	// Verify starting conditions.
	if c, _ := store.GetNumUnprocessedLogs(); c != 3 {
		t.Errorf("Expected 3 unprocessed logs, found %d", c)
	}
	if writer.NumberLines() != 0 {
		t.Errorf("Expected 0 written lines, found %d", writer.NumberLines())
	}

	mockGcs.EXPECT().NewWriter(context.Background(), "dummy_bucket",
		logPath).Return(&writer)

	mockTicker := helpers.NewMockTicker()
	testChannel := make(chan int)
	go lep.continuouslyProcessLogs(context.Background(), mockTicker, nil, testChannel)
	mockTicker.Tick()
	<-testChannel // Block on the completion of the periodicCheck.

	// Verify that no entries have been processed.
	if c, _ := store.GetNumUnprocessedLogs(); c != 3 {
		t.Errorf("Expected 3 unprocessed logs, found %d", c)
	}

	// Perform enough checks to simulate a no-progress situation.
	for i := int64(0); i < maxNoProgressCount; i++ {
		mockTicker.Tick()
		<-testChannel // Block on the completion of the periodicCheck.
	}

	// Verify the log entries have been processed.
	if c, _ := store.GetNumUnprocessedLogs(); c != 0 {
		t.Errorf("Expected 0 unprocessed logs, found %d", c)
	}
	if writer.NumberLines() != 3 {
		t.Errorf("Expected 3 written lines, found %d", writer.NumberLines())
	}
}

func TestContinuouslyProcessLogsJobRunNotification(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	store, logPath := getTestingFakeStoreAndLogPath(3)
	var writer helpers.StringWriteCloser
	mockGcs := gcloud.NewMockGCS(mockCtrl)
	lep := LogEntryProcessor{mockGcs, store}

	// Verify starting conditions.
	if c, _ := store.GetNumUnprocessedLogs(); c != 3 {
		t.Errorf("Expected 3 unprocessed logs, found %d", c)
	}
	if writer.NumberLines() != 0 {
		t.Errorf("Expected 0 written lines, found %d", writer.NumberLines())
	}

	mockGcs.EXPECT().NewWriter(context.Background(), "dummy_bucket",
		logPath).Return(&writer)

	mockTicker := helpers.NewMockTicker()
	jobrunChannel := make(chan int)
	testChannel := make(chan int)
	go lep.continuouslyProcessLogs(context.Background(), mockTicker, jobrunChannel, testChannel)

	// Perform a bunch of ticks, but not enough to trigger 'no-progress'.
	for i := int64(0); i < maxNoProgressCount/2; i++ {
		mockTicker.Tick()
		<-testChannel // Block on the completion of the periodicCheck.
	}

	// Verify that no entries have been processed.
	if c, _ := store.GetNumUnprocessedLogs(); c != 3 {
		t.Errorf("Expected 3 unprocessed logs, found %d", c)
	}

	// Trigger logs processing by sending on the jobrunChannel.
	jobrunChannel <- 0
	<-testChannel // Block on the completion of the periodicCheck.

	// Verify the log entries have been processed.
	if c, _ := store.GetNumUnprocessedLogs(); c != 0 {
		t.Errorf("Expected 0 unprocessed logs, found %d", c)
	}
	if writer.NumberLines() != 3 {
		t.Errorf("Expected 3 written lines, found %d", writer.NumberLines())
	}
}

func TestSingleLogsProcessingRun(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	store, logPath := getTestingFakeStoreAndLogPath(3)
	var writer helpers.StringWriteCloser
	mockGcs := gcloud.NewMockGCS(mockCtrl)
	lep := LogEntryProcessor{mockGcs, store}

	// Verify starting conditions.
	if c, _ := store.GetNumUnprocessedLogs(); c != 3 {
		t.Errorf("Expected 3 unprocessed logs, found %d", c)
	}
	if writer.NumberLines() != 0 {
		t.Errorf("Expected 0 written lines, found %d", writer.NumberLines())
	}

	mockGcs.EXPECT().NewWriter(context.Background(), "dummy_bucket",
		logPath).Return(&writer)
	lep.SingleLogsProcessingRun(context.Background(), 1) // Process a single log entry.

	// Verify a single log entry has been processed.
	if c, _ := store.GetNumUnprocessedLogs(); c != 2 {
		t.Errorf("Expected 2 unprocessed logs, found %d", c)
	}
	if writer.NumberLines() != 1 {
		t.Errorf("Expected 1 written line, found %d", writer.NumberLines())
	}

	mockGcs.EXPECT().NewWriter(context.Background(), "dummy_bucket",
		"cloud-ingest/logs/job_config_id_A/2009-11-10T15:00:00.000000150-08:00.log").
		Return(&writer)
	lep.SingleLogsProcessingRun(context.Background(), 100) // Process all (two) remaining log entries.

	// Verify all remaining log entries have been processed.
	if c, _ := store.GetNumUnprocessedLogs(); c != 0 {
		t.Errorf("Expected 0 unprocessed logs, found %d", c)
	}
	if writer.NumberLines() != 3 {
		t.Errorf("Expected 3 written lines, found %d", writer.NumberLines())
	}
}

func TestSanitizeFailureMessage(t *testing.T) {
	s := "This test's string is unsanitized.\nHow shameful!\n\n'!'\n"
	s = sanitizeFailureMessage(s)
	if s != "This test`s string is unsanitized.\\nHow shameful!\\n\\n`!`\\n" {
		t.Errorf("String not correctly sanitized:", s)
	}
}

func TestLogEntryRowStringer(t *testing.T) {
	taskRRStruct := NewTaskRRStruct("UNUSED", "UNUSED", "UNUSED", "task_id")

	l := LogEntryRow{
		TaskRRStruct:   *taskRRStruct,
		LogEntryID:     0,
		CreationTime:   1257894000000000000,
		CurrentStatus:  3,
		PreviousStatus: 1,
		FailureMessage: "failure_message'\n",
		LogEntry:       "key1:value1 key2:value2",
		Processed:      false,
	}
	lString := l.String()
	lExpectedString := "2009-11-10T15:00:00.000000000-08:00 task_id QUEUED->SUCCESS FailureMessage:'failure_message`\\n' WorkerLog:'key1:value1 key2:value2'"
	if lString != lExpectedString {
		t.Errorf("Expected l.String to be\n%s\ninstead got\n%s", lExpectedString, lString)
	}
	if count := strings.Count(lString, "UNUSED"); count > 0 {
		t.Errorf("Expected no instances of 'UNUSED' in log entry string, found:", count)
	}
}
