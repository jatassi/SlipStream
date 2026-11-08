// Ported from Prowlarr's src/NzbDrone.Core/Indexers/Definitions/AvistaZ.cs
// at ef9c182466ed04f34e6bcfedebb02c3c4af7eaf9. Copyright the Prowlarr
// contributors, GPL-3.0. Modified for SlipStream: rewritten in Go.

// Package avistaz is AvistaZ, on the Avistaz family.
package avistaz

import (
	_ "embed"
	"strings"

	"github.com/jatassi/slipstream/internal/newznab"
	avistazbase "github.com/jatassi/slipstream/internal/ports/internal/avistaz"
	"github.com/jatassi/slipstream/internal/ports/port"
)

//go:embed upstream.json
var upstream []byte

var Port = avistazbase.Define(port.Entry{
	ID:          "avistaz",
	Name:        "AvistaZ",
	IndexerURLs: []string{"https://avistaz.to/"},
	Description: "AvistaZ, once called AsiaTorrents, is a private Tracker for Asian movies and TV series.",
	Privacy:     port.Private,
	Capabilities: port.Capabilities{
		// SlipStream: no genre, which SearchCriteria doesn't carry.
		TvSearchParams:    []port.Param{port.Q, port.Season, port.Ep, port.ImdbID, port.TvdbID},
		MovieSearchParams: []port.Param{port.Q, port.ImdbID, port.TmdbID},
		Categories: port.Categories(
			port.Map(1, newznab.Movies, ""),
			port.Map(1, newznab.MoviesUHD, ""),
			port.Map(1, newznab.MoviesHD, ""),
			port.Map(1, newznab.MoviesSD, ""),
			port.Map(2, newznab.TV, ""),
			port.Map(2, newznab.TVUHD, ""),
			port.Map(2, newznab.TVHD, ""),
			port.Map(2, newznab.TVSD, ""),
		),
	},
	Upstream: port.ReadUpstream(upstream),
}, avistazbase.Site{
	GetEpisodeSearchTerm: getEpisodeSearchTerm,
})

// AvistaZ has episodes without season. eg Running Man E323
func getEpisodeSearchTerm(searchCriteria port.SearchCriteria) string {
	if (searchCriteria.Season == nil || *searchCriteria.Season == 0) && strings.TrimSpace(searchCriteria.Episode) != "" {
		return "E" + searchCriteria.Episode
	}
	return searchCriteria.EpisodeSearchString()
}
