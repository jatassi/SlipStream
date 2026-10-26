package movie

import (
	"context"
	"time"

	"github.com/jatassi/slipstream/internal/contenttypes/movie/internal/db"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/works"
)

// ListFields reads year, runtime, certification and genres for every one
// of ws in two queries.
func (*Movie) ListFields(ctx context.Context, r storage.Reader[file], ws []works.WorkID) (map[works.WorkID][]works.FieldValue, error) {
	q := db.New(r)
	movies, err := q.ListMovies(ctx, ws)
	if err != nil {
		return nil, err
	}
	genres, err := q.ListGenresOf(ctx, ws)
	if err != nil {
		return nil, err
	}
	of := map[works.WorkID][]string{}
	for _, g := range genres {
		of[g.WorkID] = append(of[g.WorkID], g.Name)
	}
	out := make(map[works.WorkID][]works.FieldValue, len(movies))
	for _, mv := range movies {
		var fields []works.FieldValue
		if mv.Year != nil {
			fields = append(fields, works.NumberField("year", float64(*mv.Year)))
		}
		if mv.RuntimeMinutes != nil {
			fields = append(fields, works.DurationField("runtime", time.Duration(*mv.RuntimeMinutes)*time.Minute))
		}
		if mv.Certification != nil {
			fields = append(fields, works.TextField("certification", *mv.Certification))
		}
		if g := of[mv.WorkID]; len(g) > 0 {
			fields = append(fields, works.TextsField("genres", g))
		}
		out[mv.WorkID] = fields
	}
	return out, nil
}
