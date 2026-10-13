package rpc_test

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"slices"
	"sync/atomic"
	"testing"
	"testing/synctest"

	"google.golang.org/protobuf/proto"

	"github.com/jatassi/slipstream/fakes/fault"
	"github.com/jatassi/slipstream/internal/link"
	"github.com/jatassi/slipstream/internal/link/linktest"
	rpctestv1 "github.com/jatassi/slipstream/internal/pb/slipstream/rpctest/v1"
	"github.com/jatassi/slipstream/internal/pb/slipstream/rpctest/v1/rpctestv1connect"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/rpc"
	"github.com/jatassi/slipstream/internal/storage"
)

var errSentinel = errors.New("the handler's own error")

// recorder records who called, mutates the request it was handed, keeps
// the response it returned, and fails on demand.
type recorder struct {
	calls atomic.Int32
	last  *rpctestv1.RecordResponse
	// cluster and offline are what a forwarding recorder calls: a Node in
	// the roster with no session.
	cluster *rpc.Cluster
	offline role.NodeID
}

func (r *recorder) Record(ctx context.Context, req *rpctestv1.RecordRequest) (*rpctestv1.RecordResponse, error) {
	r.calls.Add(1)
	caller, _ := rpc.CallerOf(ctx)
	text := req.GetText()
	req.Text = "mutated by the handler"
	switch req.GetFail() {
	case rpctestv1.Fail_FAIL_UNSPECIFIED:
	case rpctestv1.Fail_FAIL_CODED:
		return nil, fmt.Errorf("%w: %w", errSentinel, rpc.Errorf(rpc.CodeFailedPrecondition, "x"))
	case rpctestv1.Fail_FAIL_FORWARD:
		_, err := rpctestv1connect.PingService.On(r.cluster, r.offline).Ping(ctx, &rpctestv1.PingRequest{})
		return nil, err
	}
	r.last = &rpctestv1.RecordResponse{Text: text, Caller: caller[:]}
	return r.last, nil
}

// pinger answers with its caller's key once release allows it.
type pinger struct {
	calls   atomic.Int32
	release chan struct{}
}

func (p *pinger) Ping(ctx context.Context, _ *rpctestv1.PingRequest) (*rpctestv1.PingResponse, error) {
	p.calls.Add(1)
	if p.release != nil {
		<-p.release
	}
	caller, _ := rpc.CallerOf(ctx)
	return &rpctestv1.PingResponse{Caller: caller[:]}, nil
}

type searcher struct{ calls atomic.Int32 }

func (s *searcher) Search(ctx context.Context, _ *rpctestv1.SearchRequest) (*rpctestv1.SearchResponse, error) {
	s.calls.Add(1)
	caller, _ := rpc.CallerOf(ctx)
	return &rpctestv1.SearchResponse{Caller: caller[:]}, nil
}

type roster struct {
	keys    map[role.NodeID]link.Key
	holders map[role.Name][]role.NodeID
}

func (r roster) Key(node role.NodeID) (link.Key, bool) {
	k, ok := r.keys[node]
	return k, ok
}

func (r roster) Holders(n role.Name) []role.NodeID { return r.holders[n] }

// cluster is a Controller and a Worker over linktest, each with every test
// service mounted.
type cluster struct {
	*linktest.Pair
	controller, worker     *rpc.Dispatch
	cluster                *rpc.Cluster
	controllerID, workerID role.NodeID
	roster                 roster
	recorder               *recorder
	pingers                map[role.NodeID]*pinger
	searchers              map[role.NodeID]*searcher
	workerHolds            atomic.Bool
}

