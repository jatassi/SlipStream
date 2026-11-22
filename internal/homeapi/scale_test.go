//go:build scale

package homeapi_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/jatassi/slipstream/internal/contenttype/contenttypetest"
	"github.com/jatassi/slipstream/internal/nodes/nodestest"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/storage"
)

// Budgets for one read at scale, well above what a laptop takes, so a read
// that loads a whole table fails them.
const (
	readBudget   = 500 * time.Millisecond
	countsBudget = 250 * time.Millisecond
)

// shelf numbers each Item of the scale Works from 0, one Item per Work:
// the first 200 take the in-flight Grabs and the landings, the rest the
// failed Grabs.
const shelf = `(SELECT work_id, id AS item_id, ROW_NUMBER() OVER (ORDER BY id) - 1 AS slot FROM works_item)`

// TestScaleHome seeds 400 Works, 2,000 in-flight Grabs, half fetching,
// 20,000 failed Grabs, 100,000 landings of 10,000 Imports and 10,000
// Torrents on 10 Nodes, and times GET /home and the per-Node counts.
func TestScaleHome(t *testing.T) {
	w := seedScale(t)
	start := time.Now()
	got := w.home(t, epoch)
	took := time.Since(start)
	t.Logf("GET /home: %d jobs of %d moving, %d needs, %d failed, %d recent in %s", len(got.Jobs.Rows), got.Jobs.Moving, len(got.NeedsYou.Rows), got.NeedsYou.Failed, len(got.Recent.Rows), took)
	if took > readBudget {
		t.Errorf("GET /home took %s, over %s", took, readBudget)
	}
	if got.Jobs.Moving != 2000 || got.NeedsYou.Failed != 100 || len(got.Recent.Rows) != 8 {
		t.Errorf("GET /home = %d moving, %d failed, %d recent; want 2000, 100 and 8", got.Jobs.Moving, got.NeedsYou.Failed, len(got.Recent.Rows))
	}
	start = time.Now()
	counts, err := w.clients.NodeCounts(t.Context())
	took = time.Since(start)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("Torrent counts: %d Nodes in %s", len(counts), took)
	if took > countsBudget {
		t.Errorf("Torrent counts took %s, over %s", took, countsBudget)
	}
}

// TestScaleHomeReplaced seeds as TestScaleHome, then a newer imported Grab
// on each cell the 20,000 failed Grabs read from, so none is current, and
// times GET /home: Needs you reads current failures, not Grab history.
func TestScaleHomeReplaced(t *testing.T) {
	w := seedScale(t)
	w.exec(t, `INSERT INTO grabs_grab (id, work_id, item_id, edition, source, node_id, release_title, release_size, release_magnet, stage, torrent_id, created_at, updated_at)
		SELECT randomblob(16), s.work_id, s.item_id, 'any', 'search', t.node_id, 'Work.2160p.WEB-DL', 4096, 'magnet:?xt=urn:btih:0123456789abcdef0123456789abcdef01234567',
			'imported', t.id, 1000000 + s.slot, 1000000 + s.slot
		FROM downloadclients_torrent t JOIN `+shelf+` s ON s.slot = t.added_order - 801
		WHERE t.node_id = ? AND s.slot >= 200`, w.r.away)
	w.exec(t, `INSERT INTO grabs_target (grab_id, position, item_id, target_id, settled, settled_at)
		SELECT id, 0, item_id, ?, 'placed', created_at FROM grabs_grab WHERE stage = 'imported'`, w.hd.ID)

	start := time.Now()
	got := w.home(t, epoch)
	took := time.Since(start)
	t.Logf("GET /home with every failure replaced: %d moving, %d failed in %s", got.Jobs.Moving, got.NeedsYou.Failed, took)
	if took > readBudget {
		t.Errorf("GET /home took %s, over %s", took, readBudget)
	}
	if got.Jobs.Moving != 2000 || got.NeedsYou.Failed != 0 {
		t.Errorf("GET /home = %d moving, %d failed; want 2000 and 0", got.Jobs.Moving, got.NeedsYou.Failed)
	}
}

