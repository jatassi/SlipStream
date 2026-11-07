package indexers

import (
	"context"
	"errors"

	"github.com/jatassi/slipstream/internal/cardigann"
	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/indexers/internal/db"
	"github.com/jatassi/slipstream/internal/storage"
)

// AddCustom adds src as a custom definition, or replaces the custom
// definition with its id and moves that definition's Indexers to it. It
// refuses a definition that doesn't load with *cardigann.Unrunnable, and
// an id the catalog lists with ErrDefinitionIDTaken.
func (x *Indexers) AddCustom(ctx context.Context, _ identity.Admin, src []byte) (Entry, error) {
	d, err := cardigann.Load(src)
	if err != nil {
		return Entry{}, err
	}
	id := d.Summary().ID
	if id == "" {
		return Entry{}, &cardigann.Unrunnable{Reason: "it has no id"}
	}
	now := storage.Now()
	sha := blobSHA(src)
	var (
		entry Entry
		moved []ID
	)
	err = x.cfg.Library.Write(ctx, func(ctx context.Context, tx *storage.Tx[file]) error {
		q := db.New(tx)
		_, err := q.GetCatalogEntry(ctx, db.DefinitionID(id))
		switch {
		case err == nil:
			return ErrDefinitionIDTaken
		case !errors.Is(err, storage.ErrNoRows):
			return err
		}
		if err := q.InsertDefinition(ctx, definitionRow(id, sha, id, FromCustom, src, now)); err != nil {
			return err
		}
		if moved, err = x.moveCustomPins(ctx, tx, id, sha, now); err != nil {
			return err
		}
		if err := dropOlderCustom(ctx, q, id, sha); err != nil {
			return err
		}
		def, err := q.GetDefinition(ctx, db.GetDefinitionParams{ID: db.DefinitionID(id), Sha: sha})
		if err != nil {
			return err
		}
		entry, err = customEntry(def)
		return err
	})
	if err != nil {
		return Entry{}, err
	}
	x.catalogChanged(ctx)
	for _, id := range moved {
		x.changed(ctx, id)
	}
	return entry, nil
}

// moveCustomPins moves the Indexers on another version of the custom
// definition id to sha, answering those whose pin moved.
func (x *Indexers) moveCustomPins(ctx context.Context, tx *storage.Tx[file], id, sha string, now storage.Time) ([]ID, error) {
	pins, err := db.New(tx).ListPins(ctx)
	if err != nil {
		return nil, err
	}
	var moved []ID
	for _, p := range pins {
		if Origin(p.Origin) != FromCustom || *p.DefinitionID != id || *p.DefinitionSha == sha {
			continue
		}
		ok, err := x.movePin(ctx, tx, p, sha, now)
		if err != nil {
			return nil, err
		}
		if ok {
			moved = append(moved, p.ID)
		}
	}
	return moved, nil
}

func dropOlderCustom(ctx context.Context, q *db.Queries, id, sha string) error {
	older, err := q.ListCustom(ctx)
	if err != nil {
		return err
	}
	for _, row := range older {
		if string(row.ID) == id && row.Sha != sha {
			if err := q.DeleteDefinition(ctx, db.DeleteDefinitionParams{ID: row.ID, Sha: row.Sha}); err != nil {
				return err
			}
		}
	}
	return nil
}

// DeleteCustom deletes the custom definition id: ErrNoDefinition when
// there is none, ErrDefinitionInUse while an Indexer runs it.
func (x *Indexers) DeleteCustom(ctx context.Context, _ identity.Admin, id string) error {
	err := x.cfg.Library.Write(ctx, func(ctx context.Context, tx *storage.Tx[file]) error {
		q := db.New(tx)
		def, err := q.GetCustom(ctx, db.DefinitionID(id))
		if errors.Is(err, storage.ErrNoRows) {
			return ErrNoDefinition
		}
		if err != nil {
			return err
		}
		pinned, err := q.CountPinned(ctx, db.CountPinnedParams{DefinitionID: &id, DefinitionSha: &def.Sha})
		if err != nil {
			return err
		}
		if pinned > 0 {
			return ErrDefinitionInUse
		}
		return q.DeleteDefinition(ctx, db.DeleteDefinitionParams{ID: def.ID, Sha: def.Sha})
	})
	if err != nil {
		return err
	}
	x.catalogChanged(ctx)
	return nil
}
