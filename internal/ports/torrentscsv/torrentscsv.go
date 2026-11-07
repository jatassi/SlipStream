// Ported from Prowlarr's src/NzbDrone.Core/Indexers/Definitions/TorrentsCSV.cs
// at ef9c182466ed04f34e6bcfedebb02c3c4af7eaf9. Copyright the Prowlarr
// contributors, GPL-3.0. Modified for SlipStream: rewritten in Go.

package torrentscsv

import (
	_ "embed"
	"encoding/json"
	"net/url"
	"slices"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/jatassi/slipstream/internal/newznab"
	"github.com/jatassi/slipstream/internal/ports/port"
	"github.com/jatassi/slipstream/internal/prowlarr/stringutil"
)

//go:embed upstream.json
var upstream []byte

type ix = port.Indexer[struct{}]

var Port = port.Define(port.Entry{
	ID:           "torrentscsv",
	Name:         "TorrentsCSV",
	IndexerURLs:  []string{"https://torrents-csv.com/"},
	LegacyURLs:   []string{"https://torrents-csv.ml/"},
	Language:     "en-US",
	Description:  "TorrentsCSV is a public, open-source search engine over its own database of Releases.",
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
			port.Map(1, newznab.Other, ""),
		),
		NoRss: true,
	}
}

func getPagedRequests(c *ix, term string) []*port.Request {
	// search cannot be blank and needs at least 3 characters
	if strings.TrimSpace(term) == "" || utf8.RuneCountInString(term) < 3 {
		return nil
	}

	var qc stringutil.NameValueCollection
	qc.Add("size", "100")
	qc.Add("q", term)

	searchURL := c.URL("service/search")
	searchURL.RawQuery = qc.GetQueryString()

	return []*port.Request{port.Get(searchURL).Accept("application/json")}
}

func movieSearch(c *ix, q port.SearchCriteria) (port.Chain, error) {
	var pageableRequests port.Chain

	pageableRequests.Add(getPagedRequests(c, q.SanitizedSearchTerm())...)

	return pageableRequests, nil
}

func tvSearch(c *ix, q port.SearchCriteria) (port.Chain, error) {
	var pageableRequests port.Chain

	pageableRequests.Add(getPagedRequests(c, q.SanitizedTvSearchString())...)

	return pageableRequests, nil
}

func basicSearch(c *ix, q port.SearchCriteria) (port.Chain, error) {
	var pageableRequests port.Chain

	pageableRequests.Add(getPagedRequests(c, q.SanitizedSearchTerm())...)

	return pageableRequests, nil
}

func parseResponse(c *ix, r *port.IndexerResponse) ([]port.TorrentInfo, error) {
	var releaseInfos []port.TorrentInfo

	var jsonResponse torrentsCSVResponse
	if err := json.Unmarshal(r.Body, &jsonResponse); err != nil {
		return nil, port.IndexerError("read the Indexer's search response: %v", err)
	}

	for _, torrent := range jsonResponse.Torrents {
		if torrent == nil {
			continue
		}

		infoHash := torrent.InfoHash
		title := torrent.Name
		seeders := deref(torrent.Seeders)
		leechers := deref(torrent.Leechers)
		grabs := deref(torrent.Completed)
		publishDate := time.Unix(torrent.Created, 0).UTC()

		releaseInfos = append(releaseInfos, port.TorrentInfo{
			GUID: "magnet:?xt=urn:btih:" + infoHash,
			// SlipStream: the title is escaped into the query, where the C#
			// leaves it to Uri.
			InfoURL: port.NewURI(c.BaseURL).CombinePath("search").AddQueryParam("q", title).FullURI(), // there is no details link
			// SlipStream: a Release needs a link or a magnet, so the port
			// builds the magnet from the info hash, as the Cardigann engine
			// does for a site that isn't private.
			MagnetURL:            "magnet:?xt=urn:btih:" + infoHash + "&dn=" + url.QueryEscape(title),
			Title:                title,
			InfoHash:             infoHash,
			Categories:           []newznab.Category{newznab.Other},
			PublishDate:          publishDate,
			Size:                 torrent.Size,
			Grabs:                grabs,
			Seeders:              seeders,
			Peers:                leechers + seeders,
			DownloadVolumeFactor: 0,
			UploadVolumeFactor:   1,
		})
	}

	slices.SortStableFunc(releaseInfos, func(a, b port.TorrentInfo) int { return b.PublishDate.Compare(a.PublishDate) })

	return releaseInfos, nil
}

func deref(v *int) int {
	if v == nil {
		return 0
	}
	return *v
}

type torrentsCSVResponse struct {
	Torrents []*torrentsCSVTorrent `json:"torrents"`
}

type torrentsCSVTorrent struct {
	InfoHash  string `json:"infohash"`
	Name      string `json:"name"`
	Size      int64  `json:"size_bytes"`
	Created   int64  `json:"created_unix"`
	Leechers  *int   `json:"leechers"`
	Seeders   *int   `json:"seeders"`
	Completed *int   `json:"completed"`
}
