// Ported from Prowlarr's src/NzbDrone.Core/Indexers/Definitions/BitHDTV.cs
// at ef9c182466ed04f34e6bcfedebb02c3c4af7eaf9. Copyright the Prowlarr
// contributors, GPL-3.0. Modified for SlipStream: rewritten in Go.

package bithdtv

import (
	_ "embed"
	"fmt"
	"net/url"
	"slices"
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

// SlipStream: the cookie is the pasted session CookieLogin asks for, so
// CookieTorrentBaseSettings leaves no setting here.
type settings struct{}

type ix = port.Indexer[settings]

var Port = port.Define(port.Entry{
	ID:           "bithdtv",
	Name:         "BitHDTV",
	IndexerURLs:  []string{"https://www.bit-hdtv.com/"},
	Description:  "BitHDTV is a private Tracker for HD movies, TV, documentaries, sports and music.",
	Language:     "en-US",
	Encoding:     "iso-8859-1",
	Privacy:      port.Private,
	Capabilities: SetCapabilities(),
	Upstream:     port.ReadUpstream(upstream),
}, port.Impl[settings]{
	Login: port.CookieLogin{CheckIfLoginNeeded: CheckIfLoginNeeded},
	Movie: func(ix *ix, searchCriteria port.SearchCriteria) (port.Chain, error) {
		return GetSearchRequests(ix, searchCriteria.SanitizedSearchTerm(), searchCriteria.Categories, searchCriteria.FullImdbID())
	},
	TV: func(ix *ix, searchCriteria port.SearchCriteria) (port.Chain, error) {
		return GetSearchRequests(ix, searchCriteria.SanitizedTvSearchString(), searchCriteria.Categories, searchCriteria.FullImdbID())
	},
	Basic: func(ix *ix, searchCriteria port.SearchCriteria) (port.Chain, error) {
		return GetSearchRequests(ix, searchCriteria.SanitizedSearchTerm(), searchCriteria.Categories, "")
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
		TvSearchParams:    []port.Param{port.Q, port.Season, port.Ep, port.ImdbID},
		MovieSearchParams: []port.Param{port.Q, port.ImdbID},
		Categories: port.Categories(
			port.Map(1, newznab.TVAnime, "Anime"),
			port.Map(2, newznab.MoviesBluRay, "Movies/Blu-ray"),
			port.Map(4, newznab.TVDocumentary, "Documentaries"),
			port.Map(6, newznab.AudioLossless, "HQ Audio"),
			port.Map(7, newznab.Movies, "Movies"),
			port.Map(8, newznab.AudioVideo, "Music Videos"),
			port.Map(9, newznab.Other, "Other"),
			port.Map(5, newznab.TVSport, "Sports"),
			port.Map(10, newznab.TV, "TV"),
			port.Map(12, newznab.TV, "TV/Seasonpack"),
			port.Map(11, newznab.XXX, "XXX"),
		),
	}
}

func GetPagedRequests(ix *ix, term string, categories []newznab.ID, imdbID string) []*port.Request {
	cat := "0"
	if trackerCats := ix.Categories.TrackerIDs(categories); len(trackerCats) == 1 {
		cat = trackerCats[0]
	}

	var qc stringutil.NameValueCollection
	qc.Add("cat", cat)

	search := func() *port.Request {
		searchURL := ix.URL("torrents.php")
		searchURL.RawQuery = qc.GetQueryString()
		return port.Get(searchURL).Accept("text/html")
	}

	if strings.TrimSpace(imdbID) != "" {
		qc.Add("search", imdbID)
		qc.Add("options", "4") // Search URL field for IMDB link
		first := search()

		qc.Set("Options", "1") // Search Title and Description
		return []*port.Request{first, search()}
	}

	// Site handles empty string on search param. No need to check for IsNullOrEmpty()
	qc.Add("search", term)
	qc.Add("options", "0") // Search Title Only
	return []*port.Request{search()}
}

func GetSearchRequests(ix *ix, term string, categories []newznab.ID, imdbID string) (port.Chain, error) {
	var pageableRequests port.Chain

	pageableRequests.Add(GetPagedRequests(ix, term, categories, imdbID)...)

	return pageableRequests, nil
}

func ParseResponse(ix *ix, indexerResponse *port.IndexerResponse) ([]port.TorrentInfo, error) {
	var torrentInfos []port.TorrentInfo

	doc := dom.Parse(indexerResponse.Content)
	for _, child := range doc.QuerySelectorAll("#needseed") {
		child.Remove()
	}

	table := doc.QuerySelector("table[align=center] + br + table > tbody")

	// No results, so skip this search
	if table == nil {
		return torrentInfos, nil
	}

	rows := table.Children()
	if len(rows) > 0 {
		rows = rows[1:]
	}

	for _, row := range rows {
		var release port.TorrentInfo
		children := row.Children()
		qLink := children[2].QuerySelector("a")
		release.MinimumRatio = 1
		release.MinimumSeedTime = 172800 // 48 hours
		release.Title = qLink.GetAttribute("title")
		detailsLink, err := url.Parse(qLink.GetAttribute("href"))
		if err != nil {
			return nil, fmt.Errorf("read a details link: %w", err)
		}

		// Skip irrelevant and duplicate entries
		if slices.ContainsFunc(torrentInfos, func(r port.TorrentInfo) bool { return r.GUID == detailsLink.String() }) {
			continue
		}

		if release.Files, err = parseutil.CoerceInt(children[3].TextContent()); err != nil {
			return nil, err
		}
		if release.Grabs, err = parseutil.CoerceInt(children[7].TextContent()); err != nil {
			return nil, err
		}
		release.GUID = detailsLink.String()
		release.InfoURL = release.GUID
		release.DownloadURL = ix.URL(row.QuerySelector(`a[href^="download.php"]`).GetAttribute("href")).String()
		catURL := ix.URL(children[1].FirstElementChild().GetAttribute("href"))
		catNum := catURL.Query().Get("cat")
		release.Categories = ix.Categories.FromTracker(catNum)

		dateString := strings.TrimSpace(children[5].TextContent())
		if release.PublishDate, err = datetimeutil.ParseExact(dateString, "yyyy-MM-ddHH:mm:ss", datetimeutil.None, ix.Clock); err != nil {
			return nil, err
		}
		sizeStr := children[6].TextContent()
		if release.Size, err = parseutil.GetBytes(sizeStr); err != nil {
			return nil, err
		}
		if release.Seeders, err = parseutil.CoerceInt(strings.TrimSpace(children[8].TextContent())); err != nil {
			return nil, err
		}
		leechers, err := parseutil.CoerceInt(strings.TrimSpace(children[9].TextContent()))
		if err != nil {
			return nil, err
		}
		release.Peers = leechers + release.Seeders
		switch row.GetAttribute("bgcolor") {
		case "#DDDDDD":
			release.DownloadVolumeFactor = 1
			release.UploadVolumeFactor = 2
		case "#FFFF99":
			release.DownloadVolumeFactor = 0
			release.UploadVolumeFactor = 1
		case "#CCFF99":
			release.DownloadVolumeFactor = 0
			release.UploadVolumeFactor = 2
		default:
			release.DownloadVolumeFactor = 1
			release.UploadVolumeFactor = 1
		}

		torrentInfos = append(torrentInfos, release)
	}

	return torrentInfos, nil
}
