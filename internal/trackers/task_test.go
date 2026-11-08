package trackers_test

import (
	"context"
	"fmt"
	"slices"
	"testing"
	"time"

	"github.com/jatassi/slipstream/internal/bencode"
	"github.com/jatassi/slipstream/internal/downloadclients"
	"github.com/jatassi/slipstream/internal/indexers"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/tasks"
	"github.com/jatassi/slipstream/internal/trackers"
	"github.com/jatassi/slipstream/internal/trackers/seeding"
)

// runGoals runs every unit of the Seeding goals task once, in its order.
func (l *library) runGoals() {
	l.t.Helper()
	if err := l.goals(); err != nil {
		l.t.Fatal(err)
	}
}

// goals runs every unit of the Seeding goals task once, in its order, and
// answers the first unit's error.
func (l *library) goals() error {
	task := l.x.Task()
	var units []tasks.Unit
	err := l.lib.Read(l.t.Context(), func(ctx context.Context, r storage.Reader[storage.Library]) error {
		var err error
		units, err = task.Units(ctx, r)
		return err
	})
	if err != nil {
		return err
	}
	for _, u := range units {
		if err := task.Run(l.t.Context(), u, nil); err != nil {
			return fmt.Errorf("unit %s: %w", u.Label, err)
		}
	}
	return nil
}

// host gives tracker the announce host h.
func (l *library) host(tracker indexers.TrackerID, h string) {
	l.t.Helper()
	l.exec(`INSERT INTO indexers_tracker_host (tracker_id, host) VALUES (?, ?)`, tracker, h)
}

// untracked adds a private Torrent of name naming no Tracker and announcing
// to announce, stored as before S28: no trackers, only its metainfo.
func (l *library) untracked(name, announce string) {
	l.t.Helper()
	meta, err := bencode.Encode(map[string]any{
		"announce": announce,
		"info":     map[string]any{"name": name, "piece length": 16384, "pieces": string(make([]byte, 20)), "length": 1, "private": 1},
	})
	if err != nil {
		l.t.Fatal(err)
	}
	h := hashOf(name)
	at := storage.FromTime(time.Now().Add(-time.Hour))
	l.exec(`INSERT INTO downloadclients_torrent (id, node_id, info_hash, metainfo, name, size, private, trackers, content_type,
		wanted, added_order, added_at, uploaded, downloaded)
		VALUES (?, ?, ?, ?, ?, 1, 1, '[]', 'fixture', 'started', (SELECT coalesce(max(added_order), 0) + 1 FROM downloadclients_torrent), ?, 0, 0)`,
		storage.NewUUID(), l.node, h[:], meta, name, at)
}

func (l *library) windows(tracker indexers.TrackerID) map[string]trackers.WindowView {
	l.t.Helper()
	all, err := l.x.Windows(l.t.Context(), admin, tracker, "")
	if err != nil {
		l.t.Fatal(err)
	}
	out := map[string]trackers.WindowView{}
	for _, w := range all {
		out[w.Name] = w
	}
	return out
}

func TestTheFirstRunReadsRulesFromGrabsAndTrackersFromAnnounceHosts(t *testing.T) {
	l := newLibrary(t)
	seedpool := l.tracker("seedpool", indexers.Private)
	mirror := l.tracker("Mirror", indexers.Private)
	l.host(seedpool, "tracker.seedpool.test")
	l.host(seedpool, "shared.test")
	l.host(mirror, "shared.test")
	l.grab(seedpool, time.Now().Add(-day), nil, ptr(3*day))
	l.untracked("Orchard", "https://tracker.seedpool.test/announce/abc")
	l.untracked("Harbor", "https://shared.test/announce/abc")

	l.runGoals()

	s, err := l.x.Tracker(t.Context(), admin, seedpool)
	if err != nil {
		t.Fatal(err)
	}
	inForce(t, s, seeding.TimeRule{Seed: 3 * day}, trackers.FromReleases)
	rows := l.rows()
	if tr := rows["Orchard"].Tracker; tr == nil || *tr != (downloadclients.TrackerRef{ID: seedpool, Name: "seedpool"}) {
		t.Errorf("Orchard, announcing to seedpool's host alone, names Tracker %+v, want seedpool", tr)
	}
	if tr := rows["Harbor"].Tracker; tr != nil {
		t.Errorf("Harbor, announcing to a host seedpool and Mirror both hold, names Tracker %+v, want none", tr)
	}
}

func TestAMetWindowStaysMetWhenTheRuleIsRaised(t *testing.T) {
	l := newLibrary(t)
	seedpool := l.tracker("seedpool", indexers.Private)
	orchard := l.torrent("Orchard", &seedpool, true, time.Now().Add(-time.Hour), 4*gib, 4*gib)
	l.grabOf(seedpool, orchard, ptr(1.0), nil)

	l.runGoals()
	if w := l.windows(seedpool)["Orchard"]; w.Window.State != seeding.Met || w.Window.MetAt.IsZero() {
		t.Fatalf("at a ratio of 1.0 against a rule of 1.0 Orchard's window is %+v, want met", w.Window)
	}

	if _, err := l.x.SetRule(t.Context(), admin, seedpool, seeding.RatioRule{Ratio: 200}); err != nil {
		t.Fatal(err)
	}
	l.runGoals()
	if w := l.windows(seedpool)["Orchard"]; w.Window.State != seeding.Met || w.Owes != nil {
		t.Errorf("once the Admin raises seedpool's rule to 2.0 Orchard's window is %+v owing %+v, want still met", w.Window, w.Owes)
	}
}

