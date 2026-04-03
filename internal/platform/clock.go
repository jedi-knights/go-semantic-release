package platform

import "time"

// RealClock implements ports.Clock using the system clock.
type RealClock struct{}

// NewRealClock creates a real clock.
func NewRealClock() *RealClock {
	return &RealClock{}
}

func (c *RealClock) Now() time.Time {
	return time.Now()
}
