package role

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"maps"
	"net/http"
	"slices"
	"sync"

	"connectrpc.com/connect/v2"
	"connectrpc.com/connect/v2/connecthttp"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"

	"github.com/jatassi/slipstream/internal/link"
	nodev1 "github.com/jatassi/slipstream/internal/pb/slipstream/node/v1"
	"github.com/jatassi/slipstream/internal/pb/slipstream/node/v1/nodev1connect"
	"github.com/jatassi/slipstream/internal/role/internal/watermark"
	"github.com/jatassi/slipstream/internal/storage"
)

// nodeScope is the scope naming which Roles a Node holds. Every other scope
// is a Role's name.
const nodeScope = "node"

// Nodes is the Controller's roster, as the control half sees it.
type Nodes interface {
	Self() NodeID
	// NodeOf is the Node holding key, false for a key the Controller does
	// not know.
	NodeOf(key link.Key) (NodeID, bool)
}

type ControlsConfig struct {
	Library *storage.DB[storage.Library]
	Nodes   Nodes
	// Assigned derives the node scope: the Roles assigned to node.
	Assigned func(ctx context.Context, r storage.Reader[storage.Library], node NodeID) ([]Name, error)
	// Changed is called after a Node's session or facts change. It must not
	// block.
	Changed func(node NodeID)
	Log     *slog.Logger
}

// Controls is the control half: one session per holder, deriving its
// intent from the library and applying its results.
type Controls struct {
	cfg    ControlsConfig
	served map[string]Served

	mu       sync.Mutex
	sessions map[NodeID]*controlSession
	facts    map[NodeID]map[string]*anypb.Any
}

// NewControls serves each Role's Control. A Role served twice panics.
func NewControls(cfg ControlsConfig, served ...Served) *Controls {
	c := &Controls{
		cfg:      cfg,
		served:   map[string]Served{},
		sessions: map[NodeID]*controlSession{},
		facts:    map[NodeID]map[string]*anypb.Any{},
	}
	for _, s := range served {
		if _, dup := c.served[s.scope()]; dup {
			panic("role: " + s.scope() + " is served twice")
		}
		s.bind(c)
		c.served[s.scope()] = s
	}
	return c
}

type controlSession struct {
	node   NodeID
	cancel context.CancelFunc
	wake   chan struct{}
	// dirty and versions are guarded by Controls.mu. versions is what the
	// holder has, per scope.
	dirty    map[string]bool
	versions map[string][]byte
}

// controlStream is the control end of a Sync stream.
type controlStream interface {
	Receive() (*nodev1.Up, error)
	Send(*nodev1.Down) error
}

// Handler serves the link's Sync streams, at the path it returns, on the
// lockstep handler.
func (c *Controls) Handler() (string, http.Handler) {
	srv := connect.NewServer()
	nodev1connect.RegisterSyncServiceHandler(srv, syncService{c})
	mux := http.NewServeMux()
	connecthttp.Mount(mux, srv)
	return "/" + nodev1connect.SyncServiceName + "/", mux
}

type syncService struct{ c *Controls }

func (s syncService) Sync(ctx context.Context, stream nodev1connect.SyncServiceSyncServerStream) error {
	key, _ := link.PeerOf(ctx)
	node, ok := s.c.cfg.Nodes.NodeOf(key)
	if !ok {
		return connect.NewError(connect.CodePermissionDenied, "the Controller has no Node with this key")
	}
	return s.c.serve(ctx, node, stream)
}

// Touch tells the Controls which Roles nodes hold may have changed. Call
// it after the commit that changed them.
func (c *Controls) Touch(nodes ...NodeID) { c.touch(nodeScope, nodes) }

func (c *Controls) touch(scope string, nodes []NodeID) {
	c.mu.Lock()
	defer c.mu.Unlock()
	for _, node := range nodes {
		if s := c.sessions[node]; s != nil {
			s.dirty[scope] = true
			select {
			case s.wake <- struct{}{}:
			default:
			}
		}
	}
}

// NodeFacts is node's node-scope facts, false while it has no session.
func (c *Controls) NodeFacts(node NodeID) (*nodev1.NodeFacts, bool) {
	payload, ok := c.factsOf(node, nodeScope)
	if !ok {
		return nil, false
	}
	f := &nodev1.NodeFacts{}
	if err := payload.UnmarshalTo(f); err != nil {
		panic(fmt.Sprintf("role: node facts: %v", err))
	}
	return f, true
}

