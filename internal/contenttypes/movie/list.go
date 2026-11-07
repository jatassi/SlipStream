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
func (*Movie) ListFields(ctx context.Context, r storage.Reader[file], ws works.Among[works.WorkID]) (map[works.WorkID][]works.FieldValue, error) {
	q := db.New(r)
	movies, err := works.Read(ws, func() ([]db.ListEveryListedRow, error) { return q.ListEveryListed(ctx) }, func(ids []works.WorkID) ([]db.ListEveryListedRow, error) {
		rows, err := q.ListListedOf(ctx, ids)
		return as(rows, func(r db.ListListedOfRow) db.ListEveryListedRow { return db.ListEveryListedRow(r) }), err
	})
	if err != nil {
		return nil, err
	}
	genres, err := works.Read(ws, func() ([]db.ListEveryGenreRow, error) { return q.ListEveryGenre(ctx) }, func(ids []works.WorkID) ([]db.ListEveryGenreRow, error) {
		rows, err := q.ListGenresOf(ctx, ids)
		return as(rows, func(r db.ListGenresOfRow) db.ListEveryGenreRow { return db.ListEveryGenreRow(r) }), err
	})
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

// as converts each of rows, a query's row type, to another of its shape.
func as[F, T any](rows []F, to func(F) T) []T {
	out := make([]T, len(rows))
	for i, r := range rows {
		out[i] = to(r)
	}
	return out
}
