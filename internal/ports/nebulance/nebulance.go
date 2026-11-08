// Ported from Prowlarr's src/NzbDrone.Core/Indexers/Definitions/Nebulance.cs
// at ef9c182466ed04f34e6bcfedebb02c3c4af7eaf9. Copyright the Prowlarr
// contributors, GPL-3.0. Modified for SlipStream: rewritten in Go.

package nebulance

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/jatassi/slipstream/internal/newznab"
	"github.com/jatassi/slipstream/internal/ports/port"
	"github.com/jatassi/slipstream/internal/prowlarr/datetimeutil"
	"github.com/jatassi/slipstream/internal/prowlarr/parseutil"
	"github.com/jatassi/slipstream/internal/prowlarr/serializer"
	"github.com/jatassi/slipstream/internal/prowlarr/stringutil"
)

//go:embed upstream.json
var upstream []byte

// Port is Nebulance.
//
// SlipStream: CheckIfLoginNeeded's false and GetDownloadRequest's bare
// download are NoLogin's own behavior, so neither has code here. The
// mappings carry no description, as upstream gives none.
var Port = port.Define(port.Entry{
	ID:          "nebulance",
	Name:        "Nebulance",
	Description: "Nebulance is a private tracker for TV with no ratio to keep.",
	Language:    "en-US",
	Privacy:     port.Private,
	IndexerURLs: []string{"https://nebulance.io/"},
	Capabilities: port.Capabilities{
		TvSearchParams: []port.Param{port.Q, port.Season, port.Ep, port.ImdbID, port.TvMazeID},
		Categories: port.Categories(
			port.Map(1, newznab.TV, ""),
			port.Map(2, newznab.TVSD, ""),
			port.Map(3, newznab.TVHD, ""),
			port.Map(4, newznab.TVUHD, ""),
		),
	},
	Aliases:  []string{"nebulanceapi"},
	Upstream: port.ReadUpstream(upstream),
}, port.Impl[nebulanceSettings]{
	Login: port.NoLogin{},
	TV:    getTvSearchRequests,
	Basic: getBasicSearchRequests,
	Parse: parseResponse,
})

type indexer = port.Indexer[nebulanceSettings]

// SlipStream: the Movie, Music and Book overloads are gone. The
// capabilities declare none of them, so Prowlarr never calls them. The
// logger's warnings are gone too: a search the API can't take sends
// nothing and finds nothing.
func getTvSearchRequests(ix *indexer, searchCriteria port.SearchCriteria) (port.Chain, error) {
	var pageableRequests port.Chain

	var queryParams stringutil.NameValueCollection
	queryParams.Add("action", "search")
	queryParams.Add("age", ">0")

	if searchCriteria.TvMazeID > 0 {
		queryParams.Set("tvmaze", strconv.Itoa(searchCriteria.TvMazeID))
	} else if strings.TrimSpace(searchCriteria.ImdbID) != "" {
		queryParams.Set("imdb", searchCriteria.FullImdbID())
	}

	searchQuery := strings.TrimSpace(searchCriteria.SanitizedSearchTerm())

	if searchQuery != "" {
		queryParams.Set("release", searchQuery)
	}

	showDate, daily := datetimeutil.TryParseExact(fmt.Sprintf("%d %s", searchCriteria.Season, searchCriteria.Episode), "yyyy MM/dd", datetimeutil.None, ix.Clock)
	if searchCriteria.Season != 0 && strings.TrimSpace(searchCriteria.Episode) != "" && daily {
		if searchQuery != "" {
			queryParams.Set("name", searchQuery)
		}

		queryParams.Set("release", showDate.Format("2006.01.02"))
	} else {
		if searchCriteria.Season != 0 {
			queryParams.Set("season", strconv.Itoa(searchCriteria.Season))
		}

		if episodeNumber, err := strconv.Atoi(strings.TrimSpace(searchCriteria.Episode)); strings.TrimSpace(searchCriteria.Episode) != "" && err == nil {
			queryParams.Set("episode", strconv.Itoa(episodeNumber))
		}
	}

	if (strings.TrimSpace(queryParams.Get("season")) != "" || strings.TrimSpace(queryParams.Get("episode")) != "") &&
		strings.TrimSpace(queryParams.Get("name")) == "" &&
		strings.TrimSpace(queryParams.Get("release")) == "" &&
		strings.TrimSpace(queryParams.Get("tvmaze")) == "" &&
		strings.TrimSpace(queryParams.Get("imdb")) == "" {
		return port.Chain{}, nil
	}

	if n := len(queryParams.Get("name")); n > 0 && n < 3 {
		return port.Chain{}, nil
	}
	if n := len(queryParams.Get("release")); n > 0 && n < 3 {
		return port.Chain{}, nil
	}

	pageableRequests.Add(getPagedRequests(ix, &queryParams, searchCriteria))

	return pageableRequests, nil
}

func getBasicSearchRequests(ix *indexer, searchCriteria port.SearchCriteria) (port.Chain, error) {
	var pageableRequests port.Chain

	var queryParams stringutil.NameValueCollection
	queryParams.Add("action", "search")
	queryParams.Add("age", ">0")

	searchQuery := strings.TrimSpace(searchCriteria.SanitizedSearchTerm())

	if searchQuery != "" {
		queryParams.Set("release", searchQuery)
	}

	if n := len(queryParams.Get("release")); n > 0 && n < 3 {
		return port.Chain{}, nil
	}

	pageableRequests.Add(getPagedRequests(ix, &queryParams, searchCriteria))

	return pageableRequests, nil
}

