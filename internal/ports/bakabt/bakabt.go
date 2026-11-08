// Ported from Prowlarr's src/NzbDrone.Core/Indexers/Definitions/BakaBT.cs
// at ef9c182466ed04f34e6bcfedebb02c3c4af7eaf9. Copyright the Prowlarr
// contributors, GPL-3.0. Modified for SlipStream: rewritten in Go.

package bakabt

import (
	_ "embed"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"unicode"

	"github.com/jatassi/slipstream/internal/newznab"
	"github.com/jatassi/slipstream/internal/ports/port"
	"github.com/jatassi/slipstream/internal/prowlarr/datetimeutil"
	"github.com/jatassi/slipstream/internal/prowlarr/dom"
	"github.com/jatassi/slipstream/internal/prowlarr/encoding"
	"github.com/jatassi/slipstream/internal/prowlarr/parseutil"
)

//go:embed upstream.json
var upstream []byte

type ix = port.Indexer[BakaBTSettings]

var Port = port.Define(port.Entry{
	ID:           "bakabt",
	Name:         "BakaBT",
	Description:  "BakaBT is a private Tracker for anime, with its music, manga and light novels.",
	Language:     "en-US",
	Privacy:      port.Private,
	IndexerURLs:  []string{"https://bakabt.me/"},
	Capabilities: setCapabilities(),
	Upstream:     port.ReadUpstream(upstream),
}, port.Impl[BakaBTSettings]{
	Login: port.FormLogin[BakaBTSettings]{DoLogin: doLogin, CheckIfLoginNeeded: checkIfLoginNeeded},
	Movie: func(ix *ix, q port.SearchCriteria) (port.Chain, error) {
		return port.One(getPagedRequests(ix, q.SanitizedSearchTerm()))
	},
	TV: func(ix *ix, q port.SearchCriteria) (port.Chain, error) {
		return port.One(getPagedRequests(ix, q.SanitizedSearchTerm()))
	},
	Basic: func(ix *ix, q port.SearchCriteria) (port.Chain, error) {
		return port.One(getPagedRequests(ix, q.SanitizedSearchTerm()))
	},
	Parse:    parseResponse,
	Download: download,
})

func download(ix *ix, link *url.URL) (port.Got, error) {
	response, err := ix.Fetch(port.Get(link))
	if err != nil {
		return nil, err
	}

	doc := dom.Parse(response.Content)
	downloadLink := doc.QuerySelector(".download_link").GetAttribute("href")

	if strings.TrimSpace(downloadLink) == "" {
		return nil, port.IndexerError("the release page has no download link")
	}

	torrentURL, err := url.Parse(ix.BaseURL.String() + downloadLink)
	if err != nil {
		return nil, fmt.Errorf("parse the download link: %w", err)
	}

	// SlipStream: base.Download is the framework's own fetch, which a
	// port's Download can't call, so this is its torrent case.
	torrent, err := ix.Fetch(port.Get(torrentURL))
	switch {
	case err != nil:
		return nil, err
	case torrent.StatusCode == http.StatusNotFound:
		return nil, port.ErrReleaseUnavailable
	case torrent.StatusCode < http.StatusOK || torrent.StatusCode >= http.StatusMultipleChoices:
		return nil, port.IndexerError("the Indexer responded with status %d", torrent.StatusCode)
	}

	return port.Torrent{Raw: torrent.Body}, nil
}

func doLogin(in *port.SignIn[BakaBTSettings]) error {
	loginURL := in.BaseURL.JoinPath("login.php")
	requestBuilder := port.Post(loginURL)

	loginPage, err := in.Do(port.Get(loginURL))
	if err != nil {
		return err
	}

	doc := dom.Parse(loginPage.Content)
	if loginKey := doc.QuerySelector(`input[name="loginKey"]`); loginKey != nil {
		requestBuilder.AddFormParameter("loginKey", loginKey.GetAttribute("value"))
	}

	response, err := in.Do(requestBuilder.
		AddFormParameter("username", in.Settings.Username).
		AddFormParameter("password", string(in.Settings.Password)).
		AddFormParameter("returnto", "/index.php"))
	if err != nil {
		return err
	}

	if checkIfLoginNeeded(response) {
		document := dom.Parse(response.Content)
		// SlipStream: no fallback message, since the stop sentence names
		// the sign-in fields and Prowlarr's fallback isn't the site's words.
		errorMessage := strings.TrimSpace(document.QuerySelector("#loginError, .error").TextContent())

		return port.IndexerAuthError(errorMessage)
	}

	return nil
}