func newCluster(t *testing.T, controllerVersion, workerVersion string) *cluster {
	t.Helper()
	cm, wm := http.NewServeMux(), http.NewServeMux()
	c := &cluster{
		Pair:         linktest.New(t, linktest.Side{Version: controllerVersion, Lockstep: cm}, linktest.Side{Version: workerVersion, Lockstep: wm}),
		controllerID: role.NodeID{UUID: storage.NewUUID()},
		workerID:     role.NodeID{UUID: storage.NewUUID()},
	}
	offline := role.NodeID{UUID: storage.NewUUID()}
	c.workerHolds.Store(true)
	c.roster = roster{
		keys:    map[role.NodeID]link.Key{c.controllerID: c.Controller.Key, c.workerID: c.Worker.Key, offline: {1}},
		holders: map[role.Name][]role.NodeID{role.Searching: {c.controllerID, c.workerID}},
	}
	c.controller = rpc.NewDispatch(c.Controller.Link, c.Controller.Key, c.Controller.Key, func(role.Name) bool { return true })
	c.worker = rpc.NewDispatch(c.Worker.Link, c.Worker.Key, c.Controller.Key, func(role.Name) bool { return c.workerHolds.Load() })
	cm.Handle("/", c.controller)
	wm.Handle("/", c.worker)
	c.cluster = rpc.NewCluster(c.controller, c.roster)
	c.recorder = &recorder{cluster: c.cluster, offline: offline}

	rpctestv1connect.RecordService.Mount(c.cluster, c.recorder)
	c.pingers = map[role.NodeID]*pinger{c.controllerID: {}, c.workerID: {}}
	c.searchers = map[role.NodeID]*searcher{c.controllerID: {}, c.workerID: {}}
	for id, d := range map[role.NodeID]*rpc.Dispatch{c.controllerID: c.controller, c.workerID: c.worker} {
		rpctestv1connect.PingService.Mount(d, c.pingers[id])
		rpctestv1connect.SearchService.Mount(d, c.searchers[id])
	}
	return c
}

func TestCallsLookTheSameInProcessAndOverTheLink(t *testing.T) {
	for _, row := range []struct {
		name   string
		caller func(*cluster) (*rpc.Dispatch, link.Key)
	}{
		{"the Controller calls itself", func(c *cluster) (*rpc.Dispatch, link.Key) { return c.controller, c.Controller.Key }},
		{"a Worker calls the Controller", func(c *cluster) (*rpc.Dispatch, link.Key) { return c.worker, c.Worker.Key }},
	} {
		t.Run(row.name, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				c := newCluster(t, "", "")
				c.Connect(t)
				d, key := row.caller(c)
				svc := rpctestv1connect.RecordService.Call(d)
				checkCopies(t, c.recorder, svc, key)
				checkErrors(t, svc)
			})
		})
	}
}

// checkCopies calls svc and checks neither side sees the other's later
// changes to the messages.
func checkCopies(t *testing.T, r *recorder, svc rpctestv1connect.RecordServiceHandler, caller link.Key) {
	t.Helper()
	req := &rpctestv1.RecordRequest{Text: "hello"}
	res, err := svc.Record(t.Context(), req)
	if err != nil {
		t.Fatal(err)
	}
	if want := (&rpctestv1.RecordResponse{Text: "hello", Caller: caller[:]}); !proto.Equal(res, want) {
		t.Errorf("response = %v, want %v", res, want)
	}
	if req.GetText() != "hello" {
		t.Errorf("the caller's request reads %q after the call", req.GetText())
	}
	r.last.Text = "changed by the handler later"
	if res.GetText() != "hello" {
		t.Errorf("the caller's response reads %q after the handler changed its own", res.GetText())
	}
}

// checkErrors checks a handler's error reaches the caller as its code and
// message only.
func checkErrors(t *testing.T, svc rpctestv1connect.RecordServiceHandler) {
	t.Helper()
	_, err := svc.Record(t.Context(), &rpctestv1.RecordRequest{Fail: rpctestv1.Fail_FAIL_CODED})
	if rpc.CodeOf(err) != rpc.CodeFailedPrecondition || err.Error() != "failed_precondition: x" {
		t.Errorf("the handler's error arrived as %v (code %v)", err, rpc.CodeOf(err))
	}
	if errors.Is(err, errSentinel) {
		t.Error("the handler's sentinel reached the caller")
	}
	if why, ok := rpc.NotSent(err); ok || rpc.Lost(err) {
		t.Errorf("a call that ran reads as not sent (%v, %v) or lost (%v)", why, ok, rpc.Lost(err))
	}

	_, err = svc.Record(t.Context(), &rpctestv1.RecordRequest{Fail: rpctestv1.Fail_FAIL_FORWARD})
	if why, ok := rpc.NotSent(err); ok || !rpc.Lost(err) || err.Error() != "unavailable: no live link session to the peer" {
		t.Errorf("a handler passing on a call that never ran = %v, NotSent %v %v; want Lost", err, why, ok)
	}
}