// SlipStream: a Go zero Limit is C#'s null, which GetValueOrDefault makes
// 100.
func getPagedRequests(ix *indexer, parameters *stringutil.NameValueCollection, searchCriteria port.SearchCriteria) *port.Request {
	limit := searchCriteria.Limit
	if limit == 0 {
		limit = 100
	}

	parameters.Set("api_key", ix.Settings.APIKey)
	parameters.Set("per_page", strconv.Itoa(limit))

	if searchCriteria.Limit > 0 && searchCriteria.Offset > 0 {
		page := searchCriteria.Offset / searchCriteria.Limit
		parameters.Set("page", strconv.Itoa(page))
	}

	apiURL := ix.URL("api.php?" + parameters.GetQueryString())

	return port.Get(apiURL).Accept("application/json")
}

// SlipStream: the messages are SlipStream's own, per WRITING.md. A 401 is
// a key the site refused, which SlipStream names account_refused.
func parseResponse(ix *indexer, indexerResponse *port.IndexerResponse) ([]port.TorrentInfo, error) {
	var torrentInfos []port.TorrentInfo

	if indexerResponse.StatusCode == http.StatusUnauthorized {
		return nil, port.IndexerAuthError("")
	}

	if indexerResponse.StatusCode != http.StatusOK {
		return nil, port.IndexerError("the Indexer responded with status %d", indexerResponse.StatusCode)
	}

	var jsonResponse nebulanceResponse
	if err := json.Unmarshal(indexerResponse.Body, &jsonResponse); err != nil {
		var response struct{ Result string }
		if json.Unmarshal(indexerResponse.Body, &response) == nil && response.Result != "" {
			return nil, port.IndexerError("the Indexer responded with %q", response.Result)
		}
		return nil, port.IndexerError("read the Indexer's search response: %v", err)
	}

	if jsonResponse.Error != nil {
		return nil, port.IndexerError("the Indexer's API responded with the error %q", jsonResponse.Error.Message)
	}

	if jsonResponse.TotalResults == 0 || len(jsonResponse.Items) == 0 {
		return torrentInfos, nil
	}

	for _, row := range jsonResponse.Items {
		details := ix.BaseURL.String() + "torrents.php?id=" + strconv.FormatInt(int64(row.TorrentID), 10)

		title := row.GroupName
		if strings.TrimSpace(row.ReleaseTitle) != "" {
			title = row.ReleaseTitle
		}

		publishDate, err := datetimeutil.Parse(row.PublishDateUtc, datetimeutil.AssumeUniversal, ix.Clock)
		if err != nil {
			return nil, port.IndexerError("read torrent %d of the Indexer's search response: %v", row.TorrentID, err)
		}

		// SlipStream: IndexerBase adds Scene from TorrentInfo.Scene, which the
		// Go TorrentInfo doesn't have. It has no TvMazeId either, because a
		// Release carries none.
		var flags []port.SiteFlag
		if containsFold(row.Tags, "scene") {
			flags = append(flags, port.Scene)
		}

		var minimumSeedTime int64 = 86400 // 120 hours for seasons and 24 hours for episodes
		if strings.EqualFold(row.Category, "SEASON") {
			minimumSeedTime = 432000
		}

		release := port.TorrentInfo{
			GUID:                 details,
			InfoURL:              details,
			DownloadURL:          row.DownloadLink,
			Title:                strings.TrimSpace(title),
			Categories:           []newznab.Category{parseTvShowQuality(row.ReleaseTitle)},
			Size:                 int64(row.Size),
			Files:                len(row.FileList),
			PublishDate:          publishDate,
			Grabs:                int(row.Snatch),
			Seeders:              int(row.Seed),
			Peers:                int(row.Seed + row.Leech),
			IndexerFlags:         flags,
			MinimumRatio:         0, // ratioless
			MinimumSeedTime:      minimumSeedTime,
			DownloadVolumeFactor: 0, // ratioless tracker
			UploadVolumeFactor:   1,
		}

		if strings.TrimSpace(row.ImdbID) != "" {
			release.ImdbID, _ = parseutil.GetImdbID(row.ImdbID)
		}

		torrentInfos = append(torrentInfos, release)
	}

	return torrentInfos, nil
}

func containsFold(values []string, s string) bool {
	for _, v := range values {
		if strings.EqualFold(v, s) {
			return true
		}
	}
	return false
}

// SlipStream: the API key is required, because a search without one
// finds nothing.
type nebulanceSettings struct {
	port.APIKeyField `help:"Find your API key on Nebulance under User Settings > API Keys. The key needs the List and Download permissions."`
}

type nebulanceResponse struct {
	TotalResults serializer.STJInt      `json:"total_results"`
	Items        []nebulanceTorrent     `json:"items"`
	Error        *nebulanceErrorMessage `json:"error"`
}

type nebulanceTorrent struct {
	ReleaseTitle   string             `json:"rls_name"`
	Category       string             `json:"cat"`
	Size           serializer.STJInt  `json:"size"`
	Seed           serializer.STJInt  `json:"seed"`
	Leech          serializer.STJInt  `json:"leech"`
	Snatch         serializer.STJInt  `json:"snatch"`
	DownloadLink   string             `json:"download"`
	FileList       []string           `json:"file_list"`
	GroupName      string             `json:"group_name"`
	TorrentID      serializer.STJInt  `json:"group_id"`
	ImdbID         string             `json:"imdb_id"`
	TvMazeID       *serializer.STJInt `json:"tvmaze_id"`
	PublishDateUtc string             `json:"rls_utc"`
	Tags           []string           `json:"tags"`
}

type nebulanceErrorMessage struct {
	Message string `json:"message"`
}
