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
	dlv1 "github.com/jatassi/slipstream/internal/pb/slipstream/downloading/v1"
	"github.com/jatassi/slipstream/internal/pb/slipstream/downloading/v1/downloadingv1connect"
	"github.com/jatassi/slipstream/internal/role"
)

const liveSample = time.Second

// downloader is a Node's client downloading one torrent at 1000 bytes a
// second: each Watch answers a sample later with it 1000 bytes further.
// Restarted, it answers afresh with nothing moving.
type downloader struct {
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

// expectLast checks the last torrents.live: download at done, and the
// downloads adding up to it beside the paused one as reported.
func (a *admin) expectLast(t *testing.T, download dc.ID, done int64) {
	t.Helper()
	ticks := a.heard()
	last := ticks[len(ticks)-1]
	if len(last.Rows) != 1 || last.Rows[0].ID != download.String() || last.Rows[0].Done != done {
		t.Fatalf("the last torrents.live carried %+v, want the download at %d", last.Rows, done)
	}
	if want := (dc.Aggregate{Done: done + 200, Wanted: 2 * 1024}); last.Downloads != want {
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
		down, _ := mustAttach(t, lib, dc.Want{Node: a, Metainfo: m, Raw: raw, ContentType: contenttypetest.ID, WrittenOff: noneWrittenOff})
		m, raw = dotTorrent(t, "Meadow")
		paused, _ := mustAttach(t, lib, dc.Want{Node: a, Metainfo: m, Raw: raw, ContentType: contenttypetest.ID, WrittenOff: noneWrittenOff})
		apply(t, lib, a, down, &dlv1.TorrentReport{AtUnixMs: reportedAt.UnixMilli(), Engine: dlv1.EngineState_ENGINE_STATE_DOWNLOADING, Done: 100})
		apply(t, lib, a, paused, &dlv1.TorrentReport{AtUnixMs: reportedAt.UnixMilli(), Engine: dlv1.EngineState_ENGINE_STATE_DOWNLOADING, Hold: dlv1.Hold_HOLD_WANTED, Done: 200})
		exec(t, lib, "UPDATE downloadclients_torrent SET wanted = 'paused' WHERE id = ?", paused)
		node, listener := &downloader{id: down}, &admin{}
		c := dc.New(dc.Config{Touch: func(...role.NodeID) {}, Changed: published{}, Live: dc.LiveConfig{
			Library: lib,
			Watched: func() []role.NodeID { return []role.NodeID{a} },
			Remote:  func(role.NodeID) downloadingv1connect.DownloadingServiceHandler { return node },
			Event:   listener, Sample: liveSample, Log: slog.New(slog.DiscardHandler),
		}})
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
