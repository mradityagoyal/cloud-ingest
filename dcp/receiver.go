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
	"fmt"
	"strings"

	"golang.org/x/net/context"

	"cloud.google.com/go/pubsub"
)

// MessageHandler interface is used to abstract handling of various message
// types that correspond to various task types.
type MessageHandler interface {
	HandleMessage(jobSpec *JobSpec, task *Task) error
}

// MessageReceiver receives outstanding messages from a PubSub subscription. For
// each message, executes the handler, and then acks it. It also responsible for
// extending the message lease as needed.
// TODO(b/63014764): Add unit tests for MessageReceiver.
type MessageReceiver struct {
	Sub     *pubsub.Subscription
	Store   Store
	Handler MessageHandler
}

func (r *MessageReceiver) ReceiveMessages() error {
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
			fmt.Printf("Error Decoding msg: %v, with error: %v.\n", msgData, err)
			return
		}

		// TODO(b/63058868): Failed to handle a PubSub message will be keep
		// redelivered by the PubSub for significant amount of time (1 week).
		// Non-retriable errors should mark the task failed and ack the message.
		fmt.Printf("Handling a message: %s.\n", string(decodedMsg))
		task, err := TaskCompletionMessageJsonToTask(decodedMsg)
		if err != nil {
			fmt.Printf("Error handling the message: %s with error: %v.\n",
				string(decodedMsg), err)
			return
		}
		// TODO(b/63060838): Do caching for JobSpec, inquiry the database for each
		// task is not efficient.
		jobSpec, err := r.Store.GetJobSpec(task.JobConfigId)
		if err != nil {
			fmt.Printf("Error in getting JobSpec of JobConfig: %d, with error: %v.\n",
				task.JobConfigId, err)
			return
		}
		if err := r.Handler.HandleMessage(jobSpec, task); err != nil {
			fmt.Printf(
				"Error handling the message: %s, for with job spec: %v, and task: %v, with error: %v.\n",
				string(msg.Data), jobSpec, task, err)
			return
		}
		// TODO(b/63015042): Fix the race condition where the task is updated to
		// success or failure and then the task queuing thread updates it to Queued.
		if err := r.Store.UpdateTasks([]*Task{task}); err != nil {
			fmt.Printf("Error Updating task: %v, with error: %v.\n", task, err)
			return
		}
		fmt.Printf("Acking message: %s.\n", string(decodedMsg))
		msg.Ack()
	})
	if err != nil {
		fmt.Printf("Error receiving messages for subscription %v, with error: %v.\n",
			r.Sub, err)
	}
	return err
}
