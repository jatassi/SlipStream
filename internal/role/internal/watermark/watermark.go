// Package watermark owns the role library set: per (Node, epoch), the
// highest result seq the Controller applied.
package watermark

import (
	"context"
	"embed"
	"errors"

	"github.com/jatassi/slipstream/internal/role/internal/watermark/internal/db"
	"github.com/jatassi/slipstream/internal/storage"
)

type file = storage.Library

//go:embed schema/*.sql
var schemaFiles embed.FS

func Schema() storage.Schema[file] { return storage.NewSchema[file]("role", schemaFiles) }

// Get is the highest seq applied from node's epoch, 0 before the first.
func Get(ctx context.Context, r storage.Reader[file], node, epoch storage.UUID) (uint64, error) {
	seq, err := db.New(r).GetWatermark(ctx, db.GetWatermarkParams{NodeID: db.NodeID{UUID: node}, Epoch: epoch})
	if errors.Is(err, storage.ErrNoRows) {
		return 0, nil
	}
	return uint64(seq), err //nolint:gosec // seqs count up from 1
}

func Set(ctx context.Context, tx *storage.Tx[file], node, epoch storage.UUID, seq uint64) error {
	return db.New(tx).SetWatermark(ctx, db.SetWatermarkParams{NodeID: db.NodeID{UUID: node}, Epoch: epoch, Seq: int64(seq)}) //nolint:gosec // seqs count up from 1
}
