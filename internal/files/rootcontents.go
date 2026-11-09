package files

import (
	"context"
	"fmt"

	"github.com/jatassi/slipstream/internal/files/internal/db"
	"github.com/jatassi/slipstream/internal/roots"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/storing"
	"github.com/jatassi/slipstream/internal/works"
)

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

func (RootContents) Proof(ctx context.Context, r storage.Reader[file], root roots.ID) ([]string, error) {
	return db.New(r).ProofOfRoot(ctx, root)
}

func (RootContents) Recorded(ctx context.Context, r storage.Reader[file], root roots.ID) ([]storing.Recorded, error) {
	rows, err := db.New(r).RecordedOfRoot(ctx, root)
	if err != nil {
		return nil, err
	}
	out := make([]storing.Recorded, len(rows))
	for i, row := range rows {
		out[i].Size = row.Size
		if out[i].Path, out[i].Sits, out[i].Next, err = placesOf(row.ID, row.Path, row.Sits, row.Ring); err != nil {
			return nil, fmt.Errorf("file %s of root %s: %w", row.ID, root, err)
		}
	}
	return out, nil
}

// placesOf is where the bytes of file id, recorded at p, belong, where
// they sit and, while a move is pending, where the disk queue takes them
// next.
func placesOf(id ID, p string, sits *string, ring storage.UUID) (at, now, next storing.Rel, err error) {
	if at, err = storing.ParseRel(p); err != nil || sits == nil {
		return at, at, storing.Rel{}, err
	}
	if now, err = storing.ParseRel(*sits); err != nil {
		return storing.Rel{}, storing.Rel{}, storing.Rel{}, err
	}
	next, err = storing.ParseRel(move{file: id, path: p, sits: *sits, ring: ring}.next())
	return at, now, next, err
}

func (RootContents) ForgetGone(ctx context.Context, tx *storage.Tx[file], root roots.ID) error {
	return db.New(tx).ForgetGoneOfRoot(ctx, root)
}

func (RootContents) ForgetGoneOf(ctx context.Context, tx *storage.Tx[file], work works.WorkID) error {
	return db.New(tx).ForgetGoneOfWork(ctx, work)
}

func (RootContents) Forget(ctx context.Context, tx *storage.Tx[file], root roots.ID, paths []storing.Rel) error {
	q := db.New(tx)
	rows, err := q.RecordedOfRoot(ctx, root)
	if err != nil {
		return err
	}
	at := make(map[string]db.FileID, len(rows))
	for _, row := range rows {
		at[row.Path] = row.ID
	}
	for _, p := range paths {
		id, ok := at[p.String()]
		if !ok {
			continue
		}
		if err := q.DeleteFile(ctx, id); err != nil {
			return err
		}
	}
	return nil
}
