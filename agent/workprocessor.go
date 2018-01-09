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

package agent

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"

	"cloud.google.com/go/pubsub"
	"github.com/GoogleCloudPlatform/cloud-ingest/dcp"
	"github.com/golang/glog"
)

var workerID string

func init() {
	hostname, err := os.Hostname()
	if err != nil {
		glog.Fatal("Initialization failed, cannot get host name.")
	}
	workerID = fmt.Sprintf("%s-%d", hostname, os.Getpid())
}

// WorkHandler is an interface to handle different task types.
type WorkHandler interface {
	// Do handles the task with taskRRName and taskParams.
	Do(ctx context.Context, taskRRName string, taskParams dcp.TaskParams) dcp.TaskCompletionMessage
}

// WorkProcessor processes tasks of a certain type. It listens to subscription
// WorkSub, delegates to the Handler to do the work, and send progress messages
// to ProgressTopic.
type WorkProcessor struct {
	WorkSub       *pubsub.Subscription
	ProgressTopic *pubsub.Topic
	Handler       WorkHandler
}

func (wp *WorkProcessor) processMessage(ctx context.Context, msg *pubsub.Message) {
	glog.Infof("Handling message: %s.", string(msg.Data))

	msgMap := make(map[string]interface{})
	decoder := json.NewDecoder(bytes.NewReader(msg.Data))
	decoder.UseNumber()

	// TODO(b/70812612): Define the work messages in a struct instead of
	// map[string]interface{}.
	if err := decoder.Decode(&msgMap); err != nil {
		glog.Errorf("error decoding JSON msg string %s with error %v.",
			string(msg.Data), err)
		// Non-recoverable error. Will Ack the message to avoid delivering again.
		msg.Ack()
		return
	}
	taskRRName, ok := msgMap["task_rr_name"].(string)
	if !ok {
		glog.Errorf("Can not get the full task id from message %s.", string(msg.Data))
		// Here the taskRRName is unknown. Will Ack the message to avoid delivering again.
		msg.Ack()
		return
	}
	taskParams := msgMap["task_params"].(map[string]interface{})

	progressMsg := wp.Handler.Do(ctx, taskRRName, taskParams)

	progressMsgJSON, err := json.Marshal(progressMsg)
	if err != nil {
		glog.Errorf("Cannot marshal json %+v with err %v", progressMsg, err)
		// This may be a transient error, will not Ack the messages to retry again
		// when the message redelivered.
		return
	}
	pubResult := wp.ProgressTopic.Publish(ctx, &pubsub.Message{Data: progressMsgJSON})
	if _, err := pubResult.Get(ctx); err != nil {
		glog.Errorf("Can not publish list progress message with err: %v", err)
		// Will not Ack the messages to retry again when the message redelivered.
		return
	}
	msg.Ack()
}

func (wp *WorkProcessor) Process(ctx context.Context) {
	// Use the DefaultReceiveSettings, which is ReceiveSettings{
	// 	 MaxExtension:           10 * time.Minute,
	// 	 MaxOutstandingMessages: 1000,
	// 	 MaxOutstandingBytes:    1e9,
	// 	 NumGoroutines:          1,
	// }
	// The default settings should be safe, because of the following reasons:
	// * MaxExtension: DCP should not publish messages that estimated to take more
	//                 than 10 mins.
	// * MaxOutstandingMessages: It's also capped by the memory, and this will speed
	//                           up processing of small files.
	// * MaxOutstandingBytes: 1GB memory should not be a problem for a modern machine.
	// * NumGoroutines: Does not need more than 1 routine to pull Pub/Sub messages.
	//
	// Note that the main function can override those values when constructing the
	// WorkProcessor instance.
	err := wp.WorkSub.Receive(ctx, wp.processMessage)

	if ctx.Err() != nil {
		glog.Warningf(
			"Error receiving work messages for subscription %v, with context error: %v.",
			wp.WorkSub, ctx.Err())
	}

	// The Pub/Sub client libraries already retries on retriable errors. Panic
	// here on non-retriable errors.
	if err != nil {
		glog.Fatalf("Error receiving work messages for subscription %v, with error: %v.",
			wp.WorkSub, err)
	}
}
