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
	sendTickerMaker = func() common.Ticker {
		return common.NewClockTicker(pulseFrequency)
	}
)

const (
	pulseFrequency = 10 * time.Second // The frequency to send pulses.
)

// PulseSender periodically sends "pulses" on the topic passed in during construction.
type PulseSender struct {
	pulseTopic pubsubinternal.PSTopic // The pubsub topic to send pulses on.

	// These fields contain the contents of the pulse message.
	hostname    string
	pid         int
	logsDir     string
	prefix      string
	containerID string
	version     string

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
		hostname:     common.Hostname(),
		pid:          os.Getpid(),
		prefix:       *common.AgentIDPrefix,
		containerID:  *common.ContainerID,
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
	s := ps.statsTracker.AccumulatedPulseStats()
	return &pulsepb.Msg{
		AgentId: &pulsepb.AgentId{
			HostName:    ps.hostname,
			ProcessId:   fmt.Sprintf("%v", ps.pid),
			Prefix:      ps.prefix,
			ContainerId: ps.containerID,
		},
		AgentVersion:  ps.version,
		AgentLogsDir:  ps.logsDir,
		AgentUptimeMs: stats.DurMs(ps.startTime),

		// Accumulated stats, reset with each pulse message.
		AgentTransferredBytes:     s.CopyBytes,
		AgentTransferredListBytes: s.ListBytes,
		CopyOpenMs:                s.CopyOpenMs,
		CopyStatMs:                s.CopyStatMs,
		CopySeekMs:                s.CopySeekMs,
		CopyReadMs:                s.CopyReadMs,
		CopyWriteMs:               s.CopyWriteMs,
		CopyInternalRetries:       s.CopyInternalRetries,
		ListDirOpenMs:             s.ListDirOpenMs,
		ListDirReadMs:             s.ListDirReadMs,
		ListFileWriteMs:           s.ListFileWriteMs,
		ListDirWriteMs:            s.ListDirWriteMs,
	}
}
