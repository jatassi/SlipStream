package movie

import (
	"context"
	"strconv"

	"github.com/jatassi/slipstream/internal/contenttypes/movie/internal/db"
	"github.com/jatassi/slipstream/internal/naming"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/works"
)

var declaration = naming.Declaration{
	Tokens: []naming.Token{
		{Name: "Movie Title", Kind: naming.KindTitle, Scope: naming.WorkScope, Example: "The Quiet Orchard", About: "The movie's English title."},
		{Name: "Movie OriginalTitle", Kind: naming.KindTitle, Scope: naming.WorkScope, Example: "Le Verger", About: "The movie's title in its original language."},
		{Name: "Year", Aliases: []string{"Release Year"}, Kind: naming.KindYear, Scope: naming.WorkScope, Example: "2021", About: "The year the movie came out."},
		{Name: "ImdbId", Kind: naming.KindID, Scope: naming.WorkScope, Example: "tt0113277", About: "The movie's IMDb id."},
		{Name: "TmdbId", Kind: naming.KindID, Scope: naming.WorkScope, Example: "949", About: "The movie's TMDB id."},
	},
	Folder: "{Movie Title} {(Year)}",
	Files: []naming.Variant{
		{Key: "standard", Label: "Movie file", File: "{Movie Title} {(Year)} - {Edition Tag} {Target}", Anchor: naming.FolderAnchor()},
	},
	Samples: []naming.Sample{
		{Label: "Heat", Work: naming.Set{"Movie Title": naming.Text("Heat"), "Year": naming.Number(1995), "ImdbId": naming.Text("tt0113277"), "TmdbId": naming.Text("949")}},
		{Label: "A colon", Work: naming.Set{"Movie Title": naming.Text("Star Wars: Episode IV - A New Hope"), "Year": naming.Number(1977)}},
		{Label: "A trailing dot", Work: naming.Set{"Movie Title": naming.Text("Monsters, Inc."), "Year": naming.Number(2001)}},
		{Label: "An ampersand and an apostrophe", Work: naming.Set{"Movie Title": naming.Text("Harold & Kumar's Escape"), "Year": naming.Number(2008)}},
		{Label: "CJK", Work: naming.Set{"Movie Title": naming.Text("千と千尋の神隠し"), "Year": naming.Number(2001)}},
		{Label: "Nothing known", Missing: true, Work: naming.Set{"Movie Title": naming.Text("Heat")}},
	},
}

func (*Movie) Naming() naming.Declaration { return declaration }

// NamingValues reads every movie of ws in one query.
func (*Movie) NamingValues(ctx context.Context, r storage.Reader[file], ws works.Among[works.WorkID]) (map[works.WorkID]naming.Values, error) {
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
	out := make(map[works.WorkID]naming.Values, len(rows))
	for _, m := range rows {
		out[m.WorkID] = naming.Values{Work: naming.Set{
			"Movie Title":         naming.Text(m.TitleEnglish),
			"Movie OriginalTitle": naming.Text(m.OriginalTitle),
			"Year":                naming.NumberOr(m.Year),
			"ImdbId":              naming.TextOr((*string)(m.ImdbID)),
			"TmdbId":              naming.Text(strconv.FormatInt(int64(m.TmdbID), 10)),
		}}
	}
	return out, nil
}
