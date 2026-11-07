// Ported from Prowlarr's src/NzbDrone.Core/Indexers/Definitions/TorrentBytes.cs
// at ef9c182466ed04f34e6bcfedebb02c3c4af7eaf9. Copyright the Prowlarr
// contributors, GPL-3.0. Modified for SlipStream: rewritten in Go.

package torrentbytes

import (
	_ "embed"
	"strings"

	"github.com/jatassi/slipstream/internal/newznab"
	"github.com/jatassi/slipstream/internal/ports/port"
	"github.com/jatassi/slipstream/internal/prowlarr/datetimeutil"
	"github.com/jatassi/slipstream/internal/prowlarr/dom"
	"github.com/jatassi/slipstream/internal/prowlarr/parseutil"
)

//go:embed upstream.json
var upstream []byte

type settings struct{ port.UserPass }

type ix = port.Indexer[settings]

var Port = port.Define(port.Entry{
	ID:           "torrentbytes",
	Name:         "Torrent Bytes",
	Description:  "Torrent Bytes is a private Tracker for movies, TV, music, games and software.",
	Language:     "en-US",
	Privacy:      port.Private,
	IndexerURLs:  []string{"https://www.torrentbytes.net/"},
	Encoding:     "iso-8859-1",
	Capabilities: setCapabilities(),
	Upstream:     port.ReadUpstream(upstream),
}, port.Impl[settings]{
	Login: port.FormLogin[settings]{DoLogin: doLogin, CheckIfLoginNeeded: checkIfLoginNeeded},
	Movie: func(ix *ix, q port.SearchCriteria) (port.Chain, error) {
		return port.One(getPagedRequests(ix, q.SanitizedSearchTerm(), q.Categories, q.FullImdbID()))
	},
	TV: func(ix *ix, q port.SearchCriteria) (port.Chain, error) {
		return port.One(getPagedRequests(ix, q.SanitizedTvSearchString(), q.Categories, q.FullImdbID()))
	},
	Basic: func(ix *ix, q port.SearchCriteria) (port.Chain, error) {
		return port.One(getPagedRequests(ix, q.SanitizedSearchTerm(), q.Categories, ""))
	},
	Parse: parseResponse,
})

func doLogin(in *port.SignIn[settings]) error {
	loginURL := in.BaseURL.JoinPath("takelogin.php")
	response, err := in.Do(port.Post(loginURL).
		AddFormParameter("username", in.Settings.Username).
		AddFormParameter("password", string(in.Settings.Password)).
		AddFormParameter("returnto", "/").
		AddFormParameter("login", "Log in!").
		SetHeader("Referer", loginURL.String()))
	if err != nil {
		return err
	}

	if checkIfLoginNeeded(response) {
		doc := dom.Parse(response.Content)
		// SlipStream: no fallback message, since the stop sentence names
		// the sign-in fields and Prowlarr's fallback isn't the site's words.
		errorMessage := strings.TrimSpace(doc.QuerySelector("td.embedded").TextContent())

		return port.IndexerAuthError(errorMessage)
	}

	return nil
}

func checkIfLoginNeeded(httpResponse *port.IndexerResponse) bool {
	return !strings.Contains(httpResponse.Content, "my.php")
}

func setCapabilities() port.Capabilities {
	// SlipStream: no MusicSearchParams, since SlipStream has no music
	// search mode. Music releases still come back from Basic.
	return port.Capabilities{
		TvSearchParams:    []port.Param{port.Q, port.Season, port.Ep, port.ImdbID},
		MovieSearchParams: []port.Param{port.Q, port.ImdbID},
		Categories: port.Categories(
			port.Map(23, newznab.TVAnime, "Anime"),
			port.Map(52, newznab.PCMac, "Apple/All"),
			port.Map(22, newznab.PC, "Apps/misc"),
			port.Map(1, newznab.PC, "Apps/PC"),
			port.Map(28, newznab.TVForeign, "Foreign Titles"),
			port.Map(50, newznab.Console, "Games/Consoles"),
			port.Map(42, newznab.PCGames, "Games/Pack"),
			port.Map(4, newznab.PCGames, "Games/PC"),
			port.Map(51, newznab.PC, "Linux/All"),
			port.Map(31, newznab.OtherMisc, "Misc"),
			port.Map(20, newznab.MoviesDVD, "Movies/DVD-R"),
			port.Map(12, newznab.MoviesBluRay, "Movies/Full Blu-ray"),
			port.Map(5, newznab.MoviesHD, "Movies/HD"),
			port.Map(40, newznab.Movies, "Movies/Pack"),
			port.Map(19, newznab.MoviesSD, "Movies/SD"),
			port.Map(49, newznab.MoviesUHD, "Movies/UHD"),
			port.Map(25, newznab.Audio, "Music/DVDR"),
			port.Map(48, newznab.AudioLossless, "Music/Flac"),
			port.Map(6, newznab.AudioMP3, "Music/MP3"),
			port.Map(43, newznab.Audio, "Music/Pack"),
			port.Map(34, newznab.AudioVideo, "Music/Videos"),
			port.Map(45, newznab.MoviesBluRay, "NonScene/BRrip"),
			port.Map(46, newznab.MoviesHD, "NonScene/x264"),
			port.Map(44, newznab.MoviesSD, "NonScene/Xvid"),
			port.Map(37, newznab.TVHD, "TV/BRrip"),
			port.Map(38, newznab.TVHD, "TV/HD"),
			port.Map(41, newznab.TV, "TV/Pack"),
			port.Map(33, newznab.TVSD, "TV/SD"),
			port.Map(32, newznab.TVUHD, "TV/UHD"),
			port.Map(39, newznab.XXXx264, "XXX/HD"),
			port.Map(24, newznab.XXXImageSet, "XXX/IMGSET"),
			port.Map(21, newznab.XXXPack, "XXX/Pack"),
			port.Map(9, newznab.XXXXviD, "XXX/SD"),
			port.Map(29, newznab.XXX, "XXX/Web"),
		),
	}
}

