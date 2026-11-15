// Package grabstest is fixtures for the tests of grabs and its callers.
package grabstest

import (
	"context"
	"errors"
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
