package downloading_test

import (
	"testing"
	"testing/synctest"
	"time"

	"google.golang.org/protobuf/proto"

	dlv1 "github.com/jatassi/slipstream/internal/pb/slipstream/downloading/v1"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/torrent/engine"
	"github.com/jatassi/slipstream/internal/torrent/engine/enginetest"
)

// faultOf is the one fault torrent id reported, failing the test when its
// reports carried more than one or dropped it once carried.
func (c *client) faultOf(t *testing.T, id storage.UUID) *dlv1.Fault {
	t.Helper()
	var fault *dlv1.Fault
	for _, r := range c.h.reportsOf(id) {
		switch f := r.GetFault(); {
		case fault == nil:
			fault = f
		case !proto.Equal(f, fault):
			t.Fatalf("torrent %s reported fault %v after %v, want the first to stay", id, f, fault)
		}
	}
	return fault
}

func (c *client) after(d time.Duration) {
	time.Sleep(d)
	synctest.Wait()
}

func TestADownloadThatRunsTheFaultStallWithNoPayloadIsAFault(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		const stall = time.Hour
		n := newNode(t, enginetest.TrackerRules{Interval: 30 * time.Minute})
		running, paused := newFile(t, "Orchard.mkv"), newFile(t, "Meadow.mkv")
		runningID, pausedID := n.hold(running), n.hold(paused)
		c := n.start()
		s := live(running.started(runningID, 1), paused.paused(pausedID, 2))
		s.Intent.Thresholds = &dlv1.Thresholds{FaultStallSeconds: int64(stall / time.Second), UnregisteredGapSeconds: 1800}
		start := time.Now()
		c.send(s)

		c.after(stall - time.Minute)
		if f := c.faultOf(t, runningID); f != nil {
			t.Fatalf("a download that ran %v with no payload reports fault %v, want none before %v", time.Since(start), f, stall)
		}
		c.after(2 * time.Minute)
		f := c.faultOf(t, runningID)
		if f.GetKind() != dlv1.FaultKind_FAULT_KIND_NO_PAYLOAD || f.GetAtUnixMs() < start.Add(stall).UnixMilli() || f.GetAtUnixMs() > start.Add(stall+2*timings.Facts).UnixMilli() {
			t.Fatalf("a download that ran %v with no payload reports fault %v, want no payload from %v", time.Since(start), f, start.Add(stall))
		}
		if f := c.faultOf(t, pausedID); f != nil {
			t.Errorf("a paused download reports fault %v, want none: it never ran", f)
		}
	})
}

func TestATrackerCallingATorrentUnregisteredTwiceIsAFault(t *testing.T) {
	for reason, wantFault := range map[string]bool{"Unregistered torrent": true, "Too many requests": false} {
		t.Run(reason, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				n := newNode(t, enginetest.TrackerRules{Failure: reason})
				f := newFile(t, "Orchard.mkv")
				n.world.Payload(f.hash, f.payload)
				id := n.hold(f)
				c := n.start()
				s := live(f.started(id, 1))
				s.Intent.Thresholds = &dlv1.Thresholds{FaultStallSeconds: 12 * 3600, UnregisteredGapSeconds: 20 * 60}
				c.send(s)
				if fault := c.faultOf(t, id); fault != nil {
					t.Fatalf("after one answer %q the torrent reports fault %v, want none", reason, fault)
				}

				c.after(31 * time.Minute)
				var second time.Time
				for _, a := range n.world.Announces(trackerURL) {
					second = a.At
				}
				fault := c.faultOf(t, id)
				want := &dlv1.Fault{Kind: dlv1.FaultKind_FAULT_KIND_UNREGISTERED, AtUnixMs: second.UnixMilli()}
				if !wantFault {
					want = nil
				}
				if !proto.Equal(fault, want) {
					t.Fatalf("after two answers %q 30 minutes apart the torrent reports fault %v, want %v", reason, fault, want)
				}
			})
		})
	}
}

func TestAnEngineErrorBlamedOnTheTorrentIsAFault(t *testing.T) {
	for name, c := range map[string]struct {
		err       *engine.Error
		wantFault bool
	}{
		"a libtorrent error":         {&engine.Error{Category: engine.CategoryLibtorrent, Code: 23, Text: "invalid torrent file"}, true},
		"a system error on the Node": {&engine.Error{Category: engine.CategorySystem, Code: 13, Text: "permission denied"}, false},
	} {
		t.Run(name, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				n := newNode(t, enginetest.TrackerRules{Interval: 30 * time.Minute})
				f := newFile(t, "Orchard.mkv")
				id := n.hold(f)
				cl := n.start()
				cl.send(live(f.started(id, 1)))

				failedAt := time.Now()
				n.world.Fail(f.hash, c.err)
				synctest.Wait()
				if r := cl.h.latest(t, id); r.GetEngine() != dlv1.EngineState_ENGINE_STATE_FAILED || r.GetError() != c.err.Text {
					t.Fatalf("the failed torrent reports %v, want failed with %q", r, c.err.Text)
				}
				want := &dlv1.Fault{Kind: dlv1.FaultKind_FAULT_KIND_ENGINE_ERROR, AtUnixMs: failedAt.UnixMilli()}
				if !c.wantFault {
					want = nil
				}
				if got := cl.faultOf(t, id); !proto.Equal(got, want) {
					t.Fatalf("a torrent failed by %s reports fault %v, want %v", name, got, want)
				}
			})
		})
	}
}
