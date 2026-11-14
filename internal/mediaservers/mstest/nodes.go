package mstest

import (
	"context"
	"iter"
	"sync"
	"testing"
	"testing/synctest"
	"time"

	"github.com/jatassi/slipstream/internal/mediaserverlink"
	"github.com/jatassi/slipstream/internal/mediaservers"
	"github.com/jatassi/slipstream/internal/nodes"
	msv1 "github.com/jatassi/slipstream/internal/pb/slipstream/mediaserver/v1"
	"github.com/jatassi/slipstream/internal/pb/slipstream/mediaserver/v1/mediaserverv1connect"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/rpc"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/storage/storagetest"
)

// links are home's and seedbox's Media Server link holders, each running
// on the intent the Controller derives for it as the Role runs it, and the
// state of each Node's link as the Controller sees it.
type links struct {
	t       *testing.T
	w       *World
	cfg     mediaserverlink.Config
	mu      sync.Mutex
	held    map[role.NodeID]*holder
	link    map[role.NodeID]*mediaserverlink.Link
	local   map[role.NodeID]*storage.DB[storage.Local]
	facts   map[role.NodeID]*msv1.Facts
	away    map[role.NodeID]time.Time
	cut     map[role.NodeID]bool
	lose    map[role.NodeID]func(*msv1.Job) bool
	changed chan struct{}
}

func newLinks(t *testing.T, w *World, cfg mediaserverlink.Config) *links {
	return &links{
		t: t, w: w, cfg: cfg, held: map[role.NodeID]*holder{}, link: map[role.NodeID]*mediaserverlink.Link{},
		local: map[role.NodeID]*storage.DB[storage.Local]{}, facts: map[role.NodeID]*msv1.Facts{},
		away: map[role.NodeID]time.Time{}, cut: map[role.NodeID]bool{}, lose: map[role.NodeID]func(*msv1.Job) bool{},
		changed: make(chan struct{}),
	}
}

// start runs node's holder on its own Link, keeping node's local database
// across stops, until stop or the test's end.
func (l *links) start(node role.NodeID) {
	ml := mediaserverlink.NewLink(l.cfg)
	ctx, cancel := context.WithCancel(context.Background())
	h := &holder{node: node, l: l, poke: make(chan struct{}, 1), ctx: ctx, cancel: cancel, done: make(chan struct{})}
	l.mu.Lock()
	local, ok := l.local[node]
	if !ok {
		local = storagetest.Local(l.t, role.LocalSchema())
		l.local[node] = local
	}
	l.held[node], l.link[node] = h, ml
	l.mu.Unlock()
	control := mediaservers.ControlService{Library: l.w.Lib, Sealer: l.w.Sealer, Nodes: caller{node, l.w.Home}}
	go func() {
		defer close(h.done)
		if err := ml.Run(local, control)(ctx, h); err != nil {
			l.t.Errorf("the holder on %s stopped: %v", node, err)
		}
	}()
	l.t.Cleanup(func() { l.stop(node) })
}

// stop ends node's holder as a Node that stops does, with no last report.
func (l *links) stop(node role.NodeID) {
	l.mu.Lock()
	h, ok := l.held[node]
	delete(l.held, node)
	delete(l.facts, node)
	l.mu.Unlock()
	if ok {
		h.cancel()
		<-h.done
	}
}

// Start runs node's holder again after Stop.
func (w *World) Start(node role.NodeID) {
	w.links.start(node)
	synctest.Wait()
}

// Stop ends node's holder as a Node that stops does, with no last report,
// while the roster still sees its link as it was.
func (w *World) Stop(node role.NodeID) { w.links.stop(node) }

// touch has each node's holder read its intent again.
func (l *links) touch(ns ...role.NodeID) {
	l.mu.Lock()
	defer l.mu.Unlock()
	for _, n := range ns {
		if h, ok := l.held[n]; ok {
			select {
			case h.poke <- struct{}{}:
			default:
			}
		}
	}
}

func (l *links) factsOf(node role.NodeID) (*msv1.Facts, bool) {
	l.mu.Lock()
	defer l.mu.Unlock()
	f, ok := l.facts[node]
	return f, ok
}

