// Ported from Prowlarr's src/NzbDrone.Core/Indexers/Definitions/RevolutionTT.cs
// at ef9c182466ed04f34e6bcfedebb02c3c4af7eaf9. Copyright the Prowlarr
// contributors, GPL-3.0. Modified for SlipStream: rewritten in Go.

package revolutiontt

import (
	_ "embed"
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

type settings struct{ port.UserPass }

type ix = port.Indexer[settings]

var Port = port.Define(port.Entry{
	ID:           "revolutiontt",
	Name:         "RevolutionTT",
	Description:  "RevolutionTT is a private Tracker for movies, TV, music, games and software.",
	Language:     "en-US",
	Privacy:      port.Private,
	IndexerURLs:  []string{"https://revott.me/"},
	LegacyURLs:   []string{"https://revolutiontt.me/"},
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
	if _, err := in.Do(port.Get(in.BaseURL.JoinPath("login.php"))); err != nil {
		return err
	}

	response, err := in.Do(port.Post(in.BaseURL.JoinPath("takelogin.php")).
		AddFormParameter("username", in.Settings.Username).
		AddFormParameter("password", string(in.Settings.Password)))
	if err != nil {
		return err
	}

	if !strings.Contains(response.Content, "/logout.php") {
		// SlipStream: no message, since the stop sentence names the
		// sign-in fields and Prowlarr's isn't the site's words.
		return port.IndexerAuthError("")
	}

	return nil
}

func checkIfLoginNeeded(httpResponse *port.IndexerResponse) bool {
	return httpResponse.HasHTTPRedirect || !strings.Contains(httpResponse.Content, "/logout.php")
}

func setCapabilities() port.Capabilities {
	// SlipStream: no MusicSearchParams or BookSearchParams, since
	// SlipStream has neither search mode. Those releases still come back
	// from Basic.
	return port.Capabilities{
		TvSearchParams:    []port.Param{port.Q, port.Season, port.Ep},
		MovieSearchParams: []port.Param{port.Q, port.ImdbID},
		Categories: port.Categories(
			port.Map("23", newznab.TVAnime, ""),
			port.Map("22", newznab.PC0day, ""),
			port.Map("1", newznab.PCISO, ""),
			port.Map("36", newznab.Books, ""),
			port.Map("36", newznab.BooksEBook, ""),
			port.Map("4", newznab.PCGames, ""),
			port.Map("21", newznab.PCGames, ""),
			port.Map("16", newznab.ConsolePS3, ""),
			port.Map("40", newznab.ConsoleWii, ""),
			port.Map("39", newznab.ConsoleXBox360, ""),
			port.Map("35", newznab.ConsoleNDS, ""),
			port.Map("34", newznab.ConsolePSP, ""),
			port.Map("2", newznab.PCMac, ""),
			port.Map("10", newznab.MoviesBluRay, ""),
			port.Map("20", newznab.MoviesDVD, ""),
			port.Map("12", newznab.MoviesHD, ""),
			port.Map("44", newznab.MoviesOther, ""),
			port.Map("11", newznab.MoviesSD, ""),
			port.Map("19", newznab.MoviesSD, ""),
			port.Map("6", newznab.Audio, ""),
			port.Map("8", newznab.AudioLossless, ""),
			port.Map("46", newznab.AudioOther, ""),
			port.Map("29", newznab.AudioVideo, ""),
			port.Map("43", newznab.TVOther, ""),
			port.Map("42", newznab.TVHD, ""),
			port.Map("45", newznab.TVOther, ""),
			port.Map("41", newznab.TVSD, ""),
			port.Map("7", newznab.TVSD, ""),
			port.Map("9", newznab.XXX, ""),
			port.Map("49", newznab.XXX, ""),
			port.Map("47", newznab.XXXDVD, ""),
			port.Map("48", newznab.XXX, ""),
		),
	}
}

func getPagedRequests(ix *ix, term string, categories []newznab.ID, imdbID string) *port.Request {
	var qc stringutil.NameValueCollection
	qc.Add("incldead", "1")

	if strings.TrimSpace(imdbID) != "" {
		qc.Add("titleonly", "0")
		qc.Add("search", imdbID)
	} else {
		qc.Add("titleonly", "1")
		qc.Add("search", term)
	}

	for _, cat := range ix.Categories.TrackerIDs(categories) {
		qc.Add("c"+cat, "1")
	}

	return port.Get(ix.URL("browse.php?" + qc.GetQueryString())).Accept("text/html").NoRedirect()
}

func parseResponse(ix *ix, indexerResponse *port.IndexerResponse) ([]port.TorrentInfo, error) {
	var torrentInfos []port.TorrentInfo

	doc := dom.Parse(indexerResponse.Content)
	rows := doc.QuerySelectorAll("#torrents-table > tbody > tr")
	for _, row := range rows[min(1, len(rows)):] {
		qDetails := row.QuerySelector(".br_right > a")
		details := ix.BaseURL.String() + qDetails.GetAttribute("href")
		title := qDetails.QuerySelector("b").TextContent()

		// Remove auto-generated [REQ] tag from fulfilled requests
		title = strings.TrimPrefix(title, "[REQ] ")

		qLink := row.QuerySelector("td:nth-child(4) > a")
		if qLink == nil {
			continue // support/donation banner
		}

		link := ix.BaseURL.String() + qLink.GetAttribute("href")

		// dateString format "yyyy-MMM-dd hh:mm:ss" => eg "2015-04-25 23:38:12"
		dateString := strings.TrimSpace(row.QuerySelector("td:nth-child(6) nobr").TextContent())
		publishDate, err := datetimeutil.ParseExact(dateString, "yyyy-MM-ddHH:mm:ss", datetimeutil.None, ix.Clock)
		if err != nil {
			return nil, err
		}

		sizeHTML, _, _ := strings.Cut(row.QuerySelector("td:nth-child(7)").InnerHTML(), "<")
		size, err := parseutil.GetBytes(strings.TrimSpace(sizeHTML))
		if err != nil {
			return nil, err
		}

		// SlipStream: a cell with no number reads as 0 where C#'s cast of
		// a null long throws.
		files, _, err := parseutil.GetLongFromString(row.QuerySelector("td:nth-child(7) > a").TextContent())
		if err != nil {
			return nil, err
		}
		grabs, _, err := parseutil.GetLongFromString(row.QuerySelector("td:nth-child(8)").TextContent())
		if err != nil {
			return nil, err
		}
		seeders, err := parseutil.CoerceInt(row.QuerySelector("td:nth-child(9)").TextContent())
		if err != nil {
			return nil, err
		}
		leechers, err := parseutil.CoerceInt(row.QuerySelector("td:nth-child(10)").TextContent())
		if err != nil {
			return nil, err
		}

		category := strings.ReplaceAll(row.QuerySelector(".br_type > a").GetAttribute("href"), "browse.php?cat=", "")

		var imdb int
		if qImdb := row.QuerySelector(`a[href*="www.imdb.com/"]`); qImdb != nil {
			href := qImdb.GetAttribute("href")
			imdb, _ = parseutil.GetImdbID(href[strings.LastIndex(href, "/")+1:])
		}

		release := port.TorrentInfo{
			InfoURL:              details,
			GUID:                 details,
			Title:                title,
			DownloadURL:          link,
			PublishDate:          publishDate,
			Size:                 size,
			Seeders:              seeders,
			Peers:                seeders + leechers,
			Grabs:                port.Int(grabs),
			Files:                port.Int(files),
			Categories:           ix.Categories.FromTracker(category),
			ImdbID:               imdb,
			MinimumRatio:         1,
			MinimumSeedTime:      172800, // 48 hours
			UploadVolumeFactor:   1,
			DownloadVolumeFactor: 1,
		}

		torrentInfos = append(torrentInfos, release)
	}

	return torrentInfos, nil
}
