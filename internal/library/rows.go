package library

import (
	"context"
	"slices"

	"github.com/jatassi/slipstream/internal/files"
	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/works"
)

// Row is one Work as the Library lists it.
type Row struct {
	Work   works.Row
	Rollup files.Rollup
}

// Rows answers s from one snapshot, by sort label then id.
func (l *Library) Rows(ctx context.Context, _ identity.Admin, s works.Scope) ([]Row, error) {
	var out []Row
	err := l.cfg.Library.Read(ctx, func(ctx context.Context, r storage.Reader[storage.Library]) error {
		var err error
		out, err = l.rowsIn(ctx, r, s)
		return err
	})
	return out, err
}

func (l *Library) rowsIn(ctx context.Context, r storage.Reader[storage.Library], s works.Scope) ([]Row, error) {
	ws, err := l.cfg.Works.RowsIn(ctx, r, s)
	if err != nil {
		return nil, err
	}
	out := make([]Row, 0, len(ws))
	for chunk := range slices.Chunk(ws, works.Batch) {
		bares := make([]works.Bare, len(chunk))
		for i, w := range chunk {
			bares[i] = w.Bare
		}
		rolled, err := l.cfg.Files.Rollups(ctx, r, bares)
		if err != nil {
			return nil, err
		}
		for _, w := range chunk {
			out = append(out, Row{Work: w, Rollup: rolled[w.ID]})
		}
	}
	return out, nil
}
