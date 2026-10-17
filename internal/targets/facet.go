package targets

import (
	"context"
	"time"

	"github.com/jatassi/slipstream/internal/release"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/works"
)

// Timing is when an Item came out and how long it runs, as far as its
// Content Type knows. An unknown Released is unreleased; an unknown
// Runtime skips the size checks that need one.
type Timing struct {
	Released release.Fact[works.Date]
	Runtime  release.Fact[time.Duration]
}

// Facet is what targets asks of a Content Type. contenttype.Type embeds
// it. Timings answers every Item of work in one call. Cuts is the type's
// own cuts, beside the built-in ones; a type with none answers nil.
type Facet interface {
	Timings(ctx context.Context, r storage.Reader[storage.Library], work works.WorkID) (map[works.ItemID]Timing, error)
	Cuts() []release.Cut
}

// ReleasedBy reports whether t's Item is out on today's calendar date, in
// today's location. A date known only to the month or the year counts
// from the day after that month or year ends.
func (t Timing) ReleasedBy(today time.Time) bool {
	d, known := t.Released.Get()
	if !known {
		return false
	}
	var from time.Time
	switch {
	case d.Month == 0:
		from = time.Date(d.Year+1, time.January, 1, 0, 0, 0, 0, time.UTC)
	case d.Day == 0:
		from = time.Date(d.Year, time.Month(d.Month)+1, 1, 0, 0, 0, 0, time.UTC)
	default:
		from = time.Date(d.Year, time.Month(d.Month), d.Day, 0, 0, 0, 0, time.UTC)
	}
	y, m, day := today.Date()
	return !time.Date(y, m, day, 0, 0, 0, 0, time.UTC).Before(from)
}
