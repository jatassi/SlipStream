package movie

import (
	"context"
	"slices"

	"github.com/jatassi/slipstream/internal/calendar"
	"github.com/jatassi/slipstream/internal/contenttypes/movie/internal/db"
	"github.com/jatassi/slipstream/internal/providers/tmdb"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/works"
)

var (
	cinemaKind   = works.DatedKind{Key: "cinema", Label: "Cinema release"}
	digitalKind  = works.DatedKind{Key: "digital", Label: "Digital release"}
	physicalKind = works.DatedKind{Key: "physical", Label: "Physical release"}
)

// datedKinds maps each Calendar kind to its TMDB kinds. Premiere and TV
// dates stay off the Calendar.
var datedKinds = []struct {
	kind  works.DatedKind
	tmdbs []tmdb.ReleaseType
}{
	{cinemaKind, []tmdb.ReleaseType{tmdb.TheatricalLimited, tmdb.Theatrical}},
	{digitalKind, []tmdb.ReleaseType{tmdb.Digital}},
	{physicalKind, []tmdb.ReleaseType{tmdb.Physical}},
}

// Dated reads the movies with a release of a Calendar kind in s, then
// each of their release dates, and keeps each kind's date the
// availability rule picks, in the region else anywhere, when s holds it.
func (m *Movie) Dated(ctx context.Context, r storage.Reader[file], s calendar.Span) ([]calendar.Dated, error) {
	q := db.New(r)
	candidates, err := q.ListWorksReleasedIn(ctx, db.ListWorksReleasedInParams{FromDay: s.From().String(), ToDay: s.To().String()})
	if err != nil || len(candidates) == 0 {
		return nil, err
	}
	slices.SortFunc(candidates, func(a, b works.WorkID) int { return a.Compare(b.UUID) })
	ids := slices.Compact(candidates)
	of, err := datesOfWorks(ctx, q, ids)
	if err != nil {
		return nil, err
	}
	region, err := m.region(ctx, r)
	if err != nil {
		return nil, err
	}
	items, err := works.ItemsOf(ctx, r, works.Only(ids))
	if err != nil {
		return nil, err
	}
	var out []calendar.Dated
	for _, w := range ids {
		for _, k := range datedKinds {
			b, ok := earliest(of[w], region, k.tmdbs...)
			if !ok {
				continue
			}
			day, err := calendar.ParseDay(b.date)
			if err != nil {
				return nil, err
			}
			if s.Contains(day) {
				out = append(out, calendar.Dated{Work: w, Item: items[w][0].ID, Kind: k.kind.Key, Day: day})
			}
		}
	}
	return out, nil
}

func datesOfWorks(ctx context.Context, q *db.Queries, ids []works.WorkID) (map[works.WorkID][]releaseDate, error) {
	of := map[works.WorkID][]releaseDate{}
	for chunk := range slices.Chunk(ids, works.Batch) {
		rows, err := q.ListReleaseDatesOf(ctx, chunk)
		if err != nil {
			return nil, err
		}
		for _, rel := range rows {
			of[rel.WorkID] = append(of[rel.WorkID], releaseDate{Country: rel.Country, Kind: tmdb.ReleaseType(rel.Kind), Date: rel.Date})
		}
	}
	return of, nil
}
