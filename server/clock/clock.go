// Package clock provides time abstractions.
package clock

import "time"

// Clock provides time access for scheduling.
type Clock interface {
	Now() time.Time
}

type realClock struct{}

func (realClock) Now() time.Time {
	return time.Now()
}

// NewReal returns a Clock backed by time.Now.
func NewReal() Clock {
	return realClock{}
}
