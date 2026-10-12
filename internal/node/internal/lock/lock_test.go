package lock_test

import (
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"testing"

	"github.com/jatassi/slipstream/internal/node/internal/lock"
)

func TestSecondAcquireNamesTheHolder(t *testing.T) {
	run := filepath.Join(t.TempDir(), "data", "run")
	first, err := lock.Acquire(run)
	if err != nil {
		t.Fatal(err)
	}
	pidFile := filepath.Join(run, "node.pid")
	if b, err := os.ReadFile(pidFile); err != nil || string(b) != strconv.Itoa(os.Getpid())+"\n" {
		t.Fatalf("PID file = %q, %v; want this process's PID", b, err)
	}

	_, err = lock.Acquire(run)
	if held := new(lock.HeldError); !errors.As(err, &held) || held.PID != os.Getpid() {
		t.Fatalf("second acquire: err = %v, want held by PID %d", err, os.Getpid())
	}

	if err := os.Remove(pidFile); err != nil {
		t.Fatal(err)
	}
	_, err = lock.Acquire(run)
	if held := new(lock.HeldError); !errors.As(err, &held) || held.PID != 0 {
		t.Fatalf("acquire while the holder starts: err = %v, want held by PID 0", err)
	}

	if err := first.Release(); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(pidFile); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("PID file after release: %v, want removed", err)
	}
	again, err := lock.Acquire(run)
	if err != nil {
		t.Fatalf("acquire after release: %v", err)
	}
	if err := again.Release(); err != nil {
		t.Fatal(err)
	}
}

func TestStalePIDFileIsOverwritten(t *testing.T) {
	run := t.TempDir()
	if err := os.WriteFile(filepath.Join(run, "node.pid"), []byte("99999\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	l, err := lock.Acquire(run)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := l.Release(); err != nil {
			t.Error(err)
		}
	})
	if b, _ := os.ReadFile(filepath.Join(run, "node.pid")); string(b) != strconv.Itoa(os.Getpid())+"\n" {
		t.Fatalf("PID file = %q, want this process's PID", b)
	}
}
