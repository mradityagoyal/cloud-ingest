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
	"sync"
	"testing"

	"github.com/GoogleCloudPlatform/cloud-ingest/agent/common"
	pubsubinternal "github.com/GoogleCloudPlatform/cloud-ingest/agent/pubsub"
	"github.com/GoogleCloudPlatform/cloud-ingest/agent/stats"
	pulsepb "github.com/GoogleCloudPlatform/cloud-ingest/proto/pulse_go_proto"
	"github.com/golang/mock/gomock"
	"github.com/google/go-cmp/cmp"
)

func TestPulseSender(t *testing.T) {
	tests := []int{1, 3, 10, 100}
	for _, numPulses := range tests {
		ctx := context.Background()

		// Set up the PubSub mock.
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		mockPublishResult := pubsubinternal.NewMockPSPublishResult(ctrl)
		mockPublishResult.EXPECT().Get(ctx).MaxTimes(numPulses).MinTimes(numPulses).Return("serverid", nil)
		mockPulseTopic := pubsubinternal.NewMockPSTopic(ctrl)

		st := stats.NewTracker(ctx)
		mockSendTicker := common.NewMockTicker()
		sendTickerMaker = func() common.Ticker {
			return mockSendTicker
		}

		logsDir := "/tmp/mylogs"
		ps := NewPulseSender(ctx, mockPulseTopic, logsDir, st)
		ps.hostname = "hostname"
		ps.pid = 1234
		ps.version = "1.2.3"

		mockPulseTopic.EXPECT().Publish(ctx, gomock.Any()).MaxTimes(numPulses).MinTimes(numPulses).Return(mockPublishResult)

		// Set up the test hooks and send the pulses.
		var wg sync.WaitGroup
		ps.selectDone = func() { wg.Done() }
		for i := 0; i < numPulses; i++ {
			wg.Add(1)
			mockSendTicker.Tick()
			wg.Wait()
		}
	}
}

func TestPulseMsg(t *testing.T) {
	agentMsgCmpOpt := cmp.Comparer(func(x, y pulsepb.Msg) bool {
		return (cmp.Equal(x.AgentId, y.AgentId) && x.AgentLogsDir == y.AgentLogsDir &&
			x.AgentVersion == y.AgentVersion)
	})
	tests := []struct {
		hostname string
		pid      int
		prefix   string
		logsDir  string
		version  string
		want     *pulsepb.Msg
	}{
		{
			"hostname", 1234, "", "/tmp/mylogs", "1.2.3",
			&pulsepb.Msg{
				AgentId: &pulsepb.AgentId{
					HostName:  "hostname",
					ProcessId: "1234",
					Prefix:    "",
				},
				AgentVersion: "1.2.3",
				AgentLogsDir: "/tmp/mylogs",
			},
		},
		{
			"69c1725fc298", 11, "myagent", "/tmp/mylogs2", "5.6.7",
			&pulsepb.Msg{
				AgentId: &pulsepb.AgentId{
					HostName:  "69c1725fc298",
					ProcessId: "11",
					Prefix:    "myagent",
				},
				AgentVersion: "5.6.7",
				AgentLogsDir: "/tmp/mylogs2",
			},
		},
	}
	for _, tc := range tests {
		ps := &PulseSender{
			hostname: tc.hostname,
			pid:      tc.pid,
			prefix:   tc.prefix,
			logsDir:  tc.logsDir,
			version:  tc.version,
		}
		if got := ps.pulseMsg(); !cmp.Equal(got, tc.want, agentMsgCmpOpt) {
			t.Errorf("ps.pulseMsg() = %v, want %v", got, tc.want)
		}
	}
}
