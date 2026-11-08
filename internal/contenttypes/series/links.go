package series

import (
	"net/url"

	"github.com/jatassi/slipstream/internal/node/endpoint"
	"github.com/jatassi/slipstream/internal/works"
)

// Endpoints are the hosts a series' outbound links open in the browser.
// No Node calls them.
type Endpoints struct {
	TVDBSite endpoint.URL `yaml:"tvdb_site"`
	TMDBSite endpoint.URL `yaml:"tmdb_site"`
	IMDb     endpoint.URL `yaml:"imdb"`
	Trakt    endpoint.URL `yaml:"trakt"`
	YouTube  endpoint.URL `yaml:"youtube"`
}

func DefaultEndpoints() Endpoints {
	return Endpoints{
		TVDBSite: endpoint.MustParseURL("https://thetvdb.com"),
		TMDBSite: endpoint.MustParseURL("https://www.themoviedb.org"),
		IMDb:     endpoint.MustParseURL("https://www.imdb.com"),
		Trakt:    endpoint.MustParseURL("https://trakt.tv"),
		YouTube:  endpoint.MustParseURL("https://www.youtube.com"),
	}
}

// Links are IMDb's page from imdb_id, TVDB's and Trakt's from the TVDB id,
// TMDB's from tmdb_id, and the trailer when YouTube hosts it.
func (s *Series) Links(key works.Key, fields []works.FieldValue) []works.WebLink {
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
		out = append(out, works.WebLink{Key: "imdb", Label: "IMDb", URL: s.web.IMDb.JoinPath("title", id).String(), Kind: works.LinkPage})
	}
	out = append(out, works.WebLink{Key: "tvdb", Label: "TVDB", URL: s.web.TVDBSite.JoinPath("dereferrer", "series", string(key)).String(), Kind: works.LinkPage})
	if id := text("tmdb_id"); id != "" {
		out = append(out, works.WebLink{Key: "tmdb", Label: "TMDB", URL: s.web.TMDBSite.JoinPath("tv", id).String(), Kind: works.LinkPage})
	}
	trakt := s.web.Trakt.JoinPath("search", "tvdb", string(key))
	trakt.RawQuery = "id_type=show"
	out = append(out, works.WebLink{Key: "trakt", Label: "Trakt", URL: trakt.String(), Kind: works.LinkPage})
	if text("trailer_site") == "YouTube" && text("trailer_key") != "" {
		trailer := s.web.YouTube.JoinPath("watch")
		trailer.RawQuery = "v=" + url.QueryEscape(text("trailer_key"))
		out = append(out, works.WebLink{Key: "trailer", Label: "Trailer", URL: trailer.String(), Kind: works.LinkTrailer})
	}
	return out
}
