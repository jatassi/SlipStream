// Ported from Prowlarr's src/NzbDrone.Core/Indexers/Definitions/Shazbat.cs
// at ef9c182466ed04f34e6bcfedebb02c3c4af7eaf9. Copyright the Prowlarr
// contributors, GPL-3.0. Modified for SlipStream: rewritten in Go.

package shazbat

import (
	_ "embed"
	"net/http"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/jatassi/slipstream/internal/newznab"
	"github.com/jatassi/slipstream/internal/ports/port"
	"github.com/jatassi/slipstream/internal/prowlarr/datetimeutil"
	"github.com/jatassi/slipstream/internal/prowlarr/dom"
	"github.com/jatassi/slipstream/internal/prowlarr/parseutil"
)

//go:embed upstream.json
var upstream []byte

type ix = port.Indexer[ShazbatSettings]

var Port = port.Define(port.Entry{
	ID:           "shazbat",
	Name:         "Shazbat",
	Description:  "Shazbat is a private Tracker for curated TV.",
	Language:     "en-US",
	Privacy:      port.Private,
	IndexerURLs:  []string{"https://www.shazbat.tube/"},
	LegacyURLs:   []string{"https://www.shazbat.tv/"},
	RateLimit:    5100 * time.Millisecond,
	Capabilities: setCapabilities(),
	Upstream:     port.ReadUpstream(upstream),
}, port.Impl[ShazbatSettings]{
	Login: port.FormLogin[ShazbatSettings]{DoLogin: doLogin, CheckIfLoginNeeded: checkIfLoginNeeded},
	TV: func(ix *ix, q port.SearchCriteria) (port.Chain, error) {
		return port.One(getPagedRequests(ix, q.SanitizedSearchTerm()))
	},
	Basic: func(ix *ix, q port.SearchCriteria) (port.Chain, error) {
		return port.One(getPagedRequests(ix, q.SanitizedSearchTerm()))
	},
	Parse: parseResponse,
})

func doLogin(in *port.SignIn[ShazbatSettings]) error {
	loginURL := in.BaseURL.JoinPath("login")
	response, err := in.Do(port.Post(loginURL).
		AddFormParameter("referer", "").
		AddFormParameter("query", "").
		AddFormParameter("tv_timezone", "0").
		AddFormParameter("username", in.Settings.Username).
		AddFormParameter("password", string(in.Settings.Password)).
		SetHeader("Referer", loginURL.String()))
	if err != nil {
		return err
	}

	if checkIfLoginNeeded(response) {
		doc := dom.Parse(response.Content)
		// SlipStream: no fallback message, since the stop sentence names
		// the sign-in fields and Prowlarr's fallback isn't the site's words.
		errorMessage := strings.TrimSpace(doc.QuerySelector("div#fail .modal-body").TextContent())

		return port.IndexerAuthError(errorMessage)
	}

	return nil
}

func checkIfLoginNeeded(httpResponse *port.IndexerResponse) bool {
	content := strings.ToLower(httpResponse.Content)
	return (httpResponse.HasHTTPRedirect && strings.Contains(strings.ToLower(httpResponse.RedirectURL), "login")) ||
		strings.Contains(content, "sign in now") ||
		(strings.Contains(content, "fullredirect") && strings.Contains(content, "login"))
}

func setCapabilities() port.Capabilities {
	// SlipStream: no Flags list, since a port sets the Scene flag on each
	// release.
	return port.Capabilities{
		TvSearchParams: []port.Param{port.Q, port.Season, port.Ep},
		Categories: port.Categories(
			port.Map("1", newznab.TV, ""),
			port.Map("2", newznab.TVSD, ""),
			port.Map("3", newznab.TVHD, ""),
			port.Map("4", newznab.TVUHD, ""),
		),
	}
}

func getPagedRequests(ix *ix, term string) *port.Request {
	term = fixSearchTerm(term)

	if strings.TrimSpace(term) != "" {
		return port.Get(ix.URL("search")).
			AddQueryParam("search", term).
			AddQueryParam("portlet", "true").
			SetHeader("X-Requested-With", "XMLHttpRequest").
			SetHeader("Referer", ix.BaseURL.String()).
			Accept("text/html")
	}

	return port.Get(ix.URL("torrents")).
		SetHeader("Referer", ix.BaseURL.String()).
		Accept("text/html")
}

var (
	seasonOrEpisode = regexp.MustCompile(`(?i)\b[S|E]\d+\b`)
	yearOrDate      = regexp.MustCompile(`(.+)\b\d{4}(\.\d{2}\.\d{2})?\b`)
	termSeparators  = regexp.MustCompile(`[.\s\(\)\[\]]+`)
)

func fixSearchTerm(term string) string {
	term = seasonOrEpisode.ReplaceAllString(term, "")
	term = yearOrDate.ReplaceAllString(term, "$1")
	term = termSeparators.ReplaceAllString(term, " ")
	return strings.TrimSpace(strings.ToLower(term))
}

var (
	hdResolutions    = []string{"1080p", "1080i", "720p"}
	torrentInfoRegex = regexp.MustCompile(`\((?P<size>\d+)\)\s*:(?P<seeders>\d+) / :(?P<leechers>\d+)$`)
)

