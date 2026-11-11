package grabs_test

import (
	"testing"
	"testing/synctest"
	"time"

	"github.com/jatassi/slipstream/internal/downloadclients"
	"github.com/jatassi/slipstream/internal/grabs"
	"github.com/jatassi/slipstream/internal/grabs/grabstest"
)

// TestAFetchedTorrentOnTheBlocklistFailsAManualGrabBeforeAnyClient grabs
// by hand a magnet link listed without an info hash whose metadata names
// a blocklisted torrent: the Grab fails blocklisted and no client gets it.
func TestAFetchedTorrentOnTheBlocklistFailsAManualGrabBeforeAnyClient(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		rel := newRelease(t)
		p := newPipeline(t, answer{res: downloadclients.Resolved{Info: rel.info}})
		id := p.grab(rel.link.String(), nil)
		run(t, p.lib, sql(`UPDATE grabs_grab SET source = 'manual', grabbed_by_name = 'Jackson'`))
		grabstest.Blocked(t, p.lib, entry(rel.link.InfoHash, "The.Attic.1080p.WEB"))

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
