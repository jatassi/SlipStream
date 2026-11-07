package series

import (
	"context"
	"regexp"
	"strconv"
	"strings"

	"github.com/jatassi/slipstream/internal/contenttypes/series/internal/db"
	"github.com/jatassi/slipstream/internal/indexers"
	"github.com/jatassi/slipstream/internal/newznab"
	"github.com/jatassi/slipstream/internal/providers/tvdb"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/works"
)

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
	at := searchedAs(tvdb.Order(row.EpisodeOrder), e)
	return indexers.Terms{
		Mode:    "tvsearch",
		IDs:     map[string]string{"tvdbid": strconv.FormatInt(int64(row.TvdbID), 10)},
		Query:   row.EnglishTitle,
		Season:  new(at.season),
		Episode: strconv.Itoa(at.episode),
		Covers:  at.covers,
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

// marker is a title's first season and episode marker: S01E03, a
// multi-episode S01E03E04 or S01E03-E04, or a season pack's S01.
var marker = regexp.MustCompile(`(?i)\bS(\d{1,4})(?:E(\d{1,4})((?:-?E\d{1,4})*))?\b`)

// covers judges a release title against e: e alone, another episode or
// none, or more than one episode, by the title's first marker.
func (e episodeAt) covers(title string) indexers.Covering {
	m := marker.FindStringSubmatch(title)
	if m == nil {
		return indexers.Covering{Kind: indexers.CoversOther}
	}
	names := strings.ToUpper(m[0])
	if m[2] == "" || m[3] != "" {
		return indexers.Covering{Kind: indexers.CoversMore, Names: names}
	}
	season, _ := strconv.Atoi(m[1])
	episode, _ := strconv.Atoi(m[2])
	if (episodeAt{season, episode}) != e {
		return indexers.Covering{Kind: indexers.CoversOther, Names: names}
	}
	return indexers.Covering{Kind: indexers.CoversItem}
}
