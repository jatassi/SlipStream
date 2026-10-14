package worldtest

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/jatassi/slipstream/fakes/world"
)

var budgets struct {
	mu sync.Mutex
	m  map[*testing.T]time.Duration
}

func addBudget(t *testing.T, d time.Duration) {
	budgets.mu.Lock()
	defer budgets.mu.Unlock()
	if budgets.m == nil {
		budgets.m = map[*testing.T]time.Duration{}
	}
	if _, ok := budgets.m[t]; !ok {
		t.Cleanup(func() {
			budgets.mu.Lock()
			defer budgets.mu.Unlock()
			delete(budgets.m, t)
		})
	}
	budgets.m[t] += d
}

func budget(t *testing.T) time.Duration {
	budgets.mu.Lock()
	defer budgets.mu.Unlock()
	return budgets.m[t]
}

// Context is the only context the end-to-end tier uses. Its deadline is the
// test's, less the time stopping the test's runs may take, so a hung world
// fails the test with a message and still stops cleanly.
func Context(t *testing.T) context.Context {
	t.Helper()
	parent := context.Background()
	deadline, ok := t.Deadline()
	if !ok {
		ctx, cancel := context.WithCancel(parent)
		t.Cleanup(cancel)
		return ctx
	}
	ctx, cancel := context.WithDeadline(parent, deadline.Add(-budget(t)))
	t.Cleanup(cancel)
	return ctx
}

const tick = 25 * time.Millisecond

// Until is the tier's polling wait on a condition. It checks cond every
// 25 ms until it holds, failing the test at once when cond returns an
// error and when Context's deadline passes first. It journals one note
// into each run t launched, and the API calls cond makes journal only as
// that note.
func Until(t *testing.T, what string, cond func() (bool, error)) {
	t.Helper()
	var ends []func(int, []byte)
	for _, run := range launchedBy(t) {
		ends = append(ends, startStep(t, run, world.StepStart{Note: "until " + what}))
	}
	exit := 1
	defer func() {
		for _, end := range ends {
			end(exit, nil)
		}
	}()
	wait(t, 1)
	defer wait(t, -1)
	ctx := Context(t)
	ticker := time.NewTicker(tick)
	defer ticker.Stop()
	for {
		ok, err := cond()
		if err != nil {
			t.Fatalf("waiting until %s: %v", what, err)
		}
		if ok {
			exit = 0
			return
		}
		select {
		case <-ctx.Done():
			t.Fatalf("waiting until %s: %v", what, ctx.Err())
		case <-ticker.C:
		}
	}
}
