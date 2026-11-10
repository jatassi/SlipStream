// Package grabstest is fixtures for the tests of grabs and its callers.
package grabstest

import (
	"context"
	"testing"

	"github.com/jatassi/slipstream/internal/grabs"
	"github.com/jatassi/slipstream/internal/storage"
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
