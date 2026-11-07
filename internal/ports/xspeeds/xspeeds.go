// Ported from Prowlarr's src/NzbDrone.Core/Indexers/Definitions/XSpeeds.cs
// at ef9c182466ed04f34e6bcfedebb02c3c4af7eaf9. Copyright the Prowlarr
// contributors, GPL-3.0. Modified for SlipStream: rewritten in Go.

package xspeeds

import (
	_ "embed"
	"regexp"
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

type ix = port.Indexer[XSpeedsSettings]

var Port = port.Define(port.Entry{
	ID:           "xspeeds",
	Name:         "XSpeeds",
	Description:  "XSpeeds is a private Tracker for movies, TV and general releases.",
	Language:     "en-US",
	Privacy:      port.Private,
	IndexerURLs:  []string{"https://www.xspeeds.eu/"},
	Capabilities: setCapabilities(),
	Upstream:     port.ReadUpstream(upstream),
}, port.Impl[XSpeedsSettings]{
	Login: port.FormLogin[XSpeedsSettings]{DoLogin: doLogin, CheckIfLoginNeeded: checkIfLoginNeeded},
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

func doLogin(in *port.SignIn[XSpeedsSettings]) error {
	landingURL := in.BaseURL.JoinPath("login.php")
	if _, err := in.Do(port.Get(landingURL)); err != nil {
		return err
	}

	response, err := in.Do(port.Post(in.BaseURL.JoinPath("takelogin.php")).
		AddFormParameter("username", in.Settings.Username).
		AddFormParameter("password", string(in.Settings.Password)).
		SetHeader("Referer", landingURL.String()))
	if err != nil {
		return err
	}

	if checkIfLoginNeeded(response) {
		doc := dom.Parse(response.Content)
		errorMessage := strings.ReplaceAll(strings.TrimSpace(doc.QuerySelector(".left_side table:nth-of-type(1) tr:nth-of-type(2)").TextContent()), "\n\t", " ")
		if strings.TrimSpace(errorMessage) == "" {
			errorMessage = strings.ReplaceAll(strings.TrimSpace(doc.QuerySelector("div.notification-body").TextContent()), "\n\t", " ")
		}

		// SlipStream: no fallback message, since the stop sentence names
		// the sign-in fields and Prowlarr's fallback isn't the site's words.
		return port.IndexerAuthError(errorMessage)
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
			port.Map(70, newznab.TVAnime, "Anime"),
			port.Map(113, newznab.TVAnime, "Anime Boxsets"),
			port.Map(112, newznab.MoviesOther, "Anime Movies"),
			port.Map(111, newznab.MoviesOther, "Anime TV"),
			port.Map(150, newznab.PC, "Apps"),
			port.Map(153, newznab.Books, "Books"),
			port.Map(154, newznab.AudioAudiobook, "Books Audiobooks"),
			port.Map(155, newznab.Books, "Books eBooks & Magazines"),
			port.Map(68, newznab.MoviesOther, "Cams/TS"),
			port.Map(140, newznab.TVDocumentary, "Documentary"),
			port.Map(10, newznab.MoviesDVD, "DVDR"),
			port.Map(109, newznab.MoviesBluRay, "DVDR Bluray Disc"),
			port.Map(131, newznab.TVSport, "Fighting"),
			port.Map(134, newznab.TVSport, "Fighting Boxing"),
			port.Map(133, newznab.TVSport, "Fighting MMA"),
			port.Map(132, newznab.TVSport, "Fighting Wrestling"),
			port.Map(72, newznab.MoviesForeign, "Foreign"),
			port.Map(116, newznab.TVForeign, "Foreign Boxsets"),
			port.Map(114, newznab.MoviesForeign, "Foreign Movies"),
			port.Map(115, newznab.TVForeign, "Foreign TV"),
			port.Map(103, newznab.ConsoleOther, "Games Console"),
			port.Map(105, newznab.ConsoleOther, "Games Console Nintendo"),
			port.Map(104, newznab.ConsolePS4, "Games Console Playstation"),
			port.Map(106, newznab.ConsoleXBox, "Games Console XBOX"),
			port.Map(6, newznab.PCGames, "Games PC"),
			port.Map(108, newznab.PC, "Games PC Linux"),
			port.Map(107, newznab.PCMac, "Games PC Mac"),
			port.Map(11, newznab.Movies, "Movie Boxsets"),
			port.Map(118, newznab.MoviesUHD, "Movie Boxsets 4K"),
			port.Map(162, newznab.MoviesHD, "Movie Boxsets AV1"),
			port.Map(143, newznab.MoviesHD, "Movie Boxsets HD"),
			port.Map(119, newznab.MoviesHD, "Movie Boxsets HEVC"),
			port.Map(144, newznab.MoviesSD, "Movie Boxsets SD"),
			port.Map(12, newznab.Movies, "Movies"),
			port.Map(117, newznab.MoviesUHD, "Movies 4K"),
			port.Map(163, newznab.MoviesHD, "Movies AV1"),
			port.Map(145, newznab.MoviesHD, "Movies HD"),
			port.Map(100, newznab.MoviesHD, "Movies HEVC"),
			port.Map(146, newznab.MoviesSD, "Movies SD"),
			port.Map(13, newznab.Audio, "Music"),
			port.Map(135, newznab.AudioLossless, "Music FLAC"),
			port.Map(151, newznab.Audio, "Music Karaoke"),
			port.Map(136, newznab.Audio, "Music Boxset"),
			port.Map(148, newznab.AudioVideo, "Music Videos"),
			port.Map(9, newznab.Other, "Other"),
			port.Map(125, newznab.Other, "Other Pictures"),
			port.Map(54, newznab.TVOther, "Other Soaps"),
			port.Map(83, newznab.TVOther, "Other Specials"),
			port.Map(139, newznab.TV, "TOTM (Freeleech)"),
			port.Map(138, newznab.TV, "TOTW (x2 upload)"),
			port.Map(139, newznab.Movies, "TOTM (Freeleech)"),
			port.Map(138, newznab.Movies, "TOTW (x2 upload)"),
			port.Map(20, newznab.TVSport, "Sports"),
			port.Map(88, newznab.TVSport, "Sports/Football"),
			port.Map(86, newznab.TVSport, "Sports/MotorSports"),
			port.Map(89, newznab.TVSport, "Sports/Olympics"),
			port.Map(126, newznab.TV, "TV"),
			port.Map(127, newznab.TVUHD, "TV 4K"),
			port.Map(164, newznab.TVHD, "TV AV1"),
			port.Map(129, newznab.TVHD, "TV HD"),
			port.Map(130, newznab.TVHD, "TV HEVC"),
			port.Map(128, newznab.TVSD, "TV SD"),
			port.Map(149, newznab.TV, "TV Specials"),
			port.Map(21, newznab.TVSD, "TV Boxsets"),
			port.Map(120, newznab.TVUHD, "TV Boxset 4K"),
			port.Map(165, newznab.TVUHD, "TV Boxset AV1"),
			port.Map(76, newznab.TVHD, "TV Boxset HD"),
			port.Map(97, newznab.TVHD, "TV Boxset HEVC"),
			port.Map(147, newznab.TVSD, "TV Boxset SD"),
		),
	}
}

// SlipStream: C#'s [ -._] written with its range spelled out, space to
// dot, as .NET and Go both read it.
var termSeparators = regexp.MustCompile(`[\x20-\x2e_]+`)

func getPagedRequests(ix *ix, term string, categories []newznab.ID) *port.Request {
	category := "0"
	if categoryMapping := ix.Categories.TrackerIDs(categories); len(categoryMapping) == 1 {
		category = categoryMapping[0]
	}

	var parameters stringutil.NameValueCollection
	parameters.Add("category", category) // multi category search not supported
	parameters.Add("include_dead_torrents", "yes")
	parameters.Add("sort", "added")
	parameters.Add("order", "desc")

	term = strings.TrimSpace(termSeparators.ReplaceAllString(term, " "))

	if term != "" {
		parameters.Set("do", "search")
		parameters.Set("keywords", term)
		parameters.Set("search_type", "t_name")
	}

	searchURL := "browse.php"

	if parameters.Count() > 0 {
		searchURL += "?" + parameters.GetQueryString()
	}

	return port.Get(ix.URL(searchURL)).Accept("text/html")
}

var dateAddedRegex = regexp.MustCompile(`\d{2}-\d{2}-\d{4} \d{2}:\d{2}`)

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

		qDetails := row.QuerySelector(`div > a[href*="details.php?id="]`)
		title := strings.TrimSpace(qDetails.TextContent())
		infoURL := qDetails.GetAttribute("href")
		downloadURL := row.QuerySelector(`a[href*="download.php"]`).GetAttribute("href")
		// SlipStream: no description, which a release here doesn't carry.
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
			MinimumRatio:         0.8,
		}

		if dateAddedMatch := dateAddedRegex.FindString(strings.TrimSpace(row.QuerySelector("td:nth-of-type(2) > div:last-child").TextContent())); dateAddedMatch != "" {
			if release.PublishDate, err = datetimeutil.ParseExact(dateAddedMatch, "dd-MM-yyyy HH:mm", datetimeutil.None, ix.Clock); err != nil {
				return nil, err
			}
		}

		releaseInfos = append(releaseInfos, release)
	}

	return releaseInfos, nil
}

type XSpeedsSettings struct {
	port.UserPass
	FreeleechOnly bool `port:"freeleechOnly" label:"Freeleech only" help:"Show only the freeleech Releases of each search."`
}
