// Ported from Prowlarr's src/NzbDrone.Core/Indexers/Definitions/PrivateHD.cs
// at ef9c182466ed04f34e6bcfedebb02c3c4af7eaf9. Copyright the Prowlarr
// contributors, GPL-3.0. Modified for SlipStream: rewritten in Go.

// Package privatehd is PrivateHD, on the Avistaz family.
package privatehd

import (
	_ "embed"

	"github.com/jatassi/slipstream/internal/newznab"
	avistazbase "github.com/jatassi/slipstream/internal/ports/internal/avistaz"
	"github.com/jatassi/slipstream/internal/ports/port"
)

//go:embed upstream.json
var upstream []byte

var Port = avistazbase.Define(port.Entry{
	ID:          "privatehd",
	Name:        "PrivateHD",
	IndexerURLs: []string{"https://privatehd.to/"},
	Description: "PrivateHD, the sister site of AvistaZ, is a private Tracker for HD movies and TV series.",
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
}, avistazbase.Site{})
