// Package backoff is the one retry delay, equal jitter drawn from a seeded
// stream so a run replays. cenkalti/backoff draws from global math/rand,
// which process-globals bans.
package backoff

import (
	"math/rand/v2"
	"time"
)

// Delay is the wait before retry n, counted from 0: with
// d = min(ceiling, floor·2^n), a draw from d/2 + uniform[0, d/2).
func Delay(floor, ceiling time.Duration, n int, r *rand.Rand) time.Duration {
	d := floor
	for range n {
		if d >= ceiling {
			break
		}
		d *= 2
	}
	d = min(d, ceiling)
	return d/2 + time.Duration(r.Int64N(int64(d/2)))
}
