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
	"os"
	"time"

	"cloud.google.com/go/pubsub"
	"github.com/GoogleCloudPlatform/cloud-ingest/gcloud"
	"github.com/GoogleCloudPlatform/cloud-ingest/helpers"
	"github.com/golang/glog"
	"github.com/golang/protobuf/proto"

	pulsepb "github.com/GoogleCloudPlatform/cloud-ingest/proto/pulse_go_proto"
)

// Records all the info needed to make a pulse and publish a pulse message
type PulseHandler struct {
	PulseTopic gcloud.PSTopic
	Pulse      *pulsepb.Msg
	Frequency  int32
	Ticker     helpers.Ticker
}

// Returns Hostname as string
func GetHostName() (string, error) {
	host_name, err := os.Hostname()
	if err != nil {
		glog.Errorf("Cannot pull host name error: %v", err)
		return "error", err
	}
	return host_name, err
}

// Returns ProcessId as a string
func GetProcessId() string {
	return string(os.Getpid())
}

// Takes two strings and returns the AgentId Proto
func PulseLocalIds(host_name, process_id string) *pulsepb.AgentId {
	return &pulsepb.AgentId{host_name, process_id}
}

// Takes a pointer to an AgentId Proto and an int (frequency) returns Pulse message
func MakeAgentPulse(id *pulsepb.AgentId, frequency int32) *pulsepb.Msg {
	return &pulsepb.Msg{id, frequency}
}

// Creates the Serialized Pulse Message
func SerializePulse(pulse *pulsepb.Msg) ([]byte, error) {
	serializedPulseMessage, err := proto.Marshal(pulse)
	if err != nil {
		glog.Errorf("Cannot marshal pulsepb %+v with error:%v", pulse, err)
	}
	return serializedPulseMessage, err
}

// Creates a new PulseHandler
func NewPulseHandler(topic gcloud.PSTopic, frequency int32) (*PulseHandler, error) {
	ph := &PulseHandler{topic, nil, frequency, nil}
	host_name, err := GetHostName()
	if err != nil {
		return ph, err
	}
	pulse_local_ids := PulseLocalIds(host_name, GetProcessId())
	ph.Pulse = MakeAgentPulse(pulse_local_ids, ph.Frequency)
	ph.Ticker = helpers.NewClockTicker(time.Duration(ph.Frequency) * time.Second)
	return ph, err
}

// Publishes a Pulse based on PulseHandler's information.
func (ph *PulseHandler) PublishPulse(ctx context.Context) (gcloud.PSPublishResult, error) {
	serializedPulseMessage, err := SerializePulse(ph.Pulse)
	if err != nil {
		return nil, err
	}
	pubResult := ph.PulseTopic.Publish(ctx, &pubsub.Message{Data: serializedPulseMessage})
	_, err = pubResult.Get(ctx)
	if err != nil {
		glog.Errorf("Failed to publish pulse topic message with error:%v", err)
	}
	return pubResult, err
}

// Runs the loop to repeatedly send pulse messages.
func (ph *PulseHandler) Run(ctx context.Context) error {
	tick := ph.Ticker
	return ph.run(ctx, tick)
}

// Private form of run, takes ticker as added argument.
// Used for testing.
func (ph *PulseHandler) run(ctx context.Context, tick helpers.Ticker) error {
	for {
		select {
		case <-tick.GetChannel():
			_, err := ph.PublishPulse(ctx)
			if err != nil {
				return err
			}
		case <-ctx.Done():
			return nil
		}
	}
	return nil
}
