package backoff_test

import (
	"math/rand/v2"
	"testing"
	"time"

	"github.com/jatassi/slipstream/internal/backoff"
)

// A retry count far past the ceiling neither overflows nor passes it.
func TestDelayStopsAtTheCeiling(t *testing.T) {
	r := rand.New(rand.NewPCG(1, 2)) //nolint:gosec // a fixed draw for a literal
	if d := backoff.Delay(time.Second, time.Minute, 200, r); d != 53081198080 {
		t.Fatalf("Delay(n=200) = %d", d)
	}
}
