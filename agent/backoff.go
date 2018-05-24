package agent

import (
	"math/rand"
	"time"
)

const (
	startingDelay    = 125 * time.Millisecond
	totalDelayCutoff = 60 * time.Second
)

// Backoff provides a backoff scheme for retrying events.
type Backoff struct {
	delayCount uint64
	totalDelay time.Duration
}

// GetDelay returns a delay duration and bool indicating whether or not the caller
// should continue using the delay and retrying the event. Every iteration the
// delay grows exponentially (with jitter).
func (b *Backoff) GetDelay() (time.Duration, bool) {
	if b.totalDelay > totalDelayCutoff {
		return 0, false
	}
	delay := time.Duration((1 << b.delayCount) * int64(startingDelay))
	delay += time.Duration(rand.Int63n(int64(delay)))
	b.totalDelay += delay
	b.delayCount++
	return delay, true
}