// seedScale is the world TestScaleHome times.
func seedScale(t *testing.T) *world {
	t.Helper()
	w := newWorld(t)
	for i := range 400 {
		w.add(t, contenttypetest.ID, fmt.Sprintf("Work %d", i+1), 1)
	}
	nodes := []role.NodeID{w.r.self, w.r.away}
	err := w.lib.Write(t.Context(), func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
		for i := range 8 {
			n := role.NodeID{UUID: storage.NewUUID()}
			if err := nodestest.Insert(ctx, tx, n, fmt.Sprintf("node %d", i)); err != nil {
				return err
			}
			nodes = append(nodes, n)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	for i, n := range nodes {
		w.exec(t, `WITH RECURSIVE k(n) AS (SELECT 1 UNION ALL SELECT n + 1 FROM k WHERE n < 1000)
			INSERT INTO downloadclients_torrent (id, node_id, info_hash, name, size, private, trackers, content_type, metainfo, wanted, added_order, added_at)
			SELECT randomblob(16), ?, randomblob(20), 'Torrent ' || n, 1000, 0, '[]', 'fixture', x'00', 'started', ? + n, 0 FROM k`, n, i*1000)
	}
	w.exec(t, `INSERT INTO grabs_grab (id, work_id, item_id, edition, source, node_id, release_title, release_size, release_magnet, stage, torrent_id, created_at, updated_at)
		SELECT randomblob(16), s.work_id, s.item_id, 'any', 'search', t.node_id, 'Work.1080p.WEB-DL', 4096, 'magnet:?xt=urn:btih:0123456789abcdef0123456789abcdef01234567',
			'downloading', t.id, t.added_order, t.added_order
		FROM downloadclients_torrent t JOIN `+shelf+` s ON s.slot = t.added_order % 200
		WHERE t.node_id = ? AND t.added_order <= 1000`, w.r.self)
	w.exec(t, `WITH RECURSIVE k(n) AS (SELECT 1 UNION ALL SELECT n + 1 FROM k WHERE n < 1000)
		INSERT INTO grabs_grab (id, work_id, item_id, edition, source, node_id, release_title, release_size, release_magnet, stage, created_at, updated_at)
		SELECT randomblob(16), s.work_id, s.item_id, 'any', 'search', ?, 'Work.1080p.WEB-DL', 4096, 'magnet:?xt=urn:btih:0123456789abcdef0123456789abcdef01234567', 'fetching', n, n
		FROM k JOIN `+shelf+` s ON s.slot = n % 200`, w.r.self)
	w.exec(t, `WITH RECURSIVE k(n) AS (SELECT 1 UNION ALL SELECT n + 1 FROM k WHERE n < 100)
		INSERT INTO grabs_grab (id, work_id, item_id, edition, source, node_id, release_title, release_size, release_magnet,
			stage, failure_code, failed_stage, failure_message, created_at, updated_at)
		SELECT randomblob(16), s.work_id, s.item_id, 'any', 'search', ?, 'Work.1080p.WEB-DL', 4096, 'magnet:?xt=urn:btih:0123456789abcdef0123456789abcdef01234567',
			'failed', 'import_failed', 'importing', '', n * 1000 + s.slot, n * 1000 + s.slot
		FROM `+shelf+` s, k WHERE s.slot >= 200`, w.r.self)
	w.exec(t, `INSERT INTO grabs_target (grab_id, position, item_id, target_id) SELECT id, 0, item_id, ? FROM grabs_grab`, w.hd.ID)
	w.exec(t, `WITH RECURSIVE k(n) AS (SELECT 0 UNION ALL SELECT n + 1 FROM k WHERE n < 9999),
			f(f) AS (SELECT 0 UNION ALL SELECT f + 1 FROM f WHERE f < 9)
		INSERT INTO files_imported (id, import_id, work_id, placed_by, items, cells, replaced, quality, at)
		SELECT randomblob(16), imp, s.work_id, 'move', json_array(lower(substr(hex(s.item_id), 1, 8) || '-' || substr(hex(s.item_id), 9, 4) || '-' || substr(hex(s.item_id), 13, 4) || '-' || substr(hex(s.item_id), 17, 4) || '-' || substr(hex(s.item_id), 21))), '[]', '[]', 'webdl-1080p', n * 1000 + f
		FROM (SELECT n, randomblob(16) AS imp FROM k) JOIN `+shelf+` s ON s.slot = n % 200, f`)
	return w
}
