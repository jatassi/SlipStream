// Ported from Prowlarr's src/NzbDrone.Core/Indexers/Definitions/FileList/FileListRequestGenerator.cs
// at ef9c182466ed04f34e6bcfedebb02c3c4af7eaf9. Copyright the Prowlarr
// contributors, GPL-3.0. Modified for SlipStream: rewritten in Go.

package filelist

import (
	"encoding/base64"
	"fmt"
	"strconv"
	"strings"

	"github.com/jatassi/slipstream/internal/ports/port"
	"github.com/jatassi/slipstream/internal/prowlarr/datetimeutil"
	"github.com/jatassi/slipstream/internal/prowlarr/stringutil"
)

type indexer = port.Indexer[fileListSettings]

func getTvSearchRequests(ix *indexer, searchCriteria port.SearchCriteria) (port.Chain, error) {
	var pageableRequests port.Chain
	var parameters stringutil.NameValueCollection

	if strings.TrimSpace(searchCriteria.ImdbID) != "" || strings.TrimSpace(searchCriteria.SearchTerm) != "" {
		parameters.Set("action", "search-torrents")

		searchQuery := strings.TrimSpace(searchCriteria.SanitizedSearchTerm())

		if showDate, ok := datetimeutil.TryParseExact(fmt.Sprintf("%d %s", searchCriteria.Season, searchCriteria.Episode), "yyyy MM/dd", datetimeutil.None, ix.Clock); ok {
			if strings.TrimSpace(searchCriteria.ImdbID) != "" {
				// Skip ID searches for daily episodes
				return pageableRequests, nil
			}

			searchQuery = strings.TrimSpace(searchQuery + " " + showDate.Format("2006.01.02"))
		} else {
			if searchCriteria.Season != 0 {
				parameters.Set("season", strconv.Itoa(searchCriteria.Season))
			}

			if strings.TrimSpace(searchCriteria.Episode) != "" {
				parameters.Set("episode", searchCriteria.Episode)
			}
		}

		if strings.TrimSpace(searchCriteria.ImdbID) != "" {
			parameters.Set("type", "imdb")
			parameters.Set("query", searchCriteria.FullImdbID())
		} else if strings.TrimSpace(searchCriteria.SearchTerm) != "" {
			parameters.Set("type", "name")
			parameters.Set("query", searchQuery)
		}
	}

	pageableRequests.Add(getPagedRequests(ix, searchCriteria, &parameters))

	return pageableRequests, nil
}

func getMovieSearchRequests(ix *indexer, searchCriteria port.SearchCriteria) (port.Chain, error) {
	var pageableRequests port.Chain
	var parameters stringutil.NameValueCollection

	if strings.TrimSpace(searchCriteria.ImdbID) != "" {
		parameters.Set("action", "search-torrents")
		parameters.Set("type", "imdb")
		parameters.Set("query", searchCriteria.FullImdbID())
	} else if strings.TrimSpace(searchCriteria.SearchTerm) != "" {
		parameters.Set("action", "search-torrents")
		parameters.Set("type", "name")
		parameters.Set("query", strings.TrimSpace(searchCriteria.SanitizedSearchTerm()))
	}

	pageableRequests.Add(getPagedRequests(ix, searchCriteria, &parameters))

	return pageableRequests, nil
}

// SlipStream: the Music and Book overloads are gone. SlipStream has
// neither mode.
func getBasicSearchRequests(ix *indexer, searchCriteria port.SearchCriteria) (port.Chain, error) {
	var pageableRequests port.Chain
	var parameters stringutil.NameValueCollection

	if strings.TrimSpace(searchCriteria.SearchTerm) != "" {
		parameters.Set("action", "search-torrents")
		parameters.Set("type", "name")
		parameters.Set("query", strings.TrimSpace(searchCriteria.SanitizedSearchTerm()))
	}

	pageableRequests.Add(getPagedRequests(ix, searchCriteria, &parameters))

	return pageableRequests, nil
}

// SlipStream: BasicNetworkCredential is the Authorization header it
// sends.
func getPagedRequests(ix *indexer, searchCriteria port.SearchCriteria, parameters *stringutil.NameValueCollection) *port.Request {
	if parameters.Get("action") == "" {
		parameters.Set("action", "latest-torrents")
	}

	if len(searchCriteria.Categories) > 0 {
		parameters.Set("category", strings.Join(ix.Categories.TrackerIDs(searchCriteria.Categories), ","))
	}

	if ix.Settings.FreeleechOnly {
		parameters.Set("freeleech", "1")
	}

	searchURL := ix.URL("api.php?" + parameters.GetQueryString())

	credentials := strings.TrimSpace(ix.Settings.Username) + ":" + strings.TrimSpace(ix.Settings.Passkey)

	return port.Get(searchURL).
		Accept("application/json").
		SetHeader("Authorization", "Basic "+base64.StdEncoding.EncodeToString([]byte(credentials)))
}
