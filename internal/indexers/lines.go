package indexers

import (
	"cmp"
	"context"
	"slices"

	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/works"
)

// Lines is the lines of one search of kind k of Content Type typ, read in
// r, by priority then name, with no query reserved: for AutoSearch, every
// Indexer that is on, has Automatic search on and isn't an RSS feed; for
// ManualSearch, every one that is on, an RSS feed of typ skipped as
// RSSOnly. One that doesn't serve typ, or that the gate holds back now, is
// skipped with why. Reserve reserves each asked line when it dispatches.
func (x *Indexers) Lines(ctx context.Context, r storage.Reader[file], typ works.TypeID, k SearchKind) ([]Line, error) {
	lines, _, err := x.lines(ctx, r, typ, k)
	return lines, err
}

func (x *Indexers) lines(ctx context.Context, r storage.Reader[file], typ works.TypeID, k SearchKind) ([]Line, map[ID]Indexer, error) {
	all, err := x.all(ctx, r)
	if err != nil {
		return nil, nil, err
	}
	statuses, err := statusesIn(ctx, r)
	if err != nil {
		return nil, nil, err
	}
	slices.SortStableFunc(all, func(a, b Indexer) int { return cmp.Compare(a.Priority, b.Priority) })
	cats := x.categories(typ)
	now := storage.Now().Time()
	byID := map[ID]Indexer{}
	var lines []Line
	for _, idx := range all {
		if !idx.On || (k == AutoSearch && (!idx.AutoSearch || idx.Kind == RSS)) {
			continue
		}
		byID[idx.ID] = idx
		l := Line{Indexer: idx.ID, Name: idx.Name}
		switch {
		case idx.Kind == RSS:
			if !slices.Contains(idx.Media, typ) {
				continue
			}
			l.Skipped = &Outcome{Kind: RSSOnly}
		case len(served(idx, cats)) == 0:
			l.Skipped = &Outcome{Kind: NoCategories}
		default:
			_, l.Skipped = gate(idx, statuses[idx.ID], Usage{}, x.cfg.Roster, UseSearch, now)
		}
		lines = append(lines, l)
	}
	return lines, byID, nil
}
