package agent

import (
	"testing"
	"time"
)

func TestGetDelay(t *testing.T) {
	var b Backoff
	for i := uint(0); true; i++ {
		wantMinDelay := time.Duration((1 << i) * int64(startingDelay))
		wantMaxDelay := 2 * wantMinDelay
		gotDelay, gotRetry := b.GetDelay()
		if !gotRetry {
			break
		}
		if gotDelay < wantMinDelay {
			t.Errorf("%d: gotDelay = %v, want >= %v", i, gotDelay, wantMinDelay)
		}
		if gotDelay > wantMaxDelay {
			t.Errorf("%d: gotDelay = %v, want <= %v", i, gotDelay, wantMaxDelay)
		}
	}
	if b.totalDelay < totalDelayCutoff {
		t.Errorf("totalDelay %v, want > %v", b.totalDelay, totalDelayCutoff)
	}
}
