package targets_test

import (
	"testing"
	"time"

	"github.com/jatassi/slipstream/internal/release"
	"github.com/jatassi/slipstream/internal/targets"
	"github.com/jatassi/slipstream/internal/works"
)

func TestReleasedBy(t *testing.T) {
	day := func(y int, m time.Month, d int) time.Time { return time.Date(y, m, d, 12, 0, 0, 0, time.UTC) }
	cases := []struct {
		name     string
		released string
		today    time.Time
		want     bool
	}{
		{"the day before", "2026-10-17", day(2026, time.October, 16), false},
		{"on the day", "2026-10-17", day(2026, time.October, 17), true},
		{"after the day", "2026-10-17", day(2027, time.January, 1), true},
		{"late in its month", "2026-10", day(2026, time.October, 31), false},
		{"after its month", "2026-10", day(2026, time.November, 1), true},
		{"after a December", "2026-12", day(2027, time.January, 1), true},
		{"late in its year", "2026", day(2026, time.December, 31), false},
		{"after its year", "2026", day(2027, time.January, 1), true},
		{"a past year", "1999", day(2026, time.October, 17), true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			d, err := works.ParseDate(c.released)
			if err != nil {
				t.Fatal(err)
			}
			if got := (targets.Timing{Released: release.Known(d)}).ReleasedBy(c.today); got != c.want {
				t.Fatalf("released %s by %s = %v, want %v", c.released, c.today.Format(time.DateOnly), got, c.want)
			}
		})
	}
}

func TestAnUnknownDateIsNeverReleased(t *testing.T) {
	if (targets.Timing{Runtime: release.Known(2 * time.Hour)}).ReleasedBy(time.Date(3000, time.January, 1, 0, 0, 0, 0, time.UTC)) {
		t.Fatal("an Item with no release date is released")
	}
}

func TestTodayIsReadInItsOwnLocation(t *testing.T) {
	d, err := works.ParseDate("2026-10-17")
	if err != nil {
		t.Fatal(err)
	}
	east := time.FixedZone("UTC+10", 10*60*60)
	today := time.Date(2026, time.October, 17, 1, 0, 0, 0, east)
	if !(targets.Timing{Released: release.Known(d)}).ReleasedBy(today) {
		t.Fatalf("released 2026-10-17 by %s = false, want true: it is the 17th there", today)
	}
}
