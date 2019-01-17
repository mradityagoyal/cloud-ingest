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

package rate

import (
	"bytes"
	"context"
	"math"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"

	"golang.org/x/time/rate"

	controlpb "github.com/GoogleCloudPlatform/cloud-ingest/proto/control_go_proto"
)

func TestProcessCtrlMsg(t *testing.T) {
	jrBW0 := &controlpb.JobRunBandwidth{JobrunRelRsrcName: "job-0", Bandwidth: 0}
	jrBW1 := &controlpb.JobRunBandwidth{JobrunRelRsrcName: "job-1", Bandwidth: 10}
	jrBW2 := &controlpb.JobRunBandwidth{JobrunRelRsrcName: "job-2", Bandwidth: 20}
	tests := []struct {
		desc               string
		cm                 *controlpb.Control
		wantJobRunBW       map[string]int64
		wantProjectBWLimit rate.Limit
	}{
		{
			"empty",
			&controlpb.Control{},
			map[string]int64{},
			rate.Limit(0),
		},
		{
			"zero bandwidth jobrun",
			&controlpb.Control{JobRunsBandwidths: []*controlpb.JobRunBandwidth{jrBW0}},
			map[string]int64{"job-0": 0},
			rate.Limit(0),
		},
		{
			"one jobrun",
			&controlpb.Control{JobRunsBandwidths: []*controlpb.JobRunBandwidth{jrBW1}},
			map[string]int64{"job-1": 10},
			rate.Limit(10),
		},
		{
			"some jobruns",
			&controlpb.Control{JobRunsBandwidths: []*controlpb.JobRunBandwidth{jrBW1, jrBW2}},
			map[string]int64{"job-1": 10, "job-2": 20},
			rate.Limit(30),
		},
		{
			"mix of jobruns",
			&controlpb.Control{JobRunsBandwidths: []*controlpb.JobRunBandwidth{jrBW0, jrBW2}},
			map[string]int64{"job-0": 0, "job-2": 20},
			rate.Limit(20),
		},
	}
	for _, tc := range tests {
		ProcessCtrlMsg(tc.cm, nil)
		if got, want := jobRunBW, tc.wantJobRunBW; !cmp.Equal(got, want) {
			t.Errorf("ProcessCtrlMsg(%q): jobRunBW = %v, want: %v", tc.desc, got, want)
		}
		if got, want := projectBWLimiter.Limit(), tc.wantProjectBWLimit; got != want {
			t.Errorf("ProcessCtrlMsg(%q): Limit() = %v, want: %v", tc.desc, got, want)
		}
	}
}

func TestRateLimitingReaderReadNoBufferResize(t *testing.T) {
	readBuf := make([]byte, 1000)
	reader := bytes.NewReader(readBuf)

	projectBWLimiter = rate.NewLimiter(rate.Limit(1000), math.MaxInt32) // One byte per millisecond.
	// Drain the limiter, so we can get accurate timing.
	projectBWLimiter.WaitN(context.Background(), math.MaxInt32)

	r := NewRateLimitingReader(reader)
	if _, ok := r.(RateLimitingReader); !ok {
		t.Errorf("want type RateLimitingReader, got type %T", reader)
	}

	writeBuf := make([]byte, 10)

	start := time.Now()
	// Since the writeBuf is only 10 bytes, we expect Read to yield
	// 10 bytes and take ~10ms.
	n, err := r.Read(writeBuf)
	if err != nil {
		t.Error("Read got err:", err)
	}
	if n != 10 {
		t.Errorf("want Read 10 bytes, got %d", n)
	}
	totalTime := time.Since(start)
	if totalTime < 10*time.Millisecond {
		t.Errorf("total time want >=10ms, got %v", totalTime)
	}
}

func TestRateLimitingReaderReadBufferResize(t *testing.T) {
	readBuf := make([]byte, 2000)
	reader := bytes.NewReader(readBuf)
	projectBWLimiter = rate.NewLimiter(rate.Limit(1000), math.MaxInt32) // One byte per millisecond.
	// Drain the limiter, so we can get accurate timing.
	projectBWLimiter.WaitN(context.Background(), math.MaxInt32)

	r := NewRateLimitingReader(reader)
	if _, ok := r.(RateLimitingReader); !ok {
		t.Errorf("want type RateLimitingReader, got type %T", reader)
	}

	writeBuf := make([]byte, 2000)

	start := time.Now()
	// Since we've overridden the chunksPerSecond to be 100, we expect Read
	// to yield 10 bytes and take ~10ms.
	n, err := r.Read(writeBuf)
	if err != nil {
		t.Error("Read got err:", err)
	}
	if n != 1000 {
		t.Errorf("want Read 1000 bytes, got %d", n)
	}
	totalTime := time.Since(start)
	if totalTime < 1*time.Second {
		t.Errorf("total time want >=1s, got %v", totalTime)
	}
}
