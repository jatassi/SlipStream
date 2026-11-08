package trackers

import (
	"context"
	"fmt"
	"slices"
	"time"

	"github.com/jatassi/slipstream/internal/bencode"
	"github.com/jatassi/slipstream/internal/downloadclients"
	"github.com/jatassi/slipstream/internal/health"
	"github.com/jatassi/slipstream/internal/indexers"
	"github.com/jatassi/slipstream/internal/live"
	"github.com/jatassi/slipstream/internal/setting"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/tasks"
	"github.com/jatassi/slipstream/internal/trackers/internal/db"
	"github.com/jatassi/slipstream/internal/trackers/seeding"
)

// Cadence is how often the Seeding goals task runs.
var Cadence = setting.Duration{
	Key: "tasks." + string(TaskID) + ".cadence", Label: "Runs every",
	Default: 15 * time.Minute, Min: time.Minute, Max: 24 * time.Hour, Step: time.Minute,
}

// trackerless is the unit of the torrents with no known Tracker.
const trackerless tasks.UnitKey = "no-tracker"

// Task is Seeding goals: a unit for the torrents with no known Tracker,
// which runs first, then one per Tracker. Each unit writes the windows
// of its info hashes whose window moved since it was stored, and acts on
// each goal that is due.
func (t *Trackers) Task() tasks.Task {
	return tasks.Task{
		ID:   TaskID,
		Name: "Seeding goals",
		About: "Reads each Tracker's seeding rule from its Releases, and finds the Tracker of each torrent that has none from its announce hosts. " +
			"Then it records each private torrent's hit and run window: when it opens, when its Tracker's rule is met, and when a torrent that still owes stops seeding. " +
			"Once a torrent's goal and its Tracker's rule with the margin are both met, it pauses the torrent, removes it, or removes it and deletes its download copy, as the goal sets. " +
			"It keeps the download copy when a Grab on the torrent didn't import. On a Node draining Downloading, a goal that pauses removes the torrent and keeps its copy instead.",
		Group:   tasks.ControllerGroup,
		Cadence: tasks.Every{Interval: Cadence, AtStart: true},
		Noun:    setting.Noun{One: "Tracker", Many: "Trackers"},
		Units:   units,
		Run:     t.run,
		Follows: []live.Event{indexers.TrackersEvent},
		Health:  string(health.SeedingID),
	}
}

func units(ctx context.Context, r storage.Reader[file]) ([]tasks.Unit, error) {
	all, err := indexers.TrackersIn(ctx, r)
	if err != nil {
		return nil, err
	}
	out := []tasks.Unit{{Key: trackerless, Label: "Torrents with no known Tracker", Place: tasks.OnController()}}
	for _, tr := range all {
		out = append(out, tasks.Unit{Key: tasks.UnitKey(tr.ID.String()), Label: tr.Name, Place: tasks.OnController()})
	}
	return out, nil
}

// run is one unit. The unit of torrents with no known Tracker first gives
// each the Tracker its announce hosts name, and a Tracker's unit first
// reads its rule from its Releases.
func (t *Trackers) run(ctx context.Context, u tasks.Unit, _ *tasks.Progress) error {
	var (
		hashes []bencode.InfoHash
		named  []indexers.TrackerID
	)
	err := t.cfg.Library.Write(ctx, func(ctx context.Context, tx *storage.Tx[file]) error {
		if u.Key == trackerless {
			var err error
			hashes, named, err = assignHosts(ctx, tx)
			return err
		}
		raw, err := storage.ParseUUID(string(u.Key))
		if err != nil {
			return fmt.Errorf("read the unit's Tracker id: %w", err)
		}
		id := indexers.TrackerID{UUID: raw}
		if named, err = syncReleases(ctx, tx, t.cfg.Grabs, func(other indexers.TrackerID) bool { return other == id }); err != nil {
			return err
		}
		hashes, err = downloadclients.TrackerHashes(ctx, tx, id)
		return err
	})
	if err != nil {
		return err
	}
	for _, id := range named {
		t.publish(ctx, &id)
	}
	return t.settle(ctx, hashes)
}

