package platform_test

import (
	"testing"
	"time"

	"github.com/jedi-knights/go-semantic-release/internal/platform"
)

func TestRealClock_Now_ReturnsCurrentTime(t *testing.T) {
	clk := platform.NewRealClock()
	before := time.Now()
	got := clk.Now()
	after := time.Now()

	if got.Before(before) || got.After(after) {
		t.Errorf("Now() = %v, expected between %v and %v", got, before, after)
	}
}

func TestRealClock_Now_IsNotZero(t *testing.T) {
	clk := platform.NewRealClock()
	if clk.Now().IsZero() {
		t.Error("Now() returned zero time")
	}
}
