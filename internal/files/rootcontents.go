package files

import (
	"context"

	"github.com/jatassi/slipstream/internal/files/internal/db"
	"github.com/jatassi/slipstream/internal/roots"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/works"
)

// RootContents is what the files know about each root.
type RootContents struct{}

func (RootContents) Use(ctx context.Context, r storage.Reader[file], ids []roots.ID) (map[roots.ID]roots.Use, error) {
	q := db.New(r)
	live, err := q.UseOfRoots(ctx, ids)
	if err != nil {
		return nil, err
	}
	doomed, err := q.DoomedOfRoots(ctx, ids)
	if err != nil {
		return nil, err
	}
	out := make(map[roots.ID]roots.Use, len(ids))
	for _, row := range live {
		out[row.RootID] = roots.Use{Files: row.Files, Bytes: row.Bytes}
	}
	for _, row := range doomed {
		u := out[row.RootID]
		u.Files += row.Files
		out[row.RootID] = u
	}
	return out, nil
}

func (RootContents) HasFiles(ctx context.Context, r storage.Reader[file], work works.WorkID) (bool, error) {
	return db.New(r).HasFiles(ctx, work)
}
