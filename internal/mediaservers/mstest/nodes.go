package mstest

import (
	"context"
	"iter"
	"sync"
	"testing"

	"github.com/jatassi/slipstream/internal/mediaserverlink"
	"github.com/jatassi/slipstream/internal/mediaservers"
	msv1 "github.com/jatassi/slipstream/internal/pb/slipstream/mediaserver/v1"
	"github.com/jatassi/slipstream/internal/pb/slipstream/mediaserver/v1/mediaserverv1connect"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/rpc"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/storage/storagetest"
)

// links are home's and seedbox's Media Server link holders, each running
// on the intent the Controller derives for it, and the state of each
// Node's link as the Controller sees it.
type links struct {
	self    role.NodeID
	mu      sync.Mutex
	held    map[role.NodeID]*holder
	link    map[role.NodeID]*mediaserverlink.Link
	away    map[role.NodeID]bool
	cut     map[role.NodeID]bool
	changed chan struct{}
}

func newLinks(self role.NodeID) *links {
	return &links{
		self: self, held: map[role.NodeID]*holder{}, link: map[role.NodeID]*mediaserverlink.Link{},
		away: map[role.NodeID]bool{}, cut: map[role.NodeID]bool{}, changed: make(chan struct{}),
	}
}

// start runs node's holder on ms's intent for it, until ctx ends.
func (l *links) start(ctx context.Context, t *testing.T, w *World, node role.NodeID, cfg mediaserverlink.Config, wg *sync.WaitGroup) {
	t.Helper()
	h := &holder{node: node, w: w, poke: make(chan struct{}, 1)}
	ml := mediaserverlink.NewLink(cfg)
	l.mu.Lock()
	l.held[node], l.link[node] = h, ml
	l.mu.Unlock()
	local := storagetest.Local(t, role.LocalSchema())
	control := mediaservers.ControlService{Library: w.Lib, Sealer: w.Sealer, Nodes: caller{node, l.self}}
	wg.Go(func() { _ = ml.Run(local, control)(ctx, h) })
}

// touch has each node's holder read its intent again.
func (l *links) touch(nodes ...role.NodeID) {
	l.mu.Lock()
	defer l.mu.Unlock()
	for _, n := range nodes {
		if h, ok := l.held[n]; ok {
			select {
			case h.poke <- struct{}{}:
			default:
			}
		}
	}
}

// service is node's MediaServerLinkService as the Controller calls it: an
// rpc error while its link is away or cut.
func (l *links) service(node role.NodeID) mediaserverv1connect.MediaServerLinkServiceHandler {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.away[node] || l.cut[node] {
		return unlinked{}
	}
	return l.link[node].Service()
}

func (l *links) set(node role.NodeID, away, cut bool) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.away[node], l.cut[node] = away, cut
	close(l.changed)
	l.changed = make(chan struct{})
}

func (l *links) inStep(node role.NodeID) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	return !l.away[node]
}

func (l *links) roster() <-chan struct{} {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.changed
}

// Offline takes node's link down, as the roster sees it.
func (w *World) Offline(node role.NodeID) { w.links.set(node, true, false) }

// Online brings node's link back.
func (w *World) Online(node role.NodeID) { w.links.set(node, false, false) }

// Cut drops node's calls while the roster still sees it in step: a
// flapping link.
func (w *World) Cut(node role.NodeID) { w.links.set(node, false, true) }

// holder is the Role framework's side of a Node's link: the intent the
// Controller derives for node, read again at each Touch. Reports go
// nowhere.
type holder struct {
	node role.NodeID
	w    *World
	poke chan struct{}
}

func (h *holder) Intent(ctx context.Context) iter.Seq[role.Snapshot[*msv1.Intent]] {
	return func(yield func(role.Snapshot[*msv1.Intent]) bool) {
		for {
			var in *msv1.Intent
			err := h.w.Lib.Read(ctx, func(ctx context.Context, r storage.Reader[storage.Library]) error {
				var err error
				in, err = h.w.ms.Intent(ctx, r, h.node)
				return err
			})
			if err != nil || !yield(role.Snapshot[*msv1.Intent]{Intent: in, Standing: role.Live}) {
				return
			}
			select {
			case <-ctx.Done():
				return
			case <-h.poke:
			}
		}
	}
}

func (*holder) SetFacts(*msv1.Facts) {}

func (*holder) Report(context.Context, *storage.Tx[storage.Local], *msv1.Report) error { return nil }

// caller is the Controller's view of who calls MediaServerControlService.
type caller struct{ node, self role.NodeID }

func (c caller) Caller(context.Context) (role.NodeID, bool) { return c.node, true }
func (c caller) Self() role.NodeID                          { return c.self }

// unlinked is a Node whose link is down: every call fails as the link's.
type unlinked struct{}

func errUnlinked() error { return rpc.Errorf(rpc.CodeUnavailable, "the Node's link is down") }

func (unlinked) Reach(context.Context, *msv1.ReachRequest) (*msv1.ReachResponse, error) {
	return nil, errUnlinked()
}

func (unlinked) Check(context.Context, *msv1.CheckRequest) (*msv1.CheckResponse, error) {
	return nil, errUnlinked()
}

func (unlinked) SignIn(context.Context, *msv1.SignInRequest) (*msv1.SignInResponse, error) {
	return nil, errUnlinked()
}

func (unlinked) Advance(context.Context, *msv1.AdvanceRequest) (*msv1.AdvanceResponse, error) {
	return nil, errUnlinked()
}
