package role

import (
	"context"
	"crypto/sha256"
	"fmt"
	"iter"
	"time"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"

	"github.com/jatassi/slipstream/internal/role/internal/holding"
	"github.com/jatassi/slipstream/internal/role/internal/watermark"
	"github.com/jatassi/slipstream/internal/storage"
)

// LibrarySchema is the role library set: the Controller's watermark per
// (Node, epoch).
func LibrarySchema() storage.Schema[storage.Library] { return watermark.Schema() }

// LocalSchema is the role local set: the epoch, the intent cache and the
// result outbox.
func LocalSchema() storage.Schema[storage.Local] { return holding.Schema() }

// Timings are the Role supervisor's restart backoff bounds, how often a
// holder beats on its Sync stream, and how long the Controller lets a
// Worker's Sync stream go silent before it closes the Worker's link.
type Timings struct {
	RestartMin time.Duration `yaml:"restart_min"`
	RestartMax time.Duration `yaml:"restart_max"`
	Beat       time.Duration `yaml:"beat"`
	Stall      time.Duration `yaml:"stall"`
}

// Def binds a Role's name to its intent, facts and result messages.
type Def[I, F, R proto.Message] struct {
	Name Name
	// Notable reports whether a holder's next facts change what Admins see
	// of its Node beside its last, which are empty before its first. Nil
	// means facts never do.
	Notable func(last, next F) bool
}

// Holder is a Role's holder half, as its run function sees it.
type Holder[I, F, R proto.Message] interface {
	// Intent yields the current intent at once, then the latest after each
	// change of intent or Standing, until ctx ends.
	Intent(ctx context.Context) iter.Seq[Snapshot[I]]
	// SetFacts replaces this Role's facts on this Node. It never blocks.
	SetFacts(f F)
	// Report queues r in tx, so it reaches the Controller once tx commits,
	// and is applied there exactly once.
	Report(ctx context.Context, tx *storage.Tx[storage.Local], r R) error
}

type Snapshot[I proto.Message] struct {
	Intent   I
	Standing Standing
	// Draining: the Node no longer holds the Role and finishes what it
	// holds.
	Draining bool
}

// MayStart reports whether a job may start now: in step with the
// Controller and not draining. Otherwise, finish in-flight work and start
// nothing new.
func (s Snapshot[I]) MayStart() bool { return s.Standing == Live && !s.Draining }

// Serve is the Role's Controller half: derive computes a Node's intent from
// the library, and apply applies one of its results.
func (d Def[I, F, R]) Serve(
	derive func(context.Context, storage.Reader[storage.Library], NodeID) (I, error),
	apply func(context.Context, *storage.Tx[storage.Library], NodeID, R) error,
) *Control[I, F, R] {
	return &Control[I, F, R]{name: d.Name, derive: derive, apply: apply, notable: d.Notable}
}

// Hold is the Role's holder half: the supervisor runs run while this Node
// holds or drains the Role and its intent is cached, and calls release once
// it neither holds nor drains it. Release lets go of what the Role's runs
// left behind; a shutdown or a restart after a failure never calls it.
func (d Def[I, F, R]) Hold(run func(context.Context, Holder[I, F, R]) error, release func(context.Context) error) Held {
	return Held{name: d.Name, scoped: true, release: release, run: func(ctx context.Context, h *Holding) error {
		return run(ctx, holder[I, F, R]{h: h, scope: string(d.Name)})
	}}
}

// HoldRun is the holder half of a Role with no intent, such as Serving: the
// supervisor runs run while this Node holds or drains Role n, and calls
// release as Hold does.
func HoldRun(n Name, run, release func(context.Context) error) Held {
	return Held{name: n, release: release, run: func(ctx context.Context, _ *Holding) error { return run(ctx) }}
}

// Held is one Role's run and release functions, for the supervisor. A
// scoped Role runs only once its intent is cached.
type Held struct {
	name    Name
	scoped  bool
	run     func(context.Context, *Holding) error
	release func(context.Context) error
}

// Control is one Role's Controller half.
type Control[I, F, R proto.Message] struct {
	name     Name
	derive   func(context.Context, storage.Reader[storage.Library], NodeID) (I, error)
	apply    func(context.Context, *storage.Tx[storage.Library], NodeID, R) error
	notable  func(last, next F) bool
	controls *Controls
}

