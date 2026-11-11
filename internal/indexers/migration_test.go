package indexers_test

import (
	"context"
	"strings"
	"testing"

	"github.com/jatassi/slipstream/internal/indexers"
	"github.com/jatassi/slipstream/internal/nodes"
	"github.com/jatassi/slipstream/internal/secret"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/storage/storagetest"
	"github.com/jatassi/slipstream/internal/works"
)

func TestTheSearchWaitKeepsToTenSecondsThroughTenMinutes(t *testing.T) {
	lib := storagetest.Library(t, secret.Schema(), nodes.Schema(), works.Schema(), indexers.Schema())
	for _, c := range []struct {
		seconds any
		ok      bool
	}{{nil, true}, {10, true}, {600, true}, {9, false}, {601, false}} {
		err := lib.Write(t.Context(), func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
			_, err := tx.ExecContext(ctx, "UPDATE indexers_settings SET search_wait_seconds = ?", c.seconds)
			return err
		})
		switch {
		case c.ok && err != nil:
			t.Errorf("a search wait of %v seconds: refused: %v", c.seconds, err)
		case !c.ok && (err == nil || !strings.Contains(err.Error(), "CHECK constraint failed")):
			t.Errorf("a search wait of %v seconds: want a CHECK to refuse it, got %v", c.seconds, err)
		}
	}
}
