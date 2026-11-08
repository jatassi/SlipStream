package trackers_test

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/jatassi/slipstream/internal/bencode"
	"github.com/jatassi/slipstream/internal/downloadclients"
	"github.com/jatassi/slipstream/internal/indexers"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/secret/secrettest"
	"github.com/jatassi/slipstream/internal/storage"
	tc "github.com/jatassi/slipstream/internal/torrent/clients"
	"github.com/jatassi/slipstream/internal/trackers/seeding"
)

// privateTorrent adds a reported 4 GiB private Torrent of tracker on the
// library's Node with a real .torrent, downloaded whole and uploaded up,
// and answers it and its info hash.
func (l *library) privateTorrent(name string, tracker indexers.TrackerID, at time.Time, up int64) (downloadclients.ID, bencode.InfoHash) {
	l.t.Helper()
	info := map[string]any{"name": name, "piece length": int64(16384), "pieces": "aaaaaaaaaaaaaaaaaaaa", "length": int64(4 * gib), "private": int64(1)}
	raw, err := bencode.Encode(map[string]any{"announce": "https://tracker.test/announce/key", "info": info})
	if err != nil {
		l.t.Fatal(err)
	}
	m, err := bencode.DecodeMetainfo(raw)
	if err != nil {
		l.t.Fatal(err)
	}
	report, err := json.Marshal(downloadclients.Report{At: at, Engine: downloadclients.EngineSeeding, Complete: true})
	if err != nil {
		l.t.Fatal(err)
	}
	id := downloadclients.ID{UUID: storage.NewUUID()}
	l.exec(`INSERT INTO downloadclients_torrent (id, node_id, info_hash, metainfo, name, size, private, trackers, content_type, tracker_id,
		wanted, added_order, added_at, uploaded, downloaded, report, reported_at)
		VALUES (?, ?, ?, ?, ?, ?, 1, '["tracker.test"]', 'fixture', ?, 'started',
		(SELECT coalesce(max(added_order), 0) + 1 FROM downloadclients_torrent), ?, ?, ?, ?, ?)`,
		id, l.node, m.InfoHash[:], raw, name, int64(4*gib), tracker, storage.FromTime(at), up, int64(4*gib), string(report), storage.FromTime(at))
	return id, m.InfoHash
}

// TestASendToAnExternalClientReleasesTheWindowOfTheTorrentItRemoves sends
// a private torrent that owes seedpool from the built-in client to
// Transmission on den. The built-in client's copy is removed, and once
// the Node parts with it the window reads released, since the external
// client seeds it from then on. A remove with no copy anywhere reads
// missed.
func TestASendToAnExternalClientReleasesTheWindowOfTheTorrentItRemoves(t *testing.T) {
	l := newLibrary(t)
	seedpool := l.tracker("seedpool", indexers.Private)
	opened := time.Now().Add(-time.Hour).Truncate(time.Second)
	sent, sentHash := l.privateTorrent("Orchard", seedpool, opened, gib)
	removed, removedHash := l.privateTorrent("Harbor", seedpool, opened, gib)
	for _, h := range []bencode.InfoHash{sentHash, removedHash} {
		l.exec(`INSERT INTO trackers_window (info_hash, tracker_id, state, rule_clears, rule_ratio_centi, opened_at) VALUES (?, ?, 'owing', 'ratio', 100, ?)`,
			h[:], seedpool, storage.FromTime(opened))
	}
	den := role.NodeID{UUID: storage.NewUUID()}
	l.exec(`INSERT INTO nodes_node (id, public_key, name, paired_at) VALUES (?, randomblob(32), 'den', 0)`, den)
	address, err := tc.ParseAddress("http://192.168.1.20:9091", tc.Transmission)
	mustNot(t, err)
	label, err := tc.ParseLabel(tc.DefaultLabel, tc.Transmission)
	mustNot(t, err)

	now := storage.Now()
	err = l.lib.Write(t.Context(), func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
		transmission, err := downloadclients.Import(ctx, tx, secrettest.Sealer(t), downloadclients.Imported{
			Node: den, Kind: tc.Transmission, Name: "Transmission", Address: address, Label: label,
		}, now)
		if err != nil {
			return err
		}
		_, err = downloadclients.Retarget(ctx, tx, sent, downloadclients.Target{Node: den, Client: &transmission}, now)
		return err
	})
	mustNot(t, err)
	l.exec(`UPDATE downloadclients_torrent SET parted_at = ?, freed = 0 WHERE id = ?`, now, sent)
	l.remove(removed, now.Time())

	if w := l.viewOf(sentHash).Window; w == nil || w.State != seeding.Released {
		t.Errorf("once sent to Transmission Orchard's window is %+v, want released", w)
	}
	if w := l.viewOf(removedHash).Window; w == nil || w.State != seeding.Missed {
		t.Errorf("once removed Harbor's window is %+v, want missed", w)
	}
	l.runGoals()
	if w := l.windows(seedpool)["Orchard"].Window; w.State != seeding.Released {
		t.Errorf("after a run Orchard's stored window is %+v, want released", w)
	}
}

func (l *library) viewOf(h bencode.InfoHash) seeding.View {
	l.t.Helper()
	var v seeding.View
	err := l.lib.Read(l.t.Context(), func(ctx context.Context, r storage.Reader[storage.Library]) error {
		var err error
		v, err = l.x.ViewOf(ctx, r, h)
		return err
	})
	mustNot(l.t, err)
	return v
}
