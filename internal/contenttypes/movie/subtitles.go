package movie

import (
	"context"
	"strconv"

	"github.com/jatassi/slipstream/internal/contenttypes/movie/internal/db"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/subtitling"
	"github.com/jatassi/slipstream/internal/works"
)

// Subtitled searches a movie's files as features.
func (*Movie) Subtitled() []subtitling.Kind { return []subtitling.Kind{subtitling.Feature} }

// SubtitleSubject is the movie's ids, English title and year.
func (*Movie) SubtitleSubject(ctx context.Context, r storage.Reader[file], work works.WorkID, _ []works.ItemID) (subtitling.Subject, error) {
	mv, err := db.New(r).GetMovie(ctx, work)
	if err != nil {
		return subtitling.Subject{}, err
	}
	s := subtitling.Subject{Kind: subtitling.Feature, TMDB: strconv.FormatInt(int64(mv.TmdbID), 10), Title: mv.TitleEnglish}
	if mv.ImdbID != nil {
		s.IMDb = string(*mv.ImdbID)
	}
	if mv.Year != nil {
		s.Year = int(*mv.Year)
	}
	return s, nil
}
