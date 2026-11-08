package trackers

import (
	"context"
	"slices"
	"time"

	"github.com/jatassi/slipstream/internal/bencode"
	"github.com/jatassi/slipstream/internal/downloadclients"
	"github.com/jatassi/slipstream/internal/indexers"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/trackers/internal/db"
	"github.com/jatassi/slipstream/internal/trackers/seeding"
)

// Seeder is the Trackers' answer to downloadclients.Seeder, which every
// read of a Torrent's seeding goes through.
func (t *Trackers) Seeder() downloadclients.Seeder { return seeder{t} }

type seeder struct{ t *Trackers }

func (s seeder) Judge(ctx context.Context, r storage.Reader[file], rows []downloadclients.Row) (map[downloadclients.ID]downloadclients.Judged, error) {
	hashes := make([]bencode.InfoHash, len(rows))
	for i, row := range rows {
		hashes[i] = row.InfoHash
	}
	j, err := s.t.judge(ctx, r, hashes, s.t.cfg.Roster)
	if err != nil {
		return nil, err
	}
	out := make(map[downloadclients.ID]downloadclients.Judged, len(rows))
	for _, row := range rows {
		h := j.hashes[row.InfoHash]
		mine := func(f downloadclients.HashFact) bool { return f.ID == row.ID }
		judged := downloadclients.Judged{View: h.verdict.View, Plan: h.verdict.Plans[slices.IndexFunc(h.copies, mine)]}
		i := slices.IndexFunc(h.facts, mine)
		if t, ok := j.trackers[trackerOf(h.facts[i])]; ok {
			judged.Tracker = &downloadclients.TrackerRef{ID: t.ID, Name: t.Name}
		}
		out[row.ID] = judged
	}
	return out, nil
}

func (s seeder) EndlessUnits(ctx context.Context, r storage.Reader[file], node role.NodeID) (int64, error) {
	kept, err := downloadclients.Kept(ctx, r, node)
	if err != nil {
		return 0, err
	}
	hashes := make([]bencode.InfoHash, len(kept))
	for i, k := range kept {
		hashes[i] = k.InfoHash
	}
	j, err := s.t.judge(ctx, r, hashes, steady{})
	if err != nil {
		return 0, err
	}
	var n int64
	for _, h := range j.hashes {
		for i, c := range h.copies {
			if w, ok := h.verdict.Plans[i].(seeding.Wait); ok && c.Node == node && w.Why.Endless() {
				n++
			}
		}
	}
	return n, nil
}

// steady is every Node in step and none draining. The roster counts
// EndlessUnits while it reads itself, so they can't read it back, and no
// Node's state decides whether a Torrent's goal ever acts.
type steady struct{}

func (steady) InStep(role.NodeID) bool { return true }

func (steady) Draining(role.Name, role.NodeID) bool { return false }

// judgment is info hashes' Verdicts, the Trackers they name, and what
// the goal runs of their Torrents did.
type judgment struct {
	hashes   map[bencode.InfoHash]judgedHash
	trackers map[indexers.TrackerID]Summary
	ran      map[downloadclients.ID]seeding.Ran
}

type judgedHash struct {
	// facts are the hash's Torrents in added order, and copies the live
	// ones among them in the order of the Verdict's Plans.
	facts   []downloadclients.HashFact
	copies  []downloadclients.HashFact
	verdict seeding.Verdict
	// tracker is the hash's Tracker, the zero id for none, and stored the
	// Tracker its stored window names.
	tracker, stored indexers.TrackerID
}

// moved reports whether the hash's window differs from the stored one,
// its Tracker included.
func (h judgedHash) moved() bool {
	return h.verdict.Write || h.verdict.View.Window != nil && h.stored != h.tracker
}

// acts reports whether a goal acts on one of the hash's Torrents now.
func (h judgedHash) acts() bool {
	return slices.ContainsFunc(h.verdict.Plans, func(p seeding.Plan) bool {
		_, ok := deedOf(p)
		return ok
	})
}

