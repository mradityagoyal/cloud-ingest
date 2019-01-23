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
		desc string
		msg  []byte
		ts   time.Time
		want bool
	}{
		{"ok msg", okCtrlMsg, now, true},
		{"stale msg", okCtrlMsg, now.Add(-10 * time.Second), false},
		{"invalid msg", []byte("Invalid message"), now.Add(10 * time.Second), false},
	}
	for _, tc := range tests {
		ch := NewControlHandler(nil, nil)
		called := false
		ch.lastUpdate = now
		ch.processCtrlMsg = func(_ *controlpb.Control, _ *stats.Tracker) { called = true }
		msg := &pubsub.Message{
			Data:        tc.msg,
			PublishTime: tc.ts,
		}
		ch.processMessage(context.Background(), msg)
		if called != tc.want {
			t.Errorf("processMessage(%q) called processCtrlMsg = %t, want: %t", tc.desc, called, tc.want)
		}
	}
}
