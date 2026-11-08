package trackers_test

import (
	"context"
	"errors"
	"slices"
	"testing"
	"time"

	"github.com/jatassi/slipstream/internal/indexers"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/trackers"
	"github.com/jatassi/slipstream/internal/trackers/seeding"
)

// indexer plants an Indexer named name that reaches tracker.
func (l *library) indexer(name string, tracker indexers.TrackerID) {
	l.t.Helper()
	id := storage.NewUUID()
	l.exec(`INSERT INTO indexers_indexer (id, name, kind, home_node_id, is_on, auto_search, rss, allow_failover, priority,
		search_limit, search_window, grab_limit, grab_window, plain, secrets_sealed, feed_host, allow_no_size, caps, notes,
		revision, created_at, updated_at)
		VALUES (?, ?, 'torznab', ?, 1, 1, 1, 0, 25, 0, 'day', 0, 'day', '{}', x'', 'site.test', 0, '{}', '{}', 1, 0, 0)`, id, name, l.node)
	l.exec(`INSERT INTO indexers_reach (indexer_id, tracker_id) VALUES (?, ?)`, id, tracker)
}

// torrentOf plants a Torrent of tracker's, removed or live, and answers its
// ID.
func (l *library) torrentOf(tracker indexers.TrackerID, removed bool) storage.UUID {
	l.t.Helper()
	id := storage.NewUUID()
	var removedAt, reason any
	if removed {
		removedAt, reason = 1, "admin"
	}
	l.exec(`INSERT INTO downloadclients_torrent (id, node_id, info_hash, metainfo, name, size, private, trackers, content_type,
		wanted, added_order, added_at, removed_at, removed_reason, tracker_id)
		VALUES (?, ?, randomblob(20), x'', 'Orchard.1080p', 1, 1, '[]', 'fixture', 'started',
		(SELECT count(*) FROM downloadclients_torrent), 0, ?, ?, ?)`, id, l.node, removedAt, reason, tracker)
	return id
}

// grabAt plants a Grab of tracker's in stage, on torrent when it is set.
func (l *library) grabAt(tracker indexers.TrackerID, stage string, torrent *storage.UUID) storage.UUID {
	l.t.Helper()
	id := storage.NewUUID()
	var path, failedStage, failureCode any
	switch stage {
	case "imported":
		path = "/library/Orchard"
	case "failed":
		failedStage, failureCode = "downloading", "fixture_failure"
	}
	l.exec(`INSERT INTO grabs_grab (id, work_id, item_id, edition, source, tracker_id, node_id, release_title, release_size, release_magnet,
		stage, torrent_id, path, failed_stage, failure_code, created_at, updated_at)
		VALUES (?, ?, ?, 'any', 'search', ?, ?, 'Orchard.1080p', 1, 'magnet:?xt=urn:btih:00', ?, ?, ?, ?, ?, 0, 0)`,
		id, l.work, l.item, tracker, l.node, stage, torrent, path, failedStage, failureCode)
	return id
}

// history gives tracker an Admin rule, its own goal, an owing window and
// a goal run on torrent.
func (l *library) history(tracker indexers.TrackerID, torrent storage.UUID) {
	l.t.Helper()
	l.own(tracker, seeding.TimeRule{Seed: 72 * time.Hour}, seeding.MustFinite(seeding.Both, 300, 30*24*time.Hour, seeding.Pause))
	l.exec(`INSERT INTO trackers_window (info_hash, tracker_id, state, opened_at) VALUES (randomblob(20), ?, 'owing', 0)`, tracker)
	l.exec(`INSERT INTO trackers_goal_run (torrent_id, acted_at, action) VALUES (?, 0, 'remove')`, torrent)
}

func (l *library) count(sql string, args ...any) int {
	l.t.Helper()
	var n int
	err := l.lib.Read(l.t.Context(), func(ctx context.Context, r storage.Reader[storage.Library]) error {
		return r.QueryRowContext(ctx, sql, args...).Scan(&n)
	})
	if err != nil {
		l.t.Fatal(err)
	}
	return n
}

