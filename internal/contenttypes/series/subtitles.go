package series

import (
	"context"
	"fmt"
	"strconv"

	"github.com/jatassi/slipstream/internal/contenttypes/series/internal/db"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/subtitles"
	"github.com/jatassi/slipstream/internal/subtitling"
	"github.com/jatassi/slipstream/internal/works"
)

// Subtitled searches a series' files as episodes.
func (*Series) Subtitled() []subtitling.Kind { return []subtitling.Kind{subtitling.Episode} }

// SubtitleSubject is the series' ids, English title and first year, with
// the Aired season and numbers of items: providers number by the Aired
// order whatever order the series is set to.
func (*Series) SubtitleSubject(ctx context.Context, r storage.Reader[file], work works.WorkID, items []works.ItemID) (subtitling.Subject, error) {
	q := db.New(r)
	row, err := q.GetSeries(ctx, work)
	if err != nil {
		return subtitling.Subject{}, err
	}
	s := subtitling.Subject{Kind: subtitling.Episode, TVDB: strconv.FormatInt(int64(row.TvdbID), 10), Title: row.EnglishTitle}
	if row.TmdbID != nil {
		s.TMDB = strconv.FormatInt(int64(*row.TmdbID), 10)
	}
	if row.ImdbID != nil {
		s.IMDb = *row.ImdbID
	}
	if row.FirstAired != nil {
		first, err := works.ParseDate(*row.FirstAired)
		if err != nil {
			return subtitling.Subject{}, fmt.Errorf("series %d: %w", row.TvdbID, err)
		}
		s.Year = first.Year
	}
	for _, it := range items {
		e, err := q.GetEpisode(ctx, it)
		if err != nil {
			return subtitling.Subject{}, err
		}
		if e.AiredSeason == nil || e.AiredNumber == nil {
			return subtitling.Subject{}, subtitles.ErrNoAiredNumber
		}
		s.Season = int(*e.AiredSeason)
		s.Episodes = append(s.Episodes, int(*e.AiredNumber))
	}
	return s, nil
}
