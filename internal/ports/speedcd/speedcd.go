// Ported from Prowlarr's src/NzbDrone.Core/Indexers/Definitions/SpeedCD.cs
// at ef9c182466ed04f34e6bcfedebb02c3c4af7eaf9. Copyright the Prowlarr
// contributors, GPL-3.0. Modified for SlipStream: rewritten in Go.

package speedcd

import (
	_ "embed"
	"regexp"
	"strconv"
	"strings"

	"github.com/jatassi/slipstream/internal/newznab"
	"github.com/jatassi/slipstream/internal/ports/port"
	"github.com/jatassi/slipstream/internal/prowlarr/datetimeutil"
	"github.com/jatassi/slipstream/internal/prowlarr/dom"
	"github.com/jatassi/slipstream/internal/prowlarr/encoding"
	"github.com/jatassi/slipstream/internal/prowlarr/parseutil"
)

//go:embed upstream.json
var upstream []byte

type ix = port.Indexer[SpeedCDSettings]

var Port = port.Define(port.Entry{
	ID:          "speedcd",
	Name:        "SpeedCD",
	Description: "SpeedCD is a private Tracker for movies, TV, games, music and software.",
	Language:    "en-US",
	Privacy:     port.Private,
	IndexerURLs: []string{
		"https://speed.cd/",
		"https://speed.click/",
		"https://speeders.me/",
	},
	Capabilities: setCapabilities(),
	Upstream:     port.ReadUpstream(upstream),
}, port.Impl[SpeedCDSettings]{
	Login: port.FormLogin[SpeedCDSettings]{DoLogin: doLogin, CheckIfLoginNeeded: checkIfLoginNeeded},
	Movie: func(ix *ix, q port.SearchCriteria) (port.Chain, error) {
		term := q.SanitizedSearchTerm()
		if strings.TrimSpace(q.FullImdbID()) != "" {
			term = q.FullImdbID()
		}

		return port.One(getPagedRequests(ix, strings.TrimSpace(term), q, strings.TrimSpace(q.FullImdbID()) != ""))
	},
	TV: func(ix *ix, q port.SearchCriteria) (port.Chain, error) {
		term := q.SanitizedTvSearchString()
		if strings.TrimSpace(q.FullImdbID()) != "" {
			term = q.FullImdbID()
			if strings.TrimSpace(q.EpisodeSearchString()) != "" {
				term += " " + q.EpisodeSearchString()
				if q.Season != 0 && strings.TrimSpace(q.Episode) == "" {
					term += "*"
				}
			}
		}

		return port.One(getPagedRequests(ix, strings.TrimSpace(term), q, strings.TrimSpace(q.FullImdbID()) != ""))
	},
	Basic: func(ix *ix, q port.SearchCriteria) (port.Chain, error) {
		return port.One(getPagedRequests(ix, q.SanitizedSearchTerm(), q, false))
	},
	Parse: parseResponse,
})

var tokenRegex = regexp.MustCompile(`name=\\"a\\" value=\\"([^"]+)\\"`)

func doLogin(in *port.SignIn[SpeedCDSettings]) error {
	loginPage, err := in.Do(port.Post(in.BaseURL.JoinPath("checkpoint", "API")).
		AddFormParameter("username", in.Settings.Username))
	if err != nil {
		return err
	}

	matches := tokenRegex.FindStringSubmatch(loginPage.Content)
	if matches == nil {
		// SlipStream: no message, since Prowlarr's isn't the site's words.
		return port.IndexerAuthError("")
	}

	token := matches[1]

	response, err := in.Do(port.Post(in.BaseURL.JoinPath("checkpoint/")).
		AddFormParameter("pwd", string(in.Settings.Password)).
		AddFormParameter("a", token))
	if err != nil {
		return err
	}

	if checkIfLoginNeeded(response) {
		doc := dom.Parse(response.Content)
		errorMessage := strings.TrimSpace(doc.QuerySelector("h5").TextContent())

		if strings.Contains(response.Content, "Wrong Captcha!") {
			// SlipStream: SlipStream's words in place of Prowlarr's, which
			// name Prowlarr.
			errorMessage = "SpeedCD shows a captcha after a failed sign-in. Sign in once with a browser on this network, then test the Indexer again."
		}

		// SlipStream: no fallback message, since the stop sentence names
		// the sign-in fields and Prowlarr's fallback isn't the site's words.
		return port.IndexerAuthError(errorMessage)
	}

	return nil
}

