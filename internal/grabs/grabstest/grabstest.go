// Package grabstest is fixtures for the tests of grabs and its callers.
package grabstest

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/jatassi/slipstream/internal/grabs"
	"github.com/jatassi/slipstream/internal/history/entry"
	"github.com/jatassi/slipstream/internal/history/entry/entrytest"
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
// search's is Backlog and RSS sync's RSS, each its default when zero.
// They don't change. Backlog search next opens at NextBacklog, and reads
// as running while it is zero.
type Cadences struct {
	Backlog     time.Duration
	RSS         time.Duration
	NextBacklog time.Time
}

func (c Cadences) Next(_ context.Context, id tasks.TaskID) (time.Time, error) {
	if id != grabs.BacklogTaskID {
		return time.Time{}, tasks.ErrUnknownTask
	}
	return c.NextBacklog, nil
}

func (c Cadences) Every(_ context.Context, id tasks.TaskID) (*int64, error) {
	every := map[tasks.TaskID]time.Duration{grabs.BacklogTaskID: c.Backlog, grabs.RSSTaskID: c.RSS}[id]
	if every == 0 {
		return nil, nil
	}
	return new(int64(every / time.Second)), nil
}

func (c Cadences) EveryIn(ctx context.Context, _ storage.Reader[storage.Library], id tasks.TaskID) (*int64, error) {
	return c.Every(ctx, id)
}

func (Cadences) SetCadence(context.Context, tasks.TaskID, setting.Stored[int64]) error {
	return errors.New("grabstest: the cadences don't change")
}

// StoredCadences is the Scheduled tasks' cadences as tasks stores them:
// each at its default until SetCadence stores an override. None has a
// schedule, so Next knows no task.
type StoredCadences struct {
	mu    sync.Mutex
	every map[tasks.TaskID]*int64
}

func (*StoredCadences) Next(context.Context, tasks.TaskID) (time.Time, error) {
	return time.Time{}, tasks.ErrUnknownTask
}

func (c *StoredCadences) Every(_ context.Context, id tasks.TaskID) (*int64, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.every[id], nil
}

func (c *StoredCadences) EveryIn(ctx context.Context, _ storage.Reader[storage.Library], id tasks.TaskID) (*int64, error) {
	return c.Every(ctx, id)
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

// Notices is a Recorder for a Grabs whose events a test doesn't read.
func Notices() entry.Recorder { return entrytest.Ignored("grabs", grabs.Notices()) }
