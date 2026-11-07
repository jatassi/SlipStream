// Ported from Prowlarr's src/NzbDrone.Core/Indexers/Definitions/ImmortalSeed.cs
// at ef9c182466ed04f34e6bcfedebb02c3c4af7eaf9. Copyright the Prowlarr
// contributors, GPL-3.0. Modified for SlipStream: rewritten in Go.

package immortalseed

import (
	_ "embed"
	"regexp"
	"strings"
	"time"

	"github.com/jatassi/slipstream/internal/newznab"
	"github.com/jatassi/slipstream/internal/ports/port"
	"github.com/jatassi/slipstream/internal/prowlarr/datetimeutil"
	"github.com/jatassi/slipstream/internal/prowlarr/dom"
	"github.com/jatassi/slipstream/internal/prowlarr/parseutil"
	"github.com/jatassi/slipstream/internal/prowlarr/stringutil"
)

//go:embed upstream.json
var upstream []byte

type ix = port.Indexer[ImmortalSeedSettings]

var Port = port.Define(port.Entry{
	ID:           "immortalseed",
	Name:         "ImmortalSeed",
	Description:  "ImmortalSeed is a private Tracker for movies, TV and general releases.",
	Language:     "en-US",
	Privacy:      port.Private,
	IndexerURLs:  []string{"https://immortalseed.me/"},
	RateLimit:    5 * time.Second,
	Capabilities: setCapabilities(),
	Upstream:     port.ReadUpstream(upstream),
}, port.Impl[ImmortalSeedSettings]{
	Login: port.FormLogin[ImmortalSeedSettings]{DoLogin: doLogin, CheckIfLoginNeeded: checkIfLoginNeeded},
	Movie: func(ix *ix, q port.SearchCriteria) (port.Chain, error) {
		return port.One(getPagedRequests(ix, q.SanitizedSearchTerm(), q.Categories))
	},
	TV: func(ix *ix, q port.SearchCriteria) (port.Chain, error) {
		return port.One(getPagedRequests(ix, q.SanitizedTvSearchString(), q.Categories))
	},
	Basic: func(ix *ix, q port.SearchCriteria) (port.Chain, error) {
		return port.One(getPagedRequests(ix, q.SanitizedSearchTerm(), q.Categories))
	},
	Parse: parseResponse,
})

func doLogin(in *port.SignIn[ImmortalSeedSettings]) error {
	response, err := in.Do(port.Post(in.BaseURL.JoinPath("takelogin.php")).
		AddFormParameter("username", in.Settings.Username).
		AddFormParameter("password", string(in.Settings.Password)))
	if err != nil {
		return err
	}

	if !strings.Contains(response.Content, "logout.php") {
		// SlipStream: no message, since the stop sentence names the
		// sign-in fields and Prowlarr's isn't the site's words.
		return port.IndexerAuthError("")
	}

	return nil
}

func checkIfLoginNeeded(httpResponse *port.IndexerResponse) bool {
	return !strings.Contains(httpResponse.Content, "logout.php")
}

