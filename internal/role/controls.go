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
	"time"

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

// NodeScope is the node scope as the control half derives it: the Roles a
// Node holds, assigned or covering, the Roles it no longer holds but still
// holds work for, so it drains them, and its name.
type NodeScope struct {
	Holds    []Name
	Draining []Name
	Name     string
}

type ControlsConfig struct {
	Library *storage.DB[storage.Library]
	Nodes   Nodes
	// Scope derives the node scope.
	Scope func(ctx context.Context, r storage.Reader[storage.Library], node NodeID) (NodeScope, error)
	// Changed is called after a Node's session or node-scope facts change,
	// and after a Role's facts change in a way its Def's Notable reports.
	// It must not block.
	Changed func(node NodeID)
	// Facts is called after a holder's Role facts are stored, with the
	// Role's name. It must not block.
	Facts func(node NodeID, scope string)
	// Applied is called after a batch of node's results commits, once for
	// each Role whose results it applied. It must not block.
	Applied func(node NodeID, scope string)
	// Touched is called after a Role's Control.Touch, with the Role's name
	// and the Nodes touched. It must not block.
	Touched func(scope string, nodes []NodeID)
	// InStep is called when node's Sync session attaches and when it ends,
	// and Seen after each frame its holder sends after Open. Neither may
	// block.
	InStep func(node NodeID, attached bool)
	Seen   func(node NodeID)
	// Stalled is called when a Worker's Sync stream carried nothing for
	// Stall, before its session ends. It must not block.
	Stalled func(node NodeID)
	Stall   time.Duration
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
	for _, node := range nodes {
		if s := c.sessions[node]; s != nil {
			s.dirty[scope] = true
			select {
			case s.wake <- struct{}{}:
			default:
			}
		}
	}
	c.mu.Unlock()
	if scope != nodeScope && c.cfg.Touched != nil {
		c.cfg.Touched(scope, nodes)
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
	scope, err := c.cfg.Scope(ctx, c.cfg.Library, node)
	if err != nil {
		return nil, err
	}
	return version(nodeIntent(scope))
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
	return c.converse(ctx, sess, epoch, s, ups)
}

// converse answers the holder's frames and sends the scopes that change
// until the session ends. A holder silent for the stall loses its link;
// the Controller's own session never stalls, as it is always in step.
func (c *Controls) converse(ctx context.Context, sess *controlSession, epoch storage.UUID, s controlStream, ups *inbox[*nodev1.Up]) error {
	stall := time.NewTimer(c.cfg.Stall)
	defer stall.Stop()
	silence := stall.C
	if sess.node == c.cfg.Nodes.Self() {
		silence = nil
	}
	for {
		select {
		case <-ups.ready:
			stall.Reset(c.cfg.Stall)
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
		case <-silence:
			c.cfg.Stalled(sess.node)
			return fmt.Errorf("the holder sent nothing for %s", c.cfg.Stall)
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
	if c.cfg.InStep != nil {
		c.cfg.InStep(s.node, true)
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
	if current && c.cfg.InStep != nil {
		c.cfg.InStep(s.node, false)
	}
	if current && c.cfg.Changed != nil {
		c.cfg.Changed(s.node)
	}
}

// receive stores a batch's facts and applies its results in one Write,
// then acks them.
func (c *Controls) receive(ctx context.Context, sess *controlSession, epoch storage.UUID, s controlStream, batch []*nodev1.Up) error {
	results, err := c.sort(sess, batch)
	if err != nil {
		return err
	}
	if len(results) == 0 {
		return nil
	}
	var top uint64
	var scopes []string
	err = c.cfg.Library.Write(ctx, func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
		var err error
		top, scopes, err = c.apply(ctx, tx, sess.node, epoch, results)
		return err
	})
	if err != nil {
		return err
	}
	if c.cfg.Applied != nil {
		for _, scope := range scopes {
			c.cfg.Applied(sess.node, scope)
		}
	}
	return s.Send(&nodev1.Down{Frame: &nodev1.Down_Ack{Ack: &nodev1.Ack{Seq: top}}})
}

// apply applies the results past node's watermark for epoch and moves it
// to the last, answering it and the scopes applied.
func (c *Controls) apply(ctx context.Context, tx *storage.Tx[storage.Library], node NodeID, epoch storage.UUID, results []*nodev1.Result) (top uint64, scopes []string, err error) {
	if top, err = watermark.Get(ctx, tx, node.UUID, epoch); err != nil {
		return 0, nil, err
	}
	applied := top
	for _, r := range results {
		if r.GetSeq() <= top {
			continue
		}
		srv := c.served[r.GetScope()]
		if srv == nil {
			return 0, nil, fmt.Errorf("result %d is for %q, which this Controller does not serve", r.GetSeq(), r.GetScope())
		}
		if err := srv.applyAny(ctx, tx, node, r.GetPayload()); err != nil {
			return 0, nil, fmt.Errorf("apply %s result %d: %w", r.GetScope(), r.GetSeq(), err)
		}
		if !slices.Contains(scopes, r.GetScope()) {
			scopes = append(scopes, r.GetScope())
		}
		top = r.GetSeq()
	}
	if top == applied {
		return top, nil, nil
	}
	return top, scopes, watermark.Set(ctx, tx, node.UUID, epoch, top)
}

// sort stores a batch's facts, notes that its Node was seen and returns
// its results.
func (c *Controls) sort(sess *controlSession, batch []*nodev1.Up) ([]*nodev1.Result, error) {
	var results []*nodev1.Result
	for _, u := range batch {
		switch f := u.GetFrame().(type) {
		case *nodev1.Up_Facts:
			c.storeFacts(sess, f.Facts)
		case *nodev1.Up_Result:
			results = append(results, f.Result)
		case *nodev1.Up_Beat:
		default:
			return nil, fmt.Errorf("unexpected frame %T after Open", f)
		}
	}
	if len(batch) > 0 && c.cfg.Seen != nil {
		c.cfg.Seen(sess.node)
	}
	return results, nil
}

// storeFacts keeps f as its Node's facts for its scope, and tells the
// roster the Node changed when they are node-scope facts or notable ones.
func (c *Controls) storeFacts(sess *controlSession, f *nodev1.Facts) {
	c.mu.Lock()
	current := c.sessions[sess.node] == sess
	var last *anypb.Any
	if current {
		if c.facts[sess.node] == nil {
			c.facts[sess.node] = map[string]*anypb.Any{}
		}
		last = c.facts[sess.node][f.GetScope()]
		c.facts[sess.node][f.GetScope()] = f.GetPayload()
	}
	c.mu.Unlock()
	if !current {
		return
	}
	if f.GetScope() != nodeScope && c.cfg.Facts != nil {
		c.cfg.Facts(sess.node, f.GetScope())
	}
	if c.cfg.Changed == nil {
		return
	}
	if srv := c.served[f.GetScope()]; f.GetScope() == nodeScope || srv != nil && srv.notableAny(last, f.GetPayload()) {
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

// derive is node's intent for the scopes in dirty, as sync reads it: the
// scopes of the Roles it holds, then of those it drains, then the node
// scope.
func (c *Controls) derive(ctx context.Context, node NodeID, dirty map[string]bool) ([]*nodev1.Intent, error) {
	all := dirty == nil || dirty[nodeScope]
	scope, err := c.cfg.Scope(ctx, c.cfg.Library, node)
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
	for _, name := range slices.Concat(scope.Holds, scope.Draining) {
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
		if err := add(nodeScope, nodeIntent(scope)); err != nil {
			return nil, err
		}
	}
	return intents, nil
}

func nodeIntent(s NodeScope) *nodev1.NodeIntent {
	return &nodev1.NodeIntent{Roles: roleStrings(s.Holds), Draining: roleStrings(s.Draining), Name: s.Name}
}

func roleStrings(ns []Name) []string {
	out := make([]string, len(ns))
	for i, n := range ns {
		out[i] = string(n)
	}
	return out
}
