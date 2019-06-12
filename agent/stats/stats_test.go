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

package stats

import (
	"context"
	"math"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/GoogleCloudPlatform/cloud-ingest/agent/common"

	taskpb "github.com/GoogleCloudPlatform/cloud-ingest/proto/task_go_proto"
)

func TestTrackerRecordBWLimit(t *testing.T) {
	st := NewTracker(context.Background())
	var wg sync.WaitGroup
	st.selectDone = func() { wg.Done() } // The test hook.
	if got, want := st.lifetime.bwLimit, int64(math.MaxInt32); got != want {
		t.Fatalf("initial bwLimit = %v, want:%v", got, want)
	}
	wg.Add(1)
	st.RecordBWLimit(123456)
	wg.Wait() // Force the Tracker to collect the recorded stats.
	if got, want := st.lifetime.bwLimit, int64(123456); got != want {
		t.Errorf("bwLimit = %v, want:%v", got, want)
	}
}

func TestTrackerRecordCtrlMsg(t *testing.T) {
	st := NewTracker(context.Background())
	var wg sync.WaitGroup
	st.selectDone = func() { wg.Done() } // The test hook.
	if got, want := st.periodic.ctrlMsgsReceived, int64(0); got != want {
		t.Fatalf("initial ctrlMsgsReceived = %v, want:%v", got, want)
	}
	s := st.lifetime.ctrlMsgTime
	for i := 0; i < 10; i++ {
		wg.Add(1)
		st.RecordCtrlMsg(time.Now())
		wg.Wait() // Force the Tracker to collect the recorded stats.
	}
	if got, want := st.periodic.ctrlMsgsReceived, int64(10); got != want {
		t.Errorf("ctrlMsgsReceived = %v, want:%v", got, want)
	}
	if c := st.lifetime.ctrlMsgTime; !c.After(s) {
		t.Errorf("ctrlMsgTime %v not after starting ctrlMsgTime %v", c, s)
	}
}

var (
	psEmpty = &PulseStats{}
	ps1     = &PulseStats{1, 0, 1, 0, 1, 0, 1, 0, 0, 1, 0, 1, 0}
	ps2     = &PulseStats{0, 1, 0, 1, 0, 1, 0, 1, 1, 0, 1, 0, 1}
	ps3     = &PulseStats{1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1} // ps3 = ps1 + ps2
	ps4     = &PulseStats{1, 2, 3, 1, 2, 3, 1, 2, 2, 3, 1, 2, 3}
	ps5     = &PulseStats{2, 4, 6, 2, 4, 6, 2, 4, 4, 6, 2, 4, 6} // ps5 = ps4 + ps4
	ps6     = &PulseStats{9, 9, 9, 9, 9, 9, 9, 9, 9, 9, 9, 9, 9}
	ps7     = &PulseStats{8, 7, 6, 8, 7, 6, 8, 7, 7, 6, 8, 7, 6} // ps7 = ps6 - ps4
)

