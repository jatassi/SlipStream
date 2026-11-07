package series

import (
	"context"
	"strconv"

	"github.com/jatassi/slipstream/internal/contenttypes/series/internal/db"
	"github.com/jatassi/slipstream/internal/naming"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/works"
)

// The folder templates of a season and of the specials. naming.Templates
// has no level folders yet, so DefaultTemplates can't carry them.
const (
	SeasonFolder   = "Season {season:00}"
	SpecialsFolder = "Specials"
)

// NamingTokens name an episode's files after its series' English title and
// first aired year, its numbers in the series' order and its title.
func (*Series) NamingTokens(ctx context.Context, r storage.Reader[file], item works.ItemID) (map[string]string, error) {
	work, it, err := works.ItemOf(ctx, r, item)
	if err != nil {
		return nil, err
	}
	q := db.New(r)
	row, err := q.GetSeries(ctx, work.ID)
	if err != nil {
		return nil, err
	}
	e, err := q.GetEpisode(ctx, item)
	if err != nil {
		return nil, err
	}
	tokens := map[string]string{
		"Series Title": row.EnglishTitle, "year": "", "Episode Title": it.Label, "absolute": "",
		"season": strconv.FormatInt(e.SeasonNumber, 10), "episode": strconv.FormatInt(e.EpisodeNumber, 10),
	}
	if row.FirstAired != nil {
		first, err := works.ParseDate(*row.FirstAired)
		if err != nil {
			return nil, err
		}
		tokens["year"] = strconv.Itoa(first.Year)
	}
	if e.AbsoluteNumber != nil {
		tokens["absolute"] = strconv.FormatInt(*e.AbsoluteNumber, 10)
	}
	return tokens, nil
}

func (*Series) DefaultTemplates() naming.Templates {
	return naming.Templates{
		Folder: "{Series Title} ({year})",
		File:   "{Series Title} - S{season:00}E{episode:00} - {Episode Title} {Edition} {Target}",
	}
}
