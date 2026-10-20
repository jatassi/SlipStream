package sim_test

import (
	"context"
	"fmt"
	"io"
	"math/rand/v2"
	"net/http"
	"slices"
	"sync/atomic"
	"testing"
	"testing/synctest"
	"time"

	"github.com/jatassi/slipstream/fakes/internal/releases"
	"github.com/jatassi/slipstream/fakes/world/sim"
	"github.com/jatassi/slipstream/internal/bencode"
	"github.com/jatassi/slipstream/internal/downloadclients"
	"github.com/jatassi/slipstream/internal/downloading"
	"github.com/jatassi/slipstream/internal/link"
	"github.com/jatassi/slipstream/internal/node"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/torrent/engine"
	"github.com/jatassi/slipstream/internal/torrent/engine/enginetest"
	"github.com/jatassi/slipstream/internal/torrent/unit3d"
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
		attachSeeded(t, controller, swarm, boxID, idleTorrents)
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

// attachSeeded has the Controller hand Node box n torrents announcing to
// idleTracker, whose payloads swarm holds.
func attachSeeded(t *testing.T, controller node.Half, swarm *enginetest.World, box role.NodeID, n int) []downloadclients.ID {
	t.Helper()
	var ids []downloadclients.ID
	err := controller.Library.Write(t.Context(), func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
		ids = nil
		for i := range n {
			payload := seededPayload(i)
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
			id, _, err := downloadclients.Attach(ctx, tx, want, storage.Now())
			if err != nil {
				return err
			}
			ids = append(ids, id)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	controller.Clients.Touch(box)
	return ids
}

func seededPayload(i int) []byte { return []byte(fmt.Sprintf("idle seedbox payload %d", i)) }

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

// launchSwarm launches home with seedbox's engines in swarm, followed by
// box, and swarm's tracker at idleTracker answering by rules.
func launchSwarm(t *testing.T, rules enginetest.TrackerRules, box *engines, opts ...sim.Option) (w *sim.World, swarm *enginetest.World) {
	t.Helper()
	swarm = enginetest.NewWorld()
	swarm.Tracker(idleTracker, rules)
	opts = append(opts, onSeedbox(func(e *node.Env) { e.Engine = box.wrap(swarm.Open) }))
	return sim.Launch(t, home(t), opts...), swarm
}

// act has home's Clients run a, failing the test on any refusal.
func act(t *testing.T, w *sim.World, a downloadclients.Action) *downloadclients.Preview {
	t.Helper()
	results, preview, err := w.Node("home").Half().Clients.Act(t.Context(), a)
	if err != nil {
		t.Fatal(err)
	}
	for _, r := range results {
		if r.Refusal != nil {
			t.Fatalf("%s of %s was refused: %+v", a.Kind, r.ID, *r.Refusal)
		}
	}
	return preview
}

// announced is what swarm's tracker heard with event, oldest first.
func announced(swarm *enginetest.World, event string) []enginetest.Announce {
	return slices.DeleteFunc(swarm.Announces(idleTracker), func(a enginetest.Announce) bool { return a.Event != event })
}

// unit3dRejects replays what swarm's tracker heard through UNIT3D's rules
// for minInterval, and lists the announces UNIT3D would reject.
func unit3dRejects(swarm *enginetest.World, minInterval time.Duration) []string {
	windows := unit3d.NewWindows(minInterval, rand.New(rand.NewPCG(1, 2))) //nolint:gosec // draws window lengths
	var out []string
	for _, a := range swarm.Announces(idleTracker) {
		var peer [20]byte
		copy(peer[:], a.PeerID)
		if rule, reason := windows.Admit("passkey", a.InfoHash, peer, a.Event, a.At); rule != "" {
			out = append(out, fmt.Sprintf("%q at %s: %s", a.Event, a.At.Format(time.TimeOnly), reason))
		}
	}
	return out
}

// parted is when the Controller parted id from its Node, and the bytes
// the Node freed, zero until it confirms the removal.
func parted(t *testing.T, w *sim.World, id downloadclients.ID) (partedAt, freed int64) {
	t.Helper()
	err := w.Node("home").Half().Library.QueryRowContext(t.Context(),
		"SELECT coalesce(parted_at, 0), coalesce(freed, 0) FROM downloadclients_torrent WHERE id = ?", id).Scan(&partedAt, &freed)
	if err != nil {
		t.Fatal(err)
	}
	return partedAt, freed
}

// TestRemoveWaitsForStoppedAndConfirms pauses a seeding torrent 20 s
// after it started and resumes it, which starts it again 30 s after its
// first started, then removes it with its files: its Node holds the
// remove until UNIT3D takes a second stopped, removes it, and the
// Controller parts it once the Node confirms with the bytes it freed.
func TestRemoveWaitsForStoppedAndConfirms(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		const minInterval = 10 * time.Second
		box := &engines{}
		w, swarm := launchSwarm(t, enginetest.TrackerRules{Interval: 30 * time.Minute, MinInterval: minInterval}, box)
		ids := attachSeeded(t, w.Node("home").Half(), swarm, nodeID(t, w, "seedbox"), 1)
		eventually(t, "the torrent seeds", func() bool { return seeding(t, w.Node("home").Half()) == 1 })
		time.Sleep(time.Until(announced(swarm, "started")[0].At.Add(20 * time.Second)))
		act(t, w, downloadclients.Action{Kind: downloadclients.Pause, IDs: ids})
		eventually(t, "the torrent stops", func() bool { return len(announced(swarm, "stopped")) == 1 })
		act(t, w, downloadclients.Action{Kind: downloadclients.Resume, IDs: ids})
		eventually(t, "the torrent starts again", func() bool { return len(announced(swarm, "started")) == 2 })

		act(t, w, downloadclients.Action{Kind: downloadclients.Remove, IDs: ids, DeleteFiles: true})
		synctest.Wait()
		if at, _ := parted(t, w, ids[0]); at != 0 || box.torrents() != 1 {
			t.Fatalf("right after the remove the Torrent reads parted at %d with seedbox holding %d torrents, want Removing and still held", at, box.torrents())
		}
		eventually(t, "the Node confirms the removal", func() bool { at, _ := parted(t, w, ids[0]); return at != 0 })

		stops := announced(swarm, "stopped")
		if len(stops) != 2 || stops[1].At.Sub(stops[0].At) < unit3d.Duplicate {
			t.Fatalf("the tracker heard stopped at %v, want a second one at least %v after the first", stops, unit3d.Duplicate)
		}
		if got := box.removals(); len(got) != 1 || !got[0].deleteFiles || got[0].at.Before(stops[1].At) {
			t.Fatalf("seedbox's engine removed %+v, want one removal with its files after the stopped at %v", got, stops[1].At)
		}
		if _, freed := parted(t, w, ids[0]); freed != int64(len(seededPayload(0))) {
			t.Errorf("the Node freed %d bytes, want the payload's %d", freed, len(seededPayload(0)))
		}
		if r := unit3dRejects(swarm, minInterval); len(r) != 0 {
			t.Errorf("UNIT3D would reject %v", r)
		}
	})
}

// TestPauseAndResumeInsideTheWindowAreAccepted pauses a seeding torrent
// and resumes it at once: the Node starts it again only once UNIT3D's
// window from its first started ends.
func TestPauseAndResumeInsideTheWindowAreAccepted(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		const minInterval = 5 * time.Minute
		w, swarm := launchSwarm(t, enginetest.TrackerRules{Interval: 30 * time.Minute, MinInterval: minInterval}, &engines{})
		ids := attachSeeded(t, w.Node("home").Half(), swarm, nodeID(t, w, "seedbox"), 1)
		eventually(t, "the torrent seeds", func() bool { return seeding(t, w.Node("home").Half()) == 1 })
		act(t, w, downloadclients.Action{Kind: downloadclients.Pause, IDs: ids})
		eventually(t, "the torrent stops", func() bool { return len(announced(swarm, "stopped")) == 1 })
		act(t, w, downloadclients.Action{Kind: downloadclients.Resume, IDs: ids})
		eventually(t, "the torrent starts again", func() bool { return len(announced(swarm, "started")) == 2 })

		starts := announced(swarm, "started")
		if gap := starts[1].At.Sub(starts[0].At); gap < minInterval {
			t.Fatalf("the torrent started again %v after its first started, want at least the window's %v", gap, minInterval)
		}
		if r := unit3dRejects(swarm, minInterval); len(r) != 0 {
			t.Fatalf("UNIT3D would reject %v", r)
		}
	})
}

