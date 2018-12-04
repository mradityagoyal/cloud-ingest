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
	"context"
	"fmt"
	"math"
	"sync"
	"time"

	"cloud.google.com/go/pubsub"
	"github.com/GoogleCloudPlatform/cloud-ingest/agent/statslog"
	"github.com/golang/glog"
	"github.com/golang/protobuf/proto"
	"golang.org/x/time/rate"

	taskpb "github.com/GoogleCloudPlatform/cloud-ingest/proto/task_go_proto"
)

var (
	// Mutex to protect activeJobRuns and bwLimiter access.
	mu sync.RWMutex
	// A map between the active jobruns and the associated BW for each job run.
	activeJobRuns map[string]int64

	bwLimiter = rate.NewLimiter(rate.Limit(math.MaxInt64), math.MaxInt32)
)

// UpdateJobRunsBW updates the mapping between job runs and the associated BW.
func UpdateJobRunsBW(jobrunsBW map[string]int64) {
	// Currently, we do not have a way to set per job run BW control. The APIs only
	// allows setting project level BW. For future extensions, DCP distribute the
	// total project BW over the active job runs. Here we aggregate it again to control
	// the BW on a project level.
	var agentBW int64
	for _, bw := range jobrunsBW {
		agentBW += bw
	}
	mu.Lock()
	activeJobRuns = jobrunsBW
	if diff := math.Abs(float64(agentBW) - float64(bwLimiter.Limit())); diff > 0.0000001 {
		glog.Infof("Updating the BW limits, old: %.fMB/s, new: %.fMB/s.", bwLimiter.Limit()/1000000.0, rate.Limit(agentBW/1000000))
		burst := math.MaxInt32
		if agentBW < int64(burst) {
			burst = int(agentBW)
		}
		bwLimiter = rate.NewLimiter(rate.Limit(agentBW), burst)
	}
	mu.Unlock()
}

// WorkHandler is an interface to handle different task types.
type WorkHandler interface {
	// Do handles the TaskReqMsg and returns a TaskRespMsg.
	Do(ctx context.Context, taskReqMsg *taskpb.TaskReqMsg) *taskpb.TaskRespMsg

	// Type returns a string of the Handler's type.
	Type() string
}

// WorkProcessor processes tasks of a certain type. It listens to subscription
// WorkSub, delegates to the Handler to do the work, and send progress messages
// to ProgressTopic.
type WorkProcessor struct {
	WorkSub       *pubsub.Subscription
	ProgressTopic *pubsub.Topic
	Handler       WorkHandler
	StatsLog      *statslog.StatsLog
}

func (wp *WorkProcessor) processMessage(ctx context.Context, msg *pubsub.Message) {
	var taskReqMsg taskpb.TaskReqMsg
	if err := proto.Unmarshal(msg.Data, &taskReqMsg); err != nil {
		glog.Errorf("error decoding msg %s with error %v.", string(msg.Data), err)
		// Non-recoverable error. Will Ack the message to avoid delivering again.
		msg.Ack()
		return
	}

	mu.RLock()
	isActiveJob := activeJobRuns[taskReqMsg.JobrunRelRsrcName] != 0
	mu.RUnlock()

	var taskRespMsg *taskpb.TaskRespMsg
	if isActiveJob {
		start := time.Now()
		taskRespMsg = wp.Handler.Do(ctx, &taskReqMsg)
		wp.StatsLog.AddSample(wp.Handler.Type(), time.Now().Sub(start))
	} else {
		taskRespMsg = buildTaskRespMsg(&taskReqMsg, nil, nil, AgentError{
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
	pubResult := wp.ProgressTopic.Publish(ctx, &pubsub.Message{Data: serializedTaskRespMsg})
	if _, err := pubResult.Get(ctx); err != nil {
		glog.Errorf("Can not publish progress message with err: %v", err)
		// Don't ack the messages, retry again when the message is redelivered.
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