func setCapabilities() port.Capabilities {
	// SlipStream: no MusicSearchParams or BookSearchParams, since
	// SlipStream has neither search mode. Those releases still come back
	// from Basic.
	return port.Capabilities{
		TvSearchParams:    []port.Param{port.Q, port.Season, port.Ep},
		MovieSearchParams: []port.Param{port.Q},
		Categories: port.Categories(
			port.Map(3, newznab.Other, "Nuked"),
			port.Map(32, newznab.TVAnime, "Anime"),
			port.Map(23, newznab.PC, "Apps"),
			port.Map(35, newznab.AudioAudiobook, "Audiobooks"),
			port.Map(31, newznab.TV, "Childrens/Cartoons"),
			port.Map(54, newznab.TVDocumentary, "Documentary - HD"),
			port.Map(53, newznab.TVDocumentary, "Documentary - SD"),
			port.Map(22, newznab.BooksEBook, "Ebooks"),
			port.Map(41, newznab.BooksComics, "Comics"),
			port.Map(46, newznab.BooksMags, "Magazines"),
			port.Map(25, newznab.PCGames, "Games"),
			port.Map(61, newznab.ConsoleNDS, "Games Nintendo"),
			port.Map(26, newznab.PCGames, "Games-PC ISO"),
			port.Map(28, newznab.ConsolePS4, "Games-PSx"),
			port.Map(29, newznab.ConsoleXBox, "Games Xbox"),
			port.Map(49, newznab.PCMobileOther, "Mobile"),
			port.Map(51, newznab.PCMobileAndroid, "Android"),
			port.Map(50, newznab.PCMobileiOS, "IOS"),
			port.Map(52, newznab.PC0day, "Windows"),
			port.Map(59, newznab.MoviesUHD, "Movies-4k"),
			port.Map(60, newznab.MoviesForeign, "Non-English 4k Movies"),
			port.Map(16, newznab.MoviesHD, "Movies HD"),
			port.Map(18, newznab.MoviesForeign, "Movies HD Non-English"),
			port.Map(17, newznab.MoviesSD, "TS/CAM/PPV"),
			port.Map(34, newznab.MoviesForeign, "Movies Low Def Non-English"),
			port.Map(62, newznab.Movies, "Movies-Packs"),
			port.Map(14, newznab.MoviesSD, "Movies-SD"),
			port.Map(33, newznab.MoviesForeign, "Movies SD Non-English"),
			port.Map(30, newznab.AudioOther, "Music"),
			port.Map(37, newznab.AudioLossless, "FLAC"),
			port.Map(36, newznab.AudioMP3, "MP3"),
			port.Map(39, newznab.AudioOther, "Music Other"),
			port.Map(38, newznab.AudioVideo, "Music Video"),
			port.Map(45, newznab.Other, "Other"),
			port.Map(7, newznab.TVSport, "Sports Tv"),
			port.Map(44, newznab.TVSport, "Sports Fitness-Instructional"),
			port.Map(58, newznab.TVSport, "Olympics"),
			port.Map(67, newznab.TVSport, "World Cup"),
			port.Map(47, newznab.TVSD, "TV - 480p"),
			port.Map(64, newznab.TVUHD, "TV - 4K"),
			port.Map(8, newznab.TVHD, "TV - High Definition"),
			port.Map(48, newznab.TVSD, "TV - Standard Definition - x264"),
			port.Map(9, newznab.TVSD, "TV - Standard Definition - XviD"),
			port.Map(63, newznab.TVUHD, "TV Season Packs - 4K"),
			port.Map(4, newznab.TVHD, "TV Season Packs - HD"),
			port.Map(6, newznab.TVSD, "TV Season Packs - SD"),
		),
	}
}

// SlipStream: C#'s [ -._] written with its range spelled out, space to
// dot, as .NET and Go both read it.
var termSeparators = regexp.MustCompile(`[\x20-\x2e_]+`)

func getPagedRequests(ix *ix, term string, categories []newznab.ID) *port.Request {
	var parameters stringutil.NameValueCollection
	parameters.Add("category", "0")
	parameters.Add("include_dead_torrents", "yes")
	parameters.Add("sort", "added")
	parameters.Add("order", "desc")

	term = strings.TrimSpace(termSeparators.ReplaceAllString(term, " "))

	if term != "" {
		parameters.Set("do", "search")
		parameters.Set("keywords", term)
		parameters.Set("search_type", "t_name")
	}

	if queryCats := ix.Categories.TrackerIDs(categories); len(queryCats) > 0 {
		parameters.Set("selectedcats2", strings.Join(queryCats, ","))
	}

	searchURL := "browse.php"

	if parameters.Count() > 0 {
		searchURL += "?" + parameters.GetQueryString()
	}

	return port.Get(ix.URL(searchURL)).Accept("text/html")
}

var dateAddedRegex = regexp.MustCompile(`\d{4}-\d{2}-\d{2} \d{2}:\d{2} [AaPp][Mm]`)

