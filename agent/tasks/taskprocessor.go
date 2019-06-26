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
	"net/http"
	"time"

	"cloud.google.com/go/pubsub"
	"cloud.google.com/go/storage"
	"github.com/GoogleCloudPlatform/cloud-ingest/agent/rate"
	"github.com/GoogleCloudPlatform/cloud-ingest/agent/stats"
	"github.com/GoogleCloudPlatform/cloud-ingest/agent/tasks/common"
	"github.com/GoogleCloudPlatform/cloud-ingest/agent/tasks/copy"
	"github.com/GoogleCloudPlatform/cloud-ingest/agent/tasks/list"
	"github.com/golang/glog"
	"github.com/golang/protobuf/proto"
	"github.com/golang/protobuf/ptypes"

	taskpb "github.com/GoogleCloudPlatform/cloud-ingest/proto/task_go_proto"
)

// TaskHandler is an interface to handle different task types.
type TaskHandler interface {
	// Do handles the TaskReqMsg and returns a TaskRespMsg.
	Do(ctx context.Context, taskReqMsg *taskpb.TaskReqMsg, reqStart time.Time) *taskpb.TaskRespMsg
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

// NewListProcessor returns a TaskProcessor for handling List tasks.
// Run the Process func on the newly returned TaskProcessor to begin processing tasks.
func NewListProcessor(sc *storage.Client, sub *pubsub.Subscription, topic *pubsub.Topic, st *stats.Tracker) *TaskProcessor {
	depthFirstListHandler := list.NewDepthFirstListHandler(sc, st)
	return &TaskProcessor{
		TaskSub:       sub,
		ProgressTopic: topic,
		Handlers: NewHandlerRegistry(map[uint64]TaskHandler{
			1: depthFirstListHandler,
			2: depthFirstListHandler,
			3: list.NewListHandlerV3(sc, st),
		}),
		StatsTracker: st,
	}
}

// NewCopyProcessor returns a TaskProcessor for handling Copy tasks.
// Run the Process func on the newly returned TaskProcessor to begin processing tasks.
func NewCopyProcessor(sc *storage.Client, hc *http.Client, sub *pubsub.Subscription, topic *pubsub.Topic, st *stats.Tracker) *TaskProcessor {
	copyHandler := copy.NewCopyHandler(sc, hc, st)
	return &TaskProcessor{
		TaskSub:       sub,
		ProgressTopic: topic,
		Handlers: NewHandlerRegistry(map[uint64]TaskHandler{
			1: copyHandler,
			2: copyHandler,
			3: copyHandler,
		}),
		StatsTracker: st,
	}
}

// Process handles taskReqMsgs sent by the DCP for the given PubSub subscription and handler.
// This is a blocking function.
func (tp *TaskProcessor) Process(ctx context.Context) {
	err := tp.TaskSub.Receive(ctx, tp.processMessage)
	if err != nil && ctx.Err() == nil {
		glog.Fatalf("%s.Receive() got err: %v", tp.TaskSub.String(), err)
	}
}

func addTaskTimestamps(resp *taskpb.TaskRespMsg, reqStart, msgPublishTime time.Time) {
	reqPublishTime, err := ptypes.TimestampProto(msgPublishTime)
	if err != nil {
		glog.Errorf("could not parse request publish time %v for task %v, err: %v", msgPublishTime, resp.ReqSpec, err)
		return
	}
	reqStartTime, err := ptypes.TimestampProto(reqStart)
	if err != nil {
		glog.Errorf("could not parse task processing start time %v for task %v, err: %v", reqStartTime, resp.ReqSpec, err)
		return
	}
	now := time.Now()
	respPublishTime, err := ptypes.TimestampProto(now)
	if err != nil {
		glog.Errorf("could not parse task response publish time %v for task %v, err: %v", now, resp.ReqSpec, err)
		return
	}
	resp.ReqPublishTime = reqPublishTime
	resp.ReqStartTime = reqStartTime
	resp.RespPublishTime = respPublishTime
}

func (tp *TaskProcessor) processMessage(ctx context.Context, msg *pubsub.Message) {
	var taskReqMsg taskpb.TaskReqMsg
	if err := proto.Unmarshal(msg.Data, &taskReqMsg); err != nil {
		glog.Errorf("error decoding msg %s with error %v.", string(msg.Data), err)
		// Non-recoverable error. Will Ack the message to avoid delivering again.
		msg.Ack()
		return
	}

	reqStart := time.Now()
	var taskRespMsg *taskpb.TaskRespMsg
	if rate.IsJobRunActive(taskReqMsg.JobrunRelRsrcName) {
		handler, agentErr := tp.Handlers.HandlerForTaskReqMsg(&taskReqMsg)
		if agentErr != nil {
			taskRespMsg = common.BuildTaskRespMsg(&taskReqMsg, nil, nil, *agentErr)
		} else {
			taskRespMsg = handler.Do(ctx, &taskReqMsg, reqStart)
			tp.StatsTracker.RecordTaskResp(taskRespMsg)
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

	addTaskTimestamps(taskRespMsg, reqStart, msg.PublishTime)

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
