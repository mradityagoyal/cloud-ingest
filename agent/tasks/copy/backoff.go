package copy

import (
	"time"
)

var (
	minBackOffDelay  = 1 * time.Second
	maxBackOffDelay  = 32 * time.Second
	totalDelayCutoff = 15 * time.Minute
)

// Backoff provides a back-off scheme for retrying events.
type BackOff struct {
	prevDelay  time.Duration
	totalDelay time.Duration
}

// GetDelay returns a delay duration and bool indicating whether or not the caller
// should continue using the delay and retrying the event. Every iteration the
// delay grows exponentially up to the maxBackOffDelay.
func (b *BackOff) GetDelay() (time.Duration, bool) {
	if b.totalDelay > totalDelayCutoff {
		return 0, false
	}
	var delay time.Duration
	if b.prevDelay < minBackOffDelay {
		delay = minBackOffDelay
	} else if b.prevDelay >= maxBackOffDelay {
		delay = maxBackOffDelay
	} else {
		delay = b.prevDelay * 2
		if delay >= maxBackOffDelay {
			delay = maxBackOffDelay
		}
	}
	b.totalDelay += delay
	b.prevDelay = delay
	return delay, true
}
