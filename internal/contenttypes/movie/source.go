package movie

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"

	"github.com/jatassi/slipstream/internal/metadata"
	"github.com/jatassi/slipstream/internal/providers/omdb"
	"github.com/jatassi/slipstream/internal/providers/tmdb"
	"github.com/jatassi/slipstream/internal/works"
)

// Source is the provider half: it reads TMDB and OMDb on any Metadata
// holder.
type Source struct {
	tmdb *tmdb.Client
	omdb *omdb.Client
}

// Fetch reads the movie whose TMDB id is key in one TMDB call, and its
// OMDb ratings and awards when ask asks for ratings and the movie has an
// IMDb id. A title OMDb doesn't know is read with none; an OMDb that
// doesn't answer leaves them as stored.
func (s *Source) Fetch(ctx context.Context, ask metadata.Ask, key works.Key) (works.Fetched, error) {
	id, err := strconv.ParseInt(string(key), 10, 64)
	if err != nil {
		return works.Fetched{}, fmt.Errorf("movie %q: %w", key, works.ErrNotFound)
	}
	tm, err := s.tmdb.Movie(ctx, ask, id)
	if err != nil {
		return works.Fetched{}, err
	}
	r, err := recordOf(tm, ask)
	if err != nil {
		return works.Fetched{}, err
	}
	f := works.Fetched{
		Label:    r.Title,
		Children: []works.Entry{{Key: itemKey, Label: r.Title}},
		Ratings:  tmdbRating(tm),
		Artwork:  artworkOf(tm, ask.Language),
	}
	if ask.Ratings && r.IMDbID != nil {
		t, err := s.omdb.Title(ctx, ask, string(*r.IMDbID))
		switch {
		case err == nil:
			f.Ratings, f.RatingsRead, r.Awards = append(f.Ratings, t.Ratings...), true, &t.Awards
		case errors.Is(err, works.ErrNotFound):
			f.RatingsRead = true
		}
	}
	if f.Detail, err = json.Marshal(r); err != nil {
		return works.Fetched{}, fmt.Errorf("encode movie %s: %w", key, err)
	}
	return f, nil
}

func year(date string) *int64 {
	day, ok := dayOf(date)
	if !ok {
		return nil
	}
	y := int64(day.Year())
	return &y
}
