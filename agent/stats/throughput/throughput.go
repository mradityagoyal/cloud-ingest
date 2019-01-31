/*
Copyright 2019 Google Inc. All Rights Reserved.
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

package throughput

import (
	"context"
	"sync"
	"time"

	"github.com/GoogleCloudPlatform/cloud-ingest/helpers"
	"github.com/golang/glog"
)

const (
	tpMeasurementDuration = 10 // Throughput measurement duration, in seconds.
)

// Tracker collects bytes sent by the Agent and produces a throughput measurement.
type Tracker struct {
	throughputMu sync.RWMutex
	throughput   int64 // In bytes/second.

	bytesSentChan    chan int64 // Channel to record bytesSent.
	bytesSentRingBuf []int64    // Ring-buffer to hold bytesSent counts.

	// Testing hooks.
	selectDone  func()
	trackTicker helpers.Ticker
}

// NewTracker returns a new Tracker, which can then be used to track bytes sent
// and produce a throughput measurement.
func NewTracker(ctx context.Context) *Tracker {
	t := &Tracker{
		bytesSentChan:    make(chan int64, 100), // Large buffer to avoid blocking.
		bytesSentRingBuf: make([]int64, tpMeasurementDuration),
		selectDone:       func() {},
		trackTicker:      helpers.NewClockTicker(1 * time.Second),
	}
	go t.track(ctx)
	return t
}

// RecordBytesSent tracks bytes sent. For accurate throughput measurement this function
// should be called every time bytes are sent on the wire. More frequent and granular
// calls to this function will provide a more accurate throughput measurement.
func (t *Tracker) RecordBytesSent(bytes int64) {
	t.bytesSentChan <- bytes
}

// Throughput returns the current measured throughput in bytes/second.
func (t *Tracker) Throughput() int64 {
	t.throughputMu.RLock()
	defer t.throughputMu.RUnlock()
	return t.throughput
}

func (t *Tracker) track(ctx context.Context) {
	ringBufIdx := 0
	for {
		select {
		case <-ctx.Done():
			if err := ctx.Err(); err != nil {
				glog.Infof("throughput.Tracker track ctx ended with err: %v", err)
			}
			return
		case bytes := <-t.bytesSentChan:
			t.bytesSentRingBuf[ringBufIdx] += bytes
		case <-t.trackTicker.GetChannel():
			// Calculate the current throughput.
			var totalBytes int64
			for _, b := range t.bytesSentRingBuf {
				totalBytes += b
			}
			t.throughputMu.Lock()
			t.throughput = totalBytes / int64(len(t.bytesSentRingBuf))
			t.throughputMu.Unlock()

			// Rotate the ring-buffer, reset the new slot.
			ringBufIdx = (ringBufIdx + 1) % len(t.bytesSentRingBuf)
			t.bytesSentRingBuf[ringBufIdx] = 0
		}
		t.selectDone() // Testing hook.
	}
}
