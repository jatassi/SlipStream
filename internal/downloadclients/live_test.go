package downloadclients_test

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"sync/atomic"
	"testing"
	"testing/synctest"
	"time"

	"github.com/jatassi/slipstream/internal/contenttype/contenttypetest"
	dc "github.com/jatassi/slipstream/internal/downloadclients"
	"github.com/jatassi/slipstream/internal/health"
	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/indexers"
	"github.com/jatassi/slipstream/internal/nodes"
	dlv1 "github.com/jatassi/slipstream/internal/pb/slipstream/downloading/v1"
	"github.com/jatassi/slipstream/internal/pb/slipstream/downloading/v1/downloadingv1connect"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/roots"
	"github.com/jatassi/slipstream/internal/secret"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/storage/storagetest"
	"github.com/jatassi/slipstream/internal/works"
)

const liveSample = time.Second

// downloader is a Node's client downloading one torrent at 1000 bytes a
// second: each Watch answers a sample later with it 1000 bytes further.
// Restarted, it answers afresh with nothing moving.
type downloader struct {
	downloadingv1connect.UnimplementedDownloadingServiceHandler
	id        dc.ID
	restarted atomic.Bool

	mu    sync.Mutex
	calls int
}

func (d *downloader) Watch(ctx context.Context, req *dlv1.WatchRequest) (*dlv1.WatchResponse, error) {
	d.mu.Lock()
	d.calls++
	d.mu.Unlock()
	select {
	case <-time.After(liveSample):
	case <-ctx.Done():
		return nil, fmt.Errorf("watch: %w", ctx.Err())
	}
	if d.restarted.Load() {
		return &dlv1.WatchResponse{Sample: 1000}, nil
	}
	sample := req.GetAfter() + 1
	return &dlv1.WatchResponse{Sample: sample, DownBps: 1000, Rows: []*dlv1.Live{
		{Id: d.id.Bytes(), DownBps: 1000, Done: int64(sample) * 1000, Wanted: 1 << 20, Peers: 3}, //nolint:gosec // a test's sample count fits
	}}, nil
}

func (d *downloader) watched() int {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.calls
}

// admin is the Admins' event stream: listening while one is open.
type admin struct {
	listening atomic.Bool

	mu    sync.Mutex
	ticks []dc.LiveTick
}

func (a *admin) Publish(_ context.Context, v dc.LiveTick) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.ticks = append(a.ticks, v)
}

func (a *admin) Listening() bool { return a.listening.Load() }

func (a *admin) heard() []dc.LiveTick {
	a.mu.Lock()
	defer a.mu.Unlock()
	return append([]dc.LiveTick(nil), a.ticks...)
}

// expectLast checks the last torrents.live: download at done, or a
// sample short of it when the Node answered in the same instant the tick
// went out, and the downloads adding up to it beside the paused one as
// reported.
func (a *admin) expectLast(t *testing.T, download dc.ID, done int64) {
	t.Helper()
	ticks := a.heard()
	last := ticks[len(ticks)-1]
	if len(last.Rows) != 1 || last.Rows[0].ID != download.String() || last.Rows[0].Done != done && last.Rows[0].Done != done-1000 {
		t.Fatalf("the last torrents.live carried %+v, want the download at %d", last.Rows, done)
	}
	if want := (dc.Aggregate{Done: last.Rows[0].Done + 200, Wanted: 2 * 1024}); last.Downloads != want {
		t.Errorf("the downloads add up to %+v, want %+v: the download live, the paused one as reported", last.Downloads, want)
	}
}

