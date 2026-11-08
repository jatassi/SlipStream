package trackers

import (
	"context"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/jatassi/slipstream/internal/bencode"
	"github.com/jatassi/slipstream/internal/downloadclients"
	"github.com/jatassi/slipstream/internal/health"
	"github.com/jatassi/slipstream/internal/indexers"
	"github.com/jatassi/slipstream/internal/live"
	"github.com/jatassi/slipstream/internal/nodes"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/trackers/internal/db"
	"github.com/jatassi/slipstream/internal/trackers/seeding"
)

// HealthAbout is the Seeding category. A Node going offline or coming
// back changes whether its torrents' risk is known.
var HealthAbout = health.About{
	ID:     health.SeedingID,
	Label:  "Seeding",
	Phrase: "torrents that still owe a Tracker",
	Events: []live.Event{SeedingChangedEvent, downloadclients.ChangedEvent, nodes.ChangedEvent},
}

// HealthSource warns once per Tracker whose torrents that still owe it
// have stopped seeding for longer than WarnAfter. A torrent whose Node is
// out of step may be seeding, so it raises nothing: its Tracker is unsure
// while no torrent of it is known to be at risk.
func (t *Trackers) HealthSource() health.Source {
	return health.Source{Answer: t.healthAnswer}
}

// owed is one Tracker's info hashes that still owe it, as Health judges
// them.
type owed struct {
	// quiet is when each torrent at risk stopped seeding.
	quiet []time.Time
	// unseen are the Nodes out of step holding a torrent whose risk is
	// unknown.
	unseen []role.NodeID
}

func (t *Trackers) healthAnswer(ctx context.Context) (health.Answer, error) {
	var (
		all   []Summary
		owing map[indexers.TrackerID]*owed
		warn  time.Duration
		names map[role.NodeID]string
	)
	err := t.cfg.Library.Read(ctx, func(ctx context.Context, r storage.Reader[file]) error {
		var err error
		if all, err = summariesIn(ctx, r); err != nil {
			return err
		}
		s, err := db.New(r).GetSettings(ctx)
		if err != nil {
			return err
		}
		warn = WarnAfter.Value(s.WarnAfterSeconds)
		if owing, err = t.owingIn(ctx, r); err != nil {
			return err
		}
		names, err = nodeNames(ctx, r)
		return err
	})
	if err != nil {
		return health.Answer{}, err
	}
	var a health.Answer
	now := time.Now()
	for _, s := range all {
		o := owing[s.ID]
		if o == nil {
			continue
		}
		key := health.KeyOf(s.ID.String(), "at-risk")
		var late []time.Time
		for _, q := range o.quiet {
			due := q.Add(warn)
			if !due.After(now) {
				late = append(late, q)
			} else if a.Next.IsZero() || due.Before(a.Next) {
				a.Next = due
			}
		}
		switch {
		case len(late) > 0:
			a.Items = append(a.Items, atRisk(s, late))
		case len(o.unseen) > 0:
			a.Unsure = append(a.Unsure, health.Unsure{Key: key, Says: health.Says{
				Subject: "Torrents on " + s.Name,
				Text:    fmt.Sprintf("Whether %s's torrents on %s are seeding is unknown until %s back online.", s.Name, and(o.unseen, names), areOf(o.unseen)),
			}})
		}
	}
	return a, nil
}

func atRisk(s Summary, quiet []time.Time) health.Item {
	n := len(quiet)
	torrents, them := fmt.Sprintf("%d torrents", n), "them"
	if n == 1 {
		torrents, them = "1 torrent", "it"
	}
	return health.Item{Key: health.KeyOf(s.ID.String(), "at-risk"), Level: health.Warn, Says: health.Says{
		Subject: "Torrents on " + s.Name,
		Text: fmt.Sprintf("%s on %s stopped seeding before %s's rule was met. Resume or retry %s before %s counts a hit and run.",
			torrents, s.Name, s.Name, them, s.Name),
		Detail: []health.Fact{{Label: "Not seeding since", At: slices.MinFunc(quiet, time.Time.Compare)}},
		Action: health.LinkAction("/settings/trackers/"+s.ID.String()+"?tab=problems", "Open "+s.Name+"'s torrents"),
	}}
}

// owingIn is each Tracker's info hashes whose window owes it, at risk or
// of unknown risk, judged live in r.
func (t *Trackers) owingIn(ctx context.Context, r storage.Reader[file]) (map[indexers.TrackerID]*owed, error) {
	rows, err := t.cfg.Clients.Rows(ctx, r, downloadclients.Filter{})
	if err != nil {
		return nil, err
	}
	out := map[indexers.TrackerID]*owed{}
	seen := map[bencode.InfoHash]bool{}
	for _, row := range rows {
		v := row.Seeding
		if row.Tracker == nil || !owing(v) || v.Risk == seeding.Safe {
			continue
		}
		o := out[row.Tracker.ID]
		if o == nil {
			o = &owed{}
			out[row.Tracker.ID] = o
		}
		switch {
		case v.Risk == seeding.RiskUnknown && !t.cfg.Roster.InStep(row.Node) && !slices.Contains(o.unseen, row.Node):
			o.unseen = append(o.unseen, row.Node)
		case v.Risk == seeding.AtRisk && !seen[row.InfoHash]:
			o.quiet = append(o.quiet, v.Window.QuietSince)
		}
		seen[row.InfoHash] = true
	}
	return out, nil
}

func nodeNames(ctx context.Context, r storage.Reader[file]) (map[role.NodeID]string, error) {
	named, err := nodes.Names(ctx, r)
	if err != nil {
		return nil, err
	}
	out := make(map[role.NodeID]string, len(named))
	for _, n := range named {
		out[n.ID] = n.Name
	}
	return out, nil
}

// and names ids: "seedbox", "seedbox and den", "seedbox, den and attic".
func and(ids []role.NodeID, names map[role.NodeID]string) string {
	words := make([]string, len(ids))
	for i, id := range ids {
		words[i] = names[id]
	}
	if len(words) == 1 {
		return words[0]
	}
	return strings.Join(words[:len(words)-1], ", ") + " and " + words[len(words)-1]
}

func areOf(ids []role.NodeID) string {
	if len(ids) == 1 {
		return "it's"
	}
	return "they're"
}