func checkIfLoginNeeded(httpResponse *port.IndexerResponse) bool {
	return !strings.Contains(httpResponse.Content, "/browse.php")
}

func setCapabilities() port.Capabilities {
	// SlipStream: no MusicSearchParams or BookSearchParams, since
	// SlipStream has neither search mode. Those releases still come back
	// from Basic.
	return port.Capabilities{
		TvSearchParams:    []port.Param{port.Q, port.Season, port.Ep, port.ImdbID},
		MovieSearchParams: []port.Param{port.Q, port.ImdbID},
		Categories: port.Categories(
			port.Map(1, newznab.MoviesOther, "Movies/XviD"),
			port.Map(42, newznab.Movies, "Movies/Packs"),
			port.Map(32, newznab.Movies, "Movies/Kids"),
			port.Map(43, newznab.MoviesHD, "Movies/HD"),
			port.Map(47, newznab.Movies, "Movies/DiVERSiTY"),
			port.Map(28, newznab.MoviesBluRay, "Movies/B-Ray"),
			port.Map(48, newznab.Movies3D, "Movies/3D"),
			port.Map(40, newznab.MoviesDVD, "Movies/DVD-R"),
			port.Map(56, newznab.Movies, "Movies/Anime"),
			port.Map(50, newznab.TVSport, "TV/Sports"),
			port.Map(52, newznab.TVHD, "TV/B-Ray"),
			port.Map(53, newznab.TVSD, "TV/DVD-R"),
			port.Map(41, newznab.TV, "TV/Packs"),
			port.Map(55, newznab.TV, "TV/Kids"),
			port.Map(57, newznab.TV, "TV/DiVERSiTY"),
			port.Map(49, newznab.TVHD, "TV/HD"),
			port.Map(2, newznab.TVSD, "TV/Episodes"),
			port.Map(30, newznab.TVAnime, "TV/Anime"),
			port.Map(25, newznab.PCISO, "Games/PC ISO"),
			port.Map(39, newznab.ConsoleWii, "Games/Wii"),
			port.Map(45, newznab.ConsolePS3, "Games/PS3"),
			port.Map(35, newznab.Console, "Games/Nintendo"),
			port.Map(33, newznab.ConsoleXBox360, "Games/XboX360"),
			port.Map(46, newznab.PCMobileOther, "Mobile"),
			port.Map(24, newznab.PC0day, "Apps/0DAY"),
			port.Map(51, newznab.PCMac, "Mac"),
			port.Map(54, newznab.Books, "Educational"),
			port.Map(27, newznab.Books, "Books-Mags"),
			port.Map(26, newznab.Audio, "Music/Audio"),
			port.Map(3, newznab.Audio, "Music/Flac"),
			port.Map(44, newznab.Audio, "Music/Pack"),
			port.Map(29, newznab.AudioVideo, "Music/Video"),
		),
	}
}

func getPagedRequests(ix *ix, term string, searchCriteria port.SearchCriteria, deep bool) *port.Request {
	var parameters []string

	parameters = append(parameters, ix.Categories.TrackerIDs(searchCriteria.Categories)...)

	if ix.Settings.FreeleechOnly {
		parameters = append(parameters, "freeleech")
	}

	if ix.Settings.ExcludeArchives {
		parameters = append(parameters, "norar")
	}

	if deep {
		parameters = append(parameters, "deep")
	}

	if strings.TrimSpace(term) != "" {
		parameters = append(parameters, "q", encoding.URLEncode(term))
	}

	if searchCriteria.Limit > 0 && searchCriteria.Offset > 0 {
		page := searchCriteria.Offset/searchCriteria.Limit + 1
		parameters = append(parameters, "p", strconv.Itoa(page))
	}

	searchURL := ix.URL("browse/" + strings.Join(parameters, "/"))

	return port.Get(searchURL).Accept("text/html")
}

