package trackers_test

import (
	"context"
	"slices"
	"testing"
	"time"

	"github.com/jatassi/slipstream/internal/downloadclients"
	"github.com/jatassi/slipstream/internal/indexers"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/trackers/seeding"
)

// fate is what the goals did to a Torrent: its wanted state, why it was
// removed and whether with its download copy, and its goal run's action
// and kept, each "" for none.
type fate struct {
	wanted, removed string
	deletes         bool
	action, kept    string
}

func (l *library) fate(id downloadclients.ID) fate {
	l.t.Helper()
	var f fate
	err := l.lib.Read(l.t.Context(), func(ctx context.Context, r storage.Reader[storage.Library]) error {
		return r.QueryRowContext(ctx, `SELECT t.wanted, coalesce(t.removed_reason, ''), t.delete_files, coalesce(g.action, ''), coalesce(g.kept, '')
			FROM downloadclients_torrent t LEFT JOIN trackers_goal_run g ON g.torrent_id = t.id WHERE t.id = ?`, id).
			Scan(&f.wanted, &f.removed, &f.deletes, &f.action, &f.kept)
	})
	if err != nil {
		l.t.Fatal(err)
	}
	return f
}

// announced sets the seed time torrent's Tracker credited it.
func (l *library) announced(torrent downloadclients.ID, d time.Duration) {
	l.t.Helper()
	l.exec(`UPDATE downloadclients_torrent SET announced_seconds = ? WHERE id = ?`, int64(d/time.Second), torrent)
}

// part has the Node confirm torrent's removal.
func (l *library) part(torrent downloadclients.ID) {
	l.t.Helper()
	l.exec(`UPDATE downloadclients_torrent SET parted_at = removed_at, freed = 0 WHERE id = ?`, torrent)
}

func (l *library) expectFate(when string, id downloadclients.ID, want fate) {
	l.t.Helper()
	if got := l.fate(id); got != want {
		l.t.Fatalf("%s the Torrent reads %+v, want %+v", when, got, want)
	}
}

var seeds = fate{wanted: "started"}

func TestAGoalActsOnlyOnceTheRuleAndMarginAreMetAndCommitsWithItsGoalRun(t *testing.T) {
	l := newLibrary(t)
	seedpool := l.tracker("seedpool", indexers.Private)
	l.own(seedpool, seeding.TimeRule{Seed: 3 * day}, seeding.MustFinite(seeding.First, 0, time.Hour, seeding.RemoveDelete))
	orchard := l.torrent("Orchard", &seedpool, true, time.Now().Add(-4*day), 0, 4*gib)
	ruleAndMargin := 3*day + 2*time.Hour

	l.announced(orchard, ruleAndMargin-time.Minute)
	l.runGoals()
	l.expectFate("a minute short of seedpool's rule and the margin, past its own 1-hour goal,", orchard, seeds)

	l.announced(orchard, ruleAndMargin)
	l.exec(`CREATE TRIGGER fail_goal_run BEFORE INSERT ON trackers_goal_run BEGIN SELECT RAISE(ABORT, 'disk full'); END`)
	if err := l.goals(); err == nil {
		t.Fatal("a run whose goal run can't be written succeeded, want it to fail")
	}
	l.expectFate("after a run whose goal run failed to write", orchard, seeds)

	l.exec(`DROP TRIGGER fail_goal_run`)
	l.touched = nil
	l.runGoals()
	removed := fate{wanted: "started", removed: "goal_met", deletes: true, action: "remove-delete"}
	l.expectFate("once the rule and the margin are met", orchard, removed)
	if !slices.Contains(l.touched, role.NodeID{UUID: l.node}) {
		t.Fatalf("the run that removed Orchard told Nodes %v, want its Node told", l.touched)
	}

	l.touched = nil
	l.runGoals()
	l.expectFate("after a second run", orchard, removed)
	if runs := l.count(`SELECT count(*) FROM trackers_goal_run`); runs != 1 || len(l.touched) != 0 {
		t.Errorf("a second run left %d goal runs and told Nodes %v, want one run and no Node told", runs, l.touched)
	}
}