// NodeVersion is the node-scope version the library derives for node now.
func (c *Controls) NodeVersion(ctx context.Context, node NodeID) ([]byte, error) {
	assigned, err := c.cfg.Assigned(ctx, c.cfg.Library, node)
	if err != nil {
		return nil, err
	}
	return version(nodeIntent(assigned))
}

func (c *Controls) factsOf(node NodeID, scope string) (*anypb.Any, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	f, ok := c.facts[node][scope]
	return f, ok
}

// Lag is a scope whose intent, derived now, differs from what its holder
// has, with no Touch pending.
type Lag struct {
	Node  NodeID
	Scope string
}

// Lagging derives every live session's scopes and lists those that lag.
// Called while nothing else runs, it finds a change made without Touch.
func (c *Controls) Lagging(ctx context.Context) ([]Lag, error) {
	type held struct {
		node     NodeID
		versions map[string][]byte
	}
	c.mu.Lock()
	var sessions []held
	for node, s := range c.sessions {
		if len(s.dirty) == 0 {
			sessions = append(sessions, held{node, maps.Clone(s.versions)})
		}
	}
	c.mu.Unlock()
	var lags []Lag
	for _, s := range sessions {
		intents, err := c.derive(ctx, s.node, nil)
		if err != nil {
			return nil, err
		}
		for _, in := range intents {
			if !slices.Equal(in.GetVersion(), s.versions[in.GetScope()]) {
				lags = append(lags, Lag{Node: s.node, Scope: in.GetScope()})
			}
		}
	}
	return lags, nil
}

// serve runs one holder session for node until the holder closes it, a
// newer session for node replaces it, or ctx ends. It logs its own
// failures, such as an apply error; the stream's are the link's.
func (c *Controls) serve(ctx context.Context, node NodeID, s controlStream) error {
	err := c.session(ctx, node, s)
	switch {
	case err == nil, errors.Is(err, io.EOF), errors.Is(err, context.Canceled), connect.CodeOf(err) != connect.CodeUnknown:
	default:
		c.cfg.Log.Warn("role session failed", "node", node.String(), "err", err)
	}
	return err
}

func (c *Controls) session(ctx context.Context, node NodeID, s controlStream) error {
	first, err := s.Receive()
	if err != nil {
		return err
	}
	open := first.GetOpen()
	if open == nil {
		return errors.New("the holder's first frame is not Open")
	}
	epoch, err := storage.UUIDFromBytes(open.GetEpoch())
	if err != nil {
		return fmt.Errorf("epoch: %w", err)
	}
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	sess := &controlSession{node: node, cancel: cancel, wake: make(chan struct{}, 1), dirty: map[string]bool{}, versions: map[string][]byte{}}
	for _, h := range open.GetHave() {
		sess.versions[h.GetScope()] = h.GetVersion()
	}
	c.attach(sess)
	defer c.detach(sess)
	// The reader outlives a session served over the link: its Receive ends
	// once the handler returns.
	ups := receive(&sync.WaitGroup{}, s.Receive)

	seq, err := watermark.Get(ctx, c.cfg.Library, node.UUID, epoch)
	if err != nil {
		return err
	}
	if err := s.Send(&nodev1.Down{Frame: &nodev1.Down_Ack{Ack: &nodev1.Ack{Seq: seq}}}); err != nil {
		return err
	}
	if err := c.sync(ctx, sess, s, nil); err != nil {
		return err
	}
	if err := s.Send(&nodev1.Down{Frame: &nodev1.Down_CaughtUp{CaughtUp: &nodev1.CaughtUp{}}}); err != nil {
		return err
	}
	for {
		select {
		case <-ups.ready:
			frames, end := ups.take()
			if err := c.receive(ctx, sess, epoch, s, frames); err != nil {
				return err
			}
			if end != nil {
				return end
			}
		case <-sess.wake:
			c.mu.Lock()
			dirty := sess.dirty
			sess.dirty = map[string]bool{}
			c.mu.Unlock()
			if err := c.sync(ctx, sess, s, dirty); err != nil {
				return err
			}
		case <-ctx.Done():
			return fmt.Errorf("role session: %w", ctx.Err())
		}
	}
}

func (c *Controls) attach(s *controlSession) {
	c.mu.Lock()
	old := c.sessions[s.node]
	c.sessions[s.node] = s
	c.mu.Unlock()
	if old != nil {
		old.cancel()
	}
}

func (c *Controls) detach(s *controlSession) {
	c.mu.Lock()
	current := c.sessions[s.node] == s
	if current {
		delete(c.sessions, s.node)
		delete(c.facts, s.node)
	}
	c.mu.Unlock()
	if current && c.cfg.Changed != nil {
		c.cfg.Changed(s.node)
	}
}