func parseResponse(ix *ix, indexerResponse *port.IndexerResponse) ([]port.TorrentInfo, error) {
	var releaseInfos []port.TorrentInfo

	doc := dom.Parse(indexerResponse.Content)
	rows := doc.QuerySelectorAll("div.boxContent > table > tbody > tr")
	for _, row := range rows {
		title := cleanTitle(row.QuerySelector(`td:nth-child(2) > div > a[href^="/t/"]`).TextContent())
		downloadURL := ix.BaseURL.String() + strings.TrimLeft(row.QuerySelector(`td:nth-child(4) a[href^="/download/"]`).GetAttribute("href"), "/")
		infoURL := ix.BaseURL.String() + strings.TrimLeft(row.QuerySelector(`td:nth-child(2) > div > a[href^="/t/"]`).GetAttribute("href"), "/")
		seeders, err := parseutil.CoerceInt(row.QuerySelector("td:nth-child(8)").TextContent())
		if err != nil {
			return nil, err
		}
		leechers, err := parseutil.CoerceInt(row.QuerySelector("td:nth-child(9)").TextContent())
		if err != nil {
			return nil, err
		}
		dateAdded := strings.ReplaceAll(row.QuerySelector(`td:nth-child(2) span[class^="elapsedDate"]`).GetAttribute("title"), " at", "")
		catHref := row.QuerySelector("td:nth-child(1) a").GetAttribute("href")
		cat := catHref[strings.LastIndex(catHref, "/")+1:]
		downloadVolumeFactor := 1.0
		if row.QuerySelector(`td:nth-child(2) span:contains("[Freeleech]")`) != nil {
			downloadVolumeFactor = 0
		}

		publishDate, err := datetimeutil.ParseExact(dateAdded, "dddd, MMMM d, yyyy h:mmtt", datetimeutil.AssumeUniversal, ix.Clock)
		if err != nil {
			return nil, err
		}
		size, err := parseutil.GetBytes(row.QuerySelector("td:nth-child(6)").TextContent())
		if err != nil {
			return nil, err
		}
		grabs, err := parseutil.CoerceInt(row.QuerySelector("td:nth-child(7)").TextContent())
		if err != nil {
			return nil, err
		}

		release := port.TorrentInfo{
			GUID:                 infoURL,
			InfoURL:              infoURL,
			DownloadURL:          downloadURL,
			Title:                title,
			Categories:           ix.Categories.FromTracker(cat),
			PublishDate:          publishDate,
			Size:                 size,
			Grabs:                grabs,
			Seeders:              seeders,
			Peers:                seeders + leechers,
			DownloadVolumeFactor: downloadVolumeFactor,
			UploadVolumeFactor:   1,
			MinimumRatio:         1,
			MinimumSeedTime:      259200, // 72 hours
		}

		releaseInfos = append(releaseInfos, release)
	}

	return releaseInfos, nil
}

var requestTag = regexp.MustCompile(`(?i)\[REQ(UEST)?\]`)

func cleanTitle(title string) string {
	title = requestTag.ReplaceAllString(title, "")
	return strings.Trim(title, " .")
}

type SpeedCDSettings struct {
	port.UserPass
	FreeleechOnly   bool `port:"freeleechOnly" label:"Freeleech only" help:"Search only freeleech Releases."`
	ExcludeArchives bool `port:"excludeArchives" label:"Exclude archives" help:"Leave out Releases that hold RAR files."`
}
