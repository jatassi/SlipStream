package node

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"
)

type stopRecorder struct {
	mu    sync.Mutex
	order []string
}

func (r *stopRecorder) stop(name string) func(context.Context) error {
	return func(context.Context) error {
		r.mu.Lock()
		defer r.mu.Unlock()
		r.order = append(r.order, name)
		return nil
	}
}

func records(t *testing.T, log string) []map[string]any {
	t.Helper()
	var out []map[string]any
	for line := range strings.SplitSeq(strings.TrimSpace(log), "\n") {
		var rec map[string]any
		if err := json.Unmarshal([]byte(line), &rec); err != nil {
			t.Fatalf("log line %q: %v", line, err)
		}
		out = append(out, rec)
	}
	return out
}

func TestPanickingGoroutineFailsTheNodeAndUnwindsInReverse(t *testing.T) {
	logged := new(syncBuffer)
	s := newStack(slog.New(slog.NewJSONHandler(logged, nil)))
	r := new(stopRecorder)
	s.OnStop("first", r.stop("first"))
	s.OnStop("second", r.stop("second"))
	s.Go("loop", func(context.Context) error { panic("boom") })
	s.OnStop("third", r.stop("third"))

	select {
	case <-s.Failed():
	case <-time.After(10 * time.Second):
		t.Fatal("panic did not fail the stack")
	}
	err := errors.Join(s.Err(), s.unwind(time.Second))
	if err == nil || err.Error() != "loop: panic: boom" || ExitCode(err) != ExitFailed {
		t.Fatalf("err = %v, want loop: panic: boom with exit %d", err, ExitFailed)
	}
	if want := []string{"third", "second", "first"}; !slices.Equal(r.order, want) {
		t.Fatalf("stop order = %v, want %v", r.order, want)
	}
	rec := records(t, logged.String())[0]
	if rec["level"] != "ERROR" || rec["msg"] != "subsystem failed" || rec["err"] != "loop: panic: boom" || !strings.Contains(fmt.Sprint(rec["stack"]), "stack_test.go") {
		t.Fatalf("failure record = %v, want the error and its stack at error level", rec)
	}
}

func TestGoroutineErrorFailsAndCancelledOneDoesNot(t *testing.T) {
	s := newStack(slog.New(slog.DiscardHandler))
	s.Go("worker", func(ctx context.Context) error {
		<-ctx.Done()
		return ctx.Err()
	})
	if err := s.unwind(time.Second); err != nil || s.Err() != nil {
		t.Fatalf("stopping a worker: unwind %v, failure %v; want neither", err, s.Err())
	}

	s = newStack(slog.New(slog.DiscardHandler))
	s.Go("dialer", func(context.Context) error { return errors.New("connection refused") })
	<-s.Failed()
	if err := s.Err(); err == nil || err.Error() != "dialer: connection refused" {
		t.Fatalf("err = %v, want dialer: connection refused", err)
	}
}

func TestUnwindPastTheDeadlineNamesTheStopStillRunning(t *testing.T) {
	logged := new(syncBuffer)
	s := newStack(slog.New(slog.NewJSONHandler(logged, nil)))
	r := new(stopRecorder)
	hung := make(chan struct{})
	t.Cleanup(func() { close(hung) })
	s.OnStop("lock", r.stop("lock"))
	s.OnStop("log", r.stop("log"))
	s.OnStop("torrent", func(context.Context) error { <-hung; return nil })
	s.OnStop("control", r.stop("control"))

	err := s.unwind(50 * time.Millisecond)
	if !errors.Is(err, ErrStopDeadline) || err.Error() != "stop deadline passed after 50ms, torrent still running" {
		t.Fatalf("err = %v", err)
	}
	if want := []string{"control"}; !slices.Equal(r.order, want) {
		t.Fatalf("stops run = %v, want %v", r.order, want)
	}
	rec := records(t, logged.String())[0]
	got, err := json.Marshal(map[string]any{"msg": rec["msg"], "running": rec["running"], "not_run": rec["not_run"]})
	if err != nil {
		t.Fatal(err)
	}
	if want := `{"msg":"stop deadline passed","not_run":["log","lock"],"running":"torrent"}`; string(got) != want {
		t.Fatalf("deadline record = %s, want %s", got, want)
	}
}