func TestTrackerAccumulatedPulseStats(t *testing.T) {
	tests := []struct {
		desc   string
		inputs []interface{}
		want   *PulseStats
	}{
		{"Empty", []interface{}{"t"}, psEmpty},
		{"Empty, no accumulator tick", []interface{}{ps1}, psEmpty},
		{"Basic 1", []interface{}{ps1, "t"}, ps1},
		{"Basic 2", []interface{}{ps1, ps2, "t"}, ps3},
		{"Basic 3", []interface{}{ps1, ps2, "t", ps7}, ps3},
		{"Basic 4", []interface{}{ps3, "t", ps3, ps3, "t", ps3, ps3, ps3, "t", ps3, ps3, ps3, "t"}, ps6},
	}
	for _, tc := range tests {
		// Must be done before creating the Tracker.
		mockAccumulatorTicker := common.NewMockTicker()
		accumulatorTickerMaker = func() common.Ticker {
			return mockAccumulatorTicker
		}

		st := NewTracker(context.Background())

		var wg sync.WaitGroup
		st.selectDone = func() { wg.Done() }

		// AccumulatedPulseStats must start empty.
		if got := st.AccumulatedPulseStats(); *got != *psEmpty {
			t.Errorf("AccumulatedPulseStats got %v, want %v", got, psEmpty)
			continue
		}

		// Record all of the PulseStats and accumulator ticks.
		for _, i := range tc.inputs {
			wg.Add(1)
			switch v := i.(type) {
			case string:
				mockAccumulatorTicker.Tick()
			case *PulseStats:
				st.pulseStatsChan <- v
			default:
				t.Fatalf("Unrecognized input type: %T %v", i, i)
			}
			wg.Wait() // Allow the Tracker to collect the input.
		}

		// Validate AccumulatedPulseStats.
		if got := st.AccumulatedPulseStats(); *got != *tc.want {
			t.Errorf("AccumulatedPulseStats() got %v, want %v", got, tc.want)
		}

		// AccumulatedPulseStats should be empty again.
		if got := st.AccumulatedPulseStats(); *got != *psEmpty {
			t.Errorf("AccumulatedPulseStats got %v, want %v", got, psEmpty)
		}
	}
}

func TestPulseStatsAdd(t *testing.T) {
	tests := []struct {
		a    *PulseStats
		b    *PulseStats
		want *PulseStats
	}{
		{psEmpty, psEmpty, psEmpty},
		{ps1, ps2, ps3},
		{ps4, ps4, ps5},
	}
	for i, tc := range tests {
		got := *tc.a // Create a copy to not interfere with other tests.
		if got.add(tc.b); got != *tc.want {
			t.Errorf("%d: PulseStats.add got %v, want %v", i, got, tc.want)
		}
	}
}

func TestPulseStatsSub(t *testing.T) {
	tests := []struct {
		a    *PulseStats
		b    *PulseStats
		want *PulseStats
	}{
		{psEmpty, psEmpty, psEmpty},
		{ps3, ps2, ps1},
		{ps6, ps4, ps7},
	}
	for i, tc := range tests {
		tc.a.sub(tc.b)
		if got := tc.a; *got != *tc.want {
			t.Errorf("%d: PulseStats.sub got %v, want %v", i, got, tc.want)
		}
	}
}

var (
	copyTaskRespMsg = &taskpb.TaskRespMsg{ReqSpec: &taskpb.Spec{Spec: &taskpb.Spec_CopySpec{CopySpec: &taskpb.CopySpec{}}}}
	listTaskRespMsg = &taskpb.TaskRespMsg{ReqSpec: &taskpb.Spec{Spec: &taskpb.Spec_ListSpec{ListSpec: &taskpb.ListSpec{}}}}
)