// settle brings each of hashes to what its evaluation says, in one Write
// per hash: it writes the window if it moved since it was stored, and has
// each goal that is due act and records its goal run. Each hash is judged
// again inside its Write, so a run after a crash or a second run neither
// writes nor acts again.
func (t *Trackers) settle(ctx context.Context, hashes []bencode.InfoHash) error {
	var due []bencode.InfoHash
	err := t.cfg.Library.Read(ctx, func(ctx context.Context, r storage.Reader[file]) error {
		j, err := t.judge(ctx, r, hashes, t.cfg.Roster)
		for h, jh := range j.hashes {
			if jh.moved() || jh.acts() {
				due = append(due, h)
			}
		}
		return err
	})
	if err != nil {
		return err
	}
	slices.SortFunc(due, func(a, b bencode.InfoHash) int { return slices.Compare(a[:], b[:]) })
	changed := map[indexers.TrackerID]bool{}
	for _, h := range due {
		tracker, moved, herr := t.settleHash(ctx, h)
		if herr != nil {
			err = herr
			break
		}
		if moved {
			changed[tracker] = true
		}
	}
	for id := range changed {
		if id.IsZero() {
			t.publish(ctx, nil)
			continue
		}
		t.publish(ctx, &id)
	}
	return err
}

// settleHash settles h in one Write, and answers its Tracker and whether
// the Write changed anything.
func (t *Trackers) settleHash(ctx context.Context, h bencode.InfoHash) (tracker indexers.TrackerID, changed bool, err error) {
	var touched []downloadclients.Touched
	err = t.cfg.Library.Write(ctx, func(ctx context.Context, tx *storage.Tx[file]) error {
		touched, changed = nil, false
		j, err := t.judge(ctx, tx, []bencode.InfoHash{h}, t.cfg.Roster)
		if err != nil {
			return err
		}
		jh := j.hashes[h]
		tracker = jh.tracker
		if jh.moved() {
			if err := db.New(tx).SetWindow(ctx, windowRow(h, jh.tracker, *jh.verdict.View.Window)); err != nil {
				return err
			}
			changed = true
		}
		now := storage.Now()
		for i, plan := range jh.verdict.Plans {
			d, ok := deedOf(plan)
			if !ok {
				continue
			}
			id := jh.copies[i].ID
			done, err := t.act(ctx, tx, id, d, j.ran[id], now)
			if err != nil {
				return err
			}
			touched, changed = append(touched, done), true
		}
		return nil
	})
	if err != nil {
		return indexers.TrackerID{}, false, err
	}
	if len(touched) > 0 {
		t.cfg.Clients.Touched(touched...)
	}
	return tracker, changed, nil
}

// deed is what a due Plan does to its Torrent, as its goal run records
// it: pause it, or remove it with or without its download copy. kept is
// why a remove keeps the copy a goal asked to delete, or a pause became a
// remove.
type deed struct {
	action seeding.Action
	kept   seeding.Downgrade
}

// deedOf is what plan does now; false for a Wait.
func deedOf(plan seeding.Plan) (deed, bool) {
	switch p := plan.(type) {
	case seeding.Act:
		return deed{action: p.Do}, true
	case seeding.Keep:
		return deed{action: seeding.Remove, kept: p.Why}, true
	case seeding.Wait:
	}
	return deed{}, false
}

