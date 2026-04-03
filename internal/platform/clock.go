package platform

import (
	"time"

	"github.com/jedi-knights/go-semantic-release/internal/ports"
)

// Compile-time interface compliance check.
var _ ports.Clock = (*RealClock)(nil)

// RealClock implements ports.Clock using the system clock.
type RealClock struct{}

// NewRealClock creates a real clock.
func NewRealClock() *RealClock {
	return &RealClock{}
}

func (c *RealClock) Now() time.Time {
	return time.Now()
}
