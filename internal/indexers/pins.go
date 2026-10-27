package indexers

import (
	"cmp"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"slices"

	"github.com/jatassi/slipstream/internal/cardigann"
	"github.com/jatassi/slipstream/internal/indexers/internal/db"
	"github.com/jatassi/slipstream/internal/storage"
)

// movePins moves every Indexer pinned to a catalog version whose id the
// catalog now lists at a version that loads, inside tx. A version that
// doesn't load leaves the pin, and Health says so.
func (x *Indexers) movePins(ctx context.Context, tx *storage.Tx[file], now storage.Time) ([]ID, error) {
	q := db.New(tx)
	pins, err := q.ListPins(ctx)
	if err != nil {
		return nil, err
	}
	var moved []ID
	for _, p := range pins {
		if Origin(p.Origin) != FromCatalog {
			continue
		}
		entry, err := q.GetCatalogEntry(ctx, db.DefinitionID(*p.DefinitionID))
		switch {
		case errors.Is(err, storage.ErrNoRows):
			continue
		case err != nil:
			return nil, err
		case entry.Sha == *p.DefinitionSha:
			continue
		}
		ok, err := x.movePin(ctx, tx, p, entry.Sha, now)
		if err != nil {
			return nil, err
		}
		if ok {
			moved = append(moved, p.ID)
		}
	}
	return moved, nil
}

// movePin moves p to version sha of its definition when that version
// loads, carrying its settings over by name. It reports whether it moved.
func (x *Indexers) movePin(ctx context.Context, tx *storage.Tx[file], p db.ListPinsRow, sha string, now storage.Time) (bool, error) {
	q := db.New(tx)
	from, err := q.GetDefinition(ctx, db.GetDefinitionParams{ID: db.DefinitionID(*p.DefinitionID), Sha: *p.DefinitionSha})
	if err != nil {
		return false, fmt.Errorf("read Indexer %s's definition: %w", p.Name, err)
	}
	to, err := q.GetDefinition(ctx, db.GetDefinitionParams{ID: db.DefinitionID(*p.DefinitionID), Sha: sha})
	if err != nil {
		return false, fmt.Errorf("read Indexer %s's new definition: %w", p.Name, err)
	}
	toSummary, loads, err := summaryOf(to)
	if err != nil || !loads {
		return false, err
	}
	fromSummary, _, err := summaryOf(from)
	if err != nil {
		return false, err
	}
	var plain map[string]string
	if err := decode(p.Plain, &plain); err != nil {
		return false, fmt.Errorf("indexer %s: plain: %w", p.Name, err)
	}
	secrets, err := x.open(Indexer{ID: p.ID, Secrets: p.SecretsSealed})
	if err != nil {
		return false, err
	}
	c := carry(plain, secrets.Fields, fromSummary, toSummary)
	secrets.Fields = c.sealed
	sealed, err := x.seal(p.ID, secrets)
	if err != nil {
		return false, err
	}
	plainJSON, err := json.Marshal(c.plain)
	if err != nil {
		return false, fmt.Errorf("encode the plain settings: %w", err)
	}
	notes, err := json.Marshal(append([]string{}, c.notes...))
	if err != nil {
		return false, fmt.Errorf("encode the notes: %w", err)
	}
	n, err := q.MovePin(ctx, db.MovePinParams{
		ID: p.ID, FromSha: p.DefinitionSha, Sha: &sha, Plain: string(plainJSON), SecretsSealed: sealed, Notes: string(notes), UpdatedAt: now,
	})
	return n == 1, err
}

// carried is an Indexer's settings under a new definition version, and
// what the move changed that the Indexer page notes.
type carried struct {
	plain, sealed map[string]string
	notes         []string
}

// carry takes each setting of to by name from plain or sealed. A setting
// to lacks is dropped. A select whose stored option to no longer offers
// takes to's default, with a note.
func carry(plain, sealed map[string]string, from, to cardigann.Summary) carried {
	out := carried{plain: map[string]string{}, sealed: map[string]string{}}
	for _, f := range to.Fields {
		if !stored(f) {
			continue
		}
		v, ok := plain[f.Name]
		if !ok {
			v, ok = sealed[f.Name]
		}
		if !ok {
			continue
		}
		if f.Type == cardigann.SelectField && optionLabel(f, v) == "" {
			was := v
			if i := slices.IndexFunc(from.Fields, func(g cardigann.Field) bool { return g.Name == f.Name }); i >= 0 {
				was = cmp.Or(optionLabel(from.Fields[i], v), v)
			}
			out.notes = append(out.notes, fmt.Sprintf("The definition's new version no longer offers %q for %s. The Indexer now uses its default, %q.",
				was, f.Label, cmp.Or(optionLabel(f, f.Default), f.Default)))
			continue
		}
		if Sealed(f) {
			out.sealed[f.Name] = v
		} else {
			out.plain[f.Name] = v
		}
	}
	return out
}

// stored reports whether an Indexer stores a value for f. Info fields are
// help text, and a captcha is answered at each sign-in.
func stored(f cardigann.Field) bool {
	switch f.Type {
	case cardigann.TextField, cardigann.PasswordField, cardigann.CheckboxField, cardigann.SelectField:
		return true
	}
	return false
}

// Sealed reports whether an Indexer seals f's value: every text and
// password field but the username, since most API key fields are text.
func Sealed(f cardigann.Field) bool {
	return (f.Type == cardigann.TextField || f.Type == cardigann.PasswordField) && f.Name != "username"
}

func optionLabel(f cardigann.Field, value string) string {
	for _, o := range f.Options {
		if o.Value == value {
			return o.Label
		}
	}
	return ""
}
