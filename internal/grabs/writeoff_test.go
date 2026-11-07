package grabs_test

import (
	"testing"
	"testing/synctest"
	"time"

	"github.com/jatassi/slipstream/internal/grabs"
	"github.com/jatassi/slipstream/internal/storage"
)

// TestAGrabFailsOnceItsWrittenOffTorrentIsParted fails a Grab at probing
// whose Torrent is parted as node_written_off while the roster reads no
// write-off.
func TestAGrabFailsOnceItsWrittenOffTorrentIsParted(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		rel := newRelease(t)
		p := newPipeline(t)
		id := p.grab(rel.link.String(), &rel.link.InfoHash)
		raw, err := rel.link.Torrent(rel.info)
		if err != nil {
			t.Fatal(err)
		}
		attachTorrent(t, p, raw)
		now := storage.Now()
		run(t, p.lib,
			sql(`UPDATE downloadclients_torrent SET removed_at = ?, removed_reason = 'node_written_off', parted_at = ?`, now, now),
			sql(`UPDATE grabs_grab SET stage = 'probing', torrent_id = (SELECT id FROM downloadclients_torrent)`),
		)

		p.running()
		time.Sleep(time.Second)

		v := p.view(id)
		if v.Stage != grabs.Failed || v.Failed.Stage != grabs.Probing || v.Failed.Code != grabs.NodeWrittenOff {
			t.Fatalf("the Grab is %s with %+v, want failed at probing with node_written_off", v.Stage, v.Failed)
		}
	})
}