// judge evaluates each of hashes in r's snapshot, its Nodes as nodes
// says.
func (t *Trackers) judge(ctx context.Context, r storage.Reader[file], hashes []bencode.InfoHash, nodes Roster) (judgment, error) {
	hashes = slices.Clone(hashes)
	slices.SortFunc(hashes, func(a, b bencode.InfoHash) int { return slices.Compare(a[:], b[:]) })
	hashes = slices.Compact(hashes)
	in, err := t.policyIn(ctx, r, hashes)
	if err != nil {
		return judgment{}, err
	}
	facts, err := t.cfg.Clients.HashFacts(ctx, r, hashes)
	if err != nil {
		return judgment{}, err
	}
	var ids []downloadclients.ID
	for _, fs := range facts {
		for _, f := range fs {
			ids = append(ids, f.ID)
		}
	}
	runs, err := db.New(r).ListGoalRunsOf(ctx, ids)
	if err != nil {
		return judgment{}, err
	}
	ran := make(map[downloadclients.ID]seeding.Ran, len(runs))
	for _, run := range runs {
		ran[run.TorrentID] = ranOf(seeding.Action(run.Action))
	}
	j := judgment{hashes: make(map[bencode.InfoHash]judgedHash, len(facts)), trackers: in.trackers, ran: ran}
	for h, fs := range facts {
		jh := judgedHash{facts: fs, stored: in.windows[h].TrackerID}
		hash, tracker, err := hashOf(in, fs, ran, in.windows[h], nodes)
		if err != nil {
			return judgment{}, err
		}
		for _, f := range fs {
			if f.Removal == nil || f.Removal.Parted == nil {
				jh.copies = append(jh.copies, f)
			}
		}
		jh.verdict, jh.tracker = seeding.Evaluate(hash), tracker
		j.hashes[h] = jh
	}
	return j, nil
}

// policy is what every hash's evaluation reads besides its Torrents.
type policy struct {
	now         time.Time
	trackers    map[indexers.TrackerID]Summary
	goal        seeding.Goal
	margin      time.Duration
	reconciling bool
	windows     map[bencode.InfoHash]db.TrackersWindow
}

func (t *Trackers) policyIn(ctx context.Context, r storage.Reader[file], hashes []bencode.InfoHash) (policy, error) {
	p := policy{now: time.Now(), trackers: map[indexers.TrackerID]Summary{}, windows: map[bencode.InfoHash]db.TrackersWindow{}}
	all, err := summariesIn(ctx, r)
	if err != nil {
		return policy{}, err
	}
	for _, s := range all {
		p.trackers[s.ID] = s
	}
	if p.goal, err = defaultGoalIn(ctx, r); err != nil {
		return policy{}, err
	}
	q := db.New(r)
	s, err := q.GetSettings(ctx)
	if err != nil {
		return policy{}, err
	}
	p.margin = Margin.Value(s.MarginSeconds)
	if p.reconciling, err = t.cfg.Reconciling(ctx, r); err != nil {
		return policy{}, err
	}
	raw := make([][]byte, len(hashes))
	for i, h := range hashes {
		raw[i] = h[:]
	}
	windows, err := q.ListWindowsOf(ctx, raw)
	if err != nil {
		return policy{}, err
	}
	for _, w := range windows {
		p.windows[bencode.InfoHash(w.InfoHash)] = w
	}
	return p, nil
}

// hashOf is one hash's input to Evaluate, and its Tracker: the first one
// a Torrent of the hash names, the zero id for none.
func hashOf(p policy, facts []downloadclients.HashFact, ran map[downloadclients.ID]seeding.Ran, stored db.TrackersWindow, nodes Roster) (seeding.Hash, indexers.TrackerID, error) {
	h := seeding.Hash{Now: p.now, Goal: p.goal, Margin: p.margin, Reconciling: p.reconciling}
	var (
		tracker *Summary
		floors  []seeding.Rule
	)
	for _, f := range facts {
		if s, ok := p.trackers[trackerOf(f)]; ok && tracker == nil {
			tracker = &s
		}
		for _, u := range f.Uses {
			if u.Floor != nil {
				floors = append(floors, u.Floor)
			}
		}
		if f.Removal != nil && f.Removal.Parted != nil {
			h.Last = &seeding.Removed{At: f.Removal.At, Totals: totalsOf(f), Unregistered: unregistered(f), Continues: f.Continued}
			continue
		}
		h.Copies = append(h.Copies, copyOf(f, ran[f.ID], nodes))
	}
	h.Privacy = privacyOf(tracker, facts[0].Private, floors)
	var id indexers.TrackerID
	if tracker != nil {
		id = tracker.ID
		if tracker.Goal != nil {
			h.Goal = tracker.Goal
		}
	}
	if stored.InfoHash != nil {
		w, err := windowOf(stored)
		if err != nil {
			return seeding.Hash{}, id, err
		}
		h.Stored = &w
	}
	return h, id, nil
}

