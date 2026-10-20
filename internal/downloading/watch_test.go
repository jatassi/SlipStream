package downloading_test

import (
	"bytes"
	"testing"
	"testing/synctest"
	"time"

	"google.golang.org/protobuf/proto"

	dlv1 "github.com/jatassi/slipstream/internal/pb/slipstream/downloading/v1"
	"github.com/jatassi/slipstream/internal/torrent/engine/enginetest"
)

func (c *client) watch(t *testing.T, after uint64) *dlv1.WatchResponse {
	t.Helper()
	res, err := c.Watch(t.Context(), &dlv1.WatchRequest{After: after})
	if err != nil {
		t.Fatal(err)
	}
	return res
}

// TestWatchAnswersEachSampleWithWhatMoved downloads one torrent at 1000
// bytes a second beside a paused one. Each Watch answers one live sample
// later: first with the download alone, as the paused torrent doesn't
// move, then with whatever moved since the last answer, and with the
// torrents the Node let go.
func TestWatchAnswersEachSampleWithWhatMoved(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		n := newNode(t, enginetest.TrackerRules{Interval: 30 * time.Minute})
		down, still := newFile(t, "Orchard.mkv"), newFile(t, "Meadow.mkv")
		for _, f := range []file{down, still} {
			n.world.Payload(f.hash, f.payload)
		}
		n.world.Transfer(down.hash, 1000, 0)
		downID, stillID := n.hold(down), n.hold(still)
		c := n.start()
		c.send(live(down.started(downID, 1), still.paused(stillID, 2)))
		time.Sleep(5 * time.Second)
		synctest.Wait()

		start := time.Now()
		first := c.watch(t, 0)
		if waited := time.Since(start); waited != timings.LiveSample {
			t.Errorf("the first Watch answered after %v, want one live sample, %v", waited, timings.LiveSample)
		}
		if len(first.GetRows()) != 1 || !proto.Equal(first.GetRows()[0], &dlv1.Live{Id: downID.Bytes(), DownBps: 1000, Done: first.GetRows()[0].GetDone(), Wanted: size}) ||
			first.GetRows()[0].GetDone() == 0 || first.GetDownBps() != 1000 || first.GetUpBps() != 0 {
			t.Fatalf("the first Watch answered %v, want the download alone at 1000 bytes a second, and the Node downloading at that rate", first)
		}

		second := c.watch(t, first.GetSample())
		if second.GetSample() != first.GetSample()+1 || len(second.GetRows()) != 1 || second.GetRows()[0].GetDone() != first.GetRows()[0].GetDone()+1000 {
			t.Fatalf("the next Watch answered %v after %v, want the next sample with the download 1000 bytes further", second, first)
		}

		n.world.Transfer(down.hash, 0, 0)
		stalled := c.watch(t, second.GetSample())
		if len(stalled.GetRows()) != 1 || stalled.GetRows()[0].GetDownBps() != 0 || stalled.GetDownBps() != 0 {
			t.Fatalf("once the swarm stopped sending, Watch answered %v, want the download at rest", stalled)
		}
		quiet := c.watch(t, stalled.GetSample())
		if len(quiet.GetRows()) != 0 || len(quiet.GetGone()) != 0 || quiet.GetSample() != stalled.GetSample()+1 {
			t.Fatalf("with nothing moving Watch answered %v, want the next sample and no rows", quiet)
		}

		discarding := live(still.paused(stillID, 2))
		discarding.Intent.Discards = []*dlv1.Discard{{Id: downID.Bytes(), InfoHash: down.hash[:]}}
		c.send(discarding)
		gone := c.watch(t, quiet.GetSample())
		if len(gone.GetRows()) != 0 || len(gone.GetGone()) != 1 || !bytes.Equal(gone.GetGone()[0], downID.Bytes()) {
			t.Fatalf("once the download was discarded Watch answered %v, want it gone", gone)
		}

		n.world.Transfer(still.hash, 0, 500)
		c.send(live(still.started(stillID, 2)))
		afresh := c.watch(t, 1)
		if afresh.GetSample() == 2 || len(afresh.GetRows()) != 1 || afresh.GetRows()[0].GetUpBps() != 500 || len(afresh.GetGone()) != 0 {
			t.Fatalf("a Watch after a sample the Node never answered got %v, want a fresh start with the one torrent moving", afresh)
		}
	})
}
