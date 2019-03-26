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

package tasks

import (
	"context"
	"fmt"
	"time"

	"cloud.google.com/go/pubsub"
	"github.com/GoogleCloudPlatform/cloud-ingest/agent/rate"
	"github.com/GoogleCloudPlatform/cloud-ingest/agent/stats"
	"github.com/GoogleCloudPlatform/cloud-ingest/agent/tasks/common"
	"github.com/golang/glog"
	"github.com/golang/protobuf/proto"

	taskpb "github.com/GoogleCloudPlatform/cloud-ingest/proto/task_go_proto"
)

// TaskHandler is an interface to handle different task types.
type TaskHandler interface {
	// Do handles the TaskReqMsg and returns a TaskRespMsg.
	Do(ctx context.Context, taskReqMsg *taskpb.TaskReqMsg) *taskpb.TaskRespMsg
}

// TaskProcessor processes tasks of a certain type. It listens to subscription
// TaskSub, delegates to the Handler to do the work, and send progress messages
// to ProgressTopic.
type TaskProcessor struct {
	TaskSub       *pubsub.Subscription
	ProgressTopic *pubsub.Topic
	Handlers      *HandlerRegistry
	StatsTracker  *stats.Tracker
}

// Process handles taskReqMsgs sent by the DCP for the given PubSub subscription and handler.
// This is a blocking function.
func (tp *TaskProcessor) Process(ctx context.Context) {
	err := tp.TaskSub.Receive(ctx, tp.processMessage)
	if err != nil {
		glog.Fatalf("%s.Receive() got err: %v", tp.TaskSub.String(), err)
	}
}

func (tp *TaskProcessor) processMessage(ctx context.Context, msg *pubsub.Message) {
	var taskReqMsg taskpb.TaskReqMsg
	if err := proto.Unmarshal(msg.Data, &taskReqMsg); err != nil {
		glog.Errorf("error decoding msg %s with error %v.", string(msg.Data), err)
		// Non-recoverable error. Will Ack the message to avoid delivering again.
		msg.Ack()
		return
	}

	var taskRespMsg *taskpb.TaskRespMsg
	if rate.IsJobRunActive(taskReqMsg.JobrunRelRsrcName) {
		handler, agentErr := tp.Handlers.HandlerForTaskReqMsg(&taskReqMsg)
		if agentErr != nil {
			taskRespMsg = common.BuildTaskRespMsg(&taskReqMsg, nil, nil, *agentErr)
		} else {
			start := time.Now()
			taskRespMsg = handler.Do(ctx, &taskReqMsg)
			tp.StatsTracker.RecordTaskResp(taskRespMsg, time.Now().Sub(start))
		}
	} else {
		taskRespMsg = common.BuildTaskRespMsg(&taskReqMsg, nil, nil, common.AgentError{
			Msg:         fmt.Sprintf("job run %s is not active", taskReqMsg.JobrunRelRsrcName),
			FailureType: taskpb.FailureType_NOT_ACTIVE_JOBRUN,
		})
	}

	if !proto.Equal(taskReqMsg.Spec, taskRespMsg.ReqSpec) {
		glog.Errorf("taskRespMsg.ReqSpec = %v, want: %v", taskRespMsg.ReqSpec, taskReqMsg.Spec)
		// The taskRespMsg.ReqSpec must equal the taskReqMsg.Spec. This is an Agent
		// coding error, do not ack the PubSub message.
		return
	}
	if ctx.Err() == context.Canceled {
		glog.Errorf("Context is canceled, not sending taskRespMsg: %v", taskRespMsg)
		// When the context is canceled midway through processing a request, instead of
		// surfacing an error which propagates to the DCP just don't send the response.
		// The work will remain on PubSub and eventually be taken up by another worker.
		return
	}
	serializedTaskRespMsg, err := proto.Marshal(taskRespMsg)
	if err != nil {
		glog.Errorf("Cannot marshal pb %+v with err %v", taskRespMsg, err)
		// This may be a transient error, will not Ack the messages to retry again
		// when the message redelivered.
		return
	}
	pubResult := tp.ProgressTopic.Publish(ctx, &pubsub.Message{Data: serializedTaskRespMsg})
	if _, err := pubResult.Get(ctx); err != nil {
		glog.Errorf("Can not publish progress message with err: %v", err)
		// Don't ack the messages, retry again when the message is redelivered.
		return
	}
	msg.Ack()
}
