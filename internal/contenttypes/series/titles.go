package series

import (
	"context"
	"fmt"
	"slices"

	"github.com/jatassi/slipstream/internal/contenttypes/series/internal/db"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/works"
)

// Titles are the series' English and original titles and its aliases,
// each once.
func (*Series) Titles(ctx context.Context, r storage.Reader[file], w works.WorkID) ([]string, error) {
	row, err := db.New(r).GetSeries(ctx, w)
	if err != nil {
		return nil, err
	}
	var aliases []string
	if err := unblob(row.Aliases, &aliases); err != nil {
		return nil, fmt.Errorf("series %d: %w", row.TvdbID, err)
	}
	out := []string{row.EnglishTitle}
	for _, t := range append([]string{deref(row.OriginalTitle)}, aliases...) {
		if t != "" && !slices.Contains(out, t) {
			out = append(out, t)
		}
	}
	return out, nil
}

func deref(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
