package role

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"math/rand/v2"
	"net/netip"
	"slices"
	"sync"
	"time"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"

	"github.com/jatassi/slipstream/internal/backoff"
	"github.com/jatassi/slipstream/internal/link"
	nodev1 "github.com/jatassi/slipstream/internal/pb/slipstream/node/v1"
	"github.com/jatassi/slipstream/internal/pb/slipstream/node/v1/nodev1connect"
	"github.com/jatassi/slipstream/internal/role/internal/holding"
	"github.com/jatassi/slipstream/internal/storage"
)

type HoldingConfig struct {
	Local    *storage.DB[storage.Local]
	Upstream Upstream
	// Held is every Role this binary can run, as the composition root
	// wires them.
	Held    []Held
	Timings Timings
	// Rand is the Node's seeded role stream, safe for concurrent draws.
	Rand *rand.Rand
	Log  *slog.Logger
}

// Holding is the holder half: this Node's cached intent, its facts and
// outbox, its session to the control half, and the supervisor running each
// assigned Role.
type Holding struct {
	cfg   HoldingConfig
	epoch storage.UUID
	// post wakes the session to send changed facts or queued results.
	post chan struct{}
	// settled closes once the supervisor may act on the cached intent: a
	// session caught up, or the Controller is out of reach. Before then the
	// cache may hold an intent the Controller superseded while the Node was
	// away.
	settled chan struct{}
	settle  func()

	mu       sync.Mutex
	standing Standing
	cache    map[string]cached
	assigned []Name
	draining []Name
	name     string
	// waits holds, per scope, a channel closed at its next change.
	waits map[string]chan struct{}
	// changed is closed at any change the supervisor acts on.
	changed chan struct{}
	facts   map[string]*anypb.Any
	unsent  map[string]bool
	runs    map[Name]*run
	host    Host
}

// Host is what the holder half reports about its Node beyond its Roles.
type Host struct {
	HTTPAddress      netip.AddrPort
	SettingsRevision string
	Probes           []ProbeResult
}

// SetHost records v for the node scope's facts, which supervise sends
// when they changed.
func (h *Holding) SetHost(v Host) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.host = v
	h.wake()
}

type cached struct {
	version []byte
	payload *anypb.Any
}

// NewHolding loads local.db's epoch and intent cache, so a Node that
// starts out of reach of its Controller works from its last intent.
func NewHolding(ctx context.Context, cfg HoldingConfig) (*Holding, error) {
	settled := make(chan struct{})
	h := &Holding{
		cfg:     cfg,
		post:    make(chan struct{}, 1),
		settled: settled,
		settle:  sync.OnceFunc(func() { close(settled) }),
		cache:   map[string]cached{},
		waits:   map[string]chan struct{}{},
		changed: make(chan struct{}),
		facts:   map[string]*anypb.Any{},
		unsent:  map[string]bool{},
		runs:    map[Name]*run{},
	}
	if err := cfg.Local.Write(ctx, func(ctx context.Context, tx *storage.Tx[storage.Local]) error {
		var err error
		h.epoch, err = holding.Epoch(ctx, tx)
		return err
	}); err != nil {
		return nil, fmt.Errorf("role epoch: %w", err)
	}
	intents, err := holding.Intents(ctx, cfg.Local)
	if err != nil {
		return nil, fmt.Errorf("role intent cache: %w", err)
	}
	for _, in := range intents {
		payload := &anypb.Any{}
		if err := proto.Unmarshal(in.Payload, payload); err != nil {
			return nil, fmt.Errorf("role intent cache: %s: %w", in.Scope, err)
		}
		if err := h.remember(in.Scope, cached{version: in.Version, payload: payload}); err != nil {
			return nil, err
		}
	}
	return h, nil
}

// Run keeps a session to the control half and runs the assigned Roles
// until ctx ends.
func (h *Holding) Run(ctx context.Context) error {
	var wg sync.WaitGroup
	wg.Go(func() { h.supervise(ctx) })
	h.cfg.Upstream.run(ctx, h)
	wg.Wait()
	return nil
}

// Holds reports whether this Node's last intent assigns it r or has it
// drain r: the holder's dispatch gate.
func (h *Holding) Holds(r Name) bool {
	h.mu.Lock()
	defer h.mu.Unlock()
	return slices.Contains(h.assigned, r) || slices.Contains(h.draining, r)
}

// Name is this Node's name, as its last intent gives it.
func (h *Holding) Name() string {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.name
}

