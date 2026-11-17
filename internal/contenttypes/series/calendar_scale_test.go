//go:build scale

package series

import (
	"context"
	"testing"
	"time"

	"github.com/jatassi/slipstream/internal/calendar"
	"github.com/jatassi/slipstream/internal/files"
	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/release"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/targets"
	"github.com/jatassi/slipstream/internal/works"
)

// calendarBudget is design.md §10.4's budget for a month's read.
const calendarBudget = 300 * time.Millisecond

// TestScaleCalendarReadsAMonthOf5000Series seeds 5,000 series of 10
// episodes, airing one a day each, spread over a year, so a month holds
// about 4,300 of the episodes.
func TestScaleCalendarReadsAMonthOf5000Series(t *testing.T) {
	ctx := t.Context()
	sh := open(t)
	goals, err := targets.Open(ctx, targets.Config{Library: sh.lib, Works: sh.works, Types: map[works.TypeID]targets.Facet{Def.ID(): sh.s}, Changed: discard[targets.Changed]{}})
	if err != nil {
		t.Fatal(err)
	}
	hd, err := goals.CreateTarget(ctx, identity.Admin{}, targets.Spec{Scores: targets.Scores{Increment: 1}, Name: "HD", Tiers: [][]release.Quality{{release.WEBDL1080p}}, Cutoff: release.WEBDL1080p})
	if err != nil {
		t.Fatal(err)
	}
	p, err := goals.CreateProfile(ctx, identity.Admin{}, targets.ProfileSpec{Name: "HD", ContentType: Def.ID(), Entries: []targets.Entry{{Target: hd.ID, Sourcing: targets.Download}}})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := goals.MakeDefault(ctx, identity.Admin{}, p.ID); err != nil {
		t.Fatal(err)
	}
	first := time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC)
	seeding := time.Now()
	for i := range int64(5000) {
		r := newShow(10000+i, "Show", 0, 10)
		for e := range r.Episodes {
			r.Episodes[e].AirDate = first.AddDate(0, 0, int(i%355)+e).Format(time.DateOnly)
		}
		sh.src.put(r)
		_, _, err := sh.works.Add(ctx, identity.Admin{}, Def.ID(), keyOf(10000+i), works.Adding{
			Monitor: true,
			Also: func(ctx context.Context, tx *storage.Tx[storage.Library], w works.Summary) error {
				return targets.Assign(ctx, tx, w, nil)
			},
		})
		if err != nil {
			t.Fatal(err)
		}
	}
	t.Logf("seeded 5,000 series and 50,000 episodes in %s", time.Since(seeding))

	today := func() time.Time { return time.Date(2026, time.June, 1, 12, 0, 0, 0, time.UTC) }
	shelf, err := files.Open(ctx, files.Config{Library: sh.lib, Targets: goals, Grabs: files.NoGrabs{}, Today: today})
	if err != nil {
		t.Fatal(err)
	}
	c := calendar.New(calendar.Config{Library: sh.lib, Types: map[works.TypeID]calendar.Facet{Def.ID(): sh.s}, Works: sh.works, Files: shelf, Today: today})
	from := calendar.NewDay(2026, time.June, 1)
	span, err := calendar.NewSpan(from, from.Add(31))
	if err != nil {
		t.Fatal(err)
	}
	reading := time.Now()
	a, err := c.Read(ctx, identity.Admin{}, calendar.Query{Span: span})
	took := time.Since(reading)
	if err != nil {
		t.Fatal(err)
	}
	rows := 0
	for _, d := range a.Days {
		rows += len(d.Rows)
	}
	t.Logf("a month read answered %d rows over %d days in %s", rows, len(a.Days), took)
	if took > calendarBudget {
		t.Errorf("a month read took %s, over its %s budget", took, calendarBudget)
	}
}
