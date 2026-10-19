package movie

import (
	"context"
	"encoding/json"
	"fmt"
	"slices"
	"strconv"
	"time"

	"github.com/jatassi/slipstream/internal/contenttypes/movie/internal/db"
	"github.com/jatassi/slipstream/internal/metadata"
	"github.com/jatassi/slipstream/internal/providers/tmdb"
	"github.com/jatassi/slipstream/internal/works"
)

// Source is the provider half: it reads TMDB on any Metadata holder.
type Source struct {
	tmdb *tmdb.Client
}

// Search answers TMDB's first page, whose 20 results fit works' 25.
func (s *Source) Search(ctx context.Context, ask metadata.Ask, query string) ([]works.Candidate, error) {
	found, err := s.tmdb.SearchMovies(ctx, ask, query, 0, 1)
	if err != nil {
		return nil, err
	}
	out := make([]works.Candidate, len(found.Results))
	for i, r := range found.Results {
		out[i] = works.Candidate{Key: works.Key(strconv.FormatInt(r.ID, 10)), Label: r.Title}
		if y := year(r.ReleaseDate); y != nil {
			out[i].Fields = []works.FieldValue{works.NumberField("year", float64(*y))}
		}
	}
	return out, nil
}

// Fetch reads the movie whose TMDB id is key.
func (s *Source) Fetch(ctx context.Context, ask metadata.Ask, key works.Key) (works.Fetched, error) {
	id, err := strconv.ParseInt(string(key), 10, 64)
	if err != nil {
		return works.Fetched{}, fmt.Errorf("movie %q: %w", key, works.ErrNotFound)
	}
	tm, err := s.tmdb.Movie(ctx, ask, id)
	if err != nil {
		return works.Fetched{}, err
	}
	d := detail{TMDBID: db.TMDBID(tm.ID), Title: tm.Title, Year: year(tm.ReleaseDate)}
	if tm.ExternalIDs.IMDbID != "" {
		id := db.IMDbID(tm.ExternalIDs.IMDbID)
		d.IMDbID = &id
	}
	if released := released(tm); released != "" {
		d.Released = &released
	}
	if tm.Runtime > 0 {
		d.RuntimeMinutes = &tm.Runtime
	}
	f := works.Fetched{Label: tm.Title, Children: []works.Entry{{Key: itemKey, Label: tm.Title}}}
	if f.Detail, err = json.Marshal(d); err != nil {
		return works.Fetched{}, fmt.Errorf("encode movie %s: %w", key, err)
	}
	return f, nil
}

// released is the earliest cinema or digital date in any country, as
// YYYY-MM-DD, or empty when TMDB lists none.
func released(m tmdb.Movie) string {
	var dates []string
	for _, c := range m.ReleaseDates.Results {
		for _, r := range c.Releases {
			if (r.Type == tmdb.TheatricalLimited || r.Type == tmdb.Theatrical || r.Type == tmdb.Digital) && len(r.Date) >= len(time.DateOnly) {
				dates = append(dates, r.Date[:len(time.DateOnly)])
			}
		}
	}
	if len(dates) == 0 {
		return ""
	}
	return slices.Min(dates)
}

func year(date string) *int64 {
	t, err := time.Parse(time.DateOnly, date)
	if err != nil {
		return nil
	}
	y := int64(t.Year())
	return &y
}
