// Ported from Prowlarr's src/NzbDrone.Core/Indexers/Definitions/AnimeZ.cs
// at ef9c182466ed04f34e6bcfedebb02c3c4af7eaf9. Copyright the Prowlarr
// contributors, GPL-3.0. Modified for SlipStream: rewritten in Go.

// Package animez is AnimeZ, on the Avistaz family.
package animez

import (
	_ "embed"
	"strconv"
	"strings"

	"github.com/jatassi/slipstream/internal/newznab"
	avistazbase "github.com/jatassi/slipstream/internal/ports/internal/avistaz"
	"github.com/jatassi/slipstream/internal/ports/port"
	"github.com/jatassi/slipstream/internal/prowlarr/stringutil"
)

//go:embed upstream.json
var upstream []byte

type ix = port.Indexer[avistazbase.Settings]

// Port is AnimeZ.
// SlipStream: Download's sign-in and retry after a 401 is the
// framework's, which signs in again on a logged-out download.
var Port = avistazbase.Define(port.Entry{
	ID:           "animez",
	Name:         "AnimeZ",
	IndexerURLs:  []string{"https://animez.to/"},
	Description:  "AnimeZ, once called AnimeTorrents, is a private Tracker for anime and manga.",
	Privacy:      port.Private,
	Capabilities: setCapabilities(),
	Upstream:     port.ReadUpstream(upstream),
}, avistazbase.Site{
	GetBasicSearchParameters: getBasicSearchParameters,
	ParseCategories:          parseCategories,
	ParseTitle:               parseTitle,
	ModifyRequest:            getDownloadRequest,
})

// SlipStream: the base sets the bearer token, and ModifyRequest runs on
// every request to the Indexer's hosts, which all ask for JSON.
func getDownloadRequest(_ *ix, request *port.Request) {
	request.Accept("application/json")
}

// SlipStream: no book search, which the framework doesn't run.
func setCapabilities() port.Capabilities {
	return port.Capabilities{
		TvSearchParams:    []port.Param{port.Q, port.Season, port.Ep},
		MovieSearchParams: []port.Param{port.Q},
		Categories: port.Categories(
			port.Map("TV", newznab.TVAnime, "Anime > TV"),
			port.Map("TV_SHORT", newznab.TVAnime, "Anime > TV Short"),
			port.Map("MOVIE", newznab.Movies, "Anime > Movie"),
			port.Map("SPECIAL", newznab.TVAnime, "Anime > Special"),
			port.Map("OVA", newznab.TVAnime, "Anime > OVA"),
			port.Map("ONA", newznab.TVAnime, "Anime > ONA"),
			port.Map("MUSIC", newznab.TVAnime, "Anime > Music"),
			port.Map("MANGA", newznab.BooksComics, "Manga > Manga"),
			port.Map("NOVEL", newznab.BooksForeign, "Manga > Novel"),
			port.Map("ONE_SHOT", newznab.BooksForeign, "Manga > One-Shot"),
		),
	}
}

// getBasicSearchParameters is AnimeZRequestGenerator's.
// SlipStream: TrackerIDs is already distinct, as C#'s Distinct.
func getBasicSearchParameters(ix *ix, searchCriteria port.SearchCriteria) stringutil.KeyValuePairs {
	limit := avistazbase.PageSize
	if searchCriteria.Limit > 0 {
		limit = min(avistazbase.PageSize, searchCriteria.Limit)
	}

	parameters := stringutil.KeyValuePairs{
		{Key: "limit", Value: strconv.Itoa(limit)},
	}

	categoryMappings := ix.Categories.TrackerIDs(searchCriteria.Categories)

	for _, category := range categoryMappings {
		parameters.Add("format[]", category)
	}

	if searchCriteria.Limit > 0 && searchCriteria.Offset > 0 {
		page := searchCriteria.Offset/searchCriteria.Limit + 1
		parameters.Add("page", strconv.Itoa(page))
	}

	if ix.Settings.FreeleechOnly {
		parameters.Add("freeleech", "1")
	}

	return parameters
}

// parseCategories is AnimeZParser's.
func parseCategories(ix *ix, row avistazbase.AvistazRelease) ([]newznab.Category, error) {
	return ix.Categories.FromTracker(row.Format), nil
}

func parseTitle(row avistazbase.AvistazRelease) string {
	if strings.TrimSpace(row.ReleaseTitle) != "" {
		return row.ReleaseTitle
	}
	return row.FileName
}
