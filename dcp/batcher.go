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
	"log"
	"sync"
	"time"

	"cloud.google.com/go/pubsub"
)

const (
	batchingTimeInterval time.Duration = 1 * time.Second
)

// taskUpdatesBatcher provides a capability for batching tasks updates/inserts
// in the Spanner store, and ack'ing the Pub/Sub messages associated with them.
type taskUpdatesBatcher struct {
	started             bool
	store               Store
	pendingTasksToStore map[*TaskWithLog][]*Task
	pendingMsgsToAck    []*pubsub.Message
	mu                  sync.Mutex
	ticker              ticker
	ackMessage          func(msg *pubsub.Message)
}

// TODO(b/67495138): Avoid primitive types in the spanner store and batcher.
// This hurts the code readability. go/tott-494 for more details.
// addTaskUpdate adds a new task update to the batch. It takes the taskWithLog
// to be updated, the newTasks that is associated with this task update, and the
// Pub/Sub message to ack when the update is committed.
func (b *taskUpdatesBatcher) addTaskUpdate(
	taskWithLog *TaskWithLog, newTasks []*Task, msg *pubsub.Message) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.pendingTasksToStore[taskWithLog] = append(
		b.pendingTasksToStore[taskWithLog], newTasks...)
	b.pendingMsgsToAck = append(b.pendingMsgsToAck, msg)
}

// commitUpdates commits the pending tasks to be inserted/updated to the spanner
// store and ack's the Pub/Sub progress messages upon the success of the
// transaction. If everything is successful, it resets the pendingTasksToStore
// map and pendingMsgsToAck list.
func (b *taskUpdatesBatcher) commitUpdates() {
	b.mu.Lock()
	defer b.mu.Unlock()

	if len(b.pendingMsgsToAck) == 0 {
		return
	}

	// TODO(b/67472516): One optimization we can do here is to take a copy of
	// tasks and messages, and then release the lock before calling the
	// UpdateAndInsertTasks or ack'ing the messages. This way we guarantee that
	// other receivers will be blocked for Spanner transaction or Pub/Sub server
	// calls to complete.
	if err := b.store.UpdateAndInsertTasks(b.pendingTasksToStore); err != nil {
		fmt.Printf("Errors on the update %v\n", err)
		return
	}

	// Ack'ing the messages. This ack is asynchronous, meaning that the client
	// library marks the message as ack'ed, and will batch the actual ack's to the
	// Pub/Sub server.
	for _, msg := range b.pendingMsgsToAck {
		b.ackMessage(msg)
	}

	// Reset the pending tasks to commit and pending messages to ack.
	b.pendingTasksToStore = make(map[*TaskWithLog][]*Task)
	b.pendingMsgsToAck = b.pendingMsgsToAck[:0]
}

// TODO(b/67468360): Consider passing a context in case we need to cancel or
// terminate this thread.
// initializeAndStart initializes the taskUpdateBatcher and starts a Go routine
// to periodically commits the accumulated pending task updates.
func (b *taskUpdatesBatcher) initializeAndStart(s Store, t ticker) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.started {
		log.Println("taskUpdatesBatcher already started, ignoring this start call.")
		return
	}
	b.started = true

	b.store = s
	b.ticker = t
	b.pendingTasksToStore = make(map[*TaskWithLog][]*Task)
	b.pendingMsgsToAck = []*pubsub.Message{}
	b.ackMessage = func(msg *pubsub.Message) {
		msg.Ack()
	}

	go func() {
		c := b.ticker.getChannel()
		for {
			select {
			case <-c:
				b.commitUpdates()
			}
			// TODO(b/67468147): Implement batching transactions based on the size of
			// the transaction in addition to the interval based one.
		}
	}()
}
