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
	"fmt"
	"strconv"
	"sync"
	"testing"

	"cloud.google.com/go/pubsub"
	"github.com/GoogleCloudPlatform/cloud-ingest/helpers"
)

func initializePubSubMock() (map[string]bool, func(msg *pubsub.Message)) {
	ackedMessages := map[string]bool{}

	ackMessageFnMock := func(msg *pubsub.Message) {
		ackedMessages[msg.ID] = true
	}
	return ackedMessages, ackMessageFnMock
}

// createTaskAndTaskUpdate creates a dummy Task with status Queued, and a
// TaskUpdate to change the dummy task status to Success and create new tasks.
func createTaskAndTaskUpdate(
	configID string, numberNewTasks int) (*Task, *TaskUpdate) {
	task := Task{
		TaskFullID: *NewTaskFullID("dummy-project", configID, "dummy-run", "dummy-task"),
		Status:     Queued,
	}

	updatedTask := task
	// Change the task update status to success.
	updatedTask.Status = Success

	taskUpdate := TaskUpdate{
		Task:     &updatedTask,
		NewTasks: make([]*Task, numberNewTasks),
	}
	for i := 0; i < numberNewTasks; i++ {
		taskUpdate.NewTasks[i] = &Task{
			TaskFullID: TaskFullID{
				JobRunFullID: task.TaskFullID.JobRunFullID,
				TaskID:       fmt.Sprintf("dummy-new-task-%d", i),
			},
		}
	}
	return &task, &taskUpdate
}

func TestBatcherWithOneUpdate(t *testing.T) {
	ackedMessages, ackMessageFnMock := initializePubSubMock()
	var batcher taskUpdatesBatcher

	task, taskUpdate := createTaskAndTaskUpdate("dummy-config", 2)

	store := &FakeStore{
		tasks: map[TaskFullID]*Task{
			task.TaskFullID: task,
		},
	}

	msg := &pubsub.Message{ID: "dummy-msg"}

	mockTicker := helpers.NewMockTicker()
	testChannel := make(chan int)
	batcher.initializeAndStartInternal(store, mockTicker, testChannel)
	// Override Pub/Sub Ack function with a mock one.
	batcher.ackMessage = ackMessageFnMock
	batcher.addTaskUpdate(taskUpdate, msg)

	mockTicker.Tick()
	<-testChannel // Block until the batcher's commitUpdates call has finished.
	if len(store.tasks) != 3 {
		t.Errorf("expected 3 tasks in the store, found %v.", len(store.tasks))
	}
	if store.tasks[task.TaskFullID].Status != Success {
		t.Errorf("expected task %v to be updated success status.",
			store.tasks[task.TaskFullID])
	}
	if !ackedMessages[msg.ID] {
		t.Errorf("message %v should be ack'ed but it's not.", msg.ID)
	}
}

func TestBatcherWithMultiASyncUpdates(t *testing.T) {
	ackedMessages, ackMessageFnMock := initializePubSubMock()
	var batcher taskUpdatesBatcher

	store := &FakeStore{
		tasks: map[TaskFullID]*Task{},
	}

	mockTicker := helpers.NewMockTicker()
	testChannel := make(chan int)
	batcher.initializeAndStartInternal(store, mockTicker, testChannel)
	// Setting the max batch size to exercise commits based on the batch size.
	batcher.maxBatchSize = 13
	// Override Pub/Sub Ack function with a mock one.
	batcher.ackMessage = ackMessageFnMock

	numberOfTasks := 100

	tasks := make([]*Task, numberOfTasks)
	taskUpdates := make([]*TaskUpdate, numberOfTasks)

	// Initialize the store with new tasks
	for i := 0; i < numberOfTasks; i++ {
		tasks[i], taskUpdates[i] = createTaskAndTaskUpdate(
			fmt.Sprintf("dummy-config-%d", i), i%2)
		store.tasks[tasks[i].TaskFullID] = tasks[i]
	}

	var wg sync.WaitGroup
	// add the tasks updates in parrallel
	for i := 0; i < numberOfTasks; i++ {
		wg.Add(1)
		go func(count int) {
			defer wg.Done()
			msg := &pubsub.Message{ID: "dummy-msg-" + strconv.Itoa(count)}
			batcher.addTaskUpdate(taskUpdates[count], msg)
		}(i)
	}
	wg.Wait()

	mockTicker.Tick()
	<-testChannel // Block until the batcher's commitUpdates call has finished.

	// Make sure all the Pub/Sub update messages are ack'ed.
	for i := 0; i < numberOfTasks; i++ {
		if !ackedMessages["dummy-msg-"+strconv.Itoa(i)] {
			t.Errorf("message dummy-msg-%v should be ack'ed but it's not.", i)
		}
	}

	// Test the tasks made it to the store.
	if len(store.tasks) != numberOfTasks+numberOfTasks/2 {
		t.Errorf("expected %v tasks in the store, found %v",
			numberOfTasks+numberOfTasks/2, len(store.tasks))
	}
}

func TestBatcherMaxBatchSize(t *testing.T) {
	ackedMessages, ackMessageFnMock := initializePubSubMock()
	var batcher taskUpdatesBatcher

	task1, taskUpdate1 := createTaskAndTaskUpdate("dummy-config-1", 2)
	task2, taskUpdate2 := createTaskAndTaskUpdate("dummy-config-2", 0)

	store := &FakeStore{
		tasks: map[TaskFullID]*Task{
			task1.TaskFullID: task1,
			task2.TaskFullID: task2,
		},
	}

	msg1 := &pubsub.Message{ID: "dummy-msg-1"}
	msg2 := &pubsub.Message{ID: "dummy-msg-2"}

	mockTicker := helpers.NewMockTicker()
	testChannel := make(chan int)
	batcher.initializeAndStartInternal(store, mockTicker, testChannel)
	// Override Pub/Sub Ack function with a mock one.
	batcher.ackMessage = ackMessageFnMock
	batcher.maxBatchSize = 3
	batcher.addTaskUpdate(taskUpdate1, msg1)

	// Try adding another update task should trigger the commit.
	batcher.addTaskUpdate(taskUpdate2, msg1)

	if len(store.tasks) != 4 {
		t.Errorf("expected 4 tasks in the store, found %v.", len(store.tasks))
	}
	if store.tasks[task1.TaskFullID].Status != Success {
		t.Errorf("expected task %v to be updated success status.",
			store.tasks[task1.TaskFullID])
	}
	if !ackedMessages[msg1.ID] {
		t.Errorf("message %v should be ack'ed but it's not.", msg1.ID)
	}
	if ackedMessages[msg2.ID] {
		t.Errorf("message %v should not be ack'ed but it is.", msg2.ID)
	}
}