// service is node's MediaServerLinkService as the Controller calls it: an
// rpc error while its link is away or cut.
func (l *links) service(node role.NodeID) mediaserverv1connect.MediaServerLinkServiceHandler {
	l.mu.Lock()
	defer l.mu.Unlock()
	if _, away := l.away[node]; away || l.cut[node] {
		return unlinked{}
	}
	return losing{l.link[node].Service(), l, node}
}

// LoseAnswer drops the answer of node's next Advance whose Job matches,
// after node ran its step: a link that broke while the call ran.
func (w *World) LoseAnswer(node role.NodeID, match func(*msv1.Job) bool) {
	w.links.mu.Lock()
	defer w.links.mu.Unlock()
	w.links.lose[node] = match
}

// losing is node's service, dropping the answer LoseAnswer names.
type losing struct {
	mediaserverv1connect.MediaServerLinkServiceHandler
	l    *links
	node role.NodeID
}

func (s losing) Advance(ctx context.Context, req *msv1.AdvanceRequest) (*msv1.AdvanceResponse, error) {
	res, err := s.MediaServerLinkServiceHandler.Advance(ctx, req)
	s.l.mu.Lock()
	defer s.l.mu.Unlock()
	if match, ok := s.l.lose[s.node]; ok && match(req.GetJob()) {
		delete(s.l.lose, s.node)
		return nil, errUnlinked()
	}
	return res, err
}

func (l *links) set(node role.NodeID, away, cut bool) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if away {
		l.away[node] = time.Now()
	} else {
		delete(l.away, node)
	}
	l.cut[node] = cut
	close(l.changed)
	l.changed = make(chan struct{})
}

func (l *links) inStep(node role.NodeID) bool {
	s, _ := l.standing(node)
	return s == nodes.InStep
}

func (l *links) standing(node role.NodeID) (nodes.Standing, time.Time) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if since, ok := l.away[node]; ok {
		return nodes.Offline, since
	}
	return nodes.InStep, time.Time{}
}

func (l *links) roster() <-chan struct{} {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.changed
}

// Offline takes node's link down now, as the roster sees it.
func (w *World) Offline(node role.NodeID) { w.links.set(node, true, false) }

// Online brings node's link back.
func (w *World) Online(node role.NodeID) { w.links.set(node, false, false) }

// Cut drops node's calls while the roster still sees it in step: a
// flapping link.
func (w *World) Cut(node role.NodeID) { w.links.set(node, false, true) }

// holder is the Role framework's side of a Node's link: the intent the
// Controller derives for node, read again at each Touch, its facts kept as
// node's and its reports applied at once as node's.
type holder struct {
	node   role.NodeID
	l      *links
	poke   chan struct{}
	ctx    context.Context
	cancel context.CancelFunc
	done   chan struct{}
}

func (h *holder) Intent(ctx context.Context) iter.Seq[role.Snapshot[*msv1.Intent]] {
	return func(yield func(role.Snapshot[*msv1.Intent]) bool) {
		for {
			var in *msv1.Intent
			err := h.l.w.Lib.Read(ctx, func(ctx context.Context, r storage.Reader[storage.Library]) error {
				var err error
				in, err = h.l.w.ms.Intent(ctx, r, h.node)
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

func (h *holder) SetFacts(f *msv1.Facts) {
	h.l.mu.Lock()
	h.l.facts[h.node] = f
	h.l.mu.Unlock()
	h.l.w.ms.Noted(h.node, string(role.MediaServerLink))
}

// Report applies r at once, where the Role would apply it once the Node's
// local Write commits and its outbox reaches the Controller. A report the
// holder sends as it stops is lost, as one a stopping Node never sends.
func (h *holder) Report(_ context.Context, _ *storage.Tx[storage.Local], r *msv1.Report) error {
	err := h.l.w.Apply(h.ctx, h.node, r)
	if err != nil && h.ctx.Err() == nil {
		h.l.t.Errorf("apply %v from %s: %v", r, h.node, err)
	}
	return err
}

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