func TestTrackerDisplayStats(t *testing.T) {
	tests := []struct {
		desc        string
		inputs      []interface{}
		wantSubStrs []string
	}{
		{
			"No inputs",
			[]interface{}{},
			[]string{
				"txRate:      0B/s",
				"txSum:      0B",
				"taskResps[copy:0 list:0]",
				"ctrlMsgAge:0s (ok)",
			},
		},
		{
			"Responded tasks",
			[]interface{}{copyTaskRespMsg, copyTaskRespMsg, listTaskRespMsg},
			[]string{
				"txRate:      0B/s",
				"txSum:      0B",
				"taskResps[copy:2 list:1]",
				"ctrlMsgAge:0s (ok)",
			},
		},
		{
			"Bytes sent",
			[]interface{}{500 * 1024, 500 * 1024, 1000 * 1024},
			[]string{
				"txRate:      0B/s",
				"txSum:  2.0MiB",
				"taskResps[copy:0 list:0]",
				"ctrlMsgAge:0s (ok)",
			},
		},
		{
			"Bytes sent",
			[]interface{}{500 * 1024, 500 * 1024, 1000 * 1024},
			[]string{
				"txRate:      0B/s",
				"txSum:  2.0MiB",
				"taskResps[copy:0 list:0]",
				"ctrlMsgAge:0s (ok)",
			},
		},
		{
			"Control message (ok)",
			[]interface{}{time.Now().Add(-2 * time.Second)},
			[]string{
				"txRate:      0B/s",
				"txSum:      0B",
				"taskResps[copy:0 list:0]",
				"ctrlMsgAge:2s (ok)",
			},
		},
		{
			"Control message (??)",
			[]interface{}{time.Now().Add(-32 * time.Second)},
			[]string{
				"txRate:      0B/s",
				"txSum:      0B",
				"taskResps[copy:0 list:0]",
				"ctrlMsgAge:32s (??)",
			},
		},
		{
			"Combined",
			[]interface{}{500 * 1024, 500 * 1024, 1000 * 1024, time.Now().Add(-2 * time.Second), copyTaskRespMsg, copyTaskRespMsg, listTaskRespMsg},
			[]string{
				"txRate:      0B/s",
				"txSum:  2.0MiB",
				"taskResps[copy:2 list:1]",
				"ctrlMsgAge:2s (ok)",
			},
		},
	}
	for _, tc := range tests {
		st := NewTracker(context.Background())

		// Set up the test hooks.
		var wg sync.WaitGroup
		st.selectDone = func() { wg.Done() } // The test hook.

		// Record all the mocked inputs and ticks.
		for _, i := range tc.inputs {
			wg.Add(1)
			switch v := i.(type) {
			case *taskpb.TaskRespMsg:
				st.RecordTaskResp(v, 50*time.Millisecond)
			case int:
				st.tpTracker.RecordBytesSent(int64(v))
				st.pulseStatsChan <- &PulseStats{CopyBytes: int64(v)}
			case time.Time:
				st.RecordCtrlMsg(v)
			default:
				t.Fatalf("Unrecognized input type: %T %v", i, i)
			}
			wg.Wait() // Allow the Tracker to collect the input.
		}

		got := st.displayStats()
		for _, want := range tc.wantSubStrs {
			if !strings.Contains(got, want) {
				t.Errorf("displayStats = %q, want to contain %q", got, want)
			}
		}
	}
}

func TestByteCountBinary(t *testing.T) {
	tests := []struct {
		b    int64
		pad  int
		want string
	}{
		// Various byte size tests.
		{0, 0, "0B"},
		{10, 0, "10B"},
		{210, 0, "210B"},
		{3210, 0, "3.1KiB"},
		{43210, 0, "42.2KiB"},
		{543210, 0, "530.5KiB"},
		{6543210, 0, "6.2MiB"},
		{76543210, 0, "73.0MiB"},
		{876543210, 0, "835.9MiB"},
		{9876543210, 0, "9.2GiB"},
		{19876543210, 0, "18.5GiB"},
		{109876543210, 0, "102.3GiB"},
		{2109876543210, 0, "1.9TiB"},
		{32109876543210, 0, "29.2TiB"},
		{432109876543210, 0, "393.0TiB"},
		{5432109876543210, 0, "4.8PiB"},
		{65432109876543210, 0, "58.1PiB"},
		{765432109876543210, 0, "679.8PiB"},
		{8765432109876543210, 0, "7.6EiB"},
		// {98765432109876543210, 0, "98.8EB"}, int64 overflow.

		// Pad tests.
		{1, 3, "  1B"},
		{1, 5, "    1B"},
		{12340, 7, " 12.1KiB"},
		{12340000, 7, " 11.8MiB"},
		{2109876543210, 10, "     1.9TiB"},
	}
	for _, tc := range tests {
		got := byteCountBinary(tc.b, tc.pad)
		if got != tc.want {
			t.Errorf("byteCountBinary(%v, %v) = %q, want: %q", tc.b, tc.pad, got, tc.want)
		}
	}
}
