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

/*
Package dcp contains all the objects definition and the logic necessary for the
data control plane (dcp). DCP is responsible for managing the whole lifecyle of
transfers and so managing the transfer jobs and the tasks associated with them,
and provide a monitoring capabilities for the transfers.
*/
// TODO(b/63026027): Design a proper way of logging. Currently, everything is
// printed to stdout.
package dcp

import (
	"encoding/base64"
	"log"
	"strings"

	"golang.org/x/net/context"

	"cloud.google.com/go/pubsub"
)

// MessageHandler interface is used to abstract handling of various message
// types that correspond to various task types.
type MessageHandler interface {
	// HandleMessage processes a pubsub task progress message, and returns a list
	// of new tasks generated from this task.
	HandleMessage(jobSpec *JobSpec, taskWithLog TaskWithLog) ([]*Task, error)
}

// MessageReceiver receives outstanding messages from a PubSub subscription. For
// each message, executes the handler, and then acks it. It also responsible for
// extending the message lease as needed.
// TODO(b/63014764): Add unit tests for MessageReceiver.
type MessageReceiver struct {
	Sub     *pubsub.Subscription
	Store   Store
	Handler MessageHandler

	batcher taskUpdatesBatcher
}

func (r *MessageReceiver) ReceiveMessages() error {
	// Currently, there is a batcher for each message receiver type (list,
	// uploadGCS, loadBQ). May be we can consider only one batcher for all the
	// receiver types.
	r.batcher.initializeAndStart(r.Store, newClockTicker(batchingTimeInterval))

	ctx := context.Background()

	err := r.Sub.Receive(ctx, func(ctx context.Context, msg *pubsub.Message) {
		// TODO(b/63108335): We send a message ack and spanner database update for
		// each message(task). This is a potential scalability issue, batching of
		// task updates and messages acks is necessary to resolve that.

		// Node JS client library send the message with quotes, removing the quotes
		// from the message if exists.
		msgData := strings.Trim(string(msg.Data), "\"")

		// Decode the base64 encoded message in the Pubsub queue.
		decodedMsg, err := base64.StdEncoding.DecodeString(msgData)
		if err != nil {
			log.Printf("Error Decoding msg: %v, with error: %v.", msgData, err)
			return
		}

		// TODO(b/63058868): Failed to handle a PubSub message will be keep
		// redelivered by the PubSub for significant amount of time (1 week).
		// Non-retriable errors should mark the task failed and ack the message.
		log.Printf("Handling a message: %s.", string(decodedMsg))
		taskWithLog, err := TaskCompletionMessageJsonToTaskWithLog(decodedMsg)
		if err != nil {
			log.Printf("Error handling the message: %s with error: %v.",
				string(decodedMsg), err)
			return
		}
		task := taskWithLog.Task
		// TODO(b/63060838): Do caching for the JobSpec, querying the database for each
		// task is not efficient.
		jobSpec, err := r.Store.GetJobSpec(task.JobConfigId)
		if err != nil {
			log.Printf("Error in getting JobSpec of JobConfig: %d, with error: %v.",
				task.JobConfigId, err)
			return
		}
		newTasks, err := r.Handler.HandleMessage(jobSpec, *taskWithLog)
		if err != nil {
			log.Printf(
				"Error handling the message: %s, for with job spec: %v, and task: %v, with error: %v.",
				string(msg.Data), jobSpec, task, err)
			return
		}

		r.batcher.addTaskUpdate(taskWithLog, newTasks, msg)
	})
	if err != nil {
		log.Printf("Error receiving messages for subscription %v, with error: %v.",
			r.Sub, err)
	}
	return err
}
