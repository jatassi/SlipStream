// Ported from Prowlarr's src/NzbDrone.Core/Indexers/Definitions/Gazelle/GazelleRequestGenerator.cs
// at ef9c182466ed04f34e6bcfedebb02c3c4af7eaf9. Copyright the Prowlarr
// contributors, GPL-3.0. Modified for SlipStream: rewritten in Go.

package gazelle

import (
	"strings"

	"github.com/jatassi/slipstream/internal/ports/port"
	"github.com/jatassi/slipstream/internal/prowlarr/stringutil"
)

const apiURL = "ajax.php"

type requestGenerator[S settings] struct{ site Site[S] }

// GetRequest is the request generator's GetRequest, for a site whose
// GetSearchRequests override builds its own parameters.
func GetRequest[S any](ix *port.Indexer[S], parameters stringutil.NameValueCollection) (port.Chain, error) {
	request := port.Get(ix.URL(apiURL + "?" + parameters.GetQueryString())).
		Accept("application/json").
		NoRedirect()

	return port.One(request)
}

func (g requestGenerator[S]) movie(ix *port.Indexer[S], searchCriteria port.SearchCriteria) (port.Chain, error) {
	parameters := g.site.GetBasicSearchParameters(ix, searchCriteria, searchCriteria.SanitizedSearchTerm())

	if searchCriteria.ImdbID != "" {
		parameters.Set(ifImdbInTags(g.site.ImdbInTags, "taglist", "cataloguenumber"), searchCriteria.FullImdbID())
	}

	return GetRequest(ix, parameters)
}

// SlipStream: no music search, which the framework doesn't run.

func (g requestGenerator[S]) tv(ix *port.Indexer[S], searchCriteria port.SearchCriteria) (port.Chain, error) {
	parameters := g.site.GetBasicSearchParameters(ix, searchCriteria, searchCriteria.SanitizedTvSearchString())

	if searchCriteria.ImdbID != "" {
		parameters.Set(ifImdbInTags(g.site.ImdbInTags, "taglist", "cataloguenumber"), searchCriteria.FullImdbID())
	}

	return GetRequest(ix, parameters)
}

// SlipStream: no book search, which the framework doesn't run.

func (g requestGenerator[S]) basic(ix *port.Indexer[S], searchCriteria port.SearchCriteria) (port.Chain, error) {
	parameters := g.site.GetBasicSearchParameters(ix, searchCriteria, searchCriteria.SanitizedSearchTerm())

	return GetRequest(ix, parameters)
}

// ifImdbInTags is C#'s ImdbInTags ? a : b.
func ifImdbInTags(imdbInTags bool, a, b string) string {
	if imdbInTags {
		return a
	}
	return b
}

// hook to adjust the search term
func getSearchTerm(term string) string { return term }

// GetBasicSearchParameters is the base's, which a site's override calls
// first, as C#'s base.GetBasicSearchParameters.
func GetBasicSearchParameters[S any](ix *port.Indexer[S], searchCriteria port.SearchCriteria, term string) stringutil.NameValueCollection {
	var parameters stringutil.NameValueCollection
	parameters.Add("action", "browse")
	parameters.Add("order_by", "time")
	parameters.Add("order_way", "desc")

	searchTerm := getSearchTerm(term)

	if strings.TrimSpace(searchTerm) != "" {
		parameters.Set("searchstr", strings.ReplaceAll(searchTerm, ".", " "))
	}

	if len(searchCriteria.Categories) > 0 {
		queryCats := ix.Categories.TrackerIDs(searchCriteria.Categories)

		for _, cat := range queryCats {
			parameters.Set("filter_cat["+cat+"]", "1")
		}
	}

	return parameters
}
