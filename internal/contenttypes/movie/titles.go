package movie

import (
	"context"
	"slices"

	"github.com/jatassi/slipstream/internal/contenttypes/movie/internal/db"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/works"
)

// Titles are the movie's title in the metadata language, its English and
// original titles and its alternative titles in every country, each once.
func (*Movie) Titles(ctx context.Context, r storage.Reader[file], w works.WorkID) ([]string, error) {
	q := db.New(r)
	mv, err := q.GetMovie(ctx, w)
	if err != nil {
		return nil, err
	}
	return titlesOf(ctx, q, mv)
}

func titlesOf(ctx context.Context, q *db.Queries, mv db.MovieMovie) ([]string, error) {
	alternatives, err := q.ListTitles(ctx, mv.WorkID)
	if err != nil {
		return nil, err
	}
	var out []string
	add := func(t string) {
		if t != "" && !slices.Contains(out, t) {
			out = append(out, t)
		}
	}
	add(mv.Title)
	add(mv.TitleEnglish)
	add(mv.OriginalTitle)
	for _, t := range alternatives {
		add(t.Title)
	}
	return out, nil
}