// act does d to Torrent id in tx and records its goal run. A Torrent a
// goal already paused has that run rewritten to the kept remove.
func (t *Trackers) act(ctx context.Context, tx *storage.Tx[file], id downloadclients.ID, d deed, ran seeding.Ran, now storage.Time) (downloadclients.Touched, error) {
	ids := []downloadclients.ID{id}
	var (
		results []downloadclients.Result
		touched downloadclients.Touched
		err     error
	)
	if d.action == seeding.Pause {
		results, touched, err = t.cfg.Clients.WantIn(ctx, tx, ids, downloadclients.Paused)
	} else {
		results, touched, err = t.cfg.Clients.RemoveIn(ctx, tx, ids, downloadclients.GoalMet, d.action == seeding.RemoveDelete)
	}
	if err != nil {
		return downloadclients.Touched{}, err
	}
	if r := results[0].Refusal; r != nil {
		return downloadclients.Touched{}, fmt.Errorf("the seeding goal's %s of torrent %s was refused with %s", d.action, id, r.Code)
	}
	var kept *string
	if d.kept != "" {
		kept = new(string(d.kept))
	}
	q := db.New(tx)
	if ran == seeding.RanPause {
		return touched, q.KeepGoalRun(ctx, db.KeepGoalRunParams{Kept: kept, ActedAt: now, TorrentID: id})
	}
	return touched, q.InsertGoalRun(ctx, db.InsertGoalRunParams{TorrentID: id, ActedAt: now, Action: string(d.action), Kept: kept})
}

func windowRow(h bencode.InfoHash, tracker indexers.TrackerID, w seeding.Window) db.SetWindowParams {
	p := db.SetWindowParams{
		InfoHash: h[:], TrackerID: tracker, State: string(w.State), OpenedAt: storage.FromTime(w.OpenedAt), MetAt: storage.FromTime(w.MetAt),
		ClosedAt: storage.FromTime(w.ClosedAt), RegrabbedAt: storage.FromTime(w.RegrabbedAt), QuietSince: storage.FromTime(w.QuietSince),
	}
	if w.Rule != nil {
		clears := string(w.Rule.Clears())
		p.RuleClears = &clears
		p.RuleRatioCenti, p.RuleSeedSeconds = columnsOf(w.Rule)
	}
	if o := w.Owed; o.Clears != "" {
		clears, seconds := string(o.Clears), int64(o.Time/time.Second)
		p.OwedClears, p.OwedSeconds, p.OwedBytes = &clears, &seconds, &o.Bytes
	}
	return p
}

// assignHosts gives each Torrent with no Tracker the one Tracker holding
// one of its announce hosts, inside tx. A host two Trackers hold names
// neither, so no torrent is given a Tracker by a guess. It answers the
// info hashes of the Torrents still with no Tracker, and the Trackers it
// gave Torrents.
func assignHosts(ctx context.Context, tx *storage.Tx[file]) ([]bencode.InfoHash, []indexers.TrackerID, error) {
	all, err := indexers.TrackersIn(ctx, tx)
	if err != nil {
		return nil, nil, err
	}
	holders := map[string][]indexers.TrackerID{}
	for _, tr := range all {
		for _, h := range tr.Hosts {
			holders[h] = append(holders[h], tr.ID)
		}
	}
	torrents, err := downloadclients.TrackerlessIn(ctx, tx)
	if err != nil {
		return nil, nil, err
	}
	var (
		left []bencode.InfoHash
		gave []indexers.TrackerID
	)
	for _, tl := range torrents {
		id, ok := holderOf(holders, tl.Hosts)
		if !ok {
			left = append(left, tl.InfoHash)
			continue
		}
		if err := downloadclients.AssignTracker(ctx, tx, tl.ID, id); err != nil {
			return nil, nil, err
		}
		if !slices.Contains(gave, id) {
			gave = append(gave, id)
		}
	}
	return left, gave, nil
}

// holderOf is the one Tracker holding one of hosts, each host held by
// one Tracker alone.
func holderOf(holders map[string][]indexers.TrackerID, hosts []string) (indexers.TrackerID, bool) {
	var found []indexers.TrackerID
	for _, h := range hosts {
		if hs := holders[h]; len(hs) == 1 && !slices.Contains(found, hs[0]) {
			found = append(found, hs[0])
		}
	}
	if len(found) != 1 {
		return indexers.TrackerID{}, false
	}
	return found[0], true
}