// TestLiveSamplesFlowOnlyWhileAnAdminListens has a Node download one
// Torrent beside a paused one. Nothing calls the Node until an Admin
// listens; then its samples reach Sample, Live and torrents.live, whose
// aggregate counts the paused download as last reported. A restarted
// Node's fresh sample drops what it said before, and once no Admin
// listens the calls stop and the samples go.
func TestLiveSamplesFlowOnlyWhileAnAdminListens(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		lib := open(t)
		a := addNode(t, lib, "seedbox")
		m, raw := dotTorrent(t, "Orchard")
		down, _ := mustAttach(t, lib, dc.Want{Target: dc.Target{Node: a}, Metainfo: m, Raw: raw, ContentType: contenttypetest.ID, WrittenOff: noneWrittenOff})
		m, raw = dotTorrent(t, "Meadow")
		paused, _ := mustAttach(t, lib, dc.Want{Target: dc.Target{Node: a}, Metainfo: m, Raw: raw, ContentType: contenttypetest.ID, WrittenOff: noneWrittenOff})
		apply(t, lib, a, down, &dlv1.TorrentReport{AtUnixMs: reportedAt.UnixMilli(), Engine: dlv1.EngineState_ENGINE_STATE_DOWNLOADING, Done: 100})
		apply(t, lib, a, paused, &dlv1.TorrentReport{AtUnixMs: reportedAt.UnixMilli(), Engine: dlv1.EngineState_ENGINE_STATE_DOWNLOADING, Hold: dlv1.Hold_HOLD_WANTED, Done: 200})
		exec(t, lib, "UPDATE downloadclients_torrent SET wanted = 'paused' WHERE id = ?", paused)
		node, listener := &downloader{id: down}, &admin{}
		c := dc.New(dc.Config{
			Library: lib, Touch: func(...role.NodeID) {}, Changed: published{},
			Remote: func(role.NodeID) downloadingv1connect.DownloadingServiceHandler { return node },
			Roster: &roster{}, Log: slog.New(slog.DiscardHandler),
			Live: dc.LiveConfig{Watched: func() []role.NodeID { return []role.NodeID{a} }, Event: listener, Sample: liveSample},
			Hub:  levels{},
		})
		ctx, stop := context.WithCancel(t.Context())
		done := make(chan struct{})
		go func() {
			defer close(done)
			_ = c.Run(ctx)
		}()
		defer func() {
			stop()
			<-done
		}()

		time.Sleep(time.Minute)
		synctest.Wait()
		if n := node.watched(); n != 0 || len(listener.heard()) != 0 {
			t.Fatalf("with no Admin listening the Node was watched %d times and %d ticks went out, want none", n, len(listener.heard()))
		}

		listener.listening.Store(true)
		time.Sleep(10 * liveSample)
		synctest.Wait()
		s, ok := c.Sample(down)
		if !ok || s.DownBps != 1000 || s.Done < 1000 || s.Peers != 3 {
			t.Fatalf("Sample of the download = %+v, %v; want it moving at 1000 bytes a second", s, ok)
		}
		if l, ok := c.Live(a); !ok || l.DownBps != 1000 || l.Torrents[down.UUID].Done != s.Done {
			t.Fatalf("Live(seedbox) = %+v, %v; want its rate and the download's sample", l, ok)
		}
		listener.expectLast(t, down, s.Done)

		node.restarted.Store(true)
		time.Sleep(3 * liveSample)
		synctest.Wait()
		if l, ok := c.Live(a); !ok || len(l.Torrents) != 0 {
			t.Fatalf("after the Node restarted Live(seedbox) = %+v, %v; want no torrent moving", l, ok)
		}

		listener.listening.Store(false)
		time.Sleep(2 * liveSample)
		synctest.Wait()
		calls := node.watched()
		time.Sleep(time.Minute)
		synctest.Wait()
		if n := node.watched(); n != calls {
			t.Errorf("with no Admin listening the Node was watched %d more times, want none", n-calls)
		}
		if _, ok := c.Live(a); ok {
			t.Errorf("with no Admin listening Live(seedbox) still answers")
		}
	})
}

func TestLiveAnswersTheTorrentsNoClientRemoved(t *testing.T) {
	lib := storagetest.Library(t, secret.Schema(), identity.Schema(), nodes.Schema(), works.Schema(), health.Schema(), roots.Schema(), dc.Schema(), indexers.Schema())
	node, kept, removed, never := storage.NewUUID(), storage.NewUUID(), storage.NewUUID(), storage.NewUUID()
	err := lib.Write(t.Context(), func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
		if _, err := tx.ExecContext(ctx, `INSERT INTO nodes_node (id, public_key, name, paired_at) VALUES (?, randomblob(32), 'home', 0)`, node); err != nil {
			return err
		}
		for i, id := range []storage.UUID{kept, removed} {
			_, err := tx.ExecContext(ctx, `INSERT INTO downloadclients_torrent (id, node_id, info_hash, metainfo, name, size, private, trackers, content_type, wanted,
				added_order, added_at) VALUES (?, ?, randomblob(20), x'00', 'Orchard.WEB-DL', 1, 0, '[]', 'fixture', 'started', ?, 0)`, id, node, i)
			if err != nil {
				return err
			}
		}
		_, err := tx.ExecContext(ctx, `UPDATE downloadclients_torrent SET removed_at = 1, removed_reason = 'admin' WHERE id = ?`, removed)
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	var live map[dc.ID]bool
	err = lib.Read(t.Context(), func(ctx context.Context, r storage.Reader[storage.Library]) error {
		live, err = dc.Live(ctx, r, []dc.ID{{UUID: kept}, {UUID: removed}, {UUID: never}})
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(live) != 1 || !live[dc.ID{UUID: kept}] {
		t.Fatalf("Live answers %v, want the torrent no client removed alone", live)
	}
}
