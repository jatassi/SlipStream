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
// of its info hashes whose window moved since it was stored.
func (t *Trackers) Task() tasks.Task {
	return tasks.Task{
		ID:   TaskID,
		Name: "Seeding goals",
		About: "Reads each Tracker's seeding rule from its Releases, and finds the Tracker of each torrent that has none from its announce hosts. " +
			"Then it records each private torrent's hit and run window: when it opens, when its Tracker's rule is met, and when a torrent that still owes stops seeding.",
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

// settle writes the window of each of hashes that moved since it was
// stored, judged again inside the Write, so a run after a crash or a
// second run writes nothing new.
func (t *Trackers) settle(ctx context.Context, hashes []bencode.InfoHash) error {
	var moved []bencode.InfoHash
	err := t.cfg.Library.Read(ctx, func(ctx context.Context, r storage.Reader[file]) error {
		judged, err := t.movedIn(ctx, r, hashes)
		for h := range judged {
			moved = append(moved, h)
		}
		return err
	})
	if err != nil || len(moved) == 0 {
		return err
	}
	changed := map[indexers.TrackerID]bool{}
	err = t.cfg.Library.Write(ctx, func(ctx context.Context, tx *storage.Tx[file]) error {
		judged, err := t.movedIn(ctx, tx, moved)
		if err != nil {
			return err
		}
		q := db.New(tx)
		for h, jh := range judged {
			if err := q.SetWindow(ctx, windowRow(h, jh.tracker, *jh.verdict.View.Window)); err != nil {
				return err
			}
			changed[jh.tracker] = true
		}
		return nil
	})
	if err != nil {
		return err
	}
	for id := range changed {
		if id.IsZero() {
			t.publish(ctx, nil)
			continue
		}
		t.publish(ctx, &id)
	}
	return nil
}

// movedIn is each of hashes whose window moved since it was stored, judged
// in r.
func (t *Trackers) movedIn(ctx context.Context, r storage.Reader[file], hashes []bencode.InfoHash) (map[bencode.InfoHash]judgedHash, error) {
	j, err := t.judge(ctx, r, hashes, t.cfg.Roster)
	if err != nil {
		return nil, err
	}
	out := map[bencode.InfoHash]judgedHash{}
	for h, jh := range j.hashes {
		if jh.moved() {
			out[h] = jh
		}
	}
	return out, nil
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
