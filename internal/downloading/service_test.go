package downloading_test

import (
	"os"
	"path/filepath"
	"slices"
	"testing"
	"testing/synctest"
	"time"

	"google.golang.org/protobuf/proto"

	dlv1 "github.com/jatassi/slipstream/internal/pb/slipstream/downloading/v1"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/torrent/engine/enginetest"
)

// window is the private tracker's min interval in these tests.
const window = 30 * time.Minute

func (c *client) act(t *testing.T, act dlv1.Act, id storage.UUID) *dlv1.ActResult {
	t.Helper()
	res, err := c.Act(t.Context(), &dlv1.ActRequest{Act: act, Ids: [][]byte{id.Bytes()}})
	if err != nil {
		t.Fatal(err)
	}
	synctest.Wait()
	if len(res.GetResults()) != 1 {
		t.Fatalf("Act answered %v, want one result", res)
	}
	return res.GetResults()[0]
}

func (c *client) frees(t *testing.T, id storage.UUID) *dlv1.Freed {
	t.Helper()
	res, err := c.Frees(t.Context(), &dlv1.FreesRequest{Ids: [][]byte{id.Bytes()}})
	if err != nil {
		t.Fatal(err)
	}
	return res.GetFreed()[0]
}

func announcesOf(w *enginetest.World, f file, event string, since time.Time) int {
	n := 0
	for _, a := range w.Announces(trackerURL) {
		if a.InfoHash == f.hash && a.Event == event && !a.At.Before(since) {
			n++
		}
	}
	return n
}

// TestInspectShowsTheFilesAndTheTrackersAsThePolicySeesThem seeds a
// private torrent: Inspect reads its one file done, and its tracker
// working, answered, and closed to announces until its window ends.
func TestInspectShowsTheFilesAndTheTrackersAsThePolicySeesThem(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		n := newNode(t, enginetest.TrackerRules{Interval: window, MinInterval: window})
		f := newPrivateFile(t, "Orchard.mkv")
		n.world.Payload(f.hash, f.payload)
		id := n.hold(f)
		start := time.Now()
		c := n.start()
		c.send(live(f.started(id, 1)))

		got, err := c.Inspect(t.Context(), &dlv1.InspectRequest{Id: id.Bytes()})
		if err != nil {
			t.Fatal(err)
		}
		want := &dlv1.InspectResponse{
			Files: []*dlv1.FileProgress{{Index: 0, Done: size}},
			Trackers: []*dlv1.TrackerDetail{{
				Index: 0, Status: dlv1.TrackerStatus_TRACKER_STATUS_WORKING, AnsweredAtUnixMs: start.UnixMilli(),
				NextAnnounceUnixMs: got.GetTrackers()[0].GetNextAnnounceUnixMs(), MinIntervalSeconds: int64(window / time.Second),
				AcceptsAtUnixMs: start.Add(window).UnixMilli(),
			}},
			AnnounceAtUnixMs: start.Add(window).UnixMilli(),
		}
		if !proto.Equal(got, want) {
			t.Fatalf("Inspect = %v, want %v", got, want)
		}
		if next := time.UnixMilli(got.GetTrackers()[0].GetNextAnnounceUnixMs()); next.Before(start.Add(window)) || next.After(start.Add(window+timings.AnnounceJitter)) {
			t.Errorf("the next announce is due at %v, want the interval and some jitter after %v", next, start)
		}
		if _, err := c.Inspect(t.Context(), &dlv1.InspectRequest{Id: storage.NewUUID().Bytes()}); err == nil {
			t.Errorf("Inspect of a torrent the Node doesn't hold answered, want NotFound")
		}
	})
}

// TestAnnounceNowWaitsForEveryTrackerToAcceptIt asks a private torrent to
// announce right after it started: the client refuses until the window
// ends, says when, and then announces at once.
func TestAnnounceNowWaitsForEveryTrackerToAcceptIt(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		n := newNode(t, enginetest.TrackerRules{Interval: 2 * window, MinInterval: window})
		f := newPrivateFile(t, "Orchard.mkv")
		n.world.Payload(f.hash, f.payload)
		id := n.hold(f)
		start := time.Now()
		c := n.start()
		c.send(live(f.started(id, 1)))

		refused := c.act(t, dlv1.Act_ACT_ANNOUNCE, id)
		if want := (&dlv1.ActResult{Id: id.Bytes(), Refused: dlv1.Refusal_REFUSAL_WINDOW, AcceptsAtUnixMs: start.Add(window).UnixMilli()}); !proto.Equal(refused, want) {
			t.Fatalf("announcing inside the window = %v, want %v", refused, want)
		}
		time.Sleep(window)
		synctest.Wait()
		asked := time.Now()
		if done := c.act(t, dlv1.Act_ACT_ANNOUNCE, id); done.GetRefused() != dlv1.Refusal_REFUSAL_UNSPECIFIED {
			t.Fatalf("announcing once the window ended = %v, want it done", done)
		}
		time.Sleep(2 * time.Second)
		synctest.Wait()
		if got := announcesOf(n.world, f, "", asked); got != 1 {
			t.Fatalf("the tracker heard %d regular announces once asked, want 1 at once", got)
		}

		c.send(live(f.paused(id, 1)))
		if paused := c.act(t, dlv1.Act_ACT_ANNOUNCE, id); paused.GetRefused() != dlv1.Refusal_REFUSAL_NOT_RUNNING {
			t.Errorf("announcing a paused torrent = %v, want refused as not running", paused)
		}
		if unknown := c.act(t, dlv1.Act_ACT_ANNOUNCE, storage.NewUUID()); unknown.GetRefused() != dlv1.Refusal_REFUSAL_NOT_HELD {
			t.Errorf("announcing a torrent the Node doesn't hold = %v, want refused as not held", unknown)
		}
	})
}

