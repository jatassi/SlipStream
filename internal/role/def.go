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

// Timings are the Role supervisor's restart backoff bounds.
type Timings struct {
	RestartMin time.Duration `yaml:"restart_min"`
	RestartMax time.Duration `yaml:"restart_max"`
}

// Def binds a Role's name to its intent, facts and result messages.
type Def[I, F, R proto.Message] struct {
	Name Name
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
}

// MayStart reports whether a job may start now: in step with the
// Controller. Out of step, finish in-flight work and start nothing new.
func (s Snapshot[I]) MayStart() bool { return s.Standing == Live }

// Serve is the Role's Controller half: derive computes a Node's intent from
// the library, and apply applies one of its results.
func (d Def[I, F, R]) Serve(
	derive func(context.Context, storage.Reader[storage.Library], NodeID) (I, error),
	apply func(context.Context, *storage.Tx[storage.Library], NodeID, R) error,
) *Control[I, F, R] {
	return &Control[I, F, R]{name: d.Name, derive: derive, apply: apply}
}

// Hold is the Role's holder half: the supervisor runs run while this Node
// holds the Role and its intent is cached.
func (d Def[I, F, R]) Hold(run func(context.Context, Holder[I, F, R]) error) Held {
	return Held{name: d.Name, run: func(ctx context.Context, h *Holding) error {
		return run(ctx, holder[I, F, R]{h: h, scope: string(d.Name)})
	}}
}

// Held is one Role's run function, for the supervisor.
type Held struct {
	name Name
	run  func(context.Context, *Holding) error
}

// Control is one Role's Controller half.
type Control[I, F, R proto.Message] struct {
	name     Name
	derive   func(context.Context, storage.Reader[storage.Library], NodeID) (I, error)
	apply    func(context.Context, *storage.Tx[storage.Library], NodeID, R) error
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

type holder[I, F, R proto.Message] struct {
	h     *Holding
	scope string
}

func (h holder[I, F, R]) Intent(ctx context.Context) iter.Seq[Snapshot[I]] {
	return func(yield func(Snapshot[I]) bool) {
		for {
			payload, standing, changed := h.h.watch(h.scope)
			if payload != nil {
				i := newMessage[I]()
				if err := payload.UnmarshalTo(i); err != nil {
					panic(fmt.Sprintf("role: %s intent: %v", h.scope, err))
				}
				if !yield(Snapshot[I]{Intent: i, Standing: standing}) {
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
