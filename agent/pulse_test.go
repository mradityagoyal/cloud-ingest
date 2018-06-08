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
	"testing"

	"cloud.google.com/go/pubsub"
	"github.com/GoogleCloudPlatform/cloud-ingest/gcloud"
	"github.com/GoogleCloudPlatform/cloud-ingest/helpers"
	"github.com/golang/mock/gomock"
	"github.com/golang/protobuf/proto"

	pulsepb "github.com/GoogleCloudPlatform/cloud-ingest/proto/pulse_go_proto"
)

// Test if the AgentId proto works as expected.
func TestPulseLocalIds(t *testing.T) {
	testproto := PulseLocalIds("hostname", "pid")
	if testproto.HostName != "hostname" {
		t.Errorf("host name was incorrect, got: %s, want: %s.", testproto.HostName, "hostname")
	}
	if testproto.ProcessId != "pid" {
		t.Errorf("host name was incorrect, got: %s, want: %s.", testproto.ProcessId, "pid")
	}
}

// Test if the Pulse proto works as expected.
func TestMakeAgentPulse(t *testing.T) {
	testproto := PulseLocalIds("testing", "ids")
	testpulse := MakeAgentPulse(testproto, 10)
	if testpulse.AgentId != testproto {
		t.Errorf("AgentId was incorrect, got: %s, want: %s.", testpulse.AgentId, testproto)
	}
	if testpulse.Frequency != 10 {
		t.Errorf("host name was incorrect, got: %d, want: %d.", testpulse.Frequency, 10)
	}
}

// Test if Serialization and deserialization works and returns the same message.
func TestSerializePulse(t *testing.T) {
	testproto := PulseLocalIds("testing", "ids")
	testpulse := MakeAgentPulse(testproto, 10)
	serializedpulse, _ := SerializePulse(testpulse)
	var pulsemsg pulsepb.Msg
	err := proto.Unmarshal(serializedpulse, &pulsemsg)
	if err != nil {
		t.Errorf("error Could not unmarshal serialized pulse, returned error:%e", err)
	}
	if !proto.Equal(testpulse, &pulsemsg) {
		t.Errorf("error Serialized Pulse unmarshaled into: %v expected: %v", &pulsemsg, testpulse)
	}
}

// Test that publish pulse performs as expected
func TestPublishPulse(t *testing.T) {
	ctx := context.Background()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockpstopic := gcloud.NewMockPSTopic(ctrl)
	mockresult := gcloud.NewMockPSPublishResult(ctrl)
	mockresult.EXPECT().Get(ctx).Return("serverid", nil)

	ph, err := NewPulseHandler(mockpstopic, 10)
	if err != nil {
		t.Errorf("Could not create a PulseHandler with Topic:%v And Frequency:%v \n error:%e ", mockpstopic, int32(10), err)
	}

	msg, err := SerializePulse(ph.Pulse)
	if err != nil {
		t.Errorf("Could not Serialize Pulse:%v , returned with error:%e", ph.Pulse, err)
	}
	mockpstopic.EXPECT().Publish(ctx, &pubsub.Message{Data: msg}).Return(mockresult)
	publishresult, err := ph.PublishPulse(ctx)
	if err != nil {
		t.Errorf("Could Not Publish Pulse:%v , return with error %e", publishresult, err)
	}
}

// Test that run preforms as expected
func TestPulseHandlerRun(t *testing.T) {
	ctxparent := context.TODO()
	ctx, _ := context.WithCancel(ctxparent)
	ctrl := gomock.NewController(t)
	defer ctx.Done()
	defer ctrl.Finish()

	mockpstopic := gcloud.NewMockPSTopic(ctrl)
	mockresult := gcloud.NewMockPSPublishResult(ctrl)

	ph, err := NewPulseHandler(mockpstopic, 1)
	if err != nil {
		t.Errorf("Could not create a PulseHandler with Topic:%v And Frequency:%v \n error:%e ", mockpstopic, int32(10), err)
	}

	msg, err := SerializePulse(ph.Pulse)
	if err != nil {
		t.Errorf("Could not Serialize Pulse:%v , returned with error:%e", ph.Pulse, err)
	}

	mockresult.EXPECT().Get(ctx).MaxTimes(3).MinTimes(2).Return("serverid", nil)
	mockpstopic.EXPECT().Publish(ctx, &pubsub.Message{Data: msg}).MaxTimes(3).MinTimes(2).Return(mockresult)
	tick := helpers.NewMockTicker()

	go ph.run(ctx, tick)

	tick.Tick()
	tick.Tick()
	tick.Tick()
}
