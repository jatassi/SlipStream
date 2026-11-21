package grabs_test

import (
	"context"
	"testing"
	"testing/synctest"
	"time"

	"github.com/jatassi/slipstream/internal/downloadclients"
	"github.com/jatassi/slipstream/internal/grabs"
	"github.com/jatassi/slipstream/internal/storage"
)

// reconciled ends Reconciling as its Write does: Grabs at fetching are
// decided, then told once it commits.
func (p *pipeline) reconciled() {
	p.t.Helper()
	err := p.lib.Write(p.t.Context(), func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
		return p.g.DecideFetching(ctx, tx)
	})
	if err != nil {
		p.t.Fatal(err)
	}
	p.reconciling.Store(false)
	p.g.Reconciled(p.t.Context())
}

func TestAGrabAtFetchingWaitsThenTakesTheAdoptedTorrent(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		rel := newRelease(t)
		p := newPipeline(t, answer{err: downloadclients.ErrHeld})
		p.reconciling.Store(true)
		id := p.grab(rel.link.String(), &rel.link.InfoHash)
		p.running()
		time.Sleep(time.Hour)
		if v := p.view(id); v.Stage != grabs.Fetching || len(p.swarm.asked()) != 0 {
			t.Fatalf("while Reconciling the Grab is %s after %d calls, want fetching with none", v.Stage, len(p.swarm.asked()))
		}
		raw, err := rel.link.Torrent(rel.info)
		if err != nil {
			t.Fatal(err)
		}
		attachTorrent(t, p, raw)
		p.reconciled()
		time.Sleep(time.Minute)
		if v := p.view(id); v.Stage != grabs.Downloading || v.Torrent.InfoHash != rel.link.InfoHash {
			t.Errorf("once Reconciling ends with its torrent adopted the Grab is %s, want downloading that torrent", v.Stage)
		}
	})
}

func TestAGrabAtFetchingWaitsThenFailsRestored(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		rel := newRelease(t)
		p := newPipeline(t, answer{err: downloadclients.ErrHeld})
		p.reconciling.Store(true)
		id := p.grab(rel.link.String(), &rel.link.InfoHash)
		p.running()
		time.Sleep(time.Hour)
		p.reconciled()
		if v := p.failed(id, grabs.Restored); v.Failed.Message != "No Node had this Grab's torrent after the restore." || len(p.swarm.asked()) != 0 {
			t.Errorf("the Grab no Node held failed with %q after %d calls, want the restore's words and none", v.Failed.Message, len(p.swarm.asked()))
		}
	})
}