func TestReconcilingHoldsEveryGoal(t *testing.T) {
	l := newLibrary(t)
	nyaa := l.tracker("Nyaa", indexers.Public)
	harbor := l.torrent("Harbor", &nyaa, false, time.Now().Add(-time.Hour), 8*gib, 4*gib)
	l.reconciling = true

	l.runGoals()
	l.expectFate("while the library is reconciled", harbor, seeds)
	planned, err := l.x.Preview(t.Context(), admin)
	if err != nil {
		t.Fatal(err)
	}
	if len(planned) != 1 || planned[0].Plan != (seeding.Wait{Why: seeding.Reconciling}) {
		t.Fatalf("while the library is reconciled the preview plans %+v, want Harbor waiting", planned)
	}

	l.reconciling = false
	l.runGoals()
	l.expectFate("once reconciled", harbor, fate{wanted: "started", removed: "goal_met", deletes: true, action: "remove-delete"})
}

func TestAGoalKeepsTheDownloadCopyAGrabDidntImportAndOnADrainingNode(t *testing.T) {
	l := newLibrary(t)
	nyaa := l.tracker("Nyaa", indexers.Public)
	fixture := l.tracker("Fixture Site", indexers.Public)
	l.own(fixture, nil, seeding.MustFinite(seeding.First, 200, 0, seeding.Pause))
	start := time.Now().Add(-time.Hour)
	meadow := l.torrent("Meadow", &nyaa, false, start, 8*gib, 4*gib)
	l.failedGrabOf(meadow)
	harbor := l.torrent("Harbor", &fixture, false, start, 8*gib, 4*gib)
	l.draining = true

	l.runGoals()
	l.expectFate("Meadow, whose Grab didn't import,", meadow, fate{wanted: "started", removed: "goal_met", action: "remove", kept: "not-imported"})
	l.expectFate("Harbor, whose goal pauses on a draining Node,", harbor, fate{wanted: "started", removed: "goal_met", action: "remove", kept: "draining"})
}

func TestADrainRemovesATorrentItsGoalPausedKeepingItsCopy(t *testing.T) {
	l := newLibrary(t)
	nyaa := l.tracker("Nyaa", indexers.Public)
	l.own(nyaa, nil, seeding.MustFinite(seeding.First, 200, 0, seeding.Pause))
	harbor := l.torrent("Harbor", &nyaa, false, time.Now().Add(-time.Hour), 8*gib, 4*gib)

	l.runGoals()
	paused := fate{wanted: "paused", action: "pause"}
	l.expectFate("once its goal is met", harbor, paused)
	l.runGoals()
	l.expectFate("after a second run", harbor, paused)

	l.draining = true
	l.runGoals()
	kept := fate{wanted: "paused", removed: "goal_met", action: "remove", kept: "draining"}
	l.expectFate("once its Node drains Downloading", harbor, kept)
	l.runGoals()
	l.expectFate("after another run on the draining Node", harbor, kept)
}

func TestARegrabMeetsItsGoalAgain(t *testing.T) {
	l := newLibrary(t)
	nyaa := l.tracker("Nyaa", indexers.Public)
	start := time.Now().Add(-time.Hour)
	first := l.torrent("Harbor", &nyaa, false, start, 8*gib, 4*gib)
	l.runGoals()
	removed := fate{wanted: "started", removed: "goal_met", deletes: true, action: "remove-delete"}
	l.expectFate("once its goal is met", first, removed)
	l.part(first)

	again := l.torrent("Harbor", &nyaa, false, start.Add(time.Minute), 8*gib, 4*gib)
	l.runGoals()
	l.expectFate("grabbed again with the totals it had", again, removed)
	l.expectFate("the first", first, removed)
}