// Reset forgets every cached intent, in local.db and in memory, so the
// Node holds no Role until a Controller sends it one: its Roles stop and
// let go what they hold, and a later start can't resume them from a
// Controller that removed the Node.
func (h *Holding) Reset(ctx context.Context) error {
	if err := h.cfg.Local.Write(ctx, holding.ClearIntents); err != nil {
		return fmt.Errorf("clear the role intent cache: %w", err)
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	for scope := range h.cache {
		h.notify(scope)
	}
	clear(h.cache)
	h.assigned, h.draining, h.name = nil, nil, ""
	h.wake()
	return nil
}

// remember caches an intent in memory. The caller holds h.mu or owns h.
func (h *Holding) remember(scope string, c cached) error {
	if scope == nodeScope {
		in := &nodev1.NodeIntent{}
		if err := c.payload.UnmarshalTo(in); err != nil {
			return fmt.Errorf("node intent: %w", err)
		}
		draining := names(in.GetDraining())
		for _, r := range symmetricDifference(h.draining, draining) {
			h.notify(string(r))
		}
		h.assigned, h.draining, h.name = names(in.GetRoles()), draining, in.GetName()
	}
	h.cache[scope] = c
	h.notify(scope)
	h.wake()
	return nil
}

// notify closes the channel scope's watchers wait on. The caller holds
// h.mu or owns h.
func (h *Holding) notify(scope string) {
	if ch := h.waits[scope]; ch != nil {
		close(ch)
		delete(h.waits, scope)
	}
}

func names(roles []string) []Name {
	out := make([]Name, len(roles))
	for i, r := range roles {
		out[i] = Name(r)
	}
	return out
}

// symmetricDifference is every Name in exactly one of a and b.
func symmetricDifference(a, b []Name) []Name {
	var out []Name
	for _, n := range a {
		if !slices.Contains(b, n) {
			out = append(out, n)
		}
	}
	for _, n := range b {
		if !slices.Contains(a, n) {
			out = append(out, n)
		}
	}
	return out
}

// wake closes changed. The caller holds h.mu.
func (h *Holding) wake() {
	close(h.changed)
	h.changed = make(chan struct{})
}

// watch is scope's cached intent, the standing, whether the Node drains
// the Role, and a channel closed when any of them changes.
func (h *Holding) watch(scope string) (payload *anypb.Any, standing Standing, draining bool, changed <-chan struct{}) {
	h.mu.Lock()
	defer h.mu.Unlock()
	ch := h.waits[scope]
	if ch == nil {
		ch = make(chan struct{})
		h.waits[scope] = ch
	}
	return h.cache[scope].payload, h.standing, slices.Contains(h.draining, Name(scope)), ch
}

func (h *Holding) setStanding(s Standing) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.standing == s {
		return
	}
	h.standing = s
	for _, ch := range h.waits {
		close(ch)
	}
	clear(h.waits)
	h.wake()
}

func (h *Holding) setFacts(scope string, payload *anypb.Any) {
	h.mu.Lock()
	h.facts[scope] = payload
	h.unsent[scope] = true
	h.mu.Unlock()
	h.poke()
}

func (h *Holding) poke() {
	select {
	case h.post <- struct{}{}:
	default:
	}
}

// report queues r in tx. The session reads the outbox inside its own
// local.db Write, which waits for tx to commit, so the poke cannot arrive
// before the row.
func (h *Holding) report(ctx context.Context, tx *storage.Tx[storage.Local], scope string, r proto.Message) error {
	payload, err := proto.Marshal(mustAny(r))
	if err != nil {
		return fmt.Errorf("%s result: %w", scope, err)
	}
	if err := holding.Add(ctx, tx, scope, payload); err != nil {
		return err
	}
	h.poke()
	return nil
}

// holderStream is the holder end of a Sync stream.
type holderStream interface {
	Send(*nodev1.Up) error
	Receive() (*nodev1.Down, error)
	Close() error
}

// session runs one Sync session on s and closes it. It reports whether
// the session caught up.
func (h *Holding) session(ctx context.Context, s holderStream) (bool, error) {
	ctx, cancel := context.WithCancel(ctx)
	var reader sync.WaitGroup
	defer reader.Wait()
	defer func() { _ = s.Close() }()
	defer cancel()

	downs := receive(&reader, s.Receive)
	hs := &holderSession{h: h, s: s}
	if err := s.Send(h.open()); err != nil {
		return false, err
	}
	if err := h.sendFacts(s, true); err != nil {
		return false, err
	}
	beat := time.NewTicker(h.cfg.Timings.Beat)
	defer beat.Stop()
	for {
		var err error
		select {
		case <-downs.ready:
			err = hs.receive(ctx, downs)
		case <-h.post:
			err = hs.flush(ctx)
		case <-beat.C:
			err = s.Send(&nodev1.Up{Frame: &nodev1.Up_Beat{Beat: &nodev1.Beat{}}})
		case <-ctx.Done():
			err = fmt.Errorf("role session: %w", ctx.Err())
		}
		if err != nil {
			return hs.caughtUp, err
		}
	}
}

