package trackersapi_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/jatassi/slipstream/internal/indexers"
	"github.com/jatassi/slipstream/internal/storage"
)

func (s *served) exec(t *testing.T, sql string, args ...any) {
	t.Helper()
	err := s.lib.Write(t.Context(), func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
		_, err := tx.ExecContext(ctx, sql, args...)
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
}

// blockers plants reach Indexers reaching tracker, torrents live Torrents
// and grabs fetching Grabs naming it.
func (s *served) blockers(t *testing.T, tracker string, reach, torrents, grabs int) {
	t.Helper()
	id, err := storage.ParseUUID(tracker)
	if err != nil {
		t.Fatal(err)
	}
	node, work, item := storage.NewUUID(), storage.NewUUID(), storage.NewUUID()
	s.exec(t, `INSERT INTO nodes_node (id, public_key, name, paired_at) VALUES (?, randomblob(32), hex(?), 0)`, node, node)
	s.exec(t, `INSERT INTO works_work (id, content_type, key, label, sort_label, added_at, refreshed_at) VALUES (?, 'fixture', hex(?), 'Orchard', 'Orchard', 0, 0)`, work, work)
	s.exec(t, `INSERT INTO works_item (id, work_id, key, label, position) VALUES (?, ?, 'i1', 'Orchard', 1)`, item, work)
	for range reach {
		indexer := storage.NewUUID()
		s.exec(t, `INSERT INTO indexers_indexer (id, name, kind, home_node_id, is_on, auto_search, rss, allow_failover, priority,
			search_limit, search_window, grab_limit, grab_window, plain, secrets_sealed, feed_host, allow_no_size, caps, notes,
			revision, created_at, updated_at)
			VALUES (?, hex(?), 'torznab', ?, 1, 1, 1, 0, 25, 0, 'day', 0, 'day', '{}', x'', 'site.test', 0, '{}', '{}', 1, 0, 0)`, indexer, indexer, node)
		s.exec(t, `INSERT INTO indexers_reach (indexer_id, tracker_id) VALUES (?, ?)`, indexer, id)
	}
	for range torrents {
		s.exec(t, `INSERT INTO downloadclients_torrent (id, node_id, info_hash, metainfo, name, size, private, trackers, content_type,
			wanted, added_order, added_at, tracker_id)
			VALUES (?, ?, randomblob(20), x'', 'Orchard.1080p', 1, 1, '[]', 'fixture', 'started',
			(SELECT count(*) FROM downloadclients_torrent), 0, ?)`, storage.NewUUID(), node, id)
	}
	for range grabs {
		s.exec(t, `INSERT INTO grabs_grab (id, work_id, item_id, edition, source, tracker_id, node_id, release_title, release_size, release_magnet,
			stage, created_at, updated_at)
			VALUES (?, ?, ?, 'any', 'search', ?, ?, 'Orchard.1080p', 1, 'magnet:?xt=urn:btih:00', 'fetching', 0, 0)`,
			storage.NewUUID(), work, item, id, node)
	}
}

func TestATrackerInUseNamesWhatBlocksItsRemoval(t *testing.T) {
	for _, c := range []struct {
		name                      string
		indexers, torrents, grabs int
		detail                    string
	}{
		{"Indexers and torrents", 2, 14, 0, "seedpool is still in use. Delete its 2 Indexers and remove its 14 torrents first."},
		{"one Indexer", 1, 0, 0, "seedpool is still in use. Delete its Indexer first."},
		{"Grabs", 0, 0, 3, "seedpool is still in use. Wait for its 3 Grabs to finish first."},
		{"all three", 1, 1, 1, "seedpool is still in use. Delete its Indexer, remove its torrent and wait for its Grab to finish first."},
	} {
		t.Run(c.name, func(t *testing.T) {
			s := serve(t)
			seedpool := s.tracker(t, "seedpool", indexers.Private)
			s.blockers(t, seedpool, c.indexers, c.torrents, c.grabs)
			s.owner.Expect(t, http.MethodDelete, "/api/v1/trackers/"+seedpool, "", http.StatusConflict,
				`{"status":409,"code":"tracker_in_use","title":"This Tracker is still in use","detail":"`+c.detail+`"}`)
		})
	}
}

func TestAFreeTrackerIsRemoved(t *testing.T) {
	s := serve(t)
	seedpool := s.tracker(t, "seedpool", indexers.Private)
	if code, got := s.owner.Call(t, http.MethodDelete, "/api/v1/trackers/"+seedpool, ""); code != http.StatusNoContent {
		t.Fatalf("removeTracker = %d %s, want 204", code, got)
	}
	gone := `{"status":404,"code":"tracker_not_found","title":"That Tracker no longer exists. It may have been removed or merged into another Tracker."}`
	s.owner.Expect(t, http.MethodGet, "/api/v1/trackers/"+seedpool, "", http.StatusNotFound, gone)
	s.owner.Expect(t, http.MethodDelete, "/api/v1/trackers/"+seedpool, "", http.StatusNotFound, gone)
}
