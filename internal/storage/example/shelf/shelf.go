package shelf

import (
	"context"
	"embed"

	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/storage/example/shelf/internal/db"
)

type file = storage.Library

//go:embed schema/*.sql
var schemaFiles embed.FS

func Schema() storage.Schema[file] { return storage.NewSchema[file]("shelf", schemaFiles) }

type ID = db.ShelfID

func Add(ctx context.Context, tx *storage.Tx[file], name string) (ID, error) {
	return db.New(tx).InsertShelf(ctx, name)
}

func Count(ctx context.Context, r storage.Reader[file]) (int64, error) {
	return db.New(r).CountShelves(ctx)
}

// Names lists every shelf's name in name order.
func Names(ctx context.Context, r storage.Reader[file]) ([]string, error) {
	return db.New(r).ListShelfNames(ctx)
}
