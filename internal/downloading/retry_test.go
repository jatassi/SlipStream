package downloading_test

import (
	"testing"
	"testing/synctest"
	"time"

	dlv1 "github.com/jatassi/slipstream/internal/pb/slipstream/downloading/v1"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/torrent/engine"
	"github.com/jatassi/slipstream/internal/torrent/engine/enginetest"
)

var readError = &engine.Error{Category: engine.CategorySystem, Code: 2, Text: "No such file or directory"}

// retried is f paused with the Admin's Retry at, as the Controller's
// intent carries a pending Retry.
func (f file) retried(id storage.UUID, at time.Time) *dlv1.Torrent {
	it := f.paused(id, 1)
	it.RetryAtUnixMs = at.UnixMilli()
	return it
}

// failedSeed has a client seed f and the engine fail it reading its file.
func failedSeed(t *testing.T, n *node, f file) (*client, storage.UUID) {
	t.Helper()
	n.world.Payload(f.hash, f.payload)
	id := n.hold(f)
	c := n.start()
	c.send(live(f.started(id, 1)))
	n.world.Fail(f.hash, readError)
	synctest.Wait()
	if r := c.h.latest(t, id); r.GetEngine() != dlv1.EngineState_ENGINE_STATE_FAILED {
		t.Fatalf("the seed the engine failed reports %v, want failed", r)
	}
	return c, id
}

func TestAFailedTorrentStaysFailedUntilTheIntentCarriesARetry(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		n := newNode(t, enginetest.TrackerRules{Interval: 30 * time.Minute})
		f := newFile(t, "Orchard.mkv")
		c, id := failedSeed(t, n, f)
		c.stop()
		time.Sleep(time.Hour)

		c = n.start()
		c.send(live(f.paused(id, 1)))
		c.send(live(f.started(id, 1)))
		time.Sleep(timings.StartSpread)
		synctest.Wait()
		if r := c.h.latest(t, id); r.GetEngine() != dlv1.EngineState_ENGINE_STATE_FAILED || r.GetError() != readError.Text {
			t.Fatalf("restarted, paused and resumed with no Retry, the failed torrent reports %v, want still failed with %q", r, readError.Text)
		}
		if got := c.engine.count(); got != 0 {
			t.Fatalf("the restarted client checked the failed torrent %d times, want never before a Retry", got)
		}

		at := time.Now()
		c.send(live(f.retried(id, at)))
		r := c.h.latest(t, id)
		if r.GetEngine() != dlv1.EngineState_ENGINE_STATE_SEEDING || !r.GetComplete() || r.GetHold() != dlv1.Hold_HOLD_WANTED ||
			r.GetRetryCheckedUnixMs() != at.UnixMilli() || r.GetDone() != size || r.GetError() != "" {
			t.Fatalf("with every byte on disk, the retried torrent reports %v, want complete, held paused and its Retry checked with %d bytes found", r, size)
		}

		c.send(live(f.started(id, 1)))
		time.Sleep(timings.StartSpread)
		synctest.Wait()
		if r := c.h.latest(t, id); r.GetEngine() != dlv1.EngineState_ENGINE_STATE_SEEDING || r.GetHold() != dlv1.Hold_HOLD_UNSPECIFIED {
			t.Fatalf("resumed after its Retry, the torrent reports %v, want seeding", r)
		}
		c.stop()
		c = n.start()
		c.send(live(f.started(id, 1)))
		time.Sleep(timings.StartSpread)
		synctest.Wait()
		if r := c.h.latest(t, id); r.GetEngine() != dlv1.EngineState_ENGINE_STATE_SEEDING {
			t.Fatalf("restarted after a Retry cleared its failure, the torrent reports %v, want seeding", r)
		}
	})
}

func TestARetryFindsWhatIsLeftAndStaysPaused(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		n := newNode(t, enginetest.TrackerRules{Interval: 30 * time.Minute})
		f := newFile(t, "Orchard.mkv")
		c, id := failedSeed(t, n, f)
		if err := n.world.Lose(f.hash, size-pieceLength); err != nil {
			t.Fatal(err)
		}

		at := time.Now()
		c.send(live(f.retried(id, at)))
		r := c.h.latest(t, id)
		if r.GetEngine() != dlv1.EngineState_ENGINE_STATE_DOWNLOADING || r.GetComplete() || r.GetHold() != dlv1.Hold_HOLD_WANTED ||
			r.GetRetryCheckedUnixMs() != at.UnixMilli() || r.GetDone() != pieceLength {
			t.Fatalf("with one piece left on disk, the retried torrent reports %v, want incomplete, held paused and its Retry checked with %d bytes found", r, pieceLength)
		}
	})
}

func TestARetryTheNodeRestartsInTheMiddleOfChecksAgain(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		n := newNode(t, enginetest.TrackerRules{Interval: 30 * time.Minute})
		f := newFile(t, "Orchard.mkv")
		c, id := failedSeed(t, n, f)

		at := time.Now()
		c.engine.holdChecks()
		c.send(live(f.retried(id, at)))
		if r := c.h.latest(t, id); r.GetEngine() != dlv1.EngineState_ENGINE_STATE_CHECKING || r.GetRetryCheckedUnixMs() != 0 {
			t.Fatalf("while its Retry's check runs, the torrent reports %v, want checking and the Retry not yet checked", r)
		}
		c.stop()

		c = n.start()
		c.send(live(f.retried(id, at)))
		if got := c.engine.count(); got != 1 {
			t.Fatalf("restarted while its Retry's check ran, the client checked %d times, want the check again", got)
		}
		if r := c.h.latest(t, id); r.GetEngine() != dlv1.EngineState_ENGINE_STATE_SEEDING || r.GetRetryCheckedUnixMs() != at.UnixMilli() || r.GetDone() != size {
			t.Fatalf("once the repeated check ends, the torrent reports %v, want seeding and its Retry checked with %d bytes found", r, size)
		}
		c.send(live(f.retried(id, at)))
		if got := c.engine.count(); got != 1 {
			t.Fatalf("the same Retry again after its check, the client checked %d times, want no second check", got)
		}
	})
}
