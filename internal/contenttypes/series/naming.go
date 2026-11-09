package series

import (
	"context"
	"strconv"
	"strings"

	"github.com/jatassi/slipstream/internal/contenttypes/series/internal/db"
	"github.com/jatassi/slipstream/internal/naming"
	"github.com/jatassi/slipstream/internal/providers/tvdb"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/works"
)

var declaration = naming.Declaration{
	Tokens: []naming.Token{
		{Name: "Series Title", Kind: naming.KindTitle, Scope: naming.WorkScope, Example: "The Wire", About: "The series' English title."},
		{Name: "Series Year", Kind: naming.KindYear, Scope: naming.WorkScope, Example: "2002", About: "The year the series first aired."},
		{Name: "TvdbId", Kind: naming.KindID, Scope: naming.WorkScope, Example: "79126", About: "The series' TVDB id."},
		{Name: "TmdbId", Kind: naming.KindID, Scope: naming.WorkScope, Example: "1438", About: "The series' TMDB id."},
		{Name: "ImdbId", Kind: naming.KindID, Scope: naming.WorkScope, Example: "tt0306414", About: "The series' IMDb id."},
		{Name: "season", Kind: naming.KindNumber, Scope: naming.LevelScope, Seq: naming.SeqOuter, Example: "1", About: "The season's number in the series' order."},
		{Name: "episode", Kind: naming.KindNumber, Scope: naming.ItemScope, Seq: naming.SeqInner, Example: "1", About: "The episode's number in its season, in the series' order."},
		{Name: "absolute", Kind: naming.KindNumber, Scope: naming.ItemScope, Seq: naming.SeqAlone, Example: "1", About: "The episode's number across every season."},
		{Name: "Episode Title", Kind: naming.KindTitle, Scope: naming.ItemScope, Example: "The Target", About: "The episode's title."},
		{Name: "Air Date", Kind: naming.KindDate, Scope: naming.ItemScope, Example: "2002 06 02", About: "The day the episode first aired."},
	},
	Folder: "{Series Title} {(Series Year)}",
	Levels: []string{"Season {season:00}"},
	Aside:  "Specials",
	Files: []naming.Variant{
		{Key: standard, Label: "Episode file", File: "{Series Title} - S{season:00}E{episode:00} - {Episode Title} {Edition} {Target}", Anchor: naming.BlockAnchor()},
		{Key: absolute, Label: "Episode file, Absolute order", File: "{Series Title} - S{season:00}E{episode:00} - {absolute:000} - {Episode Title} {Edition} {Target}", Anchor: naming.BlockAnchor()},
		{Key: daily, Label: "Episode file, named by air date", File: "{Series Title} - {Air-Date} - {Episode Title} {Edition} {Target}", Anchor: naming.TokenAnchor("Air Date")},
	},
	Samples: []naming.Sample{
		{Label: "The Wire", Work: wire, Units: []naming.Unit{aired(1, 1, "The Target")}},
		{Label: "Two episodes", Work: wire, Units: []naming.Unit{aired(1, 1, "Pilot (1)"), aired(1, 2, "Pilot (2)")}},
		{Label: "Across a season", Work: wire, Units: []naming.Unit{aired(1, 13, "Sentencing"), aired(2, 1, "Ebb Tide")}},
		{Label: "A gap", Work: wire, Units: []naming.Unit{aired(1, 1, "The Target"), aired(1, 3, "The Buys")}},
		{Label: "A special", Work: wire, Units: []naming.Unit{{
			Level: naming.Set{"season": naming.Number(0)},
			Item:  naming.ItemValues{Set: naming.Set{"episode": naming.Number(1), "Episode Title": naming.Text("Inside the Wire")}},
			Aside: true,
		}}},
		{Label: "Awkward titles", Work: naming.Set{"Series Title": naming.Text("Marvel's Agents of S.H.I.E.L.D."), "Series Year": naming.Number(2013)}, Units: []naming.Unit{aired(1, 1, "Pilot: Part 1?")}},
		{Label: "A long title", Work: wire, Units: []naming.Unit{aired(1, 1, strings.Repeat("A very long episode title ", 12))}},
		{Label: "Nothing known", Missing: true, Work: naming.Set{"Series Title": naming.Text("The Wire")}, Units: []naming.Unit{{
			Level: naming.Set{"season": naming.Number(1)},
			Item:  naming.ItemValues{Set: naming.Set{"episode": naming.Number(1)}},
		}}},
	},
}

// The file variants: Standard for Aired and DVD order, Absolute for
// Absolute order, Daily for a series named by air date.
const (
	standard = "standard"
	absolute = "absolute"
	daily    = "daily"
)

