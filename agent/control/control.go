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
	"github.com/GoogleCloudPlatform/cloud-ingest/agent/rate"
	"github.com/GoogleCloudPlatform/cloud-ingest/agent/stats"
	"github.com/golang/glog"
	"github.com/golang/protobuf/proto"

	controlpb "github.com/GoogleCloudPlatform/cloud-ingest/proto/control_go_proto"
)

// ControlHandler is responsible for receiving control messages pushed by the backend service.
type ControlHandler struct {
	sub          *pubsub.Subscription
	lastUpdate   time.Time
	statsTracker *stats.Tracker

	processCtrlMsg func(cm *controlpb.Control, st *stats.Tracker) // Test hook.
}

// NewControlHandler creates an instance of ControlHandler.
func NewControlHandler(s *pubsub.Subscription, st *stats.Tracker) *ControlHandler {
	return &ControlHandler{
		sub:            s,
		lastUpdate:     time.Now(),
		statsTracker:   st,
		processCtrlMsg: rate.ProcessCtrlMsg,
	}
}

// HandleControlMessages starts handling control messages sent by the service. This
// is blocking function, it will only return in case of non-retriable errors.
// TODO(b/117972265): This method should detect control messages absence, and act accordingly.
func (ch *ControlHandler) HandleControlMessages(ctx context.Context) error {
	// Set the max outstanding messages to 1, so there is only one go routine processing
	// the messages.
	ch.sub.ReceiveSettings.MaxOutstandingMessages = 1
	return ch.sub.Receive(ctx, ch.processMessage)
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

	ch.processCtrlMsg(&controlMsg, ch.statsTracker)

	ch.lastUpdate = msg.PublishTime
	if ch.statsTracker != nil {
		ch.statsTracker.RecordCtrlMsg(msg.PublishTime)
	}
}
