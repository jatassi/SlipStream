package downloading_test

import (
	"os"
	"path/filepath"
	"testing"
	"testing/synctest"
	"time"

	"google.golang.org/protobuf/proto"

	dlv1 "github.com/jatassi/slipstream/internal/pb/slipstream/downloading/v1"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/torrent/engine/enginetest"
)

// asked is the Controller's time of every test's start ask.
const asked = 1_700_000_000_000

// arriving has the Controller answer for f as a Move's arriving Movies
// torrent whose payload already lies in its folder.
func (n *node) arriving(f file) storage.UUID {
	n.t.Helper()
	folder := filepath.Join(n.dir, "Movies", f.hash.String())
	if err := os.MkdirAll(folder, 0o750); err != nil {
		n.t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(folder, f.name), f.payload, 0o600); err != nil {
		n.t.Fatal(err)
	}
	id := storage.NewUUID()
	n.ctl.answer(id, &dlv1.TorrentResponse{Metainfo: f.meta, ContentType: "Movies", SeedMode: true})
	return id
}

func (f file) asking(id storage.UUID, window time.Duration) *dlv1.Torrent {
	t := f.started(id, 1)
	t.Arriving, t.StartAskedUnixMs, t.StartWindowMs = true, asked, window.Milliseconds()
	return t
}

func stoppedAt(w *enginetest.World, f file) []time.Time {
	var out []time.Time
	for _, a := range w.Announces(trackerURL) {
		if a.InfoHash == f.hash && a.Event == "stopped" {
			out = append(out, a.At)
		}
	}
	return out
}

func arrivalOf(t *testing.T, c *client, id storage.UUID) *dlv1.Arrival {
	t.Helper()
	return c.h.latest(t, id).GetArrival()
}

func TestAnArrivingTorrentSeedsWithoutACheckAndIsJudgedStartedOnTheReply(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		n := newNode(t, enginetest.TrackerRules{Interval: 30 * time.Minute})
		f := newFile(t, "Orchard.mkv")
		id := n.arriving(f)
		c := n.start()
		start := time.Now()
		c.send(live(f.asking(id, time.Minute)))

		if got := c.engine.count(); got != 0 {
			t.Errorf("the arriving torrent was checked %d times, want none in seed mode", got)
		}
		at := startedAnnounces(n.world, f.hash, start)
		if len(at) != 1 {
			t.Fatalf("the arriving torrent announced started at %v, want once", at)
		}
		want := &dlv1.Arrival{AskedUnixMs: asked, Result: dlv1.ArrivalResult_ARRIVAL_RESULT_STARTED, AnnouncedUnixMs: at[0].UnixMilli()}
		if got := arrivalOf(t, c, id); !proto.Equal(got, want) {
			t.Errorf("arrival %v, want %v", got, want)
		}
		if r := c.h.latest(t, id); r.GetEngine() != dlv1.EngineState_ENGINE_STATE_SEEDING {
			t.Errorf("the arriving torrent reports %v, want seeding", r)
		}
	})
}

func TestAStartIsRefusedUnlessTheTrackerAnswersItsStartedInTime(t *testing.T) {
	for _, c := range []struct {
		name   string
		rules  enginetest.TrackerRules
		window time.Duration
		reason string
		// quiet holds the arrival unjudged until the reply wait passes, and
		// late never announces started.
		quiet, late bool
	}{
		{"a failure reply", enginetest.TrackerRules{Failure: "Torrent not registered with this tracker"}, time.Minute, "Torrent not registered with this tracker", false, false},
		{"no reply in the reply wait", enginetest.TrackerRules{Interval: 30 * time.Minute, HoldStarted: true}, time.Minute, "no_answer", true, false},
		{"a window under the reply wait", enginetest.TrackerRules{Interval: 30 * time.Minute}, timings.ReplyWait - time.Second, "late", false, true},
	} {
		t.Run(c.name, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				n := newNode(t, c.rules)
				f := newFile(t, "Orchard.mkv")
				id := n.arriving(f)
				cl := n.start()
				cl.send(live(f.asking(id, c.window)))
				if c.quiet {
					time.Sleep(timings.ReplyWait - time.Second)
					synctest.Wait()
					if got := arrivalOf(t, cl, id); got != nil {
						t.Fatalf("arrival %v within the reply wait, want none yet", got)
					}
				}
				time.Sleep(timings.ReplyWait)
				synctest.Wait()
				want := &dlv1.Arrival{AskedUnixMs: asked, Result: dlv1.ArrivalResult_ARRIVAL_RESULT_REFUSED, Reason: c.reason}
				if got := arrivalOf(t, cl, id); !proto.Equal(got, want) {
					t.Errorf("arrival %v, want %v", got, want)
				}
				if at := startedAnnounces(n.world, f.hash, time.Time{}); c.late && len(at) != 0 {
					t.Errorf("a late ask announced started at %v, want never", at)
				}
			})
		})
	}
}