func checkIfLoginNeeded(httpResponse *port.IndexerResponse) bool {
	return strings.Contains(httpResponse.Content, "loginForm")
}

func setCapabilities() port.Capabilities {
	// SlipStream: no MusicSearchParams or BookSearchParams, since
	// SlipStream has neither search mode. Those releases still come back
	// from Basic.
	return port.Capabilities{
		TvSearchParams:    []port.Param{port.Q, port.Season, port.Ep},
		MovieSearchParams: []port.Param{port.Q},
		Categories: port.Categories(
			port.Map(1, newznab.TVAnime, "Anime Series"),
			port.Map(2, newznab.TVAnime, "OVA"),
			port.Map(3, newznab.AudioOther, "Soundtrack"),
			port.Map(4, newznab.BooksComics, "Manga"),
			port.Map(5, newznab.Movies, "Anime Movie"),
			port.Map(6, newznab.TVOther, "Live Action"),
			port.Map(7, newznab.BooksOther, "Artbook"),
			port.Map(8, newznab.AudioVideo, "Music Video"),
			port.Map(9, newznab.BooksEBook, "Light Novel"),
		),
	}
}

// SlipStream: C#'s .*(?=\s(?:[Ee]\d+|\d+)$) with the lookahead, which Go
// lacks, as a group before the same tail.
var episodeSuffix = regexp.MustCompile(`^(.*)\s(?:[Ee]\d+|\d+)$`)

func getPagedRequests(ix *ix, term string) *port.Request {
	searchString := term
	searchURL := "browse.php?only=0&incomplete=1&lossless=1&hd=1&multiaudio=1&bonus=1&reorder=1&q="

	if ix.Settings.AdultContent {
		searchURL = "browse.php?only=0&hentai=1&incomplete=1&lossless=1&hd=1&multiaudio=1&bonus=1&reorder=1&q="
	}

	if match := episodeSuffix.FindStringSubmatch(term); match != nil {
		searchString = match[1]
	}

	episodeSearchURL := searchURL + encoding.URLEncode(searchString)

	return port.Get(ix.URL(episodeSearchURL))
}