// holderSession is one session's progress: the highest result seq sent,
// whether the control half has acked yet, and whether it caught up.
type holderSession struct {
	h        *Holding
	s        holderStream
	sent     uint64
	acked    bool
	caughtUp bool
}

func (hs *holderSession) receive(ctx context.Context, downs *inbox[*nodev1.Down]) error {
	frames, end := downs.take()
	for _, d := range frames {
		if err := hs.frame(ctx, d); err != nil {
			return err
		}
	}
	return end
}

func (hs *holderSession) frame(ctx context.Context, d *nodev1.Down) error {
	switch f := d.GetFrame().(type) {
	case *nodev1.Down_Ack:
		hs.acked = true
		hs.sent = max(hs.sent, f.Ack.GetSeq())
		if err := hs.h.drop(ctx, f.Ack.GetSeq()); err != nil {
			return err
		}
		return hs.h.sendResults(ctx, hs.s, &hs.sent)
	case *nodev1.Down_Intent:
		return hs.h.store(ctx, f.Intent)
	case *nodev1.Down_CaughtUp:
		hs.caughtUp = true
		hs.h.setStanding(Live)
		hs.h.settle()
		return nil
	}
	return fmt.Errorf("unexpected frame %T", d.GetFrame())
}

// flush sends changed facts and, once acked, newly queued results.
func (hs *holderSession) flush(ctx context.Context) error {
	if err := hs.h.sendFacts(hs.s, false); err != nil {
		return err
	}
	if !hs.acked {
		return nil
	}
	return hs.h.sendResults(ctx, hs.s, &hs.sent)
}

func (h *Holding) open() *nodev1.Up {
	h.mu.Lock()
	defer h.mu.Unlock()
	open := &nodev1.Open{Epoch: h.epoch.Bytes()}
	for scope, c := range h.cache {
		open.Have = append(open.Have, &nodev1.Have{Scope: scope, Version: c.version})
	}
	return &nodev1.Up{Frame: &nodev1.Up_Open{Open: open}}
}

// sendFacts sends every scope's facts, or those changed since last sent.
func (h *Holding) sendFacts(s holderStream, all bool) error {
	h.mu.Lock()
	var frames []*nodev1.Up
	for scope, payload := range h.facts {
		if all || h.unsent[scope] {
			frames = append(frames, &nodev1.Up{Frame: &nodev1.Up_Facts{Facts: &nodev1.Facts{Scope: scope, Payload: payload}}})
		}
	}
	clear(h.unsent)
	h.mu.Unlock()
	for _, f := range frames {
		if err := s.Send(f); err != nil {
			return err
		}
	}
	return nil
}

// sendResults sends the queued results above *sent, in order. It reads
// them in a Write so it runs after any Report still committing.
func (h *Holding) sendResults(ctx context.Context, s holderStream, sent *uint64) error {
	var queued []holding.Result
	if err := h.cfg.Local.Write(ctx, func(ctx context.Context, tx *storage.Tx[storage.Local]) error {
		var err error
		queued, err = holding.After(ctx, tx, *sent)
		return err
	}); err != nil {
		return err
	}
	for _, r := range queued {
		payload := &anypb.Any{}
		if err := proto.Unmarshal(r.Payload, payload); err != nil {
			return fmt.Errorf("result %d: %w", r.Seq, err)
		}
		if err := s.Send(&nodev1.Up{Frame: &nodev1.Up_Result{Result: &nodev1.Result{Scope: r.Scope, Seq: r.Seq, Payload: payload}}}); err != nil {
			return err
		}
		*sent = r.Seq
	}
	return nil
}

func (h *Holding) drop(ctx context.Context, seq uint64) error {
	return h.cfg.Local.Write(ctx, func(ctx context.Context, tx *storage.Tx[storage.Local]) error {
		return holding.Drop(ctx, tx, seq)
	})
}