func getPagedRequests(ix *ix, term string, categories []newznab.ID, imdbID string) *port.Request {
	// SlipStream: AddQueryParam writes a space as %20 where
	// GetQueryString writes '+'. The site reads both as a space.
	request := port.Get(ix.URL("browse.php")).Accept("text/html").
		AddQueryParam("incldead", "1")

	if imdbID != "" {
		request.AddQueryParam("search", imdbID)
		request.AddQueryParam("sc", "2") // search in description
	} else {
		request.AddQueryParam("search", term)
		request.AddQueryParam("sc", "1") // search in title
	}

	for _, cat := range ix.Categories.TrackerIDs(categories) {
		request.AddQueryParam("c"+cat, "1")
	}

	return request
}

func parseResponse(ix *ix, indexerResponse *port.IndexerResponse) ([]port.TorrentInfo, error) {
	var torrentInfos []port.TorrentInfo

	doc := dom.Parse(indexerResponse.Content)
	rows := doc.QuerySelectorAll("table > tbody:has(tr > td.colhead) > tr:not(:has(td.colhead))")
	for _, row := range rows {
		var release port.TorrentInfo
		link := row.QuerySelector("td:nth-of-type(2) a:nth-of-type(2)")
		release.GUID = ix.BaseURL.String() + link.GetAttribute("href")
		release.InfoURL = release.GUID
		release.Title = link.GetAttribute("title")

		// There isn't a title attribute if the release name isn't truncated.
		if strings.TrimSpace(release.Title) == "" {
			release.Title = strings.TrimSpace(link.FirstChild().TextContent())
		}

		// If we search an get no results, we still get a table just with no info.
		if strings.TrimSpace(release.Title) == "" {
			break
		}

		if qCat := row.QuerySelector("td:nth-of-type(1) a"); qCat != nil {
			cat := qCat.GetAttribute("href")[15:]
			release.Categories = ix.Categories.FromTracker(cat)
		}

		qLink := row.QuerySelector("td:nth-of-type(2) a")
		release.DownloadURL = ix.BaseURL.String() + qLink.GetAttribute("href")

		added := strings.TrimSpace(row.QuerySelector("td:nth-of-type(5)").TextContent())
		publishDate, err := datetimeutil.ParseExact(added, "yyyy-MM-ddHH:mm:ss", datetimeutil.None, ix.Clock)
		if err != nil {
			return nil, err
		}
		release.PublishDate = publishDate

		sizeStr := strings.TrimSpace(row.QuerySelector("td:nth-of-type(7)").TextContent())
		if release.Size, err = parseutil.GetBytes(sizeStr); err != nil {
			return nil, err
		}
		if release.Seeders, err = parseutil.CoerceInt(strings.TrimSpace(row.QuerySelector("td:nth-of-type(9)").TextContent())); err != nil {
			return nil, err
		}
		leechers, err := parseutil.CoerceInt(strings.TrimSpace(row.QuerySelector("td:nth-of-type(10)").TextContent()))
		if err != nil {
			return nil, err
		}
		release.Peers = leechers + release.Seeders

		files := row.QuerySelector("td:nth-child(3)").TextContent()
		if release.Files, err = parseutil.CoerceInt(files); err != nil {
			return nil, err
		}

		if grabs := row.QuerySelector("td:nth-child(8)").TextContent(); grabs != "----" {
			if release.Grabs, err = parseutil.CoerceInt(grabs); err != nil {
				return nil, err
			}
		}

		release.DownloadVolumeFactor = 1
		if row.QuerySelector(`font[color="green"]:contains("F"):contains("L")`) != nil {
			release.DownloadVolumeFactor = 0
		}
		release.UploadVolumeFactor = 1

		torrentInfos = append(torrentInfos, release)
	}

	return torrentInfos, nil
}
