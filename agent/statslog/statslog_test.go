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

package statslog

import (
	"testing"
	"time"
)

func TestStatsLog(t *testing.T) {
	type sample struct {
		msgType string
		d       time.Duration
	}
	samples := []sample{
		sample{"copy", 0 * time.Millisecond},
		sample{"copy", 1 * time.Millisecond},
		sample{"copy", 2 * time.Millisecond},
		sample{"copy", 3 * time.Millisecond},
		sample{"copy", 4 * time.Millisecond},
		sample{"list", 5 * time.Millisecond},
		sample{"list", 6 * time.Millisecond},
		sample{"list", 7 * time.Millisecond},
		sample{"list", 8 * time.Millisecond},
		sample{"list", 9 * time.Millisecond},
	}
	tests := []struct {
		testDesc string
		samples  []sample
		want     string
	}{
		{"No samples", []sample{}, ""},
		{"Copy samples", samples[:5], "type(count)[time min,max,avg]:\n\tcopy(5)[0s,4ms,2ms]"},
		{"List samples", samples[5:], "type(count)[time min,max,avg]:\n\tlist(5)[5ms,9ms,7ms]"},
		{"Both samples", samples, "type(count)[time min,max,avg]:\n\tcopy(5)[0s,4ms,2ms]\n\tlist(5)[5ms,9ms,7ms]"},
	}
	for _, tc := range tests {
		sl := New()
		for _, s := range tc.samples {
			sl.AddSample(s.msgType, s.d)
		}
		got := sl.calcStatsAndLog()

		if got != tc.want {
			t.Errorf("calcStatsAndLog = %q, want %q", got, tc.want)
		}
	}
}