// TestSteadyAnnouncesAreJittered lets torrents seed for hours against a
// tracker that asks for an announce each interval: each torrent announces
// between one interval and one interval plus the jitter after the last,
// and the waits differ.
func TestSteadyAnnouncesAreJittered(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		const (
			n        = 20
			interval = 30 * time.Minute
		)
		jitter := downloading.DefaultTimings().AnnounceJitter
		w, swarm := launchSwarm(t, enginetest.TrackerRules{Interval: interval, MinInterval: interval}, &engines{})
		attachSeeded(t, w.Node("home").Half(), swarm, nodeID(t, w, "seedbox"), n)
		eventually(t, "every torrent seeds", func() bool { return seeding(t, w.Node("home").Half()) == n })
		time.Sleep(3 * interval)
		synctest.Wait()

		last := map[bencode.InfoHash]time.Time{}
		var gaps []time.Duration
		for _, a := range swarm.Announces(idleTracker) {
			if a.Event == "completed" {
				continue
			}
			if prev, ok := last[a.InfoHash]; ok {
				gaps = append(gaps, a.At.Sub(prev))
			}
			last[a.InfoHash] = a.At
		}
		if len(gaps) < 2*n {
			t.Fatalf("the tracker heard %d steady announces over %v, want at least two a torrent", len(gaps), 3*interval)
		}
		for _, g := range gaps {
			if g < interval || g > interval+jitter+2*time.Second {
				t.Fatalf("a torrent announced %v after its last, want between %v and %v", g, interval, interval+jitter)
			}
		}
		if spread := slices.Max(gaps) - slices.Min(gaps); spread < jitter/4 {
			t.Errorf("the waits between announces spread over %v, want them drawn across the jitter %v", spread, jitter)
		}
	})
}

