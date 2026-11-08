// Ported from Prowlarr's src/NzbDrone.Core/Indexers/Definitions/PixelHD.cs
// at ef9c182466ed04f34e6bcfedebb02c3c4af7eaf9. Copyright the Prowlarr
// contributors, GPL-3.0. Modified for SlipStream: rewritten in Go.

package pixelhd

import (
	_ "embed"
	"net/http"
	"strings"

	"github.com/jatassi/slipstream/internal/newznab"
	"github.com/jatassi/slipstream/internal/ports/port"
	"github.com/jatassi/slipstream/internal/prowlarr/datetimeutil"
	"github.com/jatassi/slipstream/internal/prowlarr/dom"
	"github.com/jatassi/slipstream/internal/prowlarr/parseutil"
	"github.com/jatassi/slipstream/internal/prowlarr/stringutil"
)

//go:embed upstream.json
var upstream []byte

// PixelHDSettings is empty.
// SlipStream: the cookie and its User-Agent are the pasted session that
// CookieLogin{UserAgent: true} asks for, which also requires the
// User-Agent as PixelHDSettingsValidator does, so neither is a setting
// here.
type PixelHDSettings struct{}

type ix = port.Indexer[PixelHDSettings]

var Port = port.Define(port.Entry{
	ID:           "pixelhd",
	Name:         "PiXELHD",
	IndexerURLs:  []string{"https://pixelhd.me/"},
	Description:  "PiXELHD is a private Tracker with no ratio requirement for HD movies in MP4.",
	Language:     "en-US",
	Privacy:      port.Private,
	Capabilities: SetCapabilities(),
	Upstream:     port.ReadUpstream(upstream),
}, port.Impl[PixelHDSettings]{
	Login: port.CookieLogin{UserAgent: true, CheckIfLoginNeeded: CheckIfLoginNeeded},
	Movie: func(ix *ix, searchCriteria port.SearchCriteria) (port.Chain, error) {
		return port.One(GetPagedRequests(ix, searchCriteria.SanitizedSearchTerm(), searchCriteria.FullImdbID()))
	},
	Basic: func(ix *ix, searchCriteria port.SearchCriteria) (port.Chain, error) {
		return port.One(GetPagedRequests(ix, searchCriteria.SanitizedSearchTerm(), ""))
	},
	Parse: ParseResponse,
})

// CheckIfLoginNeeded never reads a response as logged out, as upstream,
// so an expired cookie shows as a search with no Releases.
func CheckIfLoginNeeded(*port.IndexerResponse) bool {
	return false
}

func SetCapabilities() port.Capabilities {
	return port.Capabilities{
		MovieSearchParams: []port.Param{port.Q, port.ImdbID},
		Categories: port.Categories(
			port.Map("1", newznab.MoviesHD, ""),
		),
	}
}

func GetPagedRequests(ix *ix, term, imdbID string) *port.Request {
	var parameters stringutil.NameValueCollection
	parameters.Add("order_by", "time")
	parameters.Add("order_way", "desc")

	if strings.TrimSpace(imdbID) != "" {
		parameters.Set("imdbid", imdbID)
	}

	if strings.TrimSpace(term) != "" {
		parameters.Set("groupname", term)
	}

	searchURL := ix.URL("torrents.php")
	searchURL.RawQuery = parameters.GetQueryString()

	// SlipStream: the framework sends the pasted cookie's User-Agent on
	// every request, so the generator doesn't set it.
	return port.Get(searchURL).Accept("text/html")
}

func ParseResponse(ix *ix, indexerResponse *port.IndexerResponse) ([]port.TorrentInfo, error) {
	if indexerResponse.StatusCode != http.StatusOK {
		return nil, port.IndexerError("the search responded with status %d", indexerResponse.StatusCode)
	}

	var releaseInfos []port.TorrentInfo

	doc := dom.Parse(indexerResponse.Content)

	groups := doc.QuerySelectorAll("div.browsePoster")
	for _, group := range groups {
		groupName := strings.ReplaceAll(group.QuerySelector(`strong:has(a[title="View Torrent"])`).TextContent(), " ]", "]")

		imdbID, _ := parseutil.GetImdbID(lastSegment(group.QuerySelector(`a[href*="imdb.com/title/tt"]`).GetAttribute("href")))

		rows := group.QuerySelectorAll(`tr.group_torrent:has(a[href^="torrents.php?id="])`)
		for _, row := range rows {
			downloadURL := ix.BaseURL.String() + row.QuerySelector(`a[href^="torrents.php?action=download"]`).GetAttribute("href")
			infoURL := ix.BaseURL.String() + row.QuerySelector(`a[href^="torrents.php?id="]`).GetAttribute("href")

			title := strings.TrimSpace(row.QuerySelector(`a[href^="torrents.php?id="]`).TextContent())

			seeders, err := parseutil.CoerceInt(row.QuerySelector("td:nth-child(7)").TextContent())
			if err != nil {
				return nil, err
			}
			leechers, err := parseutil.CoerceInt(strings.TrimSpace(row.QuerySelector("td:nth-child(8)").TextContent()))
			if err != nil {
				return nil, err
			}
			size, err := parseutil.GetBytes(strings.TrimSpace(row.QuerySelector("td:nth-child(4)").TextContent()))
			if err != nil {
				return nil, err
			}
			grabs, err := parseutil.CoerceInt(row.QuerySelector("td:nth-child(6)").TextContent())
			if err != nil {
				return nil, err
			}

			release := port.TorrentInfo{
				GUID:                 infoURL,
				InfoURL:              infoURL,
				DownloadURL:          downloadURL,
				Title:                strings.TrimSpace(groupName + " " + title),
				Categories:           []newznab.Category{newznab.Movies, newznab.MoviesHD},
				Seeders:              seeders,
				Peers:                seeders + leechers,
				Size:                 size,
				Grabs:                grabs,
				DownloadVolumeFactor: 0,
				UploadVolumeFactor:   1,
				ImdbID:               imdbID,
				MinimumRatio:         1,
				MinimumSeedTime:      259200, // 72 hours
			}

			publishDate, ok := datetimeutil.TryParseExact(
				strings.TrimSpace(row.QuerySelector(`td:nth-child(3) span[class="time"]`).GetAttribute("title")),
				"MMM dd yyyy, HH:mm", datetimeutil.None, ix.Clock)
			if !ok {
				if publishDate, err = datetimeutil.FromTimeAgo(strings.TrimSpace(row.QuerySelector("td:nth-child(3)").TextContent()), ix.Clock); err != nil {
					return nil, err
				}
			}
			release.PublishDate = publishDate

			releaseInfos = append(releaseInfos, release)
		}
	}

	return releaseInfos, nil
}

// lastSegment is href.TrimEnd('/').Split('/').LastOrDefault().
func lastSegment(href string) string {
	href = strings.TrimRight(href, "/")
	return href[strings.LastIndex(href, "/")+1:]
}
