// Ported from Prowlarr's src/NzbDrone.Core/Indexers/Definitions/SubsPlease.cs
// at ef9c182466ed04f34e6bcfedebb02c3c4af7eaf9. Copyright the Prowlarr
// contributors, GPL-3.0. Modified for SlipStream: rewritten in Go.

package subsplease

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"strconv"
	"strings"

	"github.com/jatassi/slipstream/internal/newznab"
	"github.com/jatassi/slipstream/internal/ports/port"
	"github.com/jatassi/slipstream/internal/prowlarr/datetimeutil"
	"github.com/jatassi/slipstream/internal/prowlarr/stringutil"
)

//go:embed upstream.json
var upstream []byte

type ix = port.Indexer[struct{}]

var Port = port.Define(port.Entry{
	ID:   "subsplease",
	Name: "SubsPlease",
	IndexerURLs: []string{
		"https://subsplease.org/",
		"https://subsplease.mrunblock.bond/",
		"https://subsplease.nocensor.click/",
	},
	LegacyURLs: []string{
		"https://subsplease.nocensor.space/",
	},
	Language:     "en-US",
	Description:  "SubsPlease is a public site for new anime episodes with English subtitles.",
	Privacy:      port.Public,
	Capabilities: setCapabilities(),
	Upstream:     port.ReadUpstream(upstream),
}, port.Impl[struct{}]{
	Login: port.NoLogin{},
	Movie: movieSearch, TV: tvSearch, Basic: basicSearch,
	Parse: parseResponse,
})

func setCapabilities() port.Capabilities {
	return port.Capabilities{
		TvSearchParams:    []port.Param{port.Q, port.Season, port.Ep},
		MovieSearchParams: []port.Param{port.Q},
		Categories: port.Categories(
			port.Map(1, newznab.TVAnime, ""),
			port.Map(2, newznab.MoviesOther, ""),
		),
	}
}

var resolutionRegex = regexp.MustCompile(`(?i)\d{3,4}p`)

func movieSearch(c *ix, q port.SearchCriteria) (port.Chain, error) {
	return port.One(getSearchRequests(c, q.SanitizedSearchTerm(), q))
}

func tvSearch(c *ix, q port.SearchCriteria) (port.Chain, error) {
	searchTerm := strings.TrimSpace(q.SanitizedSearchTerm())

	// Only include season > 1 in searchTerm, format as S2 rather than S02
	if q.Season > 1 {
		searchTerm += fmt.Sprintf(" S%d", q.Season)
	}

	if episode, err := strconv.Atoi(q.Episode); err == nil && episode > 0 {
		searchTerm += fmt.Sprintf(" %02d", episode)
	}

	return port.One(getSearchRequests(c, searchTerm, q))
}

func basicSearch(c *ix, q port.SearchCriteria) (port.Chain, error) {
	return port.One(getSearchRequests(c, q.SanitizedSearchTerm(), q))
}

var subsPleaseTag = regexp.MustCompile(`(?i)\[?SubsPlease\]?\s*`)

func getSearchRequests(c *ix, term string, q port.SearchCriteria) *port.Request {
	var queryParameters stringutil.NameValueCollection
	queryParameters.Add("tz", "UTC")

	if q.IsRssSearch() {
		queryParameters.Set("f", "latest")
	} else {
		searchTerm := strings.TrimSpace(subsPleaseTag.ReplaceAllString(term, ""))

		// If the search terms contain a resolution, remove it from the query sent to the API
		if resolutionMatch := resolutionRegex.FindString(searchTerm); resolutionMatch != "" {
			searchTerm = strings.TrimSpace(strings.ReplaceAll(searchTerm, resolutionMatch, ""))
		}

		queryParameters.Set("f", "search")
		queryParameters.Set("s", searchTerm)
	}

	searchURL := c.URL("api/")
	searchURL.RawQuery = queryParameters.GetQueryString()

	return port.Get(searchURL).Accept("application/json")
}

var regexSize = regexp.MustCompile(`(?i)&xl=(?P<size>\d+)`)

