// Ported from Prowlarr's src/NzbDrone.Core/Indexers/Definitions/HDBits/HDBitsRequestGenerator.cs
// at ef9c182466ed04f34e6bcfedebb02c3c4af7eaf9. Copyright the Prowlarr
// contributors, GPL-3.0. Modified for SlipStream: rewritten in Go.

package hdbits

import (
	"regexp"
	"strconv"
	"strings"

	"github.com/jatassi/slipstream/internal/ports/port"
	"github.com/jatassi/slipstream/internal/prowlarr/datetimeutil"
	"github.com/jatassi/slipstream/internal/prowlarr/serializer"
)

type indexer = port.Indexer[hdBitsSettings]

// SlipStream: .NET's \W is any character but a Unicode letter, mark,
// digit or connector, where Go's is ASCII only.
var nonWord = regexp.MustCompile(`[^\p{L}\p{Mn}\p{Nd}\p{Pc}]+`)

func getMovieSearchRequests(ix *indexer, searchCriteria port.SearchCriteria) (port.Chain, error) {
	query := torrentQuery{}

	imdbID := searchCriteria.ImdbNumber()

	if imdbID == 0 && strings.TrimSpace(searchCriteria.SearchTerm) != "" {
		query.Search = strings.TrimSpace(nonWord.ReplaceAllString(searchCriteria.SanitizedSearchTerm(), " "))
	}

	if imdbID != 0 {
		if query.ImdbInfo == nil {
			query.ImdbInfo = &imdbInfo{}
		}
		query.ImdbInfo.ID = imdbID
	}

	return port.One(getRequest(ix, query, searchCriteria))
}

// SlipStream: the Music and Book overloads are gone. The capabilities
// declare neither, so Prowlarr never calls them.
func getTvSearchRequests(ix *indexer, searchCriteria port.SearchCriteria) (port.Chain, error) {
	query := torrentQuery{}

	tvdbID := searchCriteria.TvdbID
	imdbID := searchCriteria.ImdbNumber()

	if tvdbID == 0 && imdbID == 0 && strings.TrimSpace(searchCriteria.SearchTerm) != "" {
		query.Search = searchCriteria.SanitizedTvSearchString()
	}

	if tvdbID != 0 {
		if query.TvdbInfo == nil {
			query.TvdbInfo = &tvdbInfo{}
		}
		query.TvdbInfo.ID = &tvdbID

		if showDate, ok := datetimeutil.TryParseExact(port.Interpolated(searchCriteria.Season)+" "+searchCriteria.Episode, "yyyy MM/dd", datetimeutil.None, ix.Clock); ok {
			query.Search = showDate.Format("2006-01-02")
		} else {
			// SlipStream: an empty Episode is C#'s null, which ToJson leaves
			// out.
			query.TvdbInfo.Season = searchCriteria.Season
			query.TvdbInfo.Episode = serializer.String(searchCriteria.Episode)
		}
	}

	if imdbID != 0 {
		if query.ImdbInfo == nil {
			query.ImdbInfo = &imdbInfo{}
		}
		query.ImdbInfo.ID = imdbID
	}

	return port.One(getRequest(ix, query, searchCriteria))
}

func getBasicSearchRequests(ix *indexer, searchCriteria port.SearchCriteria) (port.Chain, error) {
	query := torrentQuery{}

	if strings.TrimSpace(searchCriteria.SearchTerm) != "" {
		query.Search = searchCriteria.SanitizedSearchTerm()
	}

	return port.One(getRequest(ix, query, searchCriteria))
}

func getRequest(ix *indexer, query torrentQuery, searchCriteria port.SearchCriteria) *port.Request {
	request := port.Post(ix.URL("api/torrents"))

	const appJSON = "application/json"
	request.Accept(appJSON)

	query.Username = ix.Settings.Username
	query.Passkey = ix.Settings.Passkey

	if len(ix.Settings.Codecs) > 0 {
		query.Codec = ints(ix.Settings.Codecs)
	}

	if len(ix.Settings.Mediums) > 0 {
		query.Medium = ints(ix.Settings.Mediums)
	}

	if len(ix.Settings.Origins) > 0 {
		query.Origin = ints(ix.Settings.Origins)
	}

	if len(searchCriteria.Categories) > 0 {
		for _, cat := range ix.Categories.TrackerIDs(searchCriteria.Categories) {
			id, _ := strconv.Atoi(cat)
			query.Category = append(query.Category, id)
		}
	}

	limit := 100
	query.Limit = &limit

	if searchCriteria.Limit > 0 && searchCriteria.Offset > 0 {
		page := searchCriteria.Offset / searchCriteria.Limit
		query.Page = &page
	}

	return request.SetJSON(query)
}

func ints[T ~int](values []T) []int {
	out := make([]int, len(values))
	for i, v := range values {
		out[i] = int(v)
	}
	return out
}
