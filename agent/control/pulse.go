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
	"flag"
	"fmt"
	"os"
	"time"

	"cloud.google.com/go/pubsub"
	"github.com/GoogleCloudPlatform/cloud-ingest/agent/common"
	pubsubinternal "github.com/GoogleCloudPlatform/cloud-ingest/agent/pubsub"
	"github.com/GoogleCloudPlatform/cloud-ingest/agent/stats"
	"github.com/GoogleCloudPlatform/cloud-ingest/agent/versions"
	"github.com/golang/glog"
	"github.com/golang/protobuf/proto"

	pulsepb "github.com/GoogleCloudPlatform/cloud-ingest/proto/pulse_go_proto"
)

var (
	agentIDPrefix = flag.String("agent-id-prefix", "", "A a prefix to include on the agent id")
	sendTickerMaker = func() common.Ticker {
		return common.NewClockTicker(pulseFrequency)
	}
)

const (
	pulseFrequency = 10 * time.Second // The frequency to send pulses.
)

// Hostname returns the hostname string.
func Hostname() string {
	hn, err := os.Hostname()
	if err != nil {
		hn = "hostnameunknown"
	}
	return hn
}

// AgentID returns the ID of this agent.
func AgentID() *pulsepb.AgentId {
	return &pulsepb.AgentId{
		HostName:  Hostname(),
		ProcessId: fmt.Sprintf("%v", os.Getpid()),
		Prefix:    *agentIDPrefix,
	}
}

// PulseSender periodically sends "pulses" on the topic passed in during construction.
type PulseSender struct {
	pulseTopic pubsubinternal.PSTopic // The pubsub topic to send pulses on.

	// These fields contain the contents of the pulse message.
	hostname string
	pid      int
	logsDir  string
	prefix   string
	version  string

	// Used to get live bandwidth measurements.
	statsTracker *stats.Tracker

	// Time of instantiation of this struct.
	startTime time.Time

	// Testing hooks.
	selectDone func()
	sendTicker common.Ticker
}

// NewPulseSender returns a new PulseSender.
func NewPulseSender(ctx context.Context, t pubsubinternal.PSTopic, logsDir string, st *stats.Tracker) *PulseSender {
	ps := &PulseSender{
		pulseTopic:   t,
		hostname:     Hostname(),
		pid:          os.Getpid(),
		prefix:       *agentIDPrefix,
		logsDir:      logsDir,
		version:      versions.AgentVersion().String(),
		statsTracker: st,
		startTime:    time.Now(),
		selectDone:   func() {},
		sendTicker:   sendTickerMaker(),
	}
	go ps.sendPulses(ctx)
	return ps
}

func (ps *PulseSender) sendPulses(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			if err := ctx.Err(); err != nil {
				glog.Infof("sendPulses ctx ended with err: %v", err)
			}
			return
		case <-ps.sendTicker.GetChannel():
			pulseMsg := ps.pulseMsg()
			serializedPulseMsg, err := proto.Marshal(pulseMsg)
			if err != nil {
				glog.Errorf("sendPulses err, proto.Marshal(%v) got err: %v", pulseMsg, err)
				break
			}
			psm := &pubsub.Message{Data: serializedPulseMsg}
			pubResult := ps.pulseTopic.Publish(ctx, psm)
			_, err = pubResult.Get(ctx)
			if err != nil {
				glog.Errorf("sendPulses err, Publish(%v) got err: %v", psm, err)
			} else {
				ps.statsTracker.RecordPulseMsg()
			}
		}
		ps.selectDone() // Testing hook.
	}
}

func (ps *PulseSender) pulseMsg() *pulsepb.Msg {
	transferredBytes := ps.statsTracker.AccumulatedBytesCopied()
	return &pulsepb.Msg{
		AgentId: &pulsepb.AgentId{
			HostName:  ps.hostname,
			ProcessId: fmt.Sprintf("%v", ps.pid),
			Prefix:    ps.prefix,
		},
		AgentVersion:          ps.version,
		AgentLogsDir:          ps.logsDir,
		AgentTransferredBytes: transferredBytes,
		// Seconds() returns the duration as a floating point
		AgentUptimeMs: int64(time.Now().Sub(ps.startTime).Seconds() * 1000),
	}
}