// SlipStream: the messages are SlipStream's own, per WRITING.md.
func parseResponse(c *ix, r *port.IndexerResponse) ([]port.TorrentInfo, error) {
	var torrentInfos []port.TorrentInfo

	if r.StatusCode != http.StatusOK {
		return nil, port.IndexerError("the Indexer responded with status %d", r.StatusCode)
	}

	// When there are no results, the API returns an empty array or empty response instead of an object
	if strings.TrimSpace(r.Content) == "" || r.Content == "[]" {
		return torrentInfos, nil
	}

	values, err := decodeReleases(r.Body)
	if err != nil {
		return nil, port.IndexerError("read the Indexer's search response: %v", err)
	}

	for _, value := range values {
		releaseDate, err := datetimeutil.Parse(value.ReleaseDate, datetimeutil.None, c.Clock)
		if err != nil {
			return nil, port.IndexerError("read the release date of %q in the Indexer's search response: %v", value.Show, err)
		}

		for _, d := range value.Downloads {
			categories := []newznab.Category{newznab.TVAnime}

			// SlipStream: a Release has no poster, so ImageUrl isn't read.

			if strings.EqualFold(value.Episode, "movie") {
				categories = append(categories, newznab.MoviesOther)
			}

			torrentInfos = append(torrentInfos, port.TorrentInfo{
				InfoURL:     fmt.Sprintf("%sshows/%s/", c.BaseURL, value.Page),
				PublishDate: releaseDate,
				Files:       1,
				Categories:  categories,
				Seeders:     1,
				Peers:       2,
				// Ex: [SubsPlease] Shingeki no Kyojin (The Final Season) - 64 (1080p)
				Title:                fmt.Sprintf("[SubsPlease] %s - %s (%sp)", value.Show, value.Episode, d.Resolution),
				MagnetURL:            d.Magnet,
				GUID:                 d.Magnet,
				Size:                 getReleaseSize(d),
				MinimumRatio:         1,
				MinimumSeedTime:      172800, // 48 hours
				DownloadVolumeFactor: 0,
				UploadVolumeFactor:   1,
			})
		}
	}

	return torrentInfos, nil
}

func getReleaseSize(info subPleaseDownloadInfo) int64 {
	if strings.TrimSpace(info.Magnet) != "" {
		if sizeMatch := regexSize.FindStringSubmatch(info.Magnet); sizeMatch != nil {
			if releaseSize, err := strconv.ParseInt(sizeMatch[regexSize.SubexpIndex("size")], 10, 64); err == nil && releaseSize > 0 {
				return releaseSize
			}
		}
	}

	// The API doesn't tell us file size, so give an estimate based on resolution
	switch info.Resolution {
	case "1080":
		return 1395864371 // 1.3 GiB, rounded as Convert.ToInt64 does
	case "720":
		return 700 << 20
	case "480":
		return 350 << 20
	default:
		return 1 << 30
	}
}

// decodeReleases is the response's Dictionary<string, SubPleaseRelease>
// as Newtonsoft fills it: its values in the order the object lists them.
func decodeReleases(body []byte) ([]subPleaseRelease, error) {
	dec := json.NewDecoder(bytes.NewReader(body))
	if open, err := dec.Token(); err != nil || open != json.Delim('{') {
		return nil, errors.New("the response isn't a JSON object")
	}
	var values []subPleaseRelease
	for dec.More() {
		if _, err := dec.Token(); err != nil {
			return nil, err //nolint:wrapcheck // the caller names the response
		}
		var value subPleaseRelease
		if err := dec.Decode(&value); err != nil {
			return nil, err //nolint:wrapcheck // the caller names the response
		}
		values = append(values, value)
	}
	return values, nil
}

type subPleaseRelease struct {
	Time        string                  `json:"time"`
	ReleaseDate string                  `json:"release_date"`
	Show        string                  `json:"show"`
	Episode     string                  `json:"episode"`
	Downloads   []subPleaseDownloadInfo `json:"downloads"`
	Xdcc        string                  `json:"xdcc"`
	ImageURL    string                  `json:"image_url"`
	Page        string                  `json:"page"`
}

type subPleaseDownloadInfo struct {
	Resolution string `json:"res"`
	Magnet     string `json:"magnet"`
}
