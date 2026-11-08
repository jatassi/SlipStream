// Ported from Prowlarr's src/NzbDrone.Core/Indexers/Definitions/CinemaZ.cs
// at ef9c182466ed04f34e6bcfedebb02c3c4af7eaf9. Copyright the Prowlarr
// contributors, GPL-3.0. Modified for SlipStream: rewritten in Go.

// Package cinemaz is CinemaZ, on the Avistaz family.
package cinemaz

import (
	_ "embed"

	"github.com/jatassi/slipstream/internal/newznab"
	avistazbase "github.com/jatassi/slipstream/internal/ports/internal/avistaz"
	"github.com/jatassi/slipstream/internal/ports/port"
)

//go:embed upstream.json
var upstream []byte

var Port = avistazbase.Define(port.Entry{
	ID:          "cinemaz",
	Name:        "CinemaZ",
	IndexerURLs: []string{"https://cinemaz.to/"},
	Description: "CinemaZ, once called EuTorrents, is a private Tracker for movies and TV series from outside Asia.",
	Privacy:     port.Private,
	Capabilities: port.Capabilities{
		// SlipStream: no genre, which SearchCriteria doesn't carry.
		TvSearchParams:    []port.Param{port.Q, port.Season, port.Ep, port.ImdbID},
		MovieSearchParams: []port.Param{port.Q, port.ImdbID},
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
