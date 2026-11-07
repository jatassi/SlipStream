package movie

import (
	"net/url"

	"github.com/jatassi/slipstream/internal/node/endpoint"
	"github.com/jatassi/slipstream/internal/works"
)

// Endpoints are the hosts a movie's outbound links open in the browser.
// No Node calls them.
type Endpoints struct {
	IMDb     endpoint.URL `yaml:"imdb"`
	TMDBSite endpoint.URL `yaml:"tmdb_site"`
	Trakt    endpoint.URL `yaml:"trakt"`
	YouTube  endpoint.URL `yaml:"youtube"`
}

func DefaultEndpoints() Endpoints {
	return Endpoints{
		IMDb:     endpoint.MustParseURL("https://www.imdb.com"),
		TMDBSite: endpoint.MustParseURL("https://www.themoviedb.org"),
		Trakt:    endpoint.MustParseURL("https://trakt.tv"),
		YouTube:  endpoint.MustParseURL("https://www.youtube.com"),
	}
}

// Links are IMDb's page from imdb_id, TMDB's and Trakt's from the TMDB id,
// and the trailer when YouTube hosts it.
func (m *Movie) Links(key works.Key, fields []works.FieldValue) []works.WebLink {
	text := func(k string) string {
		for _, f := range fields {
			if f.Key == k && f.Kind == works.KindText {
				return f.Text
			}
		}
		return ""
	}
	var out []works.WebLink
	if id := text("imdb_id"); id != "" {
		out = append(out, works.WebLink{Key: "imdb", Label: "IMDb", URL: m.web.IMDb.JoinPath("title", id).String(), Kind: works.LinkPage})
	}
	trakt := m.web.Trakt.JoinPath("search", "tmdb", string(key))
	trakt.RawQuery = "id_type=movie"
	out = append(out,
		works.WebLink{Key: "tmdb", Label: "TMDB", URL: m.web.TMDBSite.JoinPath("movie", string(key)).String(), Kind: works.LinkPage},
		works.WebLink{Key: "trakt", Label: "Trakt", URL: trakt.String(), Kind: works.LinkPage},
	)
	if text("trailer_site") == "YouTube" && text("trailer_key") != "" {
		trailer := m.web.YouTube.JoinPath("watch")
		trailer.RawQuery = "v=" + url.QueryEscape(text("trailer_key"))
		out = append(out, works.WebLink{Key: "trailer", Label: "Trailer", URL: trailer.String(), Kind: works.LinkTrailer})
	}
	return out
}
