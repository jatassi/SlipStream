package works

import (
	"context"
	"fmt"
	"slices"
	"strings"
	"unicode"

	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/works/internal/db"
)

// indexTitles writes w's search row: its label and sort label, then every
// title its type's Titles knows it by. converge calls it, so an add and a
// refresh keep the index in their own Write.
func indexTitles(ctx context.Context, tx *storage.Tx[file], c Catalog, id WorkID, label, sortLabel string) error {
	titles, err := c.Titles(ctx, tx, id)
	if err != nil {
		return fmt.Errorf("titles of work %s: %w", id, err)
	}
	labels := []string{label}
	if sortLabel != label {
		labels = append(labels, sortLabel)
	}
	var others []string
	for _, t := range titles {
		if !slices.Contains(labels, t) && !slices.Contains(others, t) {
			others = append(others, t)
		}
	}
	return db.New(tx).PutTitles(ctx, db.PutTitlesParams{WorkID: id, Labels: strings.Join(labels, "\n"), Titles: strings.Join(others, "\n")})
}

// catchUpTitles indexes, in one Write, every Work without a search row,
// as the migration that made the index leaves them. A repeat finds none.
func (w *Works) catchUpTitles(ctx context.Context) error {
	return w.cfg.Library.Write(ctx, func(ctx context.Context, tx *storage.Tx[file]) error {
		rows, err := db.New(tx).ListUntitled(ctx)
		if err != nil {
			return err
		}
		for _, row := range rows {
			if err := indexTitles(ctx, tx, w.types[row.ContentType].catalog, row.ID, row.Label, row.SortLabel); err != nil {
				return fmt.Errorf("content type %s: %w", row.ContentType, err)
			}
		}
		return nil
	})
}

// MatchIn answers the Works of t, or of every type when t is nil, whose
// titles hold each word of q as a word prefix, case and accents ignored,
// best first. q with no word matches nothing.
func (w *Works) MatchIn(ctx context.Context, r storage.Reader[file], q string, t *TypeID) ([]WorkID, error) {
	var only *string
	if t != nil {
		if _, ok := w.types[*t]; !ok {
			return nil, ErrNoType
		}
		only = new(string(*t))
	}
	query, ok := ftsQuery(q)
	if !ok {
		return nil, nil
	}
	return db.New(r).MatchTitles(ctx, db.MatchTitlesParams{Query: query, ContentType: only})
}

// ftsQuery is q as an FTS5 query that every word of q starts a word of a
// match: `amélie po` is `"amélie"* "po"*`. A word is a run of letters,
// digits and marks, as the index's tokenizer reads one, so no FTS5
// operator survives from q.
func ftsQuery(q string) (string, bool) {
	words := strings.FieldsFunc(q, func(r rune) bool { return !unicode.IsLetter(r) && !unicode.IsDigit(r) && !unicode.IsMark(r) })
	if len(words) == 0 {
		return "", false
	}
	for i, word := range words {
		words[i] = `"` + word + `"*`
	}
	return strings.Join(words, " "), true
}