func TestEachBlockerRefusesTheRemoval(t *testing.T) {
	for _, tc := range []struct {
		name  string
		plant func(*library, indexers.TrackerID)
		want  trackers.InUseError
	}{
		{"an Indexer reaches it", func(l *library, id indexers.TrackerID) {
			l.indexer("seedpool (API)", id)
			l.indexer("seedpool (RSS)", id)
		}, trackers.InUseError{Name: "seedpool", Indexers: 2}},
		{"a live torrent names it", func(l *library, id indexers.TrackerID) {
			l.torrentOf(id, false)
			l.torrentOf(id, true)
		}, trackers.InUseError{Name: "seedpool", Torrents: 1}},
		{"a Grab neither imported nor failed names it", func(l *library, id indexers.TrackerID) {
			l.grabAt(id, "fetching", nil)
			l.grabAt(id, "failed", nil)
		}, trackers.InUseError{Name: "seedpool", Grabs: 1}},
		{"all three", func(l *library, id indexers.TrackerID) {
			l.indexer("seedpool", id)
			torrent := l.torrentOf(id, false)
			l.grabAt(id, "downloading", &torrent)
		}, trackers.InUseError{Name: "seedpool", Indexers: 1, Torrents: 1, Grabs: 1}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			l := newLibrary(t)
			seedpool := l.tracker("seedpool", indexers.Private)
			tc.plant(l, seedpool)
			err := l.x.Remove(t.Context(), admin, seedpool)
			inUse, ok := errors.AsType[*trackers.InUseError](err)
			if !ok || *inUse != tc.want {
				t.Fatalf("Remove = %v (%+v), want %+v", err, inUse, tc.want)
			}
			if _, err := l.x.Tracker(t.Context(), admin, seedpool); err != nil {
				t.Fatalf("after a refused removal Tracker = %v, want seedpool kept", err)
			}
			if len(l.trackersChanged.trackers) != 0 {
				t.Fatalf("a refused removal published trackers.changed for %v", l.trackersChanged.trackers)
			}
		})
	}
}

func TestAFreeTrackerIsRemovedAndItsHistoryKeepsItsRows(t *testing.T) {
	l := newLibrary(t)
	seedpool := l.tracker("seedpool", indexers.Private)
	aither := l.tracker("aither", indexers.Private)
	for _, id := range []indexers.TrackerID{seedpool, aither} {
		l.exec(`INSERT INTO indexers_tracker_key (key, tracker_id) VALUES ('definition:' || hex(?), ?)`, id, id)
		l.exec(`INSERT INTO indexers_tracker_host (tracker_id, host) VALUES (?, hex(?) || '.test')`, id, id)
	}

	removed := l.torrentOf(seedpool, true)
	imported := l.grabAt(seedpool, "imported", &removed)
	failed := l.grabAt(seedpool, "failed", nil)
	l.history(seedpool, removed)
	other := l.torrentOf(aither, true)
	l.history(aither, other)

	if err := l.x.Remove(t.Context(), admin, seedpool); err != nil {
		t.Fatalf("Remove = %v, want seedpool removed", err)
	}
	if _, err := l.x.Tracker(t.Context(), admin, seedpool); !errors.Is(err, trackers.ErrNoTracker) {
		t.Fatalf("after its removal Tracker = %v, want ErrNoTracker", err)
	}
	if !slices.Equal(l.trackersChanged.trackers, []string{seedpool.String()}) {
		t.Fatalf("trackers.changed named %v, want seedpool once", l.trackersChanged.trackers)
	}
	for _, table := range []string{"trackers_rule", "trackers_goal", "trackers_window", "indexers_tracker_key", "indexers_tracker_host"} {
		if n := l.count(`SELECT count(*) FROM `+table+` WHERE tracker_id = ?`, seedpool); n != 0 {
			t.Fatalf("%s kept %d rows of seedpool, want none", table, n)
		}
		if n := l.count(`SELECT count(*) FROM `+table+` WHERE tracker_id = ?`, aither); n != 1 {
			t.Fatalf("%s has %d rows of aither, want its one kept", table, n)
		}
	}
	if n := l.count(`SELECT count(*) FROM trackers_goal_run WHERE torrent_id = ?`, removed); n != 0 {
		t.Fatalf("seedpool's goal run kept, want deleted")
	}
	if n := l.count(`SELECT count(*) FROM trackers_goal_run WHERE torrent_id = ?`, other); n != 1 {
		t.Fatalf("aither's goal run deleted, want kept")
	}
	for _, grab := range []storage.UUID{imported, failed} {
		if n := l.count(`SELECT count(*) FROM grabs_grab WHERE id = ? AND tracker_id IS NULL`, grab); n != 1 {
			t.Fatalf("Grab %s lost its row or kept its Tracker, want its row with no Tracker", grab)
		}
	}
	if n := l.count(`SELECT count(*) FROM downloadclients_torrent WHERE id = ? AND tracker_id IS NULL`, removed); n != 1 {
		t.Fatalf("the removed torrent lost its row or kept its Tracker, want its row with no Tracker")
	}
	if n := l.count(`SELECT count(*) FROM pragma_foreign_key_check`); n != 0 {
		t.Fatalf("%d rows break a foreign key after the removal", n)
	}
}

func TestRemovingAMissingTrackerFails(t *testing.T) {
	l := newLibrary(t)
	if err := l.x.Remove(t.Context(), admin, indexers.TrackerID{UUID: storage.NewUUID()}); !errors.Is(err, trackers.ErrNoTracker) {
		t.Fatalf("Remove = %v, want ErrNoTracker", err)
	}
}