func copyOf(f downloadclients.HashFact, ran seeding.Ran, nodes Roster) seeding.Copy {
	c := seeding.Copy{
		Totals: totalsOf(f), AddedAt: f.AddedAt, Complete: f.Complete,
		Seen:         seeding.SeenOf(string(f.State), string(f.Hold), nodes.InStep(f.Node)),
		Unregistered: unregistered(f), Paused: f.Wanted == downloadclients.Paused, Removing: f.Removal != nil,
		Draining: nodes.Draining(role.Downloading, f.Node), Ran: ran,
	}
	for _, u := range f.Uses {
		switch u.Stage {
		case downloadclients.UseDownloading:
			c.Grabs.Downloading = true
		case downloadclients.UseImporting:
			c.Grabs.Importing = true
		case downloadclients.UseFailed:
			c.Grabs.Unimported = true
		case downloadclients.UseImported:
		}
	}
	return c
}

func totalsOf(f downloadclients.HashFact) seeding.Totals {
	return seeding.Totals{
		Uploaded: f.Totals.Uploaded, Downloaded: f.Totals.Downloaded, Size: f.Size, Announced: f.Totals.Announced, Seeding: f.Totals.Seeding,
	}
}

// trackerOf is f's Tracker, the zero id for none.
func trackerOf(f downloadclients.HashFact) indexers.TrackerID {
	if f.TrackerID == nil {
		return indexers.TrackerID{}
	}
	return *f.TrackerID
}

func unregistered(f downloadclients.HashFact) bool {
	return f.Fault != nil && f.Fault.Kind == downloadclients.FaultUnregistered
}

// privacyOf is whom a hash owes. A private Tracker's torrents owe it, a
// semi-private or unknown one's owe it when the torrent's private flag is
// set, and a private torrent with no known Tracker owes one not yet known.
// A window opening now snapshots the strictest floor its Grabs stated,
// else the Tracker's rule from Releases.
func privacyOf(s *Summary, flag bool, floors []seeding.Rule) seeding.Privacy {
	switch {
	case s == nil && flag:
		return seeding.Untracked{}
	case s == nil, s.Privacy == indexers.Public, s.Privacy != indexers.Private && !flag:
		return seeding.Public{}
	}
	opening := seeding.Strictest(floors)
	if opening == nil {
		opening = s.Rules.Releases
	}
	return seeding.Private{Admin: s.Rules.Admin, Opening: opening}
}

func ranOf(a seeding.Action) seeding.Ran {
	if a == seeding.Pause {
		return seeding.RanPause
	}
	return seeding.RanRemove
}

func windowOf(w db.TrackersWindow) (seeding.Window, error) {
	out := seeding.Window{
		State: seeding.State(w.State), OpenedAt: w.OpenedAt.Time(), MetAt: w.MetAt.Time(), ClosedAt: w.ClosedAt.Time(),
		RegrabbedAt: w.RegrabbedAt.Time(), QuietSince: w.QuietSince.Time(),
	}
	if w.RuleClears != nil {
		rule, err := ruleOf(*w.RuleClears, w.RuleRatioCenti, w.RuleSeedSeconds)
		if err != nil {
			return seeding.Window{}, err
		}
		out.Rule = rule
	}
	if w.OwedClears != nil {
		out.Owed = seeding.Owes{Time: time.Duration(*w.OwedSeconds) * time.Second, Bytes: *w.OwedBytes, Clears: seeding.Clears(*w.OwedClears)}
	}
	return out, nil
}
