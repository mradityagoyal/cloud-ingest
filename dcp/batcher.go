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
	"log"
	"sync"
	"time"

	"cloud.google.com/go/pubsub"
)

const (
	batchingTimeInterval time.Duration = 1 * time.Second
	maxBatchSize                       = 1000
)

// taskUpdatesBatcher provides a capability for batching tasks updates/inserts
// in the Spanner store, and ack'ing the Pub/Sub messages associated with them.
type taskUpdatesBatcher struct {
	started             bool
	store               Store
	pendingTasksToStore *TaskUpdateCollection
	pendingMsgsToAck    []*pubsub.Message
	currUpdateSize      int
	maxBatchSize        int
	mu                  sync.Mutex
	ticker              Ticker
	ackMessage          func(msg *pubsub.Message)
}

// addTaskUpdate adds a new task update to the batch. It takes a TaskUpdate
// to be updated, and the Pub/Sub message to ack when the update is committed.
func (b *taskUpdatesBatcher) addTaskUpdate(
	taskUpdate *TaskUpdate, msg *pubsub.Message) {
	b.mu.Lock()
	defer b.mu.Unlock()
	// Check the batch size before adding the new TaskUpdate.
	if b.currUpdateSize+1+len(taskUpdate.NewTasks) > b.maxBatchSize {
		// Commit the current updates first before getting batching new ones.
		b.commitUpdates()
	}
	b.pendingTasksToStore.AddTaskUpdate(taskUpdate)
	b.pendingMsgsToAck = append(b.pendingMsgsToAck, msg)
	b.currUpdateSize += 1 + len(taskUpdate.NewTasks)
}

// commitUpdates retries commitUpdatesClosure with exponential back off
// strategy. mu Lock should be acquired before calling this function.
func (b *taskUpdatesBatcher) commitUpdates() {
	// TODO(b/69788003): RetryWithExponentialBackoff should should only retry on
	// re-triable error.
	err := RetryWithExponentialBackoff(
		time.Second,              // Initial sleep time.
		30*time.Second,           // Max sleep time.
		10,                       // Max number of failures
		"commitUpdates",          // Function name
		b.commitUpdatesClosure(), // Closure to run.
	)
	if err != nil {
		panic(err)
	}
}

// commitUpdatesClosure returns a closure that commits the pending tasks to be
// inserted/updated to the spanner store and ack's the Pub/Sub progress messages
// upon the success of the transaction. If everything is successful, it resets
// the pendingTasksToStore map and pendingMsgsToAck list.
func (b *taskUpdatesBatcher) commitUpdatesClosure() func() error {

	return func() error {
		if len(b.pendingMsgsToAck) == 0 {
			return nil
		}

		// TODO(b/67472516): One optimization we can do here is to take a copy of
		// tasks and messages, and then release the lock before calling the
		// UpdateAndInsertTasks or ack'ing the messages. This way we guarantee that
		// other receivers will be blocked for Spanner transaction or Pub/Sub server
		// calls to complete.
		if err := b.store.UpdateAndInsertTasks(b.pendingTasksToStore); err != nil {
			log.Printf("Error on UpdateAndInsertTasks: %v.", err)
			return err
		}

		// Ack'ing the messages. This ack is asynchronous, meaning that the client
		// library marks the message as ack'ed, and will batch the actual ack's to the
		// Pub/Sub server.
		for _, msg := range b.pendingMsgsToAck {
			b.ackMessage(msg)
		}

		// Reset the pending tasks to commit and pending messages to ack.
		b.pendingTasksToStore.Clear()
		b.pendingMsgsToAck = b.pendingMsgsToAck[:0]
		b.currUpdateSize = 0
		return nil
	}
}

// TODO(b/67468360): Consider passing a context in case we need to cancel or
// terminate this thread.
// initializeAndStart initializes the taskUpdateBatcher and starts a Go routine
// to periodically commits the accumulated pending task updates.
func (b *taskUpdatesBatcher) initializeAndStart(s Store) {
	b.initializeAndStartInternal(s, NewClockTicker(batchingTimeInterval), nil)
}

func (b *taskUpdatesBatcher) initializeAndStartInternal(s Store, t Ticker, testChannel chan int) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.started {
		log.Println("taskUpdatesBatcher already started, ignoring this start call.")
		return
	}
	b.started = true

	b.store = s
	b.ticker = t
	b.pendingTasksToStore = &TaskUpdateCollection{}
	b.pendingMsgsToAck = []*pubsub.Message{}
	b.currUpdateSize = 0
	b.maxBatchSize = maxBatchSize
	b.ackMessage = func(msg *pubsub.Message) {
		msg.Ack()
	}

	go func() {
		for range b.ticker.GetChannel() {
			b.mu.Lock()
			b.commitUpdates()
			b.mu.Unlock()
			if testChannel != nil {
				testChannel <- 0
			}
		}
	}()
}
