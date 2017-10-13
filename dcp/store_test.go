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
	"testing"
)

/*******************************************************************************
addJobProgressDeltaForTaskInsertsToMap Tests
*******************************************************************************/

func TestAddDeltaToMapOneInsertSingleJob(t *testing.T) {
	fullJobId := JobRunFullId{
		JobConfigId: "A",
		JobRunId:    "B",
	}
	task := &Task{
		JobConfigId: fullJobId.JobConfigId,
		JobRunId:    fullJobId.JobRunId,
	}

	progressDeltas := make(map[JobRunFullId]*JobProgressDelta)

	addJobProgressDeltaForTaskInsertsToMap([]*Task{task}, progressDeltas)

	if len(progressDeltas) != 1 {
		t.Errorf("expected progressDeltas to contain 1 delta, found %d",
			len(progressDeltas))
	}

	delta, exists := progressDeltas[fullJobId]
	if !exists {
		t.Errorf("expected progressDeltas to contain a delta for id %+v", fullJobId)
	}

	if delta.NewTasks != 1 {
		t.Errorf("expected delta.NewTasks to be 1, found %d", delta.NewTasks)
	}

	assertOtherDeltaFieldsUnchangedForInsert(t, delta)
}

func TestAddDeltaToMapMultipleInsertsSingleJob(t *testing.T) {
	fullJobId := JobRunFullId{
		JobConfigId: "A",
		JobRunId:    "B",
	}

	task1 := &Task{
		JobConfigId: fullJobId.JobConfigId,
		JobRunId:    fullJobId.JobRunId,
	}

	task2 := &Task{
		JobConfigId: fullJobId.JobConfigId,
		JobRunId:    fullJobId.JobRunId,
	}

	task3 := &Task{
		JobConfigId: fullJobId.JobConfigId,
		JobRunId:    fullJobId.JobRunId,
	}

	progressDeltas := make(map[JobRunFullId]*JobProgressDelta)

	addJobProgressDeltaForTaskInsertsToMap([]*Task{task1, task2, task3},
		progressDeltas)

	if len(progressDeltas) != 1 {
		t.Errorf("expected progressDeltas to contain 1 delta, found %d",
			len(progressDeltas))
	}

	delta, exists := progressDeltas[fullJobId]
	if !exists {
		t.Errorf("expected progressDeltas to contain a delta for id %+v", fullJobId)
	}

	if delta.NewTasks != 3 {
		t.Errorf("expected delta.NewTasks to be 3, found %d", delta.NewTasks)
	}

	assertOtherDeltaFieldsUnchangedForInsert(t, delta)
}

func TestAddDeltaToMapMultipleInsertsMultipleJobs(t *testing.T) {
	id1 := JobRunFullId{
		JobConfigId: "A",
		JobRunId:    "B",
	}

	id2 := JobRunFullId{
		JobConfigId: "C",
		JobRunId:    "B",
	}

	task1 := &Task{
		JobConfigId: id1.JobConfigId,
		JobRunId:    id1.JobRunId,
	}

	task2 := &Task{
		JobConfigId: id1.JobConfigId,
		JobRunId:    id1.JobRunId,
	}

	task3 := &Task{
		JobConfigId: id1.JobConfigId,
		JobRunId:    id1.JobRunId,
	}

	task4 := &Task{
		JobConfigId: id2.JobConfigId,
		JobRunId:    id2.JobRunId,
	}

	task5 := &Task{
		JobConfigId: id2.JobConfigId,
		JobRunId:    id2.JobRunId,
	}

	progressDeltas := make(map[JobRunFullId]*JobProgressDelta)

	addJobProgressDeltaForTaskInsertsToMap(
		[]*Task{task1, task2, task3, task4, task5}, progressDeltas)

	if len(progressDeltas) != 2 {
		t.Errorf("expected progressDeltas to contain 2 deltas, found %d",
			len(progressDeltas))
	}

	delta1, exists := progressDeltas[id1]
	if !exists {
		t.Errorf("expected progressDeltas to contain a delta for id %+v", id1)
	}

	if delta1.NewTasks != 3 {
		t.Errorf("expected delta.NewTasks to be 3, found %d", delta1.NewTasks)
	}

	assertOtherDeltaFieldsUnchangedForInsert(t, delta1)

	delta2, exists := progressDeltas[id2]
	if !exists {
		t.Errorf("expected progressDeltas to contain a delta for id %+v", id2)
	}

	if delta2.NewTasks != 2 {
		t.Errorf("expected delta.NewTasks to be 2, found %d", delta2.NewTasks)
	}

	assertOtherDeltaFieldsUnchangedForInsert(t, delta2)
}

