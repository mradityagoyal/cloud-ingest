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

package agent

import (
	"context"
	"math"
	"testing"
	"time"

	"cloud.google.com/go/pubsub"
	"github.com/golang/protobuf/proto"
	"github.com/google/go-cmp/cmp"

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
	ctx := context.Background()
	ch := NewControlHandler(nil, nil)
	now := time.Now()

	tests := []struct {
		desc string
		msg  []byte
		ts   time.Time
		want map[string]int64
	}{
		{
			desc: "first message, with no active jobs",
			msg:  marshalControlMessage(t, &controlpb.Control{}),
			ts:   now,
			want: map[string]int64{},
		},
		{
			desc: "outdated message, should not change the previous assignment",
			msg: marshalControlMessage(t, &controlpb.Control{
				JobRunsBandwidths: []*controlpb.JobRunBandwidth{
					&controlpb.JobRunBandwidth{
						JobrunRelRsrcName: "job-1",
						Bandwidth:         20,
					},
				},
			}),
			ts:   now.Add(-10 * time.Second),
			want: map[string]int64{},
		},
		{
			desc: "some active job runs",
			msg: marshalControlMessage(t, &controlpb.Control{
				JobRunsBandwidths: []*controlpb.JobRunBandwidth{
					&controlpb.JobRunBandwidth{
						JobrunRelRsrcName: "job-1",
						Bandwidth:         20,
					},
					&controlpb.JobRunBandwidth{
						JobrunRelRsrcName: "job-2",
						Bandwidth:         10,
					},
				},
			}),
			ts:   now.Add(10 * time.Second),
			want: map[string]int64{"job-1": int64(20), "job-2": int64(10)},
		},
		{
			desc: "no active job runs",
			msg:  marshalControlMessage(t, &controlpb.Control{}),
			ts:   now.Add(20 * time.Second),
			want: map[string]int64{},
		},
		{
			desc: "one active job runs",
			msg: marshalControlMessage(t, &controlpb.Control{
				JobRunsBandwidths: []*controlpb.JobRunBandwidth{
					&controlpb.JobRunBandwidth{
						JobrunRelRsrcName: "job-3",
						Bandwidth:         math.MaxInt64,
					},
				},
			}),
			ts:   now.Add(30 * time.Second),
			want: map[string]int64{"job-3": math.MaxInt64},
		},
		{
			desc: "invalid message data bytes",
			msg:  []byte("Invalid message"),
			ts:   now.Add(40 * time.Second),
			want: map[string]int64{"job-3": math.MaxInt64},
		},
	}

	for _, tc := range tests {
		msg := &pubsub.Message{
			Data:        tc.msg,
			PublishTime: tc.ts,
		}
		ch.processMessage(ctx, msg)
		if !cmp.Equal(activeJobRuns, tc.want) {
			t.Errorf("processMessage(%q) set active job run to: %v, want: %v", tc.desc, activeJobRuns, tc.want)
		}
	}
}
