package works

import (
	"cmp"
	"context"
	"slices"

	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/works/internal/db"
)

// Tally is one Content Type's share of a set of Works.
type Tally struct {
	Type  TypeID
	Count int
	// First is at most n labels, lowest sort label first.
	First []string
	IDs   []WorkID
}

// TallyOf counts ws per Content Type and keeps the first n labels and
// every id of each, reading Batch ids at a time. It is ordered by type id.
func TallyOf(ctx context.Context, r storage.Reader[file], ws []WorkID, n int) ([]Tally, error) {
	q := db.New(r)
	type first struct{ sort, label string }
	ids := map[TypeID][]WorkID{}
	firsts := map[TypeID][]first{}
	for chunk := range slices.Chunk(ws, Batch) {
		rows, err := q.ListTallyRows(ctx, chunk)
		if err != nil {
			return nil, err
		}
		for _, row := range rows {
			ids[row.ContentType] = append(ids[row.ContentType], row.ID)
			f := firsts[row.ContentType]
			f = append(f, first{row.SortLabel, row.Label})
			slices.SortFunc(f, func(a, b first) int { return cmp.Compare(a.sort, b.sort) })
			firsts[row.ContentType] = f[:min(len(f), n)]
		}
	}
	out := make([]Tally, 0, len(ids))
	for t, in := range ids {
		labels := make([]string, len(firsts[t]))
		for i, f := range firsts[t] {
			labels[i] = f.label
		}
		out = append(out, Tally{Type: t, Count: len(in), First: labels, IDs: in})
	}
	slices.SortFunc(out, func(a, b Tally) int { return cmp.Compare(a.Type, b.Type) })
	return out, nil
}
