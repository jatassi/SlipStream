// Package holding owns the role local set: the epoch of this local.db, the
// intent cache and the result outbox.
package holding

import (
	"context"
	"embed"
	"errors"

	"github.com/jatassi/slipstream/internal/role/internal/holding/internal/db"
	"github.com/jatassi/slipstream/internal/storage"
)

type file = storage.Local

//go:embed schema/*.sql
var schemaFiles embed.FS

func Schema() storage.Schema[file] { return storage.NewSchema[file]("role", schemaFiles) }

// Epoch is this local.db's epoch, drawn at its first use.
func Epoch(ctx context.Context, tx *storage.Tx[file]) (storage.UUID, error) {
	q := db.New(tx)
	epoch, err := q.GetEpoch(ctx)
	if !errors.Is(err, storage.ErrNoRows) {
		return epoch, err
	}
	epoch = storage.NewUUID()
	return epoch, q.InsertEpoch(ctx, epoch)
}

type Intent struct {
	Scope   string
	Version []byte
	Payload []byte
}

func Intents(ctx context.Context, r storage.Reader[file]) ([]Intent, error) {
	rows, err := db.New(r).ListIntents(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]Intent, len(rows))
	for i, row := range rows {
		out[i] = Intent(row)
	}
	return out, nil
}

func PutIntent(ctx context.Context, tx *storage.Tx[file], in Intent) error {
	return db.New(tx).PutIntent(ctx, db.PutIntentParams(in))
}

// ClearIntents deletes every cached intent.
func ClearIntents(ctx context.Context, tx *storage.Tx[file]) error {
	return db.New(tx).DeleteIntents(ctx)
}

type Result struct {
	Seq     uint64
	Scope   string
	Payload []byte
}

// Add queues a result under the epoch's next seq.
func Add(ctx context.Context, tx *storage.Tx[file], scope string, payload []byte) error {
	q := db.New(tx)
	seq, err := q.NextSeq(ctx)
	if err != nil {
		return err
	}
	return q.InsertOutbox(ctx, db.InsertOutboxParams{Seq: seq, Scope: scope, Payload: payload})
}

// After lists the queued results above seq, in order.
func After(ctx context.Context, tx *storage.Tx[file], seq uint64) ([]Result, error) {
	rows, err := db.New(tx).ListOutboxAfter(ctx, int64(seq)) //nolint:gosec // seqs count up from 1
	if err != nil {
		return nil, err
	}
	out := make([]Result, len(rows))
	for i, row := range rows {
		out[i] = Result{Seq: uint64(row.Seq), Scope: row.Scope, Payload: row.Payload} //nolint:gosec // seqs count up from 1
	}
	return out, nil
}

// Drop deletes the queued results through seq.
func Drop(ctx context.Context, tx *storage.Tx[file], seq uint64) error {
	return db.New(tx).DeleteOutboxThrough(ctx, int64(seq)) //nolint:gosec // seqs count up from 1
}
