package series

import (
	"context"
	"fmt"
	"strconv"

	"github.com/jatassi/slipstream/internal/contenttypes/series/internal/db"
	"github.com/jatassi/slipstream/internal/indexers"
	"github.com/jatassi/slipstream/internal/newznab"
	"github.com/jatassi/slipstream/internal/providers/tvdb"
	"github.com/jatassi/slipstream/internal/release"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/works"
)

// Expect is the series' TVDB id, titles, first-aired year and original
// language, with the type's year slack, and covers the stored episodes a
// name's marks name.
func (*Series) Expect(ctx context.Context, r storage.Reader[file], work works.WorkID) (indexers.Expect, error) {
	q := db.New(r)
	row, err := q.GetSeries(ctx, work)
	if err != nil {
		return indexers.Expect{}, err
	}
	return expect(ctx, q, row)
}

func expect(ctx context.Context, q *db.Queries, row db.SeriesSeries) (indexers.Expect, error) {
	eps, err := q.ListEpisodes(ctx, row.WorkID)
	if err != nil {
		return indexers.Expect{}, err
	}
	settings, err := q.GetSettings(ctx)
	if err != nil {
		return indexers.Expect{}, err
	}
	titles, err := titlesOf(row)
	if err != nil {
		return indexers.Expect{}, err
	}
	claim := release.Claim{
		IDs:    map[string]string{"tvdbid": strconv.FormatInt(int64(row.TvdbID), 10)},
		Titles: titles,
		Slack:  int(YearSlack.Value(settings.YearSlack)),
	}
	if row.FirstAired != nil {
		first, err := works.ParseDate(*row.FirstAired)
		if err != nil {
			return indexers.Expect{}, fmt.Errorf("series %d: %w", row.TvdbID, err)
		}
		claim.Year = release.Known(first.Year)
	}
	ex := indexers.Expect{Claim: claim, Covers: episodesOf(tvdb.Order(row.EpisodeOrder), eps).covered}
	if row.OriginalLanguage != nil {
		if l, err := release.ParseLanguage(*row.OriginalLanguage); err == nil {
			ex.Original = release.Known(l)
		}
	}
	return ex, nil
}

// SearchTerms search tvsearch by the series' TVDB id with the episode's
// season and number, else by its English title, which the free-text search
// follows with the episode, such as S01E03.
func (*Series) SearchTerms(ctx context.Context, r storage.Reader[file], item works.ItemID) (indexers.Terms, error) {
	work, _, err := works.ItemOf(ctx, r, item)
	if err != nil {
		return indexers.Terms{}, err
	}
	q := db.New(r)
	row, err := q.GetSeries(ctx, work.ID)
	if err != nil {
		return indexers.Terms{}, err
	}
	e, err := q.GetEpisode(ctx, item)
	if err != nil {
		return indexers.Terms{}, err
	}
	ex, err := expect(ctx, q, row)
	if err != nil {
		return indexers.Terms{}, err
	}
	at := searchedAs(tvdb.Order(row.EpisodeOrder), e)
	return indexers.Terms{
		Expect:  ex,
		Mode:    "tvsearch",
		Query:   row.EnglishTitle,
		Season:  new(at.season),
		Episode: strconv.Itoa(at.episode),
	}, nil
}

// Categories are the TV Newznab family.
func (*Series) Categories() []newznab.ID { return newznab.Family(5000) }

// episodeAt is an episode as releases name it: its season and its number
// in that season.
type episodeAt struct{ season, episode int }

// searchedAs is e's numbers as releases name them: its own in Aired and
// DVD order, and its Aired numbers in Absolute order, which numbers the
// main run as one season no release names. An episode Aired order doesn't
// list keeps its own.
func searchedAs(order tvdb.Order, e db.SeriesEpisode) episodeAt {
	if order == tvdb.OrderAbsolute && e.AiredSeason != nil && e.AiredNumber != nil {
		return episodeAt{int(*e.AiredSeason), int(*e.AiredNumber)}
	}
	return episodeAt{int(e.SeasonNumber), int(e.EpisodeNumber)}
}
