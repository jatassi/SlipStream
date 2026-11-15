// Package grabstest is fixtures for the tests of grabs and its callers.
package grabstest

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/jatassi/slipstream/internal/grabs"
	"github.com/jatassi/slipstream/internal/setting"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/tasks"
)

// Blocked puts e on lib's Blocklist through the Blocklist's one writer,
// with no Controller Reconciling.
func Blocked(t testing.TB, lib *storage.DB[storage.Library], e grabs.BlockEntry) {
	t.Helper()
	on := grabs.BlocklistOn{Reconciling: func(context.Context, storage.Reader[storage.Library]) (bool, error) { return false, nil }}
	err := lib.Write(t.Context(), func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
		return on.Block(ctx, tx, e)
	})
	if err != nil {
		t.Fatal(err)
	}
}

// Cadences is the Scheduled tasks' cadences as grabs reads them: Backlog
// search's is Backlog, its default when zero, and every other at its
// default. They don't change.
type Cadences struct{ Backlog time.Duration }

func (c Cadences) Every(_ context.Context, id tasks.TaskID) (*int64, error) {
	if id != grabs.BacklogTaskID || c.Backlog == 0 {
		return nil, nil
	}
	return new(int64(c.Backlog / time.Second)), nil
}

func (Cadences) SetCadence(context.Context, tasks.TaskID, setting.Stored[int64]) error {
	return errors.New("grabstest: the cadences don't change")
}

// StoredCadences is the Scheduled tasks' cadences as tasks stores them:
// each at its default until SetCadence stores an override.
type StoredCadences struct {
	mu    sync.Mutex
	every map[tasks.TaskID]*int64
}

func (c *StoredCadences) Every(_ context.Context, id tasks.TaskID) (*int64, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.every[id], nil
}

func (c *StoredCadences) SetCadence(_ context.Context, id tasks.TaskID, v setting.Stored[int64]) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.every == nil {
		c.every = map[tasks.TaskID]*int64{}
	}
	c.every[id] = v.Override()
	return nil
}