func parseResponse(ix *ix, indexerResponse *port.IndexerResponse) ([]port.TorrentInfo, error) {
	var releaseInfos []port.TorrentInfo

	doc := dom.Parse(indexerResponse.Content)
	rows := doc.QuerySelectorAll(`table#sortabletable > tbody > tr:has(a[href*="details.php?id="])`)
	for _, row := range rows {
		downloadVolumeFactor := 1.0
		if row.QuerySelector(`img[title^="Free Torrent"], img[title^="Sitewide Free Torrent"]`) != nil {
			downloadVolumeFactor = 0.0
		} else if row.QuerySelector(`img[title^="Silver Torrent"]`) != nil {
			downloadVolumeFactor = 0.5
		}

		// Skip non-freeleech results when freeleech only is set
		if ix.Settings.FreeleechOnly && downloadVolumeFactor != 0.0 {
			continue
		}

		// details link, release name gets shortened if it's to long
		qDetails := row.QuerySelector(`div > a[href*="details.php?id="]`)

		// use Title from tooltip or fallback to Details link if there's no tooltip
		qTitle := row.QuerySelector(".tooltip-content > div:nth-of-type(1)")
		if qTitle == nil {
			qTitle = qDetails
		}
		title := strings.TrimSpace(qTitle.TextContent())

		// SlipStream: no description, which a release here doesn't carry.
		infoURL := qDetails.GetAttribute("href")
		downloadURL := row.QuerySelector(`a[href*="download.php"]`).GetAttribute("href")
		seeders, err := parseutil.CoerceInt(row.QuerySelector("td:nth-of-type(7)").TextContent())
		if err != nil {
			return nil, err
		}
		leechers, err := parseutil.CoerceInt(strings.TrimSpace(row.QuerySelector("td:nth-of-type(8)").TextContent()))
		if err != nil {
			return nil, err
		}
		peers := seeders + leechers

		categoryLink := row.QuerySelector("td:nth-of-type(1) a").GetAttribute("href")
		cat, err := parseutil.GetArgumentFromQueryString(categoryLink, "category")
		if err != nil {
			return nil, err
		}

		size, err := parseutil.GetBytes(strings.TrimSpace(row.QuerySelector("td:nth-of-type(5)").TextContent()))
		if err != nil {
			return nil, err
		}
		grabs, err := parseutil.CoerceInt(row.QuerySelector("td:nth-child(6)").TextContent())
		if err != nil {
			return nil, err
		}

		uploadVolumeFactor := 1.0
		if row.QuerySelector(`img[title^="x2 Torrent"]`) != nil {
			uploadVolumeFactor = 2
		}

		release := port.TorrentInfo{
			GUID:                 infoURL,
			InfoURL:              infoURL,
			DownloadURL:          downloadURL,
			Title:                title,
			Categories:           ix.Categories.FromTracker(cat),
			Seeders:              seeders,
			Peers:                peers,
			Size:                 size,
			Grabs:                grabs,
			DownloadVolumeFactor: downloadVolumeFactor,
			UploadVolumeFactor:   uploadVolumeFactor,
			MinimumRatio:         1,
			MinimumSeedTime:      86400, // 24 hours
		}

		if dateAddedMatch := dateAddedRegex.FindString(strings.TrimSpace(row.QuerySelector("td:nth-of-type(2) > div:last-child").TextContent())); dateAddedMatch != "" {
			if release.PublishDate, err = datetimeutil.ParseExact(dateAddedMatch, "yyyy-MM-dd hh:mm tt", datetimeutil.None, ix.Clock); err != nil {
				return nil, err
			}
		}

		releaseInfos = append(releaseInfos, release)
	}

	return releaseInfos, nil
}

type ImmortalSeedSettings struct {
	port.UserPass
	FreeleechOnly bool `port:"freeleechOnly" label:"Freeleech only" help:"Show only the freeleech Releases of each search."`
}
