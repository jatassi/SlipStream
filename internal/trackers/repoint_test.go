package trackers_test

import (
	"testing"
	"time"

	"github.com/jatassi/slipstream/internal/grabs"
	"github.com/jatassi/slipstream/internal/indexers"
	"github.com/jatassi/slipstream/internal/trackers"
	"github.com/jatassi/slipstream/internal/trackers/seeding"
)

func (l *library) merge(from, into indexers.TrackerID) trackers.Summary {
	l.t.Helper()
	x := indexers.New(indexers.Config{Library: l.lib, Trackers: discard[indexers.TrackerChanged]{}, Repoints: []indexers.Repoint{grabs.RepointTracker, trackers.RepointTracker}})
	if err := x.MergeTracker(l.t.Context(), admin, from, into); err != nil {
		l.t.Fatal(err)
	}
	s, err := l.x.Tracker(l.t.Context(), admin, into)
	if err != nil {
		l.t.Fatal(err)
	}
	return s
}

// own sets Tracker id's Admin rule and own goal, each nil for none.
func (l *library) own(id indexers.TrackerID, rule seeding.Rule, g seeding.Goal) {
	l.t.Helper()
	if rule != nil {
		if _, err := l.x.SetRule(l.t.Context(), admin, id, rule); err != nil {
			l.t.Fatal(err)
		}
	}
	if g != nil {
		if _, err := l.x.SetGoal(l.t.Context(), admin, id, g); err != nil {
			l.t.Fatal(err)
		}
	}
}

func (l *library) window(hash byte, tracker indexers.TrackerID) {
	l.t.Helper()
	l.exec(`INSERT INTO trackers_window (info_hash, tracker_id, state, opened_at) VALUES (?, ?, 'owing', 0)`, append(make([]byte, 19), hash), tracker)
}

func (l *library) windowsOf(tracker indexers.TrackerID) int {
	l.t.Helper()
	var n int
	if err := l.lib.QueryRowContext(l.t.Context(), `SELECT count(*) FROM trackers_window WHERE tracker_id = ?`, tracker).Scan(&n); err != nil {
		l.t.Fatal(err)
	}
	return n
}

func TestAMergeKeepsTheAdminRuleThenTheKeptTrackersRuleAndGoal(t *testing.T) {
	start := time.Date(2026, 11, 1, 0, 0, 0, 0, time.UTC)
	keptGoal := seeding.Finite{Combine: seeding.First, Ratio: 150, Seed: 20 * day, Then: seeding.Pause}
	mergedGoal := seeding.Finite{Combine: seeding.Both, Ratio: 300, Seed: 40 * day, Then: seeding.Remove}
	mergedRule := seeding.TimeRule{Seed: 4 * day}
	keptRule := seeding.RatioRule{Ratio: 120}
	for _, c := range []struct {
		name                   string
		keptAdmin, mergedAdmin seeding.Rule
		keptOwn, mergedOwn     seeding.Goal
		rule                   seeding.Rule
		source                 trackers.Source
		goal                   seeding.Goal
	}{
		{"an Admin rule on the merged Tracker beats the kept one's from Releases", nil, mergedRule, nil, mergedGoal, mergedRule, trackers.FromAdmin, mergedGoal},
		{"between two Admin rules the kept Tracker's wins", keptRule, mergedRule, keptGoal, mergedGoal, keptRule, trackers.FromAdmin, keptGoal},
		{"between two rules from Releases the kept Tracker's wins", nil, nil, keptGoal, nil, seeding.TimeRule{Seed: day}, trackers.FromReleases, keptGoal},
	} {
		t.Run(c.name, func(t *testing.T) {
			l := newLibrary(t)
			kept, merged := l.tracker("TorrentLeech", indexers.Private), l.tracker("Torrent Leech", indexers.Private)
			l.grab(kept, start, nil, ptr(day))
			l.grab(merged, start.Add(day), nil, ptr(2*day))
			if err := l.x.SyncReleases(t.Context()); err != nil {
				t.Fatal(err)
			}
			l.own(kept, c.keptAdmin, c.keptOwn)
			l.own(merged, c.mergedAdmin, c.mergedOwn)
			l.window(1, kept)
			l.window(2, merged)

			s := l.merge(merged, kept)
			inForce(t, s, c.rule, c.source)
			if s.Goal != c.goal {
				t.Errorf("the kept Tracker's goal is %#v, want %#v", s.Goal, c.goal)
			}
			if got := l.windowsOf(kept); got != 2 {
				t.Errorf("the kept Tracker has %d windows, want both", got)
			}
		})
	}
}
