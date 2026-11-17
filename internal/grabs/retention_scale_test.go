//go:build scale

package grabs_test

import (
	"testing"
	"time"

	"github.com/jatassi/slipstream/internal/housekeeping"
)

func TestScaleFinishedGrabsPrune(t *testing.T) {
	a := newAged(t)
	a.many(1500, a.torrent())
	a.many(48500, a.gone)
	before := time.Now().Add(time.Hour)
	var batches []time.Duration
	last := time.Now()
	start := last
	tally, err := a.g.PruneOlderThan(t.Context(), before, housekeeping.DefaultTimings().PruneBatch, func() {
		now := time.Now()
		batches = append(batches, now.Sub(last))
		last = now
	})
	if err != nil {
		t.Fatal(err)
	}
	if tally != (housekeeping.Tally{Deleted: 48500, Kept: 1500}) {
		t.Fatalf("the prune deleted and kept %+v, want 48,500 deleted and 1,500 kept", tally)
	}
	t.Logf("a full Finished Grabs prune of 50,000 Grabs, 1,500 of them seeding: %v in %d batches, the slowest batch %v",
		time.Since(start), len(batches), slowest(batches))
}

func slowest(ds []time.Duration) time.Duration {
	var out time.Duration
	for _, d := range ds {
		out = max(out, d)
	}
	return out
}
