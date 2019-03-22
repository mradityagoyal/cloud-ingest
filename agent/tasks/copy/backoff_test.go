package copy

import (
	"testing"
	"time"
)

func TestGetDelay(t *testing.T) {
	tests := []struct {
		desc    string
		minBOD  time.Duration
		maxBOD  time.Duration
		totalDC time.Duration
	}{
		{"Current", 1 * time.Second, 32 * time.Second, 15 * time.Minute},
		{"Not power of 2", 1 * time.Second, 60 * time.Second, 5 * time.Minute},
		{"SubSecond minDelay", 250 * time.Millisecond, 8 * time.Second, 1 * time.Minute},
		{"MinDelay=MaxDelay", 1 * time.Second, 1 * time.Second, 10 * time.Second},
	}
	for _, tc := range tests {
		minBackOffDelay = tc.minBOD
		maxBackOffDelay = tc.maxBOD
		totalDelayCutoff = tc.totalDC

		var b BackOff
		for i := uint(0); true; i++ {
			wantDelay := (1 << i) * minBackOffDelay
			if wantDelay > maxBackOffDelay {
				wantDelay = maxBackOffDelay
			}
			gotDelay, gotRetry := b.GetDelay()
			if !gotRetry {
				break
			}
			if gotDelay != wantDelay {
				t.Errorf("%v, iteration %d: gotDelay = %v, want %v", tc.desc, i, gotDelay, wantDelay)
			}
		}
		if b.totalDelay < totalDelayCutoff {
			t.Errorf("%v, totalDelay %v, want >= %v", tc.desc, b.totalDelay, totalDelayCutoff)
		}
		if maxTotalDelay := totalDelayCutoff + maxBackOffDelay; b.totalDelay > maxTotalDelay {
			t.Errorf("%v, totalDelay %v, want <= %v", tc.desc, b.totalDelay, maxTotalDelay)
		}
	}
}
