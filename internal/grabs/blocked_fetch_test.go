package grabs_test

import (
	"testing"
	"testing/synctest"
	"time"

	"github.com/jatassi/slipstream/internal/downloadclients"
	"github.com/jatassi/slipstream/internal/grabs"
)

// TestAFetchedTorrentOnTheBlocklistFailsItsGrabBeforeAnyClient grabs a
// magnet link listed without an info hash whose metadata names a
// blocklisted torrent: the Grab fails blocklisted and no client gets it.
func TestAFetchedTorrentOnTheBlocklistFailsItsGrabBeforeAnyClient(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		rel := newRelease(t)
		p := newPipeline(t, answer{res: downloadclients.Resolved{Info: rel.info}})
		id := p.grab(rel.link.String(), nil)
		run(t, p.lib, sql(`INSERT INTO grabs_blocklist (info_hash, release_title, release_size, cause, added_by_name, added_at)
			VALUES (?, 'The.Attic.1080p.WEB', 4096, 'by_hand', 'Jackson', 0)`, rel.link.InfoHash[:]))

		p.running()
		time.Sleep(2 * hold)

		v := p.view(id)
		if v.Stage != grabs.Failed || v.Failed.Code != grabs.Blocklisted || v.Failed.Message != "The Release's torrent is on the Blocklist." {
			t.Fatalf("the Grab is %s with %+v, want failed blocklisted", v.Stage, v.Failed)
		}
		var torrents int
		if err := p.lib.QueryRowContext(t.Context(), "SELECT count(*) FROM downloadclients_torrent").Scan(&torrents); err != nil {
			t.Fatal(err)
		}
		if torrents != 0 {
			t.Fatalf("%d Torrents were added for the blocklisted torrent, want none", torrents)
		}
	})
}
