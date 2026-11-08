package trackers

import (
	"bytes"
	"context"
	"fmt"
	"slices"
	"time"

	"github.com/jatassi/slipstream/internal/bencode"
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

// Page is a Tracker with its seeding side, its stats and what needs
// attention, worst first.
type Page struct {
	Summary
	Stats     Stats
	Attention []Attention
}

type Level string

const (
	AttentionWarn Level = "warn"
	AttentionInfo Level = "info"
)

// Attention is one sentence of a Tracker page's header.
type Attention struct {
	Level Level
	Text  string
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
		var (
			c     Counts
			seeds int
		)
		for _, row := range rows {
			c = counted(c, row)
			if row.Tabs&downloadclients.TabSeeding != 0 {
				out.Stats.SeedingBytes += row.Size
			}
			if row.State == downloadclients.StateSeeding {
				seeds++
			}
		}
		out.Stats.Torrents, out.Stats.Owing = c.Torrents, c.Owing
		out.Attention = attentionOf(s, c, seeds)
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

// attentionOf is a Tracker's header sentences, worst first: "214 torrents
// are seeding. 3 still owe seedpool, and 1 of those isn't seeding." and,
// for a private Tracker with no rule, a line saying so. seeds counts its
// torrents in the seeding state.
func attentionOf(s Summary, c Counts, seeds int) []Attention {
	var out []Attention
	if c.Torrents > 0 {
		level := AttentionInfo
		if c.AtRisk > 0 {
			level = AttentionWarn
		}
		out = append(out, Attention{Level: level, Text: seedingSentence(s.Name, seeds, c)})
	}
	if rule, _ := s.Rules.InForce(); rule == nil && s.Privacy == indexers.Private {
		out = append(out, Attention{Level: AttentionInfo, Text: s.Name + "'s Releases don't state a seeding rule. Set one so its seeding goal can run."})
	}
	return out
}

func seedingSentence(name string, seeds int, c Counts) string {
	text := fmt.Sprintf("%d torrents are seeding.", seeds)
	switch seeds {
	case 0:
		text = "No torrents are seeding."
	case 1:
		text = "1 torrent is seeding."
	}
	switch {
	case c.Owing == 0:
		return text
	case c.Owing == 1 && c.AtRisk == 1:
		return text + " 1 still owes " + name + ", and it isn't seeding."
	case c.Owing == 1:
		return text + " 1 still owes " + name + "."
	}
	text += fmt.Sprintf(" %d still owe %s", c.Owing, name)
	switch c.AtRisk {
	case 0:
		return text + "."
	case 1:
		return text + ", and 1 of those isn't seeding."
	}
	return text + fmt.Sprintf(", and %d of those aren't seeding.", c.AtRisk)
}

// WindowView is an info hash's window as the Tracker page lists it.
type WindowView struct {
	InfoHash bencode.InfoHash
	// Name is its latest Torrent's.
	Name   string
	Window seeding.Window
	// Torrents are its live Torrents, in added order.
	Torrents []downloadclients.ID
	// Owes is set while it owes under a known rule.
	Owes *seeding.Owes
	Risk seeding.Risk
}

func liveIn(facts []downloadclients.HashFact) []downloadclients.ID {
	var live []downloadclients.ID
	for _, f := range facts {
		if f.Removal == nil {
			live = append(live, f.ID)
		}
	}
	return live
}

// Windows is Tracker id's windows in state, every state when state is "",
// newest opened first. Each is judged live, so a torrent removed since
// the last Seeding goals run reads missed. It fails with ErrNoTracker.
func (t *Trackers) Windows(ctx context.Context, _ identity.Admin, id indexers.TrackerID, state seeding.State) ([]WindowView, error) {
	var out []WindowView
	err := t.cfg.Library.Read(ctx, func(ctx context.Context, r storage.Reader[file]) error {
		if _, err := summaryIn(ctx, r, id); err != nil {
			return err
		}
		hashes, err := downloadclients.TrackerHashes(ctx, r, id)
		if err != nil {
			return err
		}
		j, err := t.judge(ctx, r, hashes, t.cfg.Roster)
		if err != nil {
			return err
		}
		for h, jh := range j.hashes {
			v := jh.verdict.View
			if jh.tracker != id || v.Window == nil || state != "" && v.Window.State != state {
				continue
			}
			out = append(out, WindowView{InfoHash: h, Name: jh.facts[len(jh.facts)-1].Name, Window: *v.Window, Torrents: liveIn(jh.facts), Owes: v.Owes, Risk: v.Risk})
		}
		return nil
	})
	slices.SortFunc(out, func(a, b WindowView) int {
		if c := b.Window.OpenedAt.Compare(a.Window.OpenedAt); c != 0 {
			return c
		}
		return bytes.Compare(a.InfoHash[:], b.InfoHash[:])
	})
	return out, err
}
