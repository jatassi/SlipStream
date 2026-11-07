package downloading_test

import (
	"bytes"
	"testing"
	"time"

	dlv1 "github.com/jatassi/slipstream/internal/pb/slipstream/downloading/v1"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/torrent/engine/enginetest"
)

func (c *composite) watch(t *testing.T, after uint64) *dlv1.WatchResponse {
	t.Helper()
	res, err := c.Watch(t.Context(), &dlv1.WatchRequest{After: after})
	if err != nil {
		t.Fatal(err)
	}
	return res
}

func rowOf(res *dlv1.WatchResponse, id storage.UUID) *dlv1.Live {
	for _, r := range res.GetRows() {
		if bytes.Equal(r.GetId(), id.Bytes()) {
			return r
		}
	}
	return nil
}

// TestWatchMergesTheExternalRowsAndNumbersItsOwnSamples downloads one
// torrent in the built-in client at 1000 bytes a second and one in the fake
// Transmission at 100. Each answer carries both clients' rows and rates
// under the composite's own numbering, which only ever grows: an after it
// never answered starts afresh rather than over, and a torrent that left
// the intent is gone.
func TestWatchMergesTheExternalRowsAndNumbersItsOwnSamples(t *testing.T) {
	inEachTransmission(t, func(t *testing.T, version string) {
		n := newNode(t, enginetest.TrackerRules{Interval: 30 * time.Minute})
		f := newFile(t, "Meadow.mkv")
		n.world.Payload(f.hash, f.payload)
		n.world.Transfer(f.hash, 1000, 0)
		mine := n.hold(f)
		c := n.startComposite(version, 100_000)
		theirs := c.rels["orchard"].id
		c.send(true, f.started(mine, 1), c.orchard(2))
		time.Sleep(15 * time.Second)

		first := c.watch(t, 0)
		if rowOf(first, mine).GetDownBps() != 1000 || rowOf(first, theirs).GetDownBps() != 100 || first.GetDownBps() != 1100 {
			t.Fatalf("the first Watch answered %v, want both downloads and the Node at 1100 bytes a second", first)
		}

		next := c.watch(t, first.GetSample())
		if next.GetSample() != first.GetSample()+1 {
			t.Errorf("the next Watch answered sample %d after %d, want the next", next.GetSample(), first.GetSample())
		}
		time.Sleep(10 * time.Second)
		moved := c.watch(t, next.GetSample())
		if was, now := rowOf(first, theirs).GetDone(), rowOf(moved, theirs).GetDone(); now <= was {
			t.Errorf("orchard's row moved from %d done to %d, want it further", was, now)
		}

		afresh := c.watch(t, 1)
		if afresh.GetSample() != moved.GetSample()+1 || rowOf(afresh, mine) == nil || rowOf(afresh, theirs) == nil {
			t.Errorf("a Watch after a sample the Node never answered got %v, want the next sample with every torrent moving", afresh)
		}

		c.send(true, f.started(mine, 1))
		gone := c.watch(t, afresh.GetSample())
		if len(gone.GetGone()) != 1 || !bytes.Equal(gone.GetGone()[0], theirs.Bytes()) || rowOf(gone, theirs) != nil {
			t.Errorf("once orchard left the intent Watch answered %v, want it gone", gone)
		}
	})
}