var wire = naming.Set{"Series Title": naming.Text("The Wire"), "Series Year": naming.Number(2002), "TvdbId": naming.Text("79126")}

func aired(season, episode int64, title string) naming.Unit {
	return naming.Unit{
		Level: naming.Set{"season": naming.Number(season)},
		Item: naming.ItemValues{Set: naming.Set{
			"episode":       naming.Number(episode),
			"absolute":      naming.Number(13*(season-1) + episode),
			"Episode Title": naming.Text(title),
			"Air Date":      naming.Day(works.Date{Year: 2002, Month: 6, Day: 2 + int(episode)}),
		}},
	}
}

func (*Series) Naming() naming.Declaration { return declaration }

// NamingValues reads every series of ws, its episodes and their Items in
// three queries. Each Item picks Daily when its series is named by air
// date, Absolute under Absolute order and Standard otherwise.
func (*Series) NamingValues(ctx context.Context, r storage.Reader[file], ws works.Among[works.WorkID]) (map[works.WorkID]naming.Values, error) {
	q := db.New(r)
	rows, err := works.Read(ws, func() ([]db.ListEveryNameRow, error) { return q.ListEveryName(ctx) }, func(ids []works.WorkID) ([]db.ListEveryNameRow, error) {
		rows, err := q.ListNamesOf(ctx, ids)
		out := make([]db.ListEveryNameRow, len(rows))
		for i, row := range rows {
			out[i] = db.ListEveryNameRow(row)
		}
		return out, err
	})
	if err != nil {
		return nil, err
	}
	episodes, err := works.Read(ws, func() ([]db.ListEveryEpisodeNameRow, error) { return q.ListEveryEpisodeName(ctx) }, func(ids []works.WorkID) ([]db.ListEveryEpisodeNameRow, error) {
		rows, err := q.ListEpisodeNamesOf(ctx, ids)
		out := make([]db.ListEveryEpisodeNameRow, len(rows))
		for i, row := range rows {
			out[i] = db.ListEveryEpisodeNameRow(row)
		}
		return out, err
	})
	if err != nil {
		return nil, err
	}
	items, err := works.ItemsOf(ctx, r, ws)
	if err != nil {
		return nil, err
	}
	out := make(map[works.WorkID]naming.Values, len(rows))
	variants := make(map[works.WorkID]string, len(rows))
	for _, row := range rows {
		year := naming.Value{}
		if row.FirstAired != nil {
			first, err := works.ParseDate(*row.FirstAired)
			if err != nil {
				return nil, err
			}
			year = naming.Number(int64(first.Year))
		}
		v := naming.Values{
			Work: naming.Set{
				"Series Title": naming.Text(row.EnglishTitle), "Series Year": year, "TvdbId": naming.Text(strconv.FormatInt(int64(row.TvdbID), 10)),
				"TmdbId": idOr(row.TmdbID), "ImdbId": naming.TextOr(row.ImdbID),
			},
			Levels: map[works.LevelID]naming.Set{},
			Items:  map[works.ItemID]naming.ItemValues{},
		}
		out[row.WorkID] = v
		switch {
		case row.NameByAirDate:
			variants[row.WorkID] = daily
		case tvdb.Order(row.EpisodeOrder) == tvdb.OrderAbsolute:
			variants[row.WorkID] = absolute
		default:
			variants[row.WorkID] = standard
		}
	}
	labels := map[works.ItemID]works.Item{}
	for _, its := range items {
		for _, it := range its {
			labels[it.ID] = it
		}
	}
	for _, e := range episodes {
		aired := naming.Value{}
		if e.AirDate != nil {
			d, err := works.ParseDate(*e.AirDate)
			if err != nil {
				return nil, err
			}
			aired = naming.Day(d)
		}
		v, it := out[e.WorkID], labels[e.ItemID]
		v.Levels[it.LevelID] = naming.Set{"season": naming.Number(e.SeasonNumber)}
		v.Items[e.ItemID] = naming.ItemValues{Variant: variants[e.WorkID], Set: naming.Set{
			"episode": naming.Number(e.EpisodeNumber), "absolute": naming.NumberOr(e.AbsoluteNumber),
			"Episode Title": naming.Text(it.Label), "Air Date": aired,
		}}
	}
	return out, nil
}

func idOr(id *db.TMDBID) naming.Value {
	if id == nil {
		return naming.Value{}
	}
	return naming.Text(strconv.FormatInt(int64(*id), 10))
}

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
		Levels: []string{"Season {season:00}"},
		Aside:  "Specials",
		File:   "{Series Title} - S{season:00}E{episode:00} - {Episode Title} {Edition} {Target}",
	}
}