func parseResponse(ix *ix, indexerResponse *port.IndexerResponse) ([]port.TorrentInfo, error) {
	var releaseInfos []port.TorrentInfo

	doc := dom.Parse(indexerResponse.Content)

	hasGlobalFreeleech := doc.QuerySelector(`span:contains("Freeleech until:"):has(span.datetime)`) != nil

	results, err := parseResults(ix, indexerResponse, hasGlobalFreeleech)
	if err != nil {
		return nil, err
	}
	releaseInfos = append(releaseInfos, results...)

	shows := doc.QuerySelectorAll("div.show[data-id]")
	if len(shows) > 0 {
		// SlipStream: the setting's min and max bound it to 1 through 5,
		// where C# checks the range here.
		showPagesFetchLimit := ix.Settings.ShowPagesFetchLimit
		if showPagesFetchLimit == 0 {
			showPagesFetchLimit = 2
		}

		if len(indexerResponse.Cookies()) == 0 {
			return nil, port.IndexerError("the search page set no session cookie, so SlipStream can't fetch its show pages")
		}

		for _, show := range shows[:min(showPagesFetchLimit, len(shows))] {
			showPageURL := port.Get(ix.URL("show")).AddQueryParam("id", show.GetAttribute("data-id")).URL.String()

			// SlipStream: the session carries the cookies, Fetch signs
			// in again on a logged-out page, and a 429 is already limited.
			releaseResponse, err := ix.Fetch(port.Get(ix.URL("show")).
				AddQueryParam("id", show.GetAttribute("data-id")).
				AddQueryParam("show_mode", "torrents").
				AddQueryParam("portlet", "true").
				AddQueryParam("tab", "true").
				SetHeader("X-Requested-With", "XMLHttpRequest").
				SetHeader("Referer", showPageURL).
				Accept("text/html"))
			if err != nil {
				return nil, err
			}

			if releaseResponse.StatusCode >= http.StatusBadRequest {
				return nil, port.IndexerError("the Indexer responded to a show page with status %d", releaseResponse.StatusCode)
			}

			results, err := parseResults(ix, releaseResponse, hasGlobalFreeleech)
			if err != nil {
				return nil, err
			}
			releaseInfos = append(releaseInfos, results...)
		}
	}

	return releaseInfos, nil
}

func parseResults(ix *ix, indexerResponse *port.IndexerResponse, hasGlobalFreeleech bool) ([]port.TorrentInfo, error) {
	var releaseInfos []port.TorrentInfo

	doc := dom.Parse(indexerResponse.Content)

	if !hasGlobalFreeleech {
		hasGlobalFreeleech = doc.QuerySelector(`span:contains("Freeleech until:"):has(span.datetime)`) != nil
	}

	publishDate := ix.Clock.Local()

	rows := doc.QuerySelectorAll("#torrent-table tr.eprow, table tr.eprow")
	for _, row := range rows {
		downloadURL := ix.BaseURL.String() + row.QuerySelector(`td:nth-of-type(5) a[href^="load_torrent?"]`).GetAttribute("href")
		infoURL := ix.BaseURL.String() + row.QuerySelector(`td:nth-of-type(5) [href^="torrent_info?"]`).GetAttribute("href")
		title, err := parseTitle(row.QuerySelector("td:nth-of-type(3)"))
		if err != nil {
			return nil, err
		}
		infoString := strings.TrimSpace(row.QuerySelector("td:nth-of-type(4)").TextContent())
		var size int64
		var seeders, leechers int
		if matchInfo := torrentInfoRegex.FindStringSubmatch(infoString); matchInfo != nil {
			size, _ = strconv.ParseInt(matchInfo[torrentInfoRegex.SubexpIndex("size")], 10, 64)
			seeders, _ = strconv.Atoi(matchInfo[torrentInfoRegex.SubexpIndex("seeders")])
			leechers, _ = strconv.Atoi(matchInfo[torrentInfoRegex.SubexpIndex("leechers")])
		}

		var timestamp int64
		var hasTimestamp bool
		if dateTimestamp := row.QuerySelector(".datetime[data-timestamp]"); dateTimestamp != nil {
			timestamp, hasTimestamp = parseutil.TryCoerceLong(dateTimestamp.GetAttribute("data-timestamp"))
		}
		if hasTimestamp {
			publishDate = datetimeutil.UnixTimestampToDateTime(timestamp, ix.Clock)
		} else {
			publishDate = publishDate.Add(-time.Minute)
		}

		downloadVolumeFactor := 1.0
		if hasGlobalFreeleech {
			downloadVolumeFactor = 0
		}

		// SlipStream: no Genres, which a release here doesn't carry.
		release := port.TorrentInfo{
			GUID:                 infoURL,
			InfoURL:              infoURL,
			DownloadURL:          downloadURL,
			Title:                title,
			Categories:           parseCategories(title),
			Size:                 size,
			Seeders:              seeders,
			Peers:                seeders + leechers,
			PublishDate:          publishDate,
			IndexerFlags:         []port.SiteFlag{port.Scene},
			DownloadVolumeFactor: downloadVolumeFactor,
			UploadVolumeFactor:   1,
			MinimumRatio:         1,
			MinimumSeedTime:      172800, // 48 hours
		}

		releaseInfos = append(releaseInfos, release)
	}

	return releaseInfos, nil
}

func parseTitle(titleRow *dom.Node) (string, error) {
	for _, n := range titleRow.ChildNodes() {
		if n.NodeType() == dom.Text && strings.TrimSpace(n.TextContent()) != "" {
			return strings.TrimSpace(n.TextContent()), nil
		}
	}

	return "", port.IndexerError("a search result has no title")
}

func parseCategories(title string) []newznab.Category {
	second := newznab.TVSD
	switch {
	case slices.ContainsFunc(hdResolutions, func(r string) bool { return strings.Contains(title, r) }):
		second = newznab.TVHD
	case strings.Contains(title, "2160p"):
		second = newznab.TVUHD
	}

	return []newznab.Category{newznab.TV, second}
}

type ShazbatSettings struct {
	port.UserPass
	ShowPagesFetchLimit int `port:"showPagesFetchLimit" label:"Show pages per search" default:"2" min:"1" max:"5" help:"The number of show pages SlipStream fetches for each search."`
}
