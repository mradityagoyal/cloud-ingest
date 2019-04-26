/*
Copyright 2018 Google Inc. All Rights Reserved.
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

package control

import (
	"context"
	"time"

	"cloud.google.com/go/pubsub"
	"github.com/GoogleCloudPlatform/cloud-ingest/agent/agentupdate"
	"github.com/GoogleCloudPlatform/cloud-ingest/agent/rate"
	"github.com/GoogleCloudPlatform/cloud-ingest/agent/stats"
	"github.com/golang/glog"
	"github.com/golang/protobuf/proto"

	controlpb "github.com/GoogleCloudPlatform/cloud-ingest/proto/control_go_proto"
	pulsepb "github.com/GoogleCloudPlatform/cloud-ingest/proto/pulse_go_proto"
)

// ControlHandler is responsible for receiving control messages pushed by the backend service.
type ControlHandler struct {
	sub          *pubsub.Subscription
	lastUpdate   time.Time
	statsTracker *stats.Tracker
	logDir       string

	// Test hooks.
	processJobRunBandwidths func(jobBWs []*controlpb.JobRunBandwidth, st *stats.Tracker)
	processAgentUpdateMsg   func(au *controlpb.AgentUpdate, agentID *pulsepb.AgentId, agentLogsDir string)
}

// NewControlHandler creates an instance of ControlHandler.
func NewControlHandler(s *pubsub.Subscription, st *stats.Tracker, logDir string) *ControlHandler {
	return &ControlHandler{
		sub:                     s,
		lastUpdate:              time.Now(),
		statsTracker:            st,
		logDir:                  logDir,
		processJobRunBandwidths: rate.ProcessJobRunBandwidths,
		processAgentUpdateMsg:   agentupdate.ProcessAgentUpdateMsg,
	}
}

// Process handles control messages sent by the DCP. This is a blocking function.
// TODO(b/117972265): This method should detect control messages absence, and act accordingly.
func (ch *ControlHandler) Process(ctx context.Context) {
	err := ch.sub.Receive(ctx, ch.processMessage)
	if err != nil {
		glog.Fatalf("%s.Receive() got err: %v", ch.sub.String(), err)
	}
}

func (ch *ControlHandler) processMessage(_ context.Context, msg *pubsub.Message) {
	if ch.sub != nil {
		defer msg.Ack()
	}

	var controlMsg controlpb.Control
	if err := proto.Unmarshal(msg.Data, &controlMsg); err != nil {
		glog.Errorf("error decoding msg %s, publish time: %v, error %v", string(msg.Data), msg.PublishTime, err)
		// Non-recoverable error. Will Ack the message to avoid delivering again.
		return
	}

	if msg.PublishTime.Before(ch.lastUpdate) {
		// Ignore stale messages.
		glog.Infof("Ignore stale message: %v, publish time: %v", controlMsg, msg.PublishTime)
		return
	}

	ch.processJobRunBandwidths(controlMsg.GetJobRunsBandwidths(), ch.statsTracker)
	ch.processAgentUpdateMsg(controlMsg.GetAgentUpdates(), AgentID(), ch.logDir)

	ch.lastUpdate = msg.PublishTime
	ch.statsTracker.RecordCtrlMsg(msg.PublishTime)
}