func assertOtherDeltaFieldsUnchangedForInsert(t *testing.T,
	delta *JobProgressDelta) {
	if delta.TasksCompleted != 0 {
		t.Errorf("expected delta.TasksCompleted to be 0, found %d",
			delta.TasksCompleted)
	}
	if delta.TasksFailed != 0 {
		t.Errorf("expected delta.TasksFailed to be 0, found %d",
			delta.TasksFailed)
	}
}

/*******************************************************************************
addJobProgressDeltaForTaskUpdateToMap Tests
*******************************************************************************/

func TestAddDeltaToMapUpdateQueuedToSuccess(t *testing.T) {
	fullJobId := JobRunFullId{
		JobConfigId: "A",
		JobRunId:    "B",
	}
	task := &Task{
		JobConfigId: fullJobId.JobConfigId,
		JobRunId:    fullJobId.JobRunId,
		Status:      Success,
	}

	progressDeltas := make(map[JobRunFullId]*JobProgressDelta)

	addJobProgressDeltaForTaskUpdateToMap(task, Queued, progressDeltas)

	if len(progressDeltas) != 1 {
		t.Errorf("expected progressDeltas to contain 1 delta, found %d",
			len(progressDeltas))
	}

	delta, exists := progressDeltas[fullJobId]
	if !exists {
		t.Errorf("expected progressDeltas to contain a delta for id %+v", fullJobId)
	}

	if delta.TasksCompleted != 1 {
		t.Errorf("expected delta.TasksCompleted to be 1, found %d", delta.TasksCompleted)
	}

	if delta.TasksFailed != 0 {
		t.Errorf("expected delta.TasksFailed to be 0, found %d", delta.TasksFailed)
	}

	assertOtherDeltaFieldsUnchangedForUpdate(t, delta)
}

func TestAddDeltaToMapUpdateQueuedToSuccessDeltaObjAlreadyExists(t *testing.T) {
	fullJobId := JobRunFullId{
		JobConfigId: "A",
		JobRunId:    "B",
	}
	task := &Task{
		JobConfigId: fullJobId.JobConfigId,
		JobRunId:    fullJobId.JobRunId,
		Status:      Success,
	}

	progressDeltas := make(map[JobRunFullId]*JobProgressDelta)
	progressDeltas[fullJobId] = &JobProgressDelta{}

	addJobProgressDeltaForTaskUpdateToMap(task, Queued, progressDeltas)

	if len(progressDeltas) != 1 {
		t.Errorf("expected progressDeltas to contain 1 delta, found %d",
			len(progressDeltas))
	}

	delta, exists := progressDeltas[fullJobId]
	if !exists {
		t.Errorf("expected progressDeltas to contain a delta for id %+v", fullJobId)
	}

	if delta.TasksCompleted != 1 {
		t.Errorf("expected delta.TasksCompleted to be 1, found %d", delta.TasksCompleted)
	}

	if delta.TasksFailed != 0 {
		t.Errorf("expected delta.TasksFailed to be 0, found %d", delta.TasksFailed)
	}

	assertOtherDeltaFieldsUnchangedForUpdate(t, delta)
}

