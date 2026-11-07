package works

import (
	"bytes"
	"context"
	"errors"
	"fmt"

	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/setting"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/works/internal/db"
)

// Optioning is a change to a Work's options; a nil or empty part keeps
// it.
type Optioning struct {
	LevelFolders *bool
	// Type is the type's own options to change, checked by its
	// PatchOptions.
	Type setting.Patch
}

// OptionsChange is a Work's options, and how many of its live Items a
// change to them withdraws: Items the new options' fetch doesn't list,
// which keep their files and show as no longer listed.
type OptionsChange struct {
	LevelFolders bool
	Fields       []setting.Field
	Withdraws    int
}

// SetOptions changes Work id's options. Changed type options refetch the
// Work with them, outside the Write, then converge in one Write with the
// level folder switch, so Items keep their ids and move between levels.
// preview rolls the Write back. Unchanged options change nothing and
// fetch nothing. ErrNoWork, ErrOrderUnavailable, *setting.RangeError, and
// the fetch's source errors.
func (w *Works) SetOptions(ctx context.Context, _ identity.Admin, id WorkID, o Optioning, preview bool) (OptionsChange, error) {
	row, current, next, err := w.patchOptions(ctx, id, o.Type)
	if err != nil {
		return OptionsChange{}, err
	}
	folders := row.LevelFolders
	if o.LevelFolders != nil {
		folders = *o.LevelFolders
	}
	refetch := !bytes.Equal(next, current)
	if !refetch && folders == row.LevelFolders {
		var out OptionsChange
		err := w.cfg.Library.Read(ctx, func(ctx context.Context, r storage.Reader[file]) error {
			out, err = w.OptionsOf(ctx, r, id)
			return err
		})
		return out, err
	}
	reg := w.types[row.ContentType]
	var p *parsed
	if refetch {
		fetched, err := w.fetch(ctx, row.ContentType, reg, row.Key, Fetching{Purpose: ForRefresh, RatingsReadAt: row.RatingsReadAt, Options: next})
		if err != nil {
			return OptionsChange{}, err
		}
		p = &fetched
	}
	var out OptionsChange
	err = w.cfg.Library.Write(ctx, func(ctx context.Context, tx *storage.Tx[file]) error {
		if err := db.New(tx).SetLevelFolders(ctx, db.SetLevelFoldersParams{LevelFolders: folders, ID: id}); err != nil {
			return err
		}
		withdraws, err := withdrawnBy(ctx, tx, reg.catalog, id, p)
		if err != nil {
			return err
		}
		if out, err = w.OptionsOf(ctx, tx, id); err != nil {
			return err
		}
		out.Withdraws = withdraws
		if preview {
			return errPreview
		}
		return nil
	})
	if errors.Is(err, errPreview) {
		return out, nil
	}
	if err != nil {
		return OptionsChange{}, err
	}
	w.cfg.Changed.Publish(ctx, ChangedOf(id))
	return out, nil
}

// patchOptions reads Work id and its fetch options now and as patch would
// make them, in one Read.
func (w *Works) patchOptions(ctx context.Context, id WorkID, patch setting.Patch) (row db.WorksWork, current, next []byte, err error) {
	err = w.cfg.Library.Read(ctx, func(ctx context.Context, r storage.Reader[file]) error {
		row, err = db.New(r).GetWork(ctx, id)
		if errors.Is(err, storage.ErrNoRows) {
			return ErrNoWork
		}
		if err != nil {
			return err
		}
		c := w.types[row.ContentType].catalog
		if current, err = c.FetchOptions(ctx, r, id); err != nil {
			return fmt.Errorf("content type %s: fetch options of work %s: %w", row.ContentType, id, err)
		}
		next, err = c.PatchOptions(ctx, r, id, patch)
		return err
	})
	return row, current, next, err
}

// withdrawnBy converges Work id on p and counts the Items live before
// that p withdrew; none when p is nil.
func withdrawnBy(ctx context.Context, tx *storage.Tx[file], c Catalog, id WorkID, p *parsed) (int, error) {
	if p == nil {
		return 0, nil
	}
	q := db.New(tx)
	before, err := liveItems(ctx, q, id)
	if err != nil {
		return 0, err
	}
	if _, err := converge(ctx, tx, c, id, *p, storage.Now()); err != nil {
		return 0, err
	}
	after, err := liveItems(ctx, q, id)
	if err != nil {
		return 0, err
	}
	n := 0
	for it := range before {
		if !after[it] {
			n++
		}
	}
	return n, nil
}

// liveItems is the set of Work id's live Items.
func liveItems(ctx context.Context, q *db.Queries, id WorkID) (map[ItemID]bool, error) {
	items, err := q.ListItems(ctx, id)
	if err != nil {
		return nil, err
	}
	out := make(map[ItemID]bool, len(items))
	for _, it := range items {
		if it.WithdrawnAt.IsZero() {
			out[it.ID] = true
		}
	}
	return out, nil
}

// OptionsOf reads Work id's options in r. ErrNoWork.
func (w *Works) OptionsOf(ctx context.Context, r storage.Reader[file], id WorkID) (OptionsChange, error) {
	row, err := db.New(r).GetWork(ctx, id)
	if errors.Is(err, storage.ErrNoRows) {
		return OptionsChange{}, ErrNoWork
	}
	if err != nil {
		return OptionsChange{}, err
	}
	fields, err := w.types[row.ContentType].catalog.OptionFields(ctx, r, id)
	if err != nil {
		return OptionsChange{}, fmt.Errorf("content type %s: options of work %s: %w", row.ContentType, id, err)
	}
	return OptionsChange{LevelFolders: row.LevelFolders, Fields: fields}, nil
}
