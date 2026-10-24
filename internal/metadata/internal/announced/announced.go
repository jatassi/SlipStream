// Package announced owns the metadata local set: the last calendar day the
// availability timer announced, in the Controller's local.db.
package announced

import (
	"context"
	"embed"
	"errors"
	"fmt"
	"time"

	"github.com/jatassi/slipstream/internal/metadata/internal/announced/internal/db"
	"github.com/jatassi/slipstream/internal/storage"
)

type file = storage.Local

//go:embed schema/*.sql
var schemaFiles embed.FS

func Schema() storage.Schema[file] { return storage.NewSchema[file]("metadata", schemaFiles) }

// Day is the last day announced, as a UTC midnight, false when local.db
// keeps none.
func Day(ctx context.Context, r storage.Reader[file]) (time.Time, bool, error) {
	s, err := db.New(r).GetDay(ctx)
	if errors.Is(err, storage.ErrNoRows) {
		return time.Time{}, false, nil
	}
	if err != nil {
		return time.Time{}, false, err
	}
	day, err := time.Parse(time.DateOnly, s)
	if err != nil {
		return time.Time{}, false, fmt.Errorf("read the announced day: %w", err)
	}
	return day, true, nil
}

// SetDay records day, a UTC midnight, as the last day announced.
func SetDay(ctx context.Context, tx *storage.Tx[file], day time.Time) error {
	return db.New(tx).PutDay(ctx, day.Format(time.DateOnly))
}
