// Ported from Prowlarr's src/NzbDrone.Core/Indexers/Definitions/PassThePopcorn/PassThePopcornRequestGenerator.cs
// at ef9c182466ed04f34e6bcfedebb02c3c4af7eaf9. Copyright the Prowlarr
// contributors, GPL-3.0. Modified for SlipStream: rewritten in Go.

package passthepopcorn

import (
	"strconv"

	"github.com/jatassi/slipstream/internal/ports/port"
)

type indexer = port.Indexer[passThePopcornSettings]

// SlipStream: the Music, Tv and Book overloads are gone. The capabilities
// declare no TV, music or book search, so Prowlarr never calls them.
func getSearchRequests(ix *indexer, searchCriteria port.SearchCriteria) (port.Chain, error) {
	if searchCriteria.ImdbID != "" {
		return port.One(getRequest(ix, searchCriteria.FullImdbID(), searchCriteria))
	}
	return port.One(getRequest(ix, searchCriteria.SanitizedSearchTerm(), searchCriteria))
}

func getBasicSearchRequests(ix *indexer, searchCriteria port.SearchCriteria) (port.Chain, error) {
	return port.One(getRequest(ix, searchCriteria.SanitizedSearchTerm(), searchCriteria))
}

// SlipStream: the ApiUser and ApiKey headers come from getDownloadRequest,
// which the framework runs on every request. A 429 never reaches the
// parser, because the Node reads it as a limit first.
func getRequest(ix *indexer, searchTerm string, searchCriteria port.SearchCriteria) *port.Request {
	r := port.Get(ix.URL("torrents.php")).
		AddQueryParam("action", "advanced").
		AddQueryParam("json", "noredirect").
		AddQueryParam("grouping", "0").
		AddQueryParam("order_by", "time").
		AddQueryParam("order_way", "desc").
		AddQueryParam("searchstr", searchTerm)

	if ix.Settings.FreeleechOnly {
		r.AddQueryParam("freetorrent", "1")
	}

	if ix.Settings.GoldenPopcornOnly {
		r.AddQueryParam("scene", "2")
	}

	queryCats := ix.Categories.TrackerIDs(searchCriteria.Categories)

	if searchCriteria.IsRssSearch() {
		for _, cat := range queryCats {
			r.AddQueryParam("filter_cat["+cat+"]", "1")
		}
	}

	if searchCriteria.Limit > 0 && searchCriteria.Offset > 0 {
		page := searchCriteria.Offset/searchCriteria.Limit + 1
		r.AddQueryParam("page", strconv.Itoa(page))
	}

	return r.Accept("application/json")
}
