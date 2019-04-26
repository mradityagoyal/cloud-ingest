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
	"testing"
	"time"

	"cloud.google.com/go/pubsub"
	"github.com/GoogleCloudPlatform/cloud-ingest/agent/stats"
	"github.com/golang/protobuf/proto"

	controlpb "github.com/GoogleCloudPlatform/cloud-ingest/proto/control_go_proto"
	pulsepb "github.com/GoogleCloudPlatform/cloud-ingest/proto/pulse_go_proto"
)

func marshalControlMessage(t *testing.T, msg *controlpb.Control) []byte {
	data, err := proto.Marshal(msg)
	if err != nil {
		t.Fatalf("marshal control message: %v got err: %v", msg, err)
	}
	return data
}

func TestProcessMessage(t *testing.T) {
	okCtrlMsg := marshalControlMessage(t, &controlpb.Control{
		JobRunsBandwidths: []*controlpb.JobRunBandwidth{
			&controlpb.JobRunBandwidth{JobrunRelRsrcName: "job-1", Bandwidth: 20},
		},
	})
	now := time.Now()
	tests := []struct {
		desc       string
		msg        []byte
		ts         time.Time
		wantCalled bool
	}{
		{"ok msg", okCtrlMsg, now, true},
		{"stale msg", okCtrlMsg, now.Add(-10 * time.Second), false},
		{"invalid msg", []byte("Invalid message"), now.Add(10 * time.Second), false},
	}

	logDir := "/tmp"
	for _, tc := range tests {
		ch := NewControlHandler(nil, nil, logDir)
		processJobRunBandwidthsCalled := false
		processAgentUpdateCalled := false
		ch.lastUpdate = now
		ch.processJobRunBandwidths = func(_ []*controlpb.JobRunBandwidth, _ *stats.Tracker) { processJobRunBandwidthsCalled = true }
		ch.processAgentUpdateMsg = func(_ *controlpb.AgentUpdate, _ *pulsepb.AgentId, _ string) { processAgentUpdateCalled = true }
		msg := &pubsub.Message{
			Data:        tc.msg,
			PublishTime: tc.ts,
		}
		ch.processMessage(context.Background(), msg)
		if processJobRunBandwidthsCalled != tc.wantCalled {
			t.Errorf("processMessage(%q) called processJobRunBandwidths = %t, want: %t", tc.desc, processJobRunBandwidthsCalled, tc.wantCalled)
		}
		if processAgentUpdateCalled != tc.wantCalled {
			t.Errorf("processMessage(%q) called processAgentUpdateMsg = %t, want: %t", tc.desc, processAgentUpdateCalled, tc.wantCalled)
		}
	}
}