// Touch tells the Controls the Role's intent for nodes may have changed.
// Call it after the commit that changed it.
func (c *Control[I, F, R]) Touch(nodes ...NodeID) {
	c.controls.touch(string(c.name), nodes)
}

// Facts is node's latest facts for the Role, false while it has none.
func (c *Control[I, F, R]) Facts(node NodeID) (F, bool) {
	f := newMessage[F]()
	payload, ok := c.controls.factsOf(node, string(c.name))
	if !ok {
		return f, false
	}
	if err := payload.UnmarshalTo(f); err != nil {
		panic(fmt.Sprintf("role: %s facts: %v", c.name, err))
	}
	return f, true
}

// Served is a Control, as NewControls takes it.
type Served interface {
	scope() string
	bind(c *Controls)
	deriveAny(ctx context.Context, r storage.Reader[storage.Library], node NodeID) (proto.Message, error)
	applyAny(ctx context.Context, tx *storage.Tx[storage.Library], node NodeID, payload *anypb.Any) error
	notableAny(last, next *anypb.Any) bool
}

func (c *Control[I, F, R]) scope() string     { return string(c.name) }
func (c *Control[I, F, R]) bind(cs *Controls) { c.controls = cs }

func (c *Control[I, F, R]) deriveAny(ctx context.Context, r storage.Reader[storage.Library], node NodeID) (proto.Message, error) {
	return c.derive(ctx, r, node)
}

func (c *Control[I, F, R]) applyAny(ctx context.Context, tx *storage.Tx[storage.Library], node NodeID, payload *anypb.Any) error {
	r := newMessage[R]()
	if err := payload.UnmarshalTo(r); err != nil {
		return fmt.Errorf("%s result: %w", c.name, err)
	}
	return c.apply(ctx, tx, node, r)
}

func (c *Control[I, F, R]) notableAny(last, next *anypb.Any) bool {
	if c.notable == nil {
		return false
	}
	l := newMessage[F]()
	if last != nil {
		if err := last.UnmarshalTo(l); err != nil {
			panic(fmt.Sprintf("role: %s facts: %v", c.name, err))
		}
	}
	n := newMessage[F]()
	if err := next.UnmarshalTo(n); err != nil {
		panic(fmt.Sprintf("role: %s facts: %v", c.name, err))
	}
	return c.notable(l, n)
}

type holder[I, F, R proto.Message] struct {
	h     *Holding
	scope string
}

func (h holder[I, F, R]) Intent(ctx context.Context) iter.Seq[Snapshot[I]] {
	return func(yield func(Snapshot[I]) bool) {
		for {
			payload, standing, draining, changed := h.h.watch(h.scope)
			if payload != nil {
				i := newMessage[I]()
				if err := payload.UnmarshalTo(i); err != nil {
					panic(fmt.Sprintf("role: %s intent: %v", h.scope, err))
				}
				if !yield(Snapshot[I]{Intent: i, Standing: standing, Draining: draining}) {
					return
				}
			}
			select {
			case <-changed:
			case <-ctx.Done():
				return
			}
		}
	}
}

func (h holder[I, F, R]) SetFacts(f F) {
	h.h.setFacts(h.scope, mustAny(f))
}

func (h holder[I, F, R]) Report(ctx context.Context, tx *storage.Tx[storage.Local], r R) error {
	return h.h.report(ctx, tx, h.scope, r)
}

// newMessage is a new, empty M, for M a generated message pointer.
func newMessage[M proto.Message]() M {
	var m M
	out, ok := m.ProtoReflect().Type().New().Interface().(M)
	if !ok {
		panic(fmt.Sprintf("role: %T is not a generated message pointer", m))
	}
	return out
}

func mustAny(m proto.Message) *anypb.Any {
	a, err := anypb.New(m)
	if err != nil {
		panic(fmt.Sprintf("role: pack %s: %v", m.ProtoReflect().Descriptor().FullName(), err))
	}
	return a
}

// version is a scope's version: the first 16 bytes of SHA-256 over its
// intent's deterministic encoding.
func version(m proto.Message) ([]byte, error) {
	b, err := proto.MarshalOptions{Deterministic: true}.Marshal(m)
	if err != nil {
		return nil, fmt.Errorf("encode %s: %w", m.ProtoReflect().Descriptor().FullName(), err)
	}
	sum := sha256.Sum256(b)
	return sum[:16], nil
}