// receive stores a batch's facts and applies its results in one Write,
// then acks them.
func (c *Controls) receive(ctx context.Context, sess *controlSession, epoch storage.UUID, s controlStream, batch []*nodev1.Up) error {
	var results []*nodev1.Result
	for _, u := range batch {
		switch f := u.GetFrame().(type) {
		case *nodev1.Up_Facts:
			c.storeFacts(sess, f.Facts)
		case *nodev1.Up_Result:
			results = append(results, f.Result)
		default:
			return fmt.Errorf("unexpected frame %T after Open", f)
		}
	}
	if len(results) == 0 {
		return nil
	}
	var top uint64
	err := c.cfg.Library.Write(ctx, func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
		var err error
		if top, err = watermark.Get(ctx, tx, sess.node.UUID, epoch); err != nil {
			return err
		}
		applied := top
		for _, r := range results {
			if r.GetSeq() <= top {
				continue
			}
			srv := c.served[r.GetScope()]
			if srv == nil {
				return fmt.Errorf("result %d is for %q, which this Controller does not serve", r.GetSeq(), r.GetScope())
			}
			if err := srv.applyAny(ctx, tx, sess.node, r.GetPayload()); err != nil {
				return fmt.Errorf("apply %s result %d: %w", r.GetScope(), r.GetSeq(), err)
			}
			top = r.GetSeq()
		}
		if top == applied {
			return nil
		}
		return watermark.Set(ctx, tx, sess.node.UUID, epoch, top)
	})
	if err != nil {
		return err
	}
	return s.Send(&nodev1.Down{Frame: &nodev1.Down_Ack{Ack: &nodev1.Ack{Seq: top}}})
}

func (c *Controls) storeFacts(sess *controlSession, f *nodev1.Facts) {
	c.mu.Lock()
	current := c.sessions[sess.node] == sess
	if current {
		if c.facts[sess.node] == nil {
			c.facts[sess.node] = map[string]*anypb.Any{}
		}
		c.facts[sess.node][f.GetScope()] = f.GetPayload()
	}
	c.mu.Unlock()
	if current && c.cfg.Changed != nil {
		c.cfg.Changed(sess.node)
	}
}

// sync sends the scopes in dirty, or every scope when dirty is nil or
// holds the node scope, whose version differs from the holder's.
func (c *Controls) sync(ctx context.Context, sess *controlSession, s controlStream, dirty map[string]bool) error {
	intents, err := c.derive(ctx, sess.node, dirty)
	if err != nil {
		return err
	}
	for _, in := range intents {
		c.mu.Lock()
		have := sess.versions[in.GetScope()]
		c.mu.Unlock()
		if slices.Equal(have, in.GetVersion()) {
			continue
		}
		if err := s.Send(&nodev1.Down{Frame: &nodev1.Down_Intent{Intent: in}}); err != nil {
			return err
		}
		c.mu.Lock()
		sess.versions[in.GetScope()] = in.GetVersion()
		c.mu.Unlock()
	}
	return nil
}

// derive is node's intent for the scopes in dirty, as sync reads it: Role
// scopes in assigned order, then the node scope.
func (c *Controls) derive(ctx context.Context, node NodeID, dirty map[string]bool) ([]*nodev1.Intent, error) {
	all := dirty == nil || dirty[nodeScope]
	assigned, err := c.cfg.Assigned(ctx, c.cfg.Library, node)
	if err != nil {
		return nil, fmt.Errorf("derive node intent: %w", err)
	}
	var intents []*nodev1.Intent
	add := func(scope string, msg proto.Message) error {
		v, err := version(msg)
		if err != nil {
			return err
		}
		intents = append(intents, &nodev1.Intent{Scope: scope, Version: v, Payload: mustAny(msg)})
		return nil
	}
	for _, name := range assigned {
		srv := c.served[string(name)]
		if srv == nil || !all && !dirty[string(name)] {
			continue
		}
		msg, err := srv.deriveAny(ctx, c.cfg.Library, node)
		if err != nil {
			return nil, fmt.Errorf("derive %s intent: %w", name, err)
		}
		if err := add(string(name), msg); err != nil {
			return nil, err
		}
	}
	if all {
		if err := add(nodeScope, nodeIntent(assigned)); err != nil {
			return nil, err
		}
	}
	return intents, nil
}

func nodeIntent(assigned []Name) *nodev1.NodeIntent {
	roles := make([]string, len(assigned))
	for i, n := range assigned {
		roles[i] = string(n)
	}
	return &nodev1.NodeIntent{Roles: roles}
}