func TestARunRecordsWhenATorrentThatOwesStoppedSeedingAndARerunWritesNothing(t *testing.T) {
	l := newLibrary(t)
	seedpool := l.tracker("seedpool", indexers.Private)
	orchard := l.torrent("Orchard", &seedpool, true, time.Now().Add(-time.Hour), 0, 4*gib)
	l.grabOf(seedpool, orchard, nil, ptr(3*day))

	l.runGoals()
	w := l.windows(seedpool)["Orchard"]
	if w.Window.State != seeding.Owing || !w.Window.QuietSince.IsZero() || w.Risk != seeding.Safe {
		t.Fatalf("seeding Orchard's window is %+v at risk %s, want owing and safe", w.Window, w.Risk)
	}
	l.changed.trackers = nil

	l.exec(`UPDATE downloadclients_torrent SET wanted = 'paused' WHERE id = ?`, orchard)
	paused := time.Now()
	l.runGoals()
	quiet := l.windows(seedpool)["Orchard"].Window.QuietSince
	if quiet.Before(paused.Truncate(time.Millisecond)) || l.rows()["Orchard"].Seeding.Risk != seeding.AtRisk {
		t.Fatalf("paused Orchard has been quiet since %v, want at risk since the run after its pause at %v", quiet, paused)
	}
	if len(l.changed.trackers) != 1 {
		t.Errorf("the run after the pause published %d trackers.seeding-changed, want 1 for seedpool", len(l.changed.trackers))
	}
	l.changed.trackers = nil

	l.runGoals()
	if again := l.windows(seedpool)["Orchard"].Window.QuietSince; !again.Equal(quiet) || len(l.changed.trackers) != 0 {
		t.Errorf("a rerun moved Orchard's quiet since to %v and published %d trackers.seeding-changed, want %v and none", again, len(l.changed.trackers), quiet)
	}

	l.exec(`UPDATE downloadclients_torrent SET wanted = 'started' WHERE id = ?`, orchard)
	l.runGoals()
	if w := l.windows(seedpool)["Orchard"]; !w.Window.QuietSince.IsZero() || w.Risk != seeding.Safe {
		t.Errorf("resumed Orchard's window is %+v at risk %s, want safe with no quiet since", w.Window, w.Risk)
	}
}

func TestARunLatchesAMissSoTheTrackerPageListsIt(t *testing.T) {
	l := newLibrary(t)
	seedpool := l.tracker("seedpool", indexers.Private)
	start := time.Now().Add(-time.Hour).Truncate(time.Second)
	orchard := l.torrent("Orchard", &seedpool, true, start, 0, 4*gib)
	l.grabOf(seedpool, orchard, nil, ptr(3*day))
	l.runGoals()

	removed := start.Add(time.Minute)
	l.remove(orchard, removed)
	history, err := l.x.Windows(t.Context(), admin, seedpool, trackers.WindowHistory)
	if err != nil {
		t.Fatal(err)
	}
	if len(history) != 1 || history[0].Name != "Orchard" || history[0].Window.State != seeding.Missed || !history[0].Window.ClosedAt.Equal(removed) {
		t.Fatalf("before a run the history is %+v, want Orchard's window, missed at its removal", history)
	}
	l.runGoals()
	if open, err := l.x.Windows(t.Context(), admin, seedpool, trackers.OpenWindows); err != nil || len(open) != 0 {
		t.Fatalf("after a run the open windows are %+v (%v), want none", open, err)
	}
	l.torrent("Orchard", &seedpool, true, start.Add(2*time.Minute), 0, 4*gib)
	l.torrent("Harbor", &seedpool, true, start.Add(3*time.Minute), 0, 4*gib)
	l.runGoals()
	open, err := l.x.Windows(t.Context(), admin, seedpool, trackers.OpenWindows)
	if err != nil {
		t.Fatal(err)
	}
	if names := windowNames(open); !slices.Equal(names, []string{"Harbor", "Orchard"}) {
		t.Errorf("after the regrab the open windows are %v, want Harbor's and Orchard's", names)
	}
	history, err = l.x.Windows(t.Context(), admin, seedpool, trackers.WindowHistory)
	if err != nil {
		t.Fatal(err)
	}
	if len(history) != 1 || history[0].Name != "Orchard" || history[0].Window.State != seeding.Owing || !history[0].Window.ClosedAt.Equal(removed) {
		t.Errorf("after the regrab the history is %+v, want Orchard's window alone, owing again with its miss kept", history)
	}
}

func windowNames(views []trackers.WindowView) []string {
	names := make([]string, len(views))
	for i, v := range views {
		names[i] = v.Name
	}
	return names
}
