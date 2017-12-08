package helpers

import (
	"time"
)

// Useful construct allowing us to abstract time.
type Clock interface {
	Now() time.Time
}

type realClock struct{}

func (realClock) Now() time.Time {
	return time.Now()
}

func NewClock() Clock {
	return &realClock{}
}
