package node

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"runtime/debug"
	"sync"
	"time"
)

// Stack is a Node's lifecycle: each subsystem pushes its stop function as it
// starts, and the Node pops them in reverse within one deadline.
type Stack struct {
	log   *slog.Logger
	fails *failure

	mu      sync.Mutex
	entries []stopEntry
	// unwinding is set once the Stack began to unwind, under the Node's
	// stop deadline limit.
	unwinding bool
	limit     time.Duration
}

// failure is shared by a Stack and its Subs: any goroutine failing fails
// the Node.
type failure struct {
	once   sync.Once
	err    error
	failed chan struct{}
}

type stopEntry struct {
	name string
	stop func(context.Context) error
}

var ErrStopDeadline = errors.New("stop deadline passed")

type PanicError struct {
	Value any
	Stack []byte
}

func (e *PanicError) Error() string { return fmt.Sprintf("panic: %v", e.Value) }

func newStack(log *slog.Logger) *Stack {
	return &Stack{log: log, fails: &failure{failed: make(chan struct{})}}
}

// Sub is a Stack whose stops run as one entry of s, pushed now: whatever
// is pushed onto it later still stops at this point of s's unwind. A
// subsystem that starts after the Node is ready pushes onto a Sub made at
// start, so it stops before what was there first and after what came later.
func (s *Stack) Sub(name string) *Stack {
	sub := &Stack{log: s.log, fails: s.fails}
	s.OnStop(name, func(ctx context.Context) error {
		s.mu.Lock()
		limit := s.limit
		s.mu.Unlock()
		return sub.unwindUnder(ctx, limit)
	})
	return sub
}

// OnStop pushes stop. Once the Stack began to unwind, it runs stop at once
// instead, so what started late is not left open.
func (s *Stack) OnStop(name string, stop func(context.Context) error) {
	s.mu.Lock()
	late := s.unwinding
	if !late {
		s.entries = append(s.entries, stopEntry{name, stop})
	}
	s.mu.Unlock()
	if late {
		if err := stop(context.Background()); err != nil {
			s.log.Error("stop failed", "stop", name, "err", err.Error())
		}
	}
}

// Go runs fn until the Stack unwinds to it, when its context is canceled
// and the Stack waits for it to return. If fn returns an error or panics
// first, the Node fails. Once the Stack began to unwind, fn's context is
// canceled from the start and Go waits for fn to return.
func (s *Stack) Go(name string, fn func(context.Context) error) {
	ctx, cancel := context.WithCancel(context.Background())
	s.mu.Lock()
	if s.unwinding {
		cancel()
	}
	s.mu.Unlock()
	done := make(chan struct{})
	go func() {
		defer close(done)
		if err := recovered(ctx, fn); err != nil && ctx.Err() == nil {
			s.fail(fmt.Errorf("%s: %w", name, err))
		}
	}()
	s.OnStop(name, func(stopCtx context.Context) error {
		cancel()
		select {
		case <-done:
			return nil
		case <-stopCtx.Done():
			return fmt.Errorf("wait for %s: %w", name, stopCtx.Err())
		}
	})
}

func recovered(ctx context.Context, fn func(context.Context) error) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = &PanicError{Value: r, Stack: debug.Stack()}
		}
	}()
	return fn(ctx)
}

func (s *Stack) fail(err error) {
	s.fails.once.Do(func() {
		attrs := []any{"err", err.Error()}
		if p, ok := errors.AsType[*PanicError](err); ok {
			attrs = append(attrs, "stack", string(p.Stack))
		}
		s.log.Error("subsystem failed", attrs...)
		s.fails.err = err
		close(s.fails.failed)
	})
}

// Failed is closed when a goroutine run through the Stack fails.
func (s *Stack) Failed() <-chan struct{} { return s.fails.failed }

// Err is the failure that closed Failed.
func (s *Stack) Err() error {
	select {
	case <-s.fails.failed:
		return s.fails.err
	default:
		return nil
	}
}

// unwind runs the stop functions in reverse. Past the deadline it logs how
// long each finished one took, the one still running and for how long, and
// those it never reached, and returns ErrStopDeadline without waiting
// further.
func (s *Stack) unwind(deadline time.Duration) error {
	ctx, cancel := context.WithTimeout(context.Background(), deadline)
	defer cancel()
	return s.unwindUnder(ctx, deadline)
}

// unwindUnder is unwind under ctx, which ends limit after the Node's
// unwind began.
func (s *Stack) unwindUnder(ctx context.Context, limit time.Duration) error {
	s.mu.Lock()
	entries := s.entries
	s.entries = nil
	s.unwinding, s.limit = true, limit
	s.mu.Unlock()

	var errs []error
	took := make([]string, 0, len(entries))
	for i := len(entries) - 1; i >= 0; i-- {
		e := entries[i]
		start := time.Now()
		res := make(chan error, 1)
		go func() { res <- e.stop(ctx) }()
		select {
		case err := <-res:
			took = append(took, e.name+" "+time.Since(start).String())
			if err != nil {
				s.log.Error("stop failed", "stop", e.name, "err", err.Error())
				errs = append(errs, fmt.Errorf("stop %s: %w", e.name, err))
			}
		case <-ctx.Done():
			pending := make([]string, 0, i)
			for j := i - 1; j >= 0; j-- {
				pending = append(pending, entries[j].name)
			}
			s.log.Error("stop deadline passed", "deadline", limit.String(), "took", took, "running", e.name, "running_for", time.Since(start).String(), "not_run", pending)
			return fmt.Errorf("%w after %s, %s still running", ErrStopDeadline, limit, e.name)
		}
	}
	return errors.Join(errs...)
}
