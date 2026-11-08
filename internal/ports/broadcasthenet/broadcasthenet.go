// Ported from Prowlarr's src/NzbDrone.Core/Indexers/Definitions/BroadcastheNet/BroadcastheNet.cs
// at ef9c182466ed04f34e6bcfedebb02c3c4af7eaf9. Copyright the Prowlarr
// contributors, GPL-3.0. Modified for SlipStream: rewritten in Go.

package broadcasthenet

import (
	_ "embed"
	"time"

	"github.com/jatassi/slipstream/internal/newznab"
	"github.com/jatassi/slipstream/internal/ports/port"
)

//go:embed upstream.json
var upstream []byte

// Port is BroadcasTheNet.
//
// SlipStream: Capabilities' LimitsDefault of 100 is PageSize, which the
// framework passes as each search's Limit.
var Port = port.Define(port.Entry{
	ID:          "broadcasthenet",
	Name:        "BroadcasTheNet",
	Description: "BroadcasTheNet is a private Tracker for TV.",
	Privacy:     port.Private,
	IndexerURLs: []string{"https://api.broadcasthe.net/"},
	LegacyURLs:  []string{"http://api.broadcasthe.net/"},
	RateLimit:   5 * time.Second,
	PageSize:    100,
	QueryLimit:  port.PerHour(150),
	Capabilities: port.Capabilities{
		TvSearchParams: []port.Param{port.Q, port.Season, port.Ep, port.TvdbID, port.ImdbID, port.RID},
		Categories: port.Categories(
			port.Map("SD", newznab.TVSD, "SD"),
			port.Map("720p", newznab.TVHD, "720p"),
			port.Map("1080p", newznab.TVHD, "1080p"),
			port.Map("1080i", newznab.TVHD, "1080i"),
			port.Map("2160p", newznab.TVUHD, "2160p"),
			port.Map("Portable Device", newznab.TVSD, "Portable Device"),
		),
	},
	Upstream: port.ReadUpstream(upstream),
}, port.Impl[broadcastheNetSettings]{
	Login: port.NoLogin{},
	TV:    getTvSearchRequests,
	Basic: getBasicSearchRequests,
	Parse: parseResponse,
})
