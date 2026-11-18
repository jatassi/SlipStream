package pacing_test

import (
	"testing"
	"time"

	"github.com/jatassi/slipstream/internal/pacing"
)

const (
	interval = 8 * time.Hour
	longest  = 7 * 24 * time.Hour
)

func TestSpreadIsOneFixedOffsetInsideTheInterval(t *testing.T) {
	spread := pacing.Spread("work", "l:01", interval)
	if spread < 0 || spread >= interval || spread != pacing.Spread("work", "l:01", interval) {
		t.Fatalf("Spread = %v, want one fixed offset inside %v", spread, interval)
	}
	if other := pacing.Spread("work", "l:02", interval); other == spread {
		t.Fatalf("Spread of another key = %v, want another offset", other)
	}
}

func TestGapsDoubleToTheLongestAndAnUnheardRunLeavesThemAlone(t *testing.T) {
	for _, c := range []struct {
		name   string
		prev   time.Duration
		r      pacing.Result
		fresh  bool
		capped time.Duration
		want   time.Duration
	}{
		{"answered at the interval", interval, pacing.Answered, false, 0, 2 * interval},
		{"answered near the longest", 5 * 24 * time.Hour, pacing.Answered, false, 0, longest},
		{"answered at the longest", longest, pacing.Answered, false, 0, longest},
		{"unheard", 4 * interval, pacing.Unheard, false, 0, 4 * interval},
		{"took", 4 * interval, pacing.Took, false, 0, interval},
		{"answered while fresh", 4 * interval, pacing.Answered, true, 0, interval},
		{"answered under a cap", 4 * interval, pacing.Answered, false, 3 * interval, 3 * interval},
	} {
		if got := pacing.Gap(c.prev, interval, longest, c.r, c.fresh, c.capped); got != c.want {
			t.Errorf("%s: gap %v, want %v", c.name, got, c.want)
		}
	}
}