func parseResponse(ix *ix, indexerResponse *port.IndexerResponse) ([]port.TorrentInfo, error) {
	var releaseInfos []port.TorrentInfo

	doc := dom.Parse(indexerResponse.Content)
	rows := doc.QuerySelectorAll(".torrents tr.torrent, .torrents tr.torrent_alt")
	currentCategories := []newznab.Category{newznab.TVAnime}

	for _, row := range rows {
		downloadVolumeFactor := 1.0
		if row.QuerySelector("span.freeleech") != nil {
			downloadVolumeFactor = 0
		}

		// Skip non-freeleech results when freeleech only is set
		if ix.Settings.FreeleechOnly && downloadVolumeFactor != 0 {
			continue
		}

		qTitleLink := row.QuerySelector("a.title, a.alt_title")
		if qTitleLink == nil {
			continue
		}

		title := strings.TrimSpace(qTitleLink.TextContent())

		// Insert before the release info
		taidx := strings.IndexByte(title, '(')
		tbidx := strings.IndexByte(title, '[')

		if taidx == -1 {
			taidx = len(title)
		}

		if tbidx == -1 {
			tbidx = len(title)
		}

		titleSplit := min(taidx, tbidx)
		titleSeries := title[:titleSplit]
		releaseInfo := title[titleSplit:]

		currentCategories = getNextCategory(ix, row, currentCategories)

		var titles []string
		for _, t := range strings.Split(titleSeries, " | ") {
			if t != "" {
				titles = append(titles, t)
			}
		}

		if len(titles) > 1 && !ix.Settings.AddRomajiTitle {
			titles = titles[1:]
		}

		for _, name := range titles {
			var release port.TorrentInfo
			release.Title = strings.TrimSpace(name + releaseInfo)

			// Ensure the season is defined as this tracker only deals with full seasons
			if !strings.Contains(release.Title, "Season") && ix.Settings.AppendSeason {
				// Insert before the release info
				aidx := strings.IndexByte(release.Title, '(')
				bidx := strings.IndexByte(release.Title, '[')

				if aidx == -1 {
					aidx = len(release.Title)
				}

				if bidx == -1 {
					bidx = len(release.Title)
				}

				insertPoint := min(aidx, bidx)
				release.Title = release.Title[:insertPoint] + " Season 1 " + release.Title[insertPoint:]
			}

			release.Categories = currentCategories
			// SlipStream: no description, which a release here doesn't carry.
			release.GUID = ix.BaseURL.String() + qTitleLink.GetAttribute("href")
			release.InfoURL = release.GUID
			release.DownloadURL = ix.BaseURL.String() + row.QuerySelector(".peers a").GetAttribute("href")

			grabs := strings.TrimRightFunc(strings.TrimRight(strings.TrimRightFunc(row.QuerySelectorAll(".peers")[0].FirstChild().NodeValue(), unicode.IsSpace), "/"), unicode.IsSpace)
			grabs = strings.ReplaceAll(grabs, "k", "000")
			var err error
			if release.Grabs, err = strconv.Atoi(grabs); err != nil {
				return nil, fmt.Errorf("read the grabs %q: %w", grabs, err)
			}
			peerLinks := row.QuerySelectorAll(".peers a")
			if release.Seeders, err = strconv.Atoi(peerLinks[0].TextContent()); err != nil {
				return nil, fmt.Errorf("read the seeders: %w", err)
			}
			leechers, err := strconv.Atoi(peerLinks[1].TextContent())
			if err != nil {
				return nil, fmt.Errorf("read the leechers: %w", err)
			}
			release.Peers = release.Seeders + leechers
			release.MinimumRatio = 1
			release.MinimumSeedTime = 172800 // 48 hours

			size := row.QuerySelector(".size").TextContent()
			if release.Size, err = parseutil.GetBytes(size); err != nil {
				return nil, err
			}

			// 22 Jul 15
			dateStr := strings.ReplaceAll(row.QuerySelector(".added").TextContent(), "'", "")
			if first, _, _ := strings.Cut(dateStr, " "); len(first) == 1 {
				dateStr = "0" + dateStr
			}

			switch {
			case strings.EqualFold(dateStr, "yesterday"):
				release.PublishDate = ix.Clock.Local().AddDate(0, 0, -1)
			case strings.EqualFold(dateStr, "today"):
				release.PublishDate = ix.Clock.Local()
			default:
				if release.PublishDate, err = datetimeutil.ParseExact(dateStr, "dd MMM yy", datetimeutil.None, ix.Clock); err != nil {
					return nil, err
				}
			}

			release.DownloadVolumeFactor = downloadVolumeFactor
			release.UploadVolumeFactor = 1

			releaseInfos = append(releaseInfos, release)
		}
	}

	return releaseInfos, nil
}

func getNextCategory(ix *ix, row *dom.Node, currentCategories []newznab.Category) []newznab.Category {
	if nextCategoryName := getCategoryName(row); nextCategoryName != "" {
		currentCategories = ix.Categories.FromDesc(nextCategoryName)
		if len(currentCategories) == 0 {
			return []newznab.Category{newznab.TVAnime}
		}
	}

	return currentCategories
}

func getCategoryName(row *dom.Node) string {
	categoryElement := row.QuerySelector("td.category span")
	if categoryElement == nil {
		return ""
	}

	categoryName := categoryElement.GetAttribute("title")
	if strings.TrimSpace(categoryName) == "" {
		return ""
	}

	return categoryName
}

type BakaBTSettings struct {
	port.UserPass
	FreeleechOnly  bool `port:"freeleechOnly" label:"Freeleech only" help:"Show only the freeleech Releases of each search."`
	AddRomajiTitle bool `port:"addRomajiTitle" label:"Add Romaji titles" help:"Also list each Release under its Romaji title."`
	AppendSeason   bool `port:"appendSeason" label:"Append season" help:"Add Season 1 to each title that names no season, since BakaBT lists only full seasons."`
	AdultContent   bool `port:"adultContent" label:"Adult content" help:"Include adult Releases. Turn on adult content in your BakaBT settings too."`
}
