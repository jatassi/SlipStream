package library

import (
	"context"

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
	bares := make([]works.Bare, len(ws))
	ids := make([]works.WorkID, len(ws))
	for i, w := range ws {
		bares[i], ids[i] = w.Bare, w.ID
	}
	rolled, err := l.cfg.Files.Rollups(ctx, r, bares, s.Among(ids))
	if err != nil {
		return nil, err
	}
	out := make([]Row, len(ws))
	for i, w := range ws {
		out[i] = Row{Work: w, Rollup: rolled[i]}
	}
	return out, nil
}
