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
	"strconv"
	"sync"
	"testing"

	"cloud.google.com/go/pubsub"
)

func initializePubSubMock() (map[string]bool, func(msg *pubsub.Message)) {
	ackedMessages := map[string]bool{}

	ackMessageFnMock := func(msg *pubsub.Message) {
		ackedMessages[msg.ID] = true
	}
	return ackedMessages, ackMessageFnMock
}

// TODO(b/68757834): This test sometimes fails and sometimes doesn't.
func TestBatcherWithOneUpdate(t *testing.T) {
	ackedMessages, ackMessageFnMock := initializePubSubMock()
	var batcher taskUpdatesBatcher

	task := &Task{
		JobConfigId: "dummy-config",
		JobRunId:    "dummy-run",
		TaskId:      "dummy-task",
		Status:      Queued,
	}

	store := &FakeStore{
		tasks: map[string]*Task{
			task.getTaskFullId(): task,
		},
	}

	updatedTask := *task
	updatedTask.Status = Success

	task1 := &Task{
		JobConfigId: "dummy-config",
		JobRunId:    "dummy-run",
		TaskId:      "new-task-1",
	}

	task2 := &Task{
		JobConfigId: "dummy-config",
		JobRunId:    "dummy-run",
		TaskId:      "new-task-2",
	}

	taskUpdate := &TaskUpdate{
		Task:     &updatedTask,
		NewTasks: []*Task{task1, task2},
	}

	msg := &pubsub.Message{ID: "dummy-msg"}

	mockTicker := newMockTicker()
	batcher.initializeAndStart(store, mockTicker)
	// Override Pub/Sub Ack function with a mock one.
	batcher.ackMessage = ackMessageFnMock
	batcher.addTaskUpdate(taskUpdate, msg)

	mockTicker.tick()
	if len(store.tasks) != 3 {
		t.Errorf("expected 3 tasks in the store, found %v.", len(store.tasks))
	}
	if store.tasks[task.getTaskFullId()].Status != Success {
		t.Errorf("expected task %v to be updated success status.",
			store.tasks[task.getTaskFullId()])
	}
	if !ackedMessages[msg.ID] {
		t.Errorf("message %v should be ack'ed but it's not.", msg.ID)
	}
}

func TestBatcherWithMultiASyncUpdates(t *testing.T) {
	ackedMessages, ackMessageFnMock := initializePubSubMock()
	var batcher taskUpdatesBatcher

	store := &FakeStore{
		tasks: map[string]*Task{},
	}

	mockTicker := newMockTicker()
	batcher.initializeAndStart(store, mockTicker)
	// Override Pub/Sub Ack function with a mock one.
	batcher.ackMessage = ackMessageFnMock

	numberOfTasks := 100

	// Initialize the store with new tasks
	for i := 0; i < numberOfTasks; i++ {
		task := &Task{
			JobConfigId: "dummy-config",
			JobRunId:    "dummy-run",
			TaskId:      "dummy-task-" + strconv.Itoa(i),
			Status:      Queued,
		}
		store.tasks[task.getTaskFullId()] = task
	}

	var wg sync.WaitGroup
	// add the tasks updates in parrallel
	for i := 0; i < numberOfTasks; i++ {
		wg.Add(1)
		go func(count int) {
			defer wg.Done()
			updatedTask := &Task{
				JobConfigId: "dummy-config",
				JobRunId:    "dummy-run",
				TaskId:      "dummy-task-" + strconv.Itoa(count),
				Status:      Success,
			}

			newTasks := []*Task{}
			// Half of the tasks generate new tasks.
			if count%2 != 0 {
				newTasks = append(newTasks, &Task{
					JobConfigId: "dummy-config",
					JobRunId:    "dummy-run",
					TaskId:      "new-task-" + strconv.Itoa(count),
					Status:      Unqueued,
				})
			}
			taskUpdate := &TaskUpdate{
				Task:     updatedTask,
				NewTasks: newTasks,
			}
			msg := &pubsub.Message{ID: "dummy-msg-" + strconv.Itoa(count)}
			batcher.addTaskUpdate(taskUpdate, msg)
		}(i)
	}
	wg.Wait()

	// Note that we can not use mockTicker.tick because that will run
	// commitUpdates asynchronously, and you will end up with accessing the
	// ackedMessages map (that is not thread safe) from commitUpdates and the for
	// loop below simultaneously.
	batcher.commitUpdates()

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
