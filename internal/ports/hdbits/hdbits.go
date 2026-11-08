// Ported from Prowlarr's src/NzbDrone.Core/Indexers/Definitions/HDBits/HDBits.cs
// at ef9c182466ed04f34e6bcfedebb02c3c4af7eaf9. Copyright the Prowlarr
// contributors, GPL-3.0. Modified for SlipStream: rewritten in Go.

package hdbits

import (
	_ "embed"

	"github.com/jatassi/slipstream/internal/newznab"
	"github.com/jatassi/slipstream/internal/ports/port"
)

//go:embed upstream.json
var upstream []byte

var Port = port.Define(port.Entry{
	ID:          "hdbits",
	Name:        "HDBits",
	Description: "HDBits is a private tracker for HD movies and TV.",
	Privacy:     port.Private,
	IndexerURLs: []string{"https://hdbits.org/"},
	LegacyURLs:  []string{"https://hdbits.org"},
	PageSize:    100,
	Capabilities: port.Capabilities{
		TvSearchParams:    []port.Param{port.Q, port.Season, port.Ep, port.TvdbID},
		MovieSearchParams: []port.Param{port.Q, port.ImdbID},
		Categories: port.Categories(
			port.Map(1, newznab.Movies, "Movie"),
			port.Map(2, newznab.TV, "TV"),
			port.Map(3, newznab.TVDocumentary, "Documentary"),
			port.Map(4, newznab.Audio, "Music"),
			port.Map(5, newznab.TVSport, "Sport"),
			port.Map(6, newznab.Audio, "Audio Track"),
			port.Map(7, newznab.XXX, "XXX"),
			port.Map(8, newznab.Other, "Misc/Demo"),
		),
	},
	Aliases:  []string{"hdbitsapi"},
	Upstream: port.ReadUpstream(upstream),
}, port.Impl[hdBitsSettings]{
	Login: port.NoLogin{},
	Movie: getMovieSearchRequests,
	TV:    getTvSearchRequests,
	Basic: getBasicSearchRequests,
	Parse: parseResponse,
})