// TestARecheckPausesChecksAndStartsThroughThePolicy rechecks a private
// torrent seeding a file that was damaged by hand: the client stops it,
// checks it without announcing, finds it incomplete, and starts it again
// once the tracker's window ends.
func TestARecheckPausesChecksAndStartsThroughThePolicy(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		n := newNode(t, enginetest.TrackerRules{Interval: 2 * window, MinInterval: window})
		f := newPrivateFile(t, "Orchard.mkv")
		n.world.Payload(f.hash, f.payload)
		id := n.hold(f)
		start := time.Now()
		c := n.start()
		c.send(live(f.started(id, 1)))
		path, err := c.Path(f.hash, 0)
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, make([]byte, size), 0o600); err != nil {
			t.Fatal(err)
		}
		n.world.Transfer(f.hash, 0, 0)
		checks := c.engine.count()

		if r := c.act(t, dlv1.Act_ACT_RECHECK, id); r.GetRefused() != dlv1.Refusal_REFUSAL_UNSPECIFIED {
			t.Fatalf("Recheck = %v, want it done", r)
		}
		if got := c.engine.count() - checks; got != 1 {
			t.Fatalf("the client checked %d times, want once", got)
		}
		if got := announcesOf(n.world, f, "stopped", start); got != 1 {
			t.Fatalf("the tracker heard %d stopped, want the pause's one", got)
		}
		last := c.h.latest(t, id)
		if last.GetComplete() || last.GetHold() != dlv1.Hold_HOLD_WINDOW || last.GetUntilUnixMs() != start.Add(window).UnixMilli() {
			t.Fatalf("checked, the torrent reports %v; want incomplete and held for the window until %v", last, start.Add(window))
		}
		if _, err := c.Path(f.hash, 0); err == nil {
			t.Errorf("Path of a torrent found incomplete answered, want ErrNotPresent")
		}

		time.Sleep(window)
		synctest.Wait()
		if got := announcesOf(n.world, f, "started", start); got != 2 {
			t.Fatalf("the tracker heard %d started, want a second once the window ended", got)
		}
		if r := c.h.latest(t, id); r.GetHold() != dlv1.Hold_HOLD_UNSPECIFIED || r.GetEngine() != dlv1.EngineState_ENGINE_STATE_DOWNLOADING {
			t.Errorf("once started again the torrent reports %v, want downloading", r)
		}
	})
}

// TestFreesCountsOnlyFilesWithNoOtherLink seeds a torrent, then links its
// file into a library: deleting the download copy frees its bytes before,
// nothing after, and the removal frees what Frees said.
func TestFreesCountsOnlyFilesWithNoOtherLink(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		n := newNode(t, enginetest.TrackerRules{Interval: window})
		f := newFile(t, "Orchard.mkv")
		n.world.Payload(f.hash, f.payload)
		id := n.hold(f)
		c := n.start()
		c.send(live(f.started(id, 1)))

		if got, want := c.frees(t, id), (&dlv1.Freed{Id: id.Bytes(), Bytes: size}); !proto.Equal(got, want) {
			t.Fatalf("Frees of a download only the client holds = %v, want %v", got, want)
		}
		path, err := c.Path(f.hash, 0)
		if err != nil {
			t.Fatal(err)
		}
		library := filepath.Join(t.TempDir(), f.name)
		if err := os.Link(path, library); err != nil {
			t.Fatal(err)
		}
		if got, want := c.frees(t, id), (&dlv1.Freed{Id: id.Bytes(), Linked: true}); !proto.Equal(got, want) {
			t.Fatalf("Frees once the library links the file = %v, want %v", got, want)
		}

		discarding := live()
		discarding.Intent.Discards = []*dlv1.Discard{{Id: id.Bytes(), InfoHash: f.hash[:], DeleteFiles: true}}
		c.send(discarding)
		if got := c.h.parted(); !slices.ContainsFunc(got, func(p *dlv1.Parted) bool { return proto.Equal(p, &dlv1.Parted{Id: id.Bytes()}) }) {
			t.Fatalf("confirmed %v, want the removal freeing nothing", got)
		}
		if data, err := os.ReadFile(library); err != nil || len(data) != size {
			t.Errorf("the library's file reads %d bytes, %v; want it whole", len(data), err)
		}
	})
}
