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
	"testing"

	"github.com/GoogleCloudPlatform/cloud-ingest/agent/common"
)

func TestThroughputTracker(t *testing.T) {
	// A sample 'input stream', to help construct some of the test cases.
	var iStream []interface{}
	for i := 0; i < tpMeasurementDuration; i++ {
		iStream = append(iStream, 10)
		iStream = append(iStream, "t")
	}
	for i := 0; i < tpMeasurementDuration; i++ {
		iStream = append(iStream, "t")
	}

	tests := []struct {
		desc   string
		inputs []interface{}
		want   int64
	}{
		// The 'inputs' are a stream of commands, either bytes to send to RecordBytesSent,
		// or a string "t" for trackTicker ticks.

		{"Zero, just a tick", []interface{}{"t"}, 0},
		{"Zero, bytes with no tick", []interface{}{10}, 0},
		{"Zero, no tick after bytes", []interface{}{"t", 10}, 0},

		{"Basic 1", []interface{}{10, "t"}, 10 / tpMeasurementDuration},
		{"Basic 2", []interface{}{10, 10, "t"}, 20 / tpMeasurementDuration},
		{"Basic 3", []interface{}{20, "t"}, 20 / tpMeasurementDuration},
		{"Basic 4", []interface{}{20, "t", 1000}, 20 / tpMeasurementDuration},

		{"Continuous stream", iStream[:tpMeasurementDuration*2], 10},
		{"Continuous stream, empty ticks ", iStream, 0},
	}
	for _, tc := range tests {
		// Must be done before creating the Tracker.
		mockTrackTicker := common.NewMockTicker()
		trackTickerMaker = func() common.Ticker {
			return mockTrackTicker
		}
		bwt := NewTracker(context.Background())
		var wg sync.WaitGroup
		bwt.selectDone = func() { wg.Done() }

		// Record all the mocked inputs and ticks.
		for _, i := range tc.inputs {
			wg.Add(1)
			switch v := i.(type) {
			case int:
				bwt.RecordBytesSent(int64(v))
			case string:
				mockTrackTicker.Tick()
			default:
				t.Fatalf("Unrecognized input type: %T %v", i, i)
			}
			wg.Wait() // Allow the Tracker to collect the input.
		}

		got := bwt.Throughput()
		if got != tc.want {
			t.Errorf("test %q: Throughput = %v, want %v", tc.desc, got, tc.want)
		}
	}
}
