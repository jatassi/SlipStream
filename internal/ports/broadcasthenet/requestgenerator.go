// Ported from Prowlarr's src/NzbDrone.Core/Indexers/Definitions/BroadcastheNet/BroadcastheNetRequestGenerator.cs
// at ef9c182466ed04f34e6bcfedebb02c3c4af7eaf9. Copyright the Prowlarr
// contributors, GPL-3.0. Modified for SlipStream: rewritten in Go.

package broadcasthenet

import (
	"crypto/rand"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/jatassi/slipstream/internal/ports/port"
	"github.com/jatassi/slipstream/internal/prowlarr/datetimeutil"
)

type indexer = port.Indexer[broadcastheNetSettings]

// SlipStream: the Movie, Music and Book overloads are gone. The
// capabilities declare none of them, so Prowlarr never calls them.
// SearchCriteria carries no RId, so the Tvrage branch is gone too.
func getTvSearchRequests(ix *indexer, searchCriteria port.SearchCriteria) (port.Chain, error) {
	var pageableRequests port.Chain

	parameters := broadcastheNetTorrentQuery{}

	searchTerm := searchCriteria.SearchTerm

	btnResults := searchCriteria.Limit

	btnOffset := searchCriteria.Offset

	if searchCriteria.TvdbID > 0 {
		parameters.Tvdb = strconv.Itoa(searchCriteria.TvdbID)
	} else if searchCriteria.FullImdbID() != "" {
		parameters.Imdb = searchCriteria.FullImdbID()
	}

	if strings.TrimSpace(searchTerm) != "" {
		parameters.Search = strings.ReplaceAll(searchTerm, " ", "%")
	}

	showDate, daily := datetimeutil.TryParseExact(fmt.Sprintf("%d %s", searchCriteria.Season, searchCriteria.Episode), "yyyy MM/dd", datetimeutil.None, ix.Clock)
	episode, episodeErr := strconv.Atoi(strings.TrimSpace(searchCriteria.Episode))
	_, termErr := strconv.Atoi(strings.TrimSpace(searchTerm))

	switch {
	// If only the season/episode is searched for then change format to match expected format
	case searchCriteria.Season > 0 && strings.TrimSpace(searchCriteria.Episode) == "":
		// Search Season
		parameters.Category = "Season"
		parameters.Name = fmt.Sprintf("Season %d%%", searchCriteria.Season)
		pageableRequests.Add(getPagedRequests(ix, parameters, btnResults, btnOffset))

		// Search Episode
		parameters.Category = "Episode"
		parameters.Name = fmt.Sprintf("S%02dE%%", searchCriteria.Season)
		pageableRequests.Add(getPagedRequests(ix, parameters, btnResults, btnOffset))
	case daily:
		// Daily Episode
		parameters.Name = showDate.Format("2006.01.02") + "%"
		parameters.Category = "Episode"
		pageableRequests.Add(getPagedRequests(ix, parameters, btnResults, btnOffset))
	case searchCriteria.Season > 0 && episodeErr == nil && episode > 0:
		// Standard (S/E) Episode
		if episode > 0 && episode < 100 {
			parameters.Name = fmt.Sprintf("S%02dE%%%02d%%", searchCriteria.Season, episode)
		} else {
			parameters.Name = fmt.Sprintf("S%02dE%02d%%", searchCriteria.Season, episode)
		}
		parameters.Category = "Episode"
		pageableRequests.Add(getPagedRequests(ix, parameters, btnResults, btnOffset))
	case strings.TrimSpace(searchTerm) != "" && termErr == nil && (searchCriteria.TvdbID > 0 || searchCriteria.FullImdbID() != ""):
		// Disable ID-based searches for episodes with absolute episode number
		return port.Chain{}, nil
	default:
		// Neither a season only search nor daily nor standard, fall back to query
		pageableRequests.Add(getPagedRequests(ix, parameters, btnResults, btnOffset))
	}

	return pageableRequests, nil
}

func getBasicSearchRequests(ix *indexer, searchCriteria port.SearchCriteria) (port.Chain, error) {
	var pageableRequests port.Chain

	parameters := broadcastheNetTorrentQuery{}

	searchTerm := searchCriteria.SearchTerm

	btnResults := searchCriteria.Limit

	btnOffset := searchCriteria.Offset

	if strings.TrimSpace(searchTerm) != "" {
		parameters.Search = strings.ReplaceAll(searchTerm, " ", "%")
	}

	pageableRequests.Add(getPagedRequests(ix, parameters, btnResults, btnOffset))

	return pageableRequests, nil
}

// SlipStream: Prowlarr's HttpIndexerBase throws on a 503 before the
// parser, which then never reads it as the request limit it means. The
// port lets it through so the parser does.
func getPagedRequests(ix *indexer, parameters broadcastheNetTorrentQuery, results, offset int) *port.Request {
	return jsonRPCCall(ix.BaseURL, "getTorrents", ix.Settings.APIKey, parameters, results, offset).
		SuppressHTTPErrorStatusCodes(http.StatusServiceUnavailable)
}

// jsonRPCCall is JsonRpcRequestBuilder.Call: a POST of a JSON-RPC 2.0
// message with an id of eight random characters.
func jsonRPCCall(baseURL *url.URL, method string, parameters ...any) *port.Request {
	return port.Post(baseURL).SetJSON(map[string]any{
		"jsonrpc": "2.0",
		"method":  method,
		"params":  parameters,
		"id":      rand.Text()[:8],
	})
}
