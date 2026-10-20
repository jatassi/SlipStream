package sim_test

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"sync/atomic"
	"testing"
	"testing/synctest"
	"time"

	"github.com/jatassi/slipstream/fakes/internal/releases"
	"github.com/jatassi/slipstream/fakes/world/sim"
	"github.com/jatassi/slipstream/internal/bencode"
	"github.com/jatassi/slipstream/internal/downloadclients"
	"github.com/jatassi/slipstream/internal/link"
	"github.com/jatassi/slipstream/internal/node"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/torrent/engine"
	"github.com/jatassi/slipstream/internal/torrent/engine/enginetest"
	"github.com/jatassi/slipstream/internal/works"
)

// reads counts how often seedbox's client reads its engine's payload
// counters: twice a live sample, at the window's start and end.
func reads(n *atomic.Int64) sim.Option {
	return onSeedbox(func(e *node.Env) {
		open := e.Engine
		e.Engine = func(cfg engine.Config) (engine.Engine, error) {
			eng, err := open(cfg)
			return counted{Engine: eng, reads: n}, err
		}
	})
}

type counted struct {
	engine.Engine
	reads *atomic.Int64
}

func (c counted) PostStats() error {
	c.reads.Add(1)
	return c.Engine.PostStats()
}

// TestWatchSendsNothingWhileNoAdminListens has seedbox's client sample
// only while an Admin's live stream is open on home: never before, about
// once a live sample during, and never once it closes.
func TestWatchSendsNothingWhileNoAdminListens(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		var n atomic.Int64
		w := sim.Launch(t, home(t), reads(&n))
		time.Sleep(5 * time.Minute)
		synctest.Wait()
		if got := n.Load(); got != 0 {
			t.Fatalf("with no Admin listening seedbox read its counters %d times, want never", got)
		}

		home := entryOf(t, w, "home")
		stream := home.send(http.MethodGet, "/api/v1/events", "", home.signIn().session, "")
		defer func() { _ = stream.Body.Close() }()
		go func() { _, _ = io.Copy(io.Discard, stream.Body) }()
		time.Sleep(10 * time.Second)
		synctest.Wait()
		if got := n.Load(); got < 2*5 || got > 2*11 {
			t.Fatalf("over ten seconds of an Admin listening seedbox read its counters %d times, want about one sample a second", got)
		}

		_ = stream.Body.Close()
		time.Sleep(5 * time.Second)
		synctest.Wait()
		closed := n.Load()
		time.Sleep(5 * time.Minute)
		synctest.Wait()
		if got := n.Load(); got != closed {
			t.Fatalf("once the Admin's stream closed seedbox read its counters %d more times, want none", got-closed)
		}
	})
}

const (
	idleTorrents = 300
	idleBudgeted = 3000
	idleTracker  = "https://tracker.test/announce/passkey"
)

// TestAnIdleSeedboxOf3000TorrentsSendsUnder1KBps hands seedbox 300
// torrents that finish at once, whose tracker answers every 30 minutes,
// and lets them settle past the start spread and one interval. With no
// Admin listening, it then measures what seedbox sends home over ten
// minutes, as the link counts its bytes. Ten times that bounds what 3000
// torrents send, since only part of it grows with the torrents, and the
// bound is under 1 KB a second. 300 rather than 3000 keeps the test
// within the race detector's time: the client settles every torrent on
// each event.
func TestAnIdleSeedboxOf3000TorrentsSendsUnder1KBps(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		swarm := enginetest.NewWorld()
		swarm.Tracker(idleTracker, enginetest.TrackerRules{Interval: 30 * time.Minute, MinInterval: 30 * time.Minute})
		w := sim.Launch(t, home(t), onSeedbox(func(e *node.Env) { e.Engine = swarm.Open }))
		box := seedbox(t, w.API())
		id, err := storage.ParseUUID(box.ID)
		if err != nil {
			t.Fatal(err)
		}
		boxID := role.NodeID{UUID: id}
		controller := w.Node("home").Half()
		attachIdle(t, controller, swarm, boxID)
		controller.Controls.Touch(boxID)
		eventually(t, "every torrent seeds", func() bool { return seeding(t, controller) == idleTorrents })
		time.Sleep(35 * time.Minute)
		synctest.Wait()

		rate := received(t, controller, w.Node("seedbox"), 10*time.Minute)
		bound := rate * idleBudgeted / idleTorrents
		t.Logf("an idle seedbox of %d torrents sends %.0f B/s, %.3f B/s a torrent; %d send at most %.0f B/s",
			idleTorrents, rate, rate/idleTorrents, idleBudgeted, bound)
		if bound >= 1024 {
			t.Errorf("an idle seedbox of %d torrents sends %.0f B/s, so %d may send %.0f B/s, want under 1 KB/s", idleTorrents, rate, idleBudgeted, bound)
		}
	})
}

// attachIdle has the Controller hand Node box idleTorrents torrents whose
// payloads swarm holds.
func attachIdle(t *testing.T, controller node.Half, swarm *enginetest.World, box role.NodeID) {
	t.Helper()
	err := controller.Library.Write(t.Context(), func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
		for i := range idleTorrents {
			payload := []byte(fmt.Sprintf("idle seedbox payload %d", i))
			raw, hash, err := releases.Metainfo(fmt.Sprintf("Idle.%d.mkv", i), payload, idleTracker)
			if err != nil {
				return err
			}
			swarm.Payload(hash, payload)
			m, err := bencode.DecodeMetainfo(raw)
			if err != nil {
				return err
			}
			want := downloadclients.Want{Node: box, Metainfo: m, Raw: raw, ContentType: works.TypeID("movie"), WrittenOff: func(role.NodeID) bool { return false }}
			if _, _, err := downloadclients.Attach(ctx, tx, want, storage.Now()); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

// received is the bytes a second the Controller's link reads from the Node
// from over window, as one session.
func received(t *testing.T, controller node.Half, from *sim.Node, window time.Duration) float64 {
	t.Helper()
	private, err := link.OpenKey(from.DataDir().Path())
	if err != nil {
		t.Fatal(err)
	}
	key := private.Public()
	before, ok := controller.Link.Stats(key)
	time.Sleep(window)
	synctest.Wait()
	after, still := controller.Link.Stats(key)
	if !ok || !still || after.Received < before.Received {
		t.Fatalf("the link read %+v, then %+v; want one session throughout", before, after)
	}
	return float64(after.Received-before.Received) / window.Seconds()
}

// seeding counts the Torrents whose Node reported them complete.
func seeding(t *testing.T, h node.Half) int {
	t.Helper()
	var n int
	err := h.Library.Read(t.Context(), func(ctx context.Context, r storage.Reader[storage.Library]) error {
		return r.QueryRowContext(ctx, "SELECT count(*) FROM downloadclients_torrent WHERE completion IS NOT NULL").Scan(&n)
	})
	if err != nil {
		t.Fatal(err)
	}
	return n
}
