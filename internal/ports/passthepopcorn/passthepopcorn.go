// Ported from Prowlarr's src/NzbDrone.Core/Indexers/Definitions/PassThePopcorn/PassThePopcorn.cs
// at ef9c182466ed04f34e6bcfedebb02c3c4af7eaf9. Copyright the Prowlarr
// contributors, GPL-3.0. Modified for SlipStream: rewritten in Go.

package passthepopcorn

import (
	_ "embed"
	"time"

	"github.com/jatassi/slipstream/internal/newznab"
	"github.com/jatassi/slipstream/internal/ports/port"
)

//go:embed upstream.json
var upstream []byte

var Port = port.Define(port.Entry{
	ID:          "passthepopcorn",
	Name:        "PassThePopcorn",
	Description: "PassThePopcorn is a private tracker for movies.",
	Privacy:     port.Private,
	IndexerURLs: []string{"https://passthepopcorn.me"},
	RateLimit:   4 * time.Second,
	PageSize:    50,
	QueryLimit:  port.PerHour(150),
	Capabilities: port.Capabilities{
		MovieSearchParams: []port.Param{port.Q, port.ImdbID},
		Categories: port.Categories(
			port.Map(1, newznab.Movies, "Feature Film"),
			port.Map(2, newznab.Movies, "Short Film"),
			port.Map(3, newznab.Movies, "Miniseries"),
			port.Map(4, newznab.Movies, "Stand-up Comedy"),
			port.Map(5, newznab.Movies, "Live Performance"),
			port.Map(6, newznab.Movies, "Movie Collection"),
		),
	},
	Upstream: port.ReadUpstream(upstream),
}, port.Impl[passThePopcornSettings]{
	Login:         port.NoLogin{},
	Movie:         getSearchRequests,
	Basic:         getBasicSearchRequests,
	Parse:         parseResponse,
	ModifyRequest: getDownloadRequest,
})

// SlipStream: ModifyRequest runs on every request to the Indexer's hosts,
// so the generator's searches get these headers here too.
func getDownloadRequest(ix *port.Indexer[passThePopcornSettings], r *port.Request) {
	r.SetHeader("ApiUser", ix.Settings.APIUser).
		SetHeader("ApiKey", ix.Settings.APIKey)
}
