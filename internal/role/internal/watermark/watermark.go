// Package watermark owns the role library set: per (Node, epoch), the
// highest result seq the Controller applied, and each result it skipped
// because it can't apply it.
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

// Reject records result seq of node's epoch, for scope, as skipped because
// of reason.
func Reject(ctx context.Context, tx *storage.Tx[file], node, epoch storage.UUID, seq uint64, scope, reason string) error {
	return db.New(tx).InsertRejected(ctx, db.InsertRejectedParams{
		NodeID: db.NodeID{UUID: node}, Epoch: epoch, Seq: int64(seq), Scope: scope, Reason: reason, RejectedAt: storage.Now(), //nolint:gosec // seqs count up from 1
	})
}

// Rejected is how many results from a Node the Controller skipped, and
// when it last skipped one.
type Rejected struct {
	Count int
	Last  storage.Time
}

// ListRejected is each Node's skipped results, for every Node with any.
func ListRejected(ctx context.Context, r storage.Reader[file]) (map[storage.UUID]Rejected, error) {
	rows, err := db.New(r).ListRejected(ctx)
	if err != nil {
		return nil, err
	}
	out := map[storage.UUID]Rejected{}
	for _, row := range rows {
		out[row.NodeID.UUID] = Rejected{Count: out[row.NodeID.UUID].Count + 1, Last: row.RejectedAt}
	}
	return out, nil
}

// Forget deletes node's skipped results.
func Forget(ctx context.Context, tx *storage.Tx[file], node storage.UUID) error {
	return db.New(tx).DeleteRejected(ctx, db.NodeID{UUID: node})
}
