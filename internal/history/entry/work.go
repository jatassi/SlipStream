package entry

import (
	"context"
	"fmt"

	"github.com/jatassi/slipstream/internal/metadata/artwork"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/works"
)

// Work is a Work as facts: enough to name it, link it and show its poster
// after it is gone.
type Work struct {
	ID     works.WorkID `json:"id"`
	Label  string       `json:"label"`
	Type   works.TypeID `json:"type"`
	Nouns  Nouns        `json:"nouns"`
	Poster artwork.Ref  `json:"poster"`
}

// Nouns are the Content Type's words, kept so a render never asks for the
// type.
type Nouns struct {
	Work   works.Noun   `json:"work"`
	Levels []works.Noun `json:"levels"`
	Item   works.Noun   `json:"item"`
	// Single: one Item per Work and no levels, the single-file shape.
	Single bool `json:"single"`
}

// Describer is what entry reads of the Content Types and the Works:
// *works.Works.
type Describer interface {
	Describe(t works.TypeID) (works.Description, bool)
	RowsIn(ctx context.Context, r storage.Reader[storage.Library], s works.Scope) ([]works.Row, error)
}

// WorkIn reads w's facts through r, a Write's tx included.
func WorkIn(ctx context.Context, r storage.Reader[storage.Library], d Describer, w works.WorkID) (Work, error) {
	rows, err := d.RowsIn(ctx, r, works.Scope{Works: []works.WorkID{w}})
	if err != nil {
		return Work{}, err
	}
	if len(rows) == 0 {
		return Work{}, fmt.Errorf("read the Work %s: %w", w, storage.ErrNoRows)
	}
	row := rows[0]
	return Work{ID: row.ID, Label: row.Label, Type: row.ContentType, Nouns: NounsOf(d, row.ContentType), Poster: row.Poster}, nil
}

// NounsOf is t's nouns.
func NounsOf(d Describer, t works.TypeID) Nouns {
	desc, _ := d.Describe(t)
	return Nouns{
		Work:   desc.Work,
		Levels: desc.Levels,
		Item:   desc.Item,
		Single: desc.Items == works.ItemsOne && len(desc.Levels) == 0,
	}
}
