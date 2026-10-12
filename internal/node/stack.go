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
	log *slog.Logger

	mu      sync.Mutex
	entries []stopEntry

	failOnce sync.Once
	failErr  error
	failed   chan struct{}
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
	return &Stack{log: log, failed: make(chan struct{})}
}

func (s *Stack) OnStop(name string, stop func(context.Context) error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.entries = append(s.entries, stopEntry{name, stop})
}

// Go runs fn until the Stack unwinds to it, when its context is canceled
// and the Stack waits for it to return. If fn returns an error or panics
// first, the Node fails.
func (s *Stack) Go(name string, fn func(context.Context) error) {
	ctx, cancel := context.WithCancel(context.Background())
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
	s.failOnce.Do(func() {
		attrs := []any{"err", err.Error()}
		if p, ok := errors.AsType[*PanicError](err); ok {
			attrs = append(attrs, "stack", string(p.Stack))
		}
		s.log.Error("subsystem failed", attrs...)
		s.failErr = err
		close(s.failed)
	})
}

// Failed is closed when a goroutine run through the Stack fails.
func (s *Stack) Failed() <-chan struct{} { return s.failed }

// Err is the failure that closed Failed.
func (s *Stack) Err() error {
	select {
	case <-s.failed:
		return s.failErr
	default:
		return nil
	}
}

// unwind runs the stop functions in reverse. Past the deadline it logs the
// one still running and those it never reached, and returns ErrStopDeadline
// without waiting further.
func (s *Stack) unwind(deadline time.Duration) error {
	s.mu.Lock()
	entries := s.entries
	s.entries = nil
	s.mu.Unlock()

	ctx, cancel := context.WithTimeout(context.Background(), deadline)
	defer cancel()
	var errs []error
	for i := len(entries) - 1; i >= 0; i-- {
		e := entries[i]
		res := make(chan error, 1)
		go func() { res <- e.stop(ctx) }()
		select {
		case err := <-res:
			if err != nil {
				s.log.Error("stop failed", "stop", e.name, "err", err.Error())
				errs = append(errs, fmt.Errorf("stop %s: %w", e.name, err))
			}
		case <-ctx.Done():
			pending := make([]string, 0, i)
			for j := i - 1; j >= 0; j-- {
				pending = append(pending, entries[j].name)
			}
			s.log.Error("stop deadline passed", "deadline", deadline.String(), "running", e.name, "not_run", pending)
			return fmt.Errorf("%w after %s, %s still running", ErrStopDeadline, deadline, e.name)
		}
	}
	return errors.Join(errs...)
}