// store caches an intent in local.db, then wakes its watchers.
func (h *Holding) store(ctx context.Context, in *nodev1.Intent) error {
	payload, err := proto.Marshal(in.GetPayload())
	if err != nil {
		return fmt.Errorf("%s intent: %w", in.GetScope(), err)
	}
	if err := h.cfg.Local.Write(ctx, func(ctx context.Context, tx *storage.Tx[storage.Local]) error {
		return holding.PutIntent(ctx, tx, holding.Intent{Scope: in.GetScope(), Version: in.GetVersion(), Payload: payload})
	}); err != nil {
		return err
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.remember(in.GetScope(), cached{version: in.GetVersion(), payload: in.GetPayload()})
}

// restartDelay is the wait before the next try after n failed ones.
func (h *Holding) restartDelay(n int) time.Duration {
	return backoff.Delay(h.cfg.Timings.RestartMin, h.cfg.Timings.RestartMax, n, h.cfg.Rand)
}

// Upstream is where the holder half's sessions go: the Controller's own
// control half in process, or the Controller over the link.
type Upstream struct {
	run func(ctx context.Context, h *Holding)
}

// InProcess is the Controller's own holder half, in session with c.
func InProcess(c *Controls) Upstream {
	return Upstream{run: func(ctx context.Context, h *Holding) {
		self := c.cfg.Nodes.Self()
		n := 0
		for {
			holderEnd, controlEnd := newPipe()
			served := make(chan struct{})
			go func() {
				defer close(served)
				_ = c.serve(ctx, self, controlEnd)
				controlEnd.close()
			}()
			caughtUp, _ := h.session(ctx, holderEnd)
			h.setStanding(Unreachable)
			<-served
			if caughtUp {
				n = 0
			}
			if !sleep(ctx, h.restartDelay(n)) {
				return
			}
			n++
		}
	}}
}

// OverLink is a Worker's holder half, in session with the Controller
// holding controller whenever the link to it is live and in step. The
// Worker acts on its cached intent once a session caught up, or once the
// Controller is out of reach: the link joins another version, or no
// session caught up within the longest either side takes to dial under t,
// the Controller's longest redial and one handshake.
func OverLink(l *link.Link, controller link.Key, t link.Timings) Upstream {
	return Upstream{run: func(ctx context.Context, h *Holding) {
		reach := time.AfterFunc(t.RedialMax+t.Handshake, h.settle)
		defer reach.Stop()
		changes := l.Subscribe(ctx)
		n := 0
		for {
			wait := time.Duration(0)
			switch st := l.State(controller).(type) {
			case link.Live:
				if !st.Matched {
					h.setStanding(Mismatched)
					h.settle()
					break
				}
				caughtUp := h.overLink(ctx, l, controller)
				h.setStanding(Unreachable)
				if caughtUp {
					n = 0
				}
				if l.State(controller) == st {
					wait = h.restartDelay(n)
					n++
				}
			case link.Down:
				h.setStanding(Unreachable)
			}
			if !awaitChange(ctx, changes, controller, wait) {
				return
			}
		}
	}}
}

func (h *Holding) overLink(ctx context.Context, l *link.Link, controller link.Key) bool {
	stream, err := nodev1connect.NewSyncServiceClient(l.Client(controller)).Sync(ctx)
	if err != nil {
		h.cfg.Log.Debug("role session not opened", "err", err)
		return false
	}
	caughtUp, err := h.session(ctx, stream)
	h.cfg.Log.Debug("role session ended", "err", err)
	return caughtUp
}

// awaitChange waits for a change of key's state, or wait when it is not
// zero. It is false once ctx ends or the link closes.
func awaitChange(ctx context.Context, changes <-chan link.Change, key link.Key, wait time.Duration) bool {
	var timeout <-chan time.Time
	if wait > 0 {
		t := time.NewTimer(wait)
		defer t.Stop()
		timeout = t.C
	}
	for {
		select {
		case c, ok := <-changes:
			if !ok {
				return false
			}
			if c.Key == key {
				return true
			}
		case <-timeout:
			return true
		case <-ctx.Done():
			return false
		}
	}
}

// sleep waits d, false when ctx ends first.
func sleep(ctx context.Context, d time.Duration) bool {
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-t.C:
		return true
	case <-ctx.Done():
		return false
	}
}

// pipe is an in-process Sync stream: the same frames, never marshaled.
type pipe struct {
	up   chan *nodev1.Up
	down chan *nodev1.Down
	done chan struct{}
	once *sync.Once
}

type holderEnd struct{ p pipe }

type controlEnd struct{ p pipe }

func newPipe() (holderEnd, controlEnd) {
	p := pipe{up: make(chan *nodev1.Up), down: make(chan *nodev1.Down), done: make(chan struct{}), once: &sync.Once{}}
	return holderEnd{p}, controlEnd{p}
}

func (p pipe) close() { p.once.Do(func() { close(p.done) }) }

func (e holderEnd) Send(u *nodev1.Up) error {
	select {
	case e.p.up <- u:
		return nil
	case <-e.p.done:
		return io.EOF
	}
}

func (e holderEnd) Receive() (*nodev1.Down, error) {
	select {
	case d := <-e.p.down:
		return d, nil
	case <-e.p.done:
		return nil, io.EOF
	}
}

func (e holderEnd) Close() error {
	e.p.close()
	return nil
}

func (e controlEnd) Send(d *nodev1.Down) error {
	select {
	case e.p.down <- d:
		return nil
	case <-e.p.done:
		return io.EOF
	}
}

func (e controlEnd) Receive() (*nodev1.Up, error) {
	select {
	case u := <-e.p.up:
		return u, nil
	case <-e.p.done:
		return nil, io.EOF
	}
}

func (e controlEnd) close() { e.p.close() }
