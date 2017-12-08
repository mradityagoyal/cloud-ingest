package helpers

import (
	"testing"
	"time"
)

func TestRealClock_Now(t *testing.T) {
	// Check that clock returns the current time. Since this
	// is time-dependent, just check that it's in between two
	// times we look up before and after.
	low := time.Now()
	now := NewClock().Now()
	high := time.Now()

	if low.After(now) || high.Before(now) {
		t.Errorf("wanted result in range [%v, %v], but got %v", low, high, now)
	}
}