// TestRemoveFailsDownloadingGrabsWithRemoved removes the Torrent a Grab
// downloads: the preview names the Grab it cancels, and the remove fails
// it with removed and a Detail that reads as a sentence.
func TestRemoveFailsDownloadingGrabsWithRemoved(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		w := sim.Launch(t, home(t), sim.WithWalk(), onSeedbox(func(e *node.Env) { e.Engine = enginetest.Open }))
		search(t, w, addOrchard(t, w))
		eventually(t, "the Grab downloads", func() bool {
			gs := grabsListed(t, w)
			return len(gs) == 1 && gs[0].Stage == "downloading"
		})
		var id downloadclients.ID
		if err := w.Node("home").Half().Library.QueryRowContext(t.Context(), "SELECT id FROM downloadclients_torrent").Scan(&id); err != nil {
			t.Fatal(err)
		}

		preview := act(t, w, downloadclients.Action{Kind: downloadclients.Remove, IDs: []downloadclients.ID{id}, DryRun: true})
		if len(preview.Cancels) != 1 || preview.Cancels[0].Release != release || preview.Cancels[0].Stage != downloadclients.UseDownloading {
			t.Fatalf("the remove's preview cancels %+v, want the Grab downloading %s", preview.Cancels, release)
		}
		act(t, w, downloadclients.Action{Kind: downloadclients.Remove, IDs: []downloadclients.ID{id}})
		expectFailed(t, await(t, w), "downloading", "removed", "The Admin removed this Grab's torrent")
	})
}
