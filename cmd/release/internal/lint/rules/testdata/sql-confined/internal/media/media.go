package media

import (
	"context"
	"database/sql/driver"

	"github.com/pressly/goose/v3"
	sqlite "modernc.org/sqlite/lib"
)

var _ driver.Valuer

func probe(ctx context.Context, conn interface {
	PrepareContext(context.Context, string) (any, error)
	QueryContext(context.Context, string) (any, error)
}) {
	_, _ = conn.PrepareContext(ctx, "SELECT 1")
	_, _ = conn.
		QueryContext(ctx, "SELECT 2")
	_ = goose.Up
	_ = sqlite.SQLITE_OK
}
