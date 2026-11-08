// Ported from Prowlarr's src/NzbDrone.Core/Indexers/Definitions/RetroFlix.cs
// at ef9c182466ed04f34e6bcfedebb02c3c4af7eaf9. Copyright the Prowlarr
// contributors, GPL-3.0. Modified for SlipStream: rewritten in Go.

// Package retroflix is RetroFlix, on the SpeedApp family.
package retroflix

import (
	_ "embed"
	"time"

	"github.com/jatassi/slipstream/internal/newznab"
	speedappbase "github.com/jatassi/slipstream/internal/ports/internal/speedapp"
	"github.com/jatassi/slipstream/internal/ports/port"
)

//go:embed upstream.json
var upstream []byte

var Port = speedappbase.Define(port.Entry{
	ID:           "retroflix",
	Name:         "RetroFlix",
	IndexerURLs:  []string{"https://retroflix.club/"},
	LegacyURLs:   []string{"https://retroflix.net/"},
	Description:  "RetroFlix is a private Tracker for classic movies, TV and general Releases.",
	Privacy:      port.Private,
	RateLimit:    2100 * time.Millisecond,
	Capabilities: setCapabilities(),
	Upstream:     port.ReadUpstream(upstream),
}, speedappbase.Site{
	MinimumSeedTime: 432000, // 120 hours
})

// SlipStream: no MusicSearchParams or BookSearchParams, since SlipStream
// has no music or book search mode. Music and book releases still come
// back from Basic.
func setCapabilities() port.Capabilities {
	return port.Capabilities{
		TvSearchParams:    []port.Param{port.Q, port.ImdbID, port.Season, port.Ep},
		MovieSearchParams: []port.Param{port.Q, port.ImdbID},
		Categories: port.Categories(
			port.Map(401, newznab.Movies, "Movies"),
			port.Map(402, newznab.TV, "TV Series"),
			port.Map(406, newznab.AudioVideo, "Music Videos"),
			port.Map(407, newznab.TVSport, "Sports"),
			port.Map(409, newznab.Books, "Books"),
			port.Map(408, newznab.Audio, "HQ Audio"),
		),
	}
}