func TestTheControllerCallsNodesItPicks(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		c := newCluster(t, "", "")
		c.Connect(t)
		for _, node := range []role.NodeID{c.controllerID, c.workerID} {
			res, err := rpctestv1connect.PingService.On(c.cluster, node).Ping(t.Context(), &rpctestv1.PingRequest{})
			if err != nil || link.Key(res.GetCaller()) != c.Controller.Key {
				t.Errorf("Ping on %v = %v, %v; want the Controller as caller", node, res, err)
			}
			if got := c.pingers[node].calls.Load(); got != 1 {
				t.Errorf("node %v's pinger ran %d times, want 1", node, got)
			}
		}
		if got := rpctestv1connect.SearchService.Holders(c.cluster); !slices.Equal(got, []role.NodeID{c.controllerID, c.workerID}) {
			t.Errorf("Holders = %v", got)
		}
		res, err := rpctestv1connect.SearchService.On(c.cluster, c.workerID).Search(t.Context(), &rpctestv1.SearchRequest{})
		if err != nil || link.Key(res.GetCaller()) != c.Controller.Key || c.searchers[c.workerID].calls.Load() != 1 {
			t.Errorf("Search on the Worker = %v, %v", res, err)
		}
	})
}

func TestCallsThatNeverRan(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		c := newCluster(t, "", "")
		_, err := rpctestv1connect.RecordService.Call(c.worker).Record(t.Context(), &rpctestv1.RecordRequest{})
		if why, ok := rpc.NotSent(err); why != rpc.Disconnected || !ok || err.Error() != "unavailable: no live link session to the peer" {
			t.Errorf("a call with no session = %v, NotSent %v %v", err, why, ok)
		}

		c.Connect(t)
		c.roster.holders[role.Searching] = []role.NodeID{c.controllerID}
		_, err = rpctestv1connect.SearchService.On(c.cluster, c.workerID).Search(t.Context(), &rpctestv1.SearchRequest{})
		want := "failed_precondition: node " + c.workerID.String() + " does not hold the searching Role"
		if why, ok := rpc.NotSent(err); why != rpc.NotHolder || !ok || err.Error() != want {
			t.Errorf("a Role call the roster refuses = %v, NotSent %v %v", err, why, ok)
		}

		c.roster.holders[role.Searching] = []role.NodeID{c.controllerID, c.workerID}
		c.workerHolds.Store(false)
		_, err = rpctestv1connect.SearchService.On(c.cluster, c.workerID).Search(t.Context(), &rpctestv1.SearchRequest{})
		if why, ok := rpc.NotSent(err); why != rpc.NotHolder || !ok || err.Error() != "failed_precondition: this Node does not hold the searching Role" || rpc.Lost(err) {
			t.Errorf("a Role call the target refuses = %v, NotSent %v %v", err, why, ok)
		}
		if got := c.searchers[c.workerID].calls.Load(); got != 0 {
			t.Errorf("the Worker's searcher ran %d times", got)
		}

		stranger := role.NodeID{UUID: storage.NewUUID()}
		_, err = rpctestv1connect.PingService.On(c.cluster, stranger).Ping(t.Context(), &rpctestv1.PingRequest{})
		if why, ok := rpc.NotSent(err); why != rpc.NotHolder || !ok || err.Error() != "failed_precondition: node "+stranger.String()+" is not in the roster" {
			t.Errorf("a call to a Node outside the roster = %v, NotSent %v %v", err, why, ok)
		}
	})
}

func TestCallsBetweenVersionsNeverRun(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		c := newCluster(t, "0.0.0-a", "0.0.0-b")
		c.Connect(t)
		_, err := rpctestv1connect.RecordService.Call(c.worker).Record(t.Context(), &rpctestv1.RecordRequest{})
		if why, ok := rpc.NotSent(err); why != rpc.Mismatched || !ok {
			t.Errorf("a call between versions = %v, NotSent %v %v", err, why, ok)
		}
		if got := c.recorder.calls.Load(); got != 0 {
			t.Errorf("the Controller's recorder ran %d times", got)
		}
	})
}

func TestACallWhoseLinkDiesIsLost(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		c := newCluster(t, "", "")
		c.Connect(t)
		release := make(chan struct{})
		c.pingers[c.workerID].release = release
		done := make(chan error)
		go func() {
			_, err := rpctestv1connect.PingService.On(c.cluster, c.workerID).Ping(context.Background(), &rpctestv1.PingRequest{})
			done <- err
		}()
		synctest.Wait()
		c.Net.Path(linktest.ControllerAddr, linktest.WorkerAddr).Set(fault.Reset{})
		err := <-done
		close(release)
		if _, notSent := rpc.NotSent(err); notSent || !rpc.Lost(err) {
			t.Errorf("a call whose link died after sending = %v (code %v), want Lost", err, rpc.CodeOf(err))
		}
	})
}
