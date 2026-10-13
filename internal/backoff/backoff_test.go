package backoff_test

import (
	"crypto/sha256"
	"math/rand/v2"
	"slices"
	"testing"
	"time"

	"github.com/jatassi/slipstream/internal/backoff"
)

// stream is the Node's stream named name for seed, as node.Streams derives it.
func stream(seed, name string) *rand.Rand {
	return rand.New(rand.NewChaCha8( //nolint:gosec // seeded on purpose, as the Node's streams are
		sha256.Sum256([]byte(seed + "\x00" + name))))
}

func TestRedialDelaysForSeedFixed(t *testing.T) {
	r := stream("fixed", "link")
	var got []time.Duration
	for n := range 5 {
		got = append(got, backoff.Delay(time.Second, time.Minute, n, r))
	}
	want := []time.Duration{616506727, 1275342738, 2398390352, 7826944387, 8637373429}
	if !slices.Equal(got, want) {
		t.Fatalf("delays %v, want %v", got, want)
	}
}

// A retry count far past the ceiling neither overflows nor passes it.
func TestDelayStopsAtTheCeiling(t *testing.T) {
	if d := backoff.Delay(time.Second, time.Minute, 200, stream("fixed", "link")); d != 36990403670 {
		t.Fatalf("Delay(n=200) = %d", d)
	}
}
