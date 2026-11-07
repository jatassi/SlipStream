package engine

import (
	"context"
	"time"
)

// reporter sets facts at once after an urgent change, a wait, a route or
// a run starting or stopping, and at most once per interval after
// progress.
type reporter struct {
	progressed, changed chan struct{}
}

func newReporter() *reporter {
	return &reporter{progressed: make(chan struct{}, 1), changed: make(chan struct{}, 1)}
}

func (r *reporter) progress() { poke(r.progressed) }

func (r *reporter) urgent() { poke(r.changed) }

func poke(c chan struct{}) {
	select {
	case c <- struct{}{}:
	default:
	}
}

func (r *reporter) run(ctx context.Context, interval time.Duration, set func()) {
	var last time.Time
	for {
		select {
		case <-ctx.Done():
			return
		case <-r.changed:
		case <-r.progressed:
			if wait := time.Until(last.Add(interval)); wait > 0 {
				t := time.NewTimer(wait)
				select {
				case <-ctx.Done():
					t.Stop()
					return
				case <-r.changed:
				case <-t.C:
				}
				t.Stop()
			}
		}
		set()
		last = time.Now()
	}
}
