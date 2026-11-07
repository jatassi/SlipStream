package works

import (
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
func (w *Works) SetOptions(ctx context.Context, by identity.Admin, id WorkID, o Optioning, preview bool) (OptionsChange, error) {
	return OptionsChange{}, errors.New("not implemented")
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