func TestTheWindowsEndPausesAnAskStillOpenAndNotOneTheIntentDropped(t *testing.T) {
	const window = time.Minute
	t.Run("still open", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			n := newNode(t, enginetest.TrackerRules{Interval: 30 * time.Minute})
			f := newFile(t, "Orchard.mkv")
			id := n.arriving(f)
			c := n.start()
			received := time.Now()
			c.send(live(f.asking(id, window)))
			time.Sleep(window - time.Second)
			synctest.Wait()
			if at := stoppedAt(n.world, f); len(at) != 0 {
				t.Fatalf("the torrent announced stopped at %v inside its window, want it seeding", at)
			}
			time.Sleep(time.Minute)
			synctest.Wait()

			if at := stoppedAt(n.world, f); len(at) != 1 || at[0].Sub(received) != window {
				t.Errorf("the torrent announced stopped at %v, want once at the window's end %v", at, received.Add(window))
			}
			want := &dlv1.Arrival{AskedUnixMs: asked, Result: dlv1.ArrivalResult_ARRIVAL_RESULT_REFUSED, Reason: "no_answer"}
			if got := arrivalOf(t, c, id); !proto.Equal(got, want) {
				t.Errorf("arrival %v once the window ended with the ask open, want %v", got, want)
			}
		})
	})
	t.Run("dropped", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			n := newNode(t, enginetest.TrackerRules{Interval: 30 * time.Minute})
			f := newFile(t, "Orchard.mkv")
			id := n.arriving(f)
			c := n.start()
			c.send(live(f.asking(id, window)))
			c.send(live(f.started(id, 1)))
			time.Sleep(2 * window)
			synctest.Wait()

			if at := stoppedAt(n.world, f); len(at) != 0 {
				t.Errorf("the torrent announced stopped at %v once the intent dropped the ask, want it seeding on", at)
			}
			if got := arrivalOf(t, c, id); got != nil {
				t.Errorf("arrival %v once the intent dropped the ask, want none", got)
			}
		})
	})
}

func TestARestartJudgesTheAskAgainInsideTheFreshWindow(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		n := newNode(t, enginetest.TrackerRules{Interval: 30 * time.Minute})
		f := newFile(t, "Orchard.mkv")
		id := n.arriving(f)
		first := n.start()
		first.send(live(f.asking(id, 10*time.Minute)))
		first.stop()
		time.Sleep(time.Minute)

		const fresh = 2 * time.Minute
		c := n.start()
		received := time.Now()
		c.send(live(f.asking(id, fresh)))
		at := startedAnnounces(n.world, f.hash, received)
		if len(at) != 1 {
			t.Fatalf("after the restart the torrent announced started at %v, want once", at)
		}
		want := &dlv1.Arrival{AskedUnixMs: asked, Result: dlv1.ArrivalResult_ARRIVAL_RESULT_STARTED, AnnouncedUnixMs: at[0].UnixMilli()}
		if got := arrivalOf(t, c, id); !proto.Equal(got, want) {
			t.Errorf("arrival after the restart %v, want %v", got, want)
		}
		time.Sleep(fresh + time.Minute)
		synctest.Wait()

		var since []time.Time
		for _, s := range stoppedAt(n.world, f) {
			if !s.Before(received) {
				since = append(since, s)
			}
		}
		if len(since) != 1 || since[0].Sub(received) != fresh {
			t.Errorf("after the restart the torrent announced stopped at %v, want once at the fresh window's end %v", since, received.Add(fresh))
		}
	})
}