func TestAddDeltaToMapUpdateFailedToSuccess(t *testing.T) {
	fullJobId := JobRunFullId{
		JobConfigId: "A",
		JobRunId:    "B",
	}
	task := &Task{
		JobConfigId: fullJobId.JobConfigId,
		JobRunId:    fullJobId.JobRunId,
		Status:      Success,
	}

	progressDeltas := make(map[JobRunFullId]*JobProgressDelta)

	addJobProgressDeltaForTaskUpdateToMap(task, Failed, progressDeltas)

	if len(progressDeltas) != 1 {
		t.Errorf("expected progressDeltas to contain 1 delta, found %d",
			len(progressDeltas))
	}

	delta, exists := progressDeltas[fullJobId]
	if !exists {
		t.Errorf("expected progressDeltas to contain a delta for id %+v", fullJobId)
	}

	if delta.TasksCompleted != 1 {
		t.Errorf("expected delta.TasksCompleted to be 1, found %d", delta.TasksCompleted)
	}

	if delta.TasksFailed != -1 {
		t.Errorf("expected delta.TasksFailed to be -1, found %d", delta.TasksFailed)
	}

	assertOtherDeltaFieldsUnchangedForUpdate(t, delta)
}

func TestAddDeltaToMapUpdateUnqueuedToSuccess(t *testing.T) {
	fullJobId := JobRunFullId{
		JobConfigId: "A",
		JobRunId:    "B",
	}
	task := &Task{
		JobConfigId: fullJobId.JobConfigId,
		JobRunId:    fullJobId.JobRunId,
		Status:      Success,
	}

	progressDeltas := make(map[JobRunFullId]*JobProgressDelta)

	addJobProgressDeltaForTaskUpdateToMap(task, Unqueued, progressDeltas)

	if len(progressDeltas) != 1 {
		t.Errorf("expected progressDeltas to contain 1 delta, found %d",
			len(progressDeltas))
	}

	delta, exists := progressDeltas[fullJobId]
	if !exists {
		t.Errorf("expected progressDeltas to contain a delta for id %+v", fullJobId)
	}

	if delta.TasksCompleted != 1 {
		t.Errorf("expected delta.TasksCompleted to be 1, found %d", delta.TasksCompleted)
	}

	if delta.TasksFailed != 0 {
		t.Errorf("expected delta.TasksFailed to be 0, found %d", delta.TasksFailed)
	}

	assertOtherDeltaFieldsUnchangedForUpdate(t, delta)
}

func TestAddDeltaToMapUpdateUnqueuedToFailed(t *testing.T) {
	fullJobId := JobRunFullId{
		JobConfigId: "A",
		JobRunId:    "B",
	}
	task := &Task{
		JobConfigId: fullJobId.JobConfigId,
		JobRunId:    fullJobId.JobRunId,
		Status:      Failed,
	}

	progressDeltas := make(map[JobRunFullId]*JobProgressDelta)

	addJobProgressDeltaForTaskUpdateToMap(task, Unqueued, progressDeltas)

	if len(progressDeltas) != 1 {
		t.Errorf("expected progressDeltas to contain 1 delta, found %d",
			len(progressDeltas))
	}

	delta, exists := progressDeltas[fullJobId]
	if !exists {
		t.Errorf("expected progressDeltas to contain a delta for id %+v", fullJobId)
	}

	if delta.TasksCompleted != 0 {
		t.Errorf("expected delta.TasksCompleted to be 0, found %d", delta.TasksCompleted)
	}

	if delta.TasksFailed != 1 {
		t.Errorf("expected delta.TasksFailed to be 1, found %d", delta.TasksFailed)
	}

	assertOtherDeltaFieldsUnchangedForUpdate(t, delta)
}

func TestAddDeltaToMapUpdateUnqueuedToQueued(t *testing.T) {
	fullJobId := JobRunFullId{
		JobConfigId: "A",
		JobRunId:    "B",
	}
	task := &Task{
		JobConfigId: fullJobId.JobConfigId,
		JobRunId:    fullJobId.JobRunId,
		Status:      Queued,
	}

	progressDeltas := make(map[JobRunFullId]*JobProgressDelta)

	addJobProgressDeltaForTaskUpdateToMap(task, Unqueued, progressDeltas)

	if len(progressDeltas) != 0 {
		t.Errorf("expected progressDeltas to be empty, found %d",
			len(progressDeltas))
	}
}

func assertOtherDeltaFieldsUnchangedForUpdate(t *testing.T,
	delta *JobProgressDelta) {
	if delta.NewTasks != 0 {
		t.Errorf("expected delta.NewTasks to be 0, found %d",
			delta.NewTasks)
	}
}
