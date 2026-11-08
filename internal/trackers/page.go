package trackers

import (
	"context"
	"time"

	"github.com/jatassi/slipstream/internal/downloadclients"
	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/indexers"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/trackers/seeding"
)

// Counts are a Tracker's Torrents, live or Removing, as the Torrents view
// counts them.
type Counts struct {
	Torrents, Seeding int
	// Owing is the ones whose window owes the Tracker, AtRisk those of
	// them at risk of a hit and run.
	Owing, AtRisk, GoalMet int
}

// Listed is a Tracker with its seeding side and its Torrents' counts.
type Listed struct {
	Summary
	Counts Counts
}

// List is every Tracker by name with its seeding side and counts.
func (t *Trackers) List(ctx context.Context, _ identity.Admin) ([]Listed, error) {
	var out []Listed
	err := t.cfg.Library.Read(ctx, func(ctx context.Context, r storage.Reader[file]) error {
		all, err := summariesIn(ctx, r)
		if err != nil {
			return err
		}
		rows, err := t.cfg.Clients.Rows(ctx, r, downloadclients.Filter{})
		if err != nil {
			return err
		}
		counts := map[indexers.TrackerID]Counts{}
		for _, row := range rows {
			if row.Tracker != nil {
				counts[row.Tracker.ID] = counted(counts[row.Tracker.ID], row)
			}
		}
		out = make([]Listed, len(all))
		for i, s := range all {
			out[i] = Listed{Summary: s, Counts: counts[s.ID]}
		}
		return nil
	})
	return out, err
}

func counted(c Counts, row downloadclients.Row) Counts {
	c.Torrents++
	if row.Tabs&downloadclients.TabSeeding != 0 {
		c.Seeding++
	}
	if owing(row.Seeding) {
		c.Owing++
	}
	if row.Seeding.Risk == seeding.AtRisk {
		c.AtRisk++
	}
	if row.Seeding.GoalMet {
		c.GoalMet++
	}
	return c
}

func owing(v seeding.View) bool { return v.Window != nil && v.Window.State == seeding.Owing }

// Stats are SlipStream's own sums over a Tracker's Torrents.
type Stats struct {
	// Uploaded and Downloaded count each info hash once, at the largest
	// totals of its Torrents, live or removed. Ratio is uploaded over
	// their downloaded bytes, or size where that is larger.
	Uploaded, Downloaded int64
	Ratio                seeding.Ratio
	// Torrents is its Torrents live or Removing, and SeedingBytes the size
	// of the ones seeding.
	Torrents     int
	SeedingBytes int64
	// Grabs30d is its Grabs made in the last 30 days.
	Grabs30d int
	Owing    int
}

// Page is a Tracker with its seeding side and its stats.
type Page struct {
	Summary
	Stats Stats
}

// Page is Tracker id's page, or ErrNoTracker.
func (t *Trackers) Page(ctx context.Context, _ identity.Admin, id indexers.TrackerID) (Page, error) {
	var out Page
	err := t.cfg.Library.Read(ctx, func(ctx context.Context, r storage.Reader[file]) error {
		s, err := summaryIn(ctx, r, id)
		if err != nil {
			return err
		}
		out = Page{Summary: s}
		rows, err := t.cfg.Clients.Rows(ctx, r, downloadclients.Filter{Tracker: &id})
		if err != nil {
			return err
		}
		var c Counts
		for _, row := range rows {
			c = counted(c, row)
			if row.Tabs&downloadclients.TabSeeding != 0 {
				out.Stats.SeedingBytes += row.Size
			}
		}
		out.Stats.Torrents, out.Stats.Owing = c.Torrents, c.Owing
		if err := t.sumsIn(ctx, r, id, &out.Stats); err != nil {
			return err
		}
		grabbed, err := t.cfg.Grabs.Grabbed(ctx, r, time.Now().Add(-30*24*time.Hour))
		out.Stats.Grabs30d = grabbed[id]
		return err
	})
	return out, err
}

func (t *Trackers) sumsIn(ctx context.Context, r storage.Reader[file], id indexers.TrackerID, st *Stats) error {
	hashes, err := downloadclients.TrackerHashes(ctx, r, id)
	if err != nil {
		return err
	}
	facts, err := t.cfg.Clients.HashFacts(ctx, r, hashes)
	if err != nil {
		return err
	}
	var base int64
	for _, fs := range facts {
		var up, down, size int64
		for _, f := range fs {
			up, down, size = max(up, f.Totals.Uploaded), max(down, f.Totals.Downloaded), max(size, f.Size)
		}
		st.Uploaded, st.Downloaded, base = st.Uploaded+up, st.Downloaded+down, base+max(down, size)
	}
	st.Ratio = seeding.RatioOf(seeding.Totals{Uploaded: st.Uploaded, Downloaded: base})
	return nil
}
