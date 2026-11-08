// Ported from Prowlarr's src/NzbDrone.Core/Indexers/Definitions/PreToMe.cs
// at ef9c182466ed04f34e6bcfedebb02c3c4af7eaf9. Copyright the Prowlarr
// contributors, GPL-3.0. Modified for SlipStream: rewritten in Go.

package pretome

import (
	_ "embed"
	"regexp"
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

type ix = port.Indexer[PreToMeSettings]

var Port = port.Define(port.Entry{
	ID:           "pretome",
	Name:         "PreToMe",
	Description:  "PreToMe is a ratioless private Tracker for 0day and general Releases.",
	Language:     "en-US",
	Privacy:      port.Private,
	IndexerURLs:  []string{"https://pretome.info/"},
	Encoding:     "iso-8859-1",
	Capabilities: setCapabilities(),
	Upstream:     port.ReadUpstream(upstream),
}, port.Impl[PreToMeSettings]{
	Login: port.FormLogin[PreToMeSettings]{DoLogin: doLogin, CheckIfLoginNeeded: checkIfLoginNeeded},
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

func doLogin(in *port.SignIn[PreToMeSettings]) error {
	if _, err := in.Do(port.Get(in.BaseURL.JoinPath("login.php"))); err != nil {
		return err
	}

	response, err := in.Do(port.Post(in.BaseURL.JoinPath("takelogin.php")).
		AddFormParameter("username", in.Settings.Username).
		AddFormParameter("password", string(in.Settings.Password)).
		AddFormParameter("login_pin", in.Settings.PIN).
		AddFormParameter("returnto", "%2F").
		AddFormParameter("login", "Login"))
	if err != nil {
		return err
	}

	if response.Content == "" {
		// SlipStream: no message, since Prowlarr's isn't the site's words.
		return port.IndexerAuthError("")
	}

	if checkIfLoginNeeded(response) {
		doc := dom.Parse(response.Content)
		// SlipStream: no fallback message, since the stop sentence names
		// the sign-in fields and Prowlarr's fallback isn't the site's words.
		errorMessage := strings.TrimSpace(doc.QuerySelector(`table.body_table font[color~="red"]`).TextContent())

		return port.IndexerAuthError(errorMessage)
	}

	return nil
}

func checkIfLoginNeeded(httpResponse *port.IndexerResponse) bool {
	return httpResponse.HasHTTPRedirect || !strings.Contains(httpResponse.Content, "logout.php")
}

func setCapabilities() port.Capabilities {
	// SlipStream: no MusicSearchParams or BookSearchParams, since
	// SlipStream has neither search mode. Those releases still come back
	// from Basic.
	return port.Capabilities{
		TvSearchParams:    []port.Param{port.Q, port.Season, port.Ep, port.ImdbID},
		MovieSearchParams: []port.Param{port.Q, port.ImdbID},
		Categories: port.Categories(

			port.Map("cat[]=22", newznab.PC, "Applications"),
			port.Map("cat[]=22&tags=Windows", newznab.PC0day, "Applications/Windows"),
			port.Map("cat[]=22&tags=MAC", newznab.PCMac, "Applications/MAC"),
			port.Map("cat[]=22&tags=Linux", newznab.PC, "Applications/Linux"),

			port.Map("cat[]=27", newznab.BooksEBook, "Ebooks"),

			port.Map("cat[]=4", newznab.Console, "Games"),
			port.Map("cat[]=4&tags=PC", newznab.PCGames, "Games/PC"),
			port.Map("cat[]=4&tags=RIP", newznab.PCGames, "Games/RIP"),
			port.Map("cat[]=4&tags=ISO", newznab.PCGames, "Games/ISO"),
			port.Map("cat[]=4&tags=XBOX360", newznab.ConsoleXBox360, "Games/XBOX360"),
			port.Map("cat[]=4&tags=PS3", newznab.ConsolePS3, "Games/PS3"),
			port.Map("cat[]=4&tags=Wii", newznab.ConsoleWii, "Games/Wii"),
			port.Map("cat[]=4&tags=PSP", newznab.ConsolePSP, "Games/PSP"),
			port.Map("cat[]=4&tags=GAMES-NSW", newznab.ConsoleOther, "Games/NSW"),
			port.Map("cat[]=4&tags=NDS", newznab.ConsoleNDS, "Games/NDS"),
			port.Map("cat[]=4&tags=Xbox", newznab.ConsoleXBox, "Games/Xbox"),
			port.Map("cat[]=4&tags=NSW", newznab.ConsoleOther, "Games/NSW"),
			port.Map("cat[]=4&tags=PS2", newznab.ConsoleOther, "Games/PS2"),

			port.Map("cat[]=31", newznab.Other, "Miscellaneous"),
			port.Map("cat[]=31&tags=Ebook", newznab.BooksEBook, "Miscellaneous/Ebook"),
			port.Map("cat[]=31&tags=RARFiX", newznab.Other, "Miscellaneous/RARFiX"),

			port.Map("cat[]=19", newznab.Movies, "Movies"),
			port.Map("cat[]=19&tags=x264", newznab.MoviesHD, "Movies/x264"),
			port.Map("cat[]=19&tags=720p", newznab.MoviesHD, "Movies/720p"),
			port.Map("cat[]=19&tags=XviD", newznab.MoviesSD, "Movies/XviD"),
			port.Map("cat[]=19&tags=BluRay", newznab.MoviesHD, "Movies/BluRay"),
			port.Map("cat[]=19&tags=DVDRiP", newznab.MoviesSD, "Movies/DVDRiP"),
			port.Map("cat[]=19&tags=1080p", newznab.MoviesHD, "Movies/1080p"),
			port.Map("cat[]=19&tags=DVD", newznab.MoviesSD, "Movies/DVD"),
			port.Map("cat[]=19&tags=DVDR", newznab.MoviesSD, "Movies/DVDR"),
			port.Map("cat[]=19&tags=WMV", newznab.Movies, "Movies/WMV"),
			port.Map("cat[]=19&tags=CAM", newznab.Movies, "Movies/CAM"),
			port.Map("cat[]=19&tags=2160p", newznab.MoviesUHD, "Movies/2160p"),
			port.Map("cat[]=19&tags=DolbyVision", newznab.Movies, "Movies/DolbyVision"),
			port.Map("cat[]=19&tags=x265", newznab.MoviesHD, "Movies/x265"),

			port.Map("cat[]=6", newznab.Audio, "Music"),
			port.Map("cat[]=6&tags=MP3", newznab.AudioMP3, "Music/MP3"),
			port.Map("cat[]=6&tags=V2", newznab.AudioMP3, "Music/V2"),
			port.Map("cat[]=6&tags=FLAC", newznab.AudioLossless, "Music/FLAC"),
			port.Map("cat[]=6&tags=320kbps", newznab.AudioMP3, "Music/320kbps"),

			port.Map("cat[]=52", newznab.TVSport, "Sport"),
			port.Map("cat[]=52&tags=1080p", newznab.TVSport, "Sport/1080p"),
			port.Map("cat[]=52&tags=720p", newznab.TVSport, "Sport/720p"),
			port.Map("cat[]=52&tags=2160p", newznab.TVSport, "Sport/2160p"),

			port.Map("cat[]=7", newznab.TV, "TV"),
			port.Map("cat[]=7&tags=x264", newznab.TVHD, "TV/x264"),
			port.Map("cat[]=7&tags=720p", newznab.TVHD, "TV/720p"),
			port.Map("cat[]=7&tags=HDTV", newznab.TVHD, "TV/HDTV"),
			port.Map("cat[]=7&tags=XviD", newznab.TVSD, "TV/XviD"),
			port.Map("cat[]=7&tags=BluRay", newznab.TVHD, "TV/BluRay"),
			port.Map("cat[]=7&tags=DVDRiP", newznab.TVSD, "TV/DVDRiP"),
			port.Map("cat[]=7&tags=DVD", newznab.TVSD, "TV/DVD"),
			port.Map("cat[]=7&tags=Documentary", newznab.TVDocumentary, "TV/Documentary"),
			port.Map("cat[]=7&tags=PDTV", newznab.TVSD, "TV/PDTV"),
			port.Map("cat[]=7&tags=HD-DVD", newznab.TVSD, "TV/HD-DVD"),
			port.Map("cat[]=7&tags=1080p", newznab.TVHD, "TV/1080p"),
			port.Map("cat[]=7&tags=2160p", newznab.TVUHD, "TV/2160p"),
			port.Map("cat[]=7&tags=DolbyVision", newznab.TVUHD, "TV/DolbyVision"),
			port.Map("cat[]=7&tags=x265", newznab.TVHD, "TV/x265"),

			port.Map("cat[]=51", newznab.XXX, "XXX"),
			port.Map("cat[]=51&tags=XviD", newznab.XXXXviD, "XXX/XviD"),
			port.Map("cat[]=51&tags=DVDRiP", newznab.XXXDVD, "XXX/DVDRiP"),
			port.Map("cat[]=51&tags=1080p", newznab.XXXx264, "XXX/1080p"),
			port.Map("cat[]=51&tags=2160p", newznab.XXXUHD, "XXX/2160p"),
			port.Map("cat[]=51&tags=720p", newznab.XXXx264, "XXX/720p"),
			port.Map("cat[]=51&tags=VR180", newznab.XXXOther, "XXX/VR180"),
			port.Map("cat[]=51&tags=x264", newznab.XXXx264, "XXX/x264"),
			port.Map("cat[]=51&tags=x265", newznab.XXXx264, "XXX/x265"),
			port.Map("cat[]=51&tags=XXX-VR180", newznab.XXXOther, "XXX/VR180"),
		),
	}
}

func getPagedRequests(ix *ix, term string, categories []newznab.ID, imdbID string) *port.Request {
	// NameValueCollection don't support cat[]=19&cat[]=6
	parameters := stringutil.KeyValuePairs{
		{Key: "st", Value: "1"}, // search in title
	}

	if strings.TrimSpace(imdbID) != "" {
		parameters.Add("search", imdbID)
		parameters.Add("sd", "1") // search in description
	} else {
		parameters.Add("search", term)
	}

	// parse categories and tags
	var catGroups, tagGroups []string
	cats := ix.Categories.TrackerIDs(categories)
	for _, cat := range cats {
		// "cat[]=7&tags=x264"
		cSplit := strings.Split(cat, "&")
		if gSplit := strings.Split(cSplit[0], "="); len(gSplit) > 1 && !slices.Contains(catGroups, gSplit[1]) {
			catGroups = append(catGroups, gSplit[1]) // category = 7
		}

		if len(cSplit) > 1 {
			if tSplit := strings.Split(cSplit[1], "="); len(tSplit) > 1 && !slices.Contains(tagGroups, tSplit[1]) {
				tagGroups = append(tagGroups, tSplit[1]) // tag = x264
			}
		}
	}

	// add categories
	for _, cat := range catGroups {
		parameters.Add("cat[]", cat)
	}

	// do not include too many tags as it'll mess with their servers
	if len(tagGroups) < 7 {
		parameters.Add("tags", strings.Join(tagGroups, ","))
		// if tags are specified match any
		// if no tags are specified match all, with any we get random results
		tf := "all"
		if len(tagGroups) > 0 {
			tf = "any"
		}
		parameters.Add("tf", tf)
	}

	searchURL := "browse.php"

	if len(parameters) > 0 {
		searchURL += "?" + parameters.GetQueryString()
	}

	return port.Get(ix.URL(searchURL)).Accept("text/html").NoRedirect()
}

var lineBreak = regexp.MustCompile(`<br\s?/?>`)

func parseResponse(ix *ix, indexerResponse *port.IndexerResponse) ([]port.TorrentInfo, error) {
	var releaseInfos []port.TorrentInfo

	doc := dom.Parse(indexerResponse.Content)
	rows := doc.QuerySelectorAll("table > tbody > tr.browse")
	for _, row := range rows {
		qDetails := row.QuerySelector(`a[href^="details.php?id="]`)
		title := qDetails.GetAttribute("title")
		infoURL := ix.BaseURL.String() + qDetails.GetAttribute("href")
		downloadURL := ix.BaseURL.String() + row.QuerySelector(`a[href^="download.php"]`).GetAttribute("href")
		dateAdded := strings.TrimSpace(lineBreak.ReplaceAllString(row.QuerySelector("td:nth-of-type(6)").InnerHTML(), " "))
		seeders, err := parseutil.CoerceInt(row.QuerySelector("td:nth-of-type(10)").TextContent())
		if err != nil {
			return nil, err
		}
		leechers, err := parseutil.CoerceInt(row.QuerySelector("td:nth-of-type(11)").TextContent())
		if err != nil {
			return nil, err
		}
		catHref := row.QuerySelector(`td:nth-of-type(1) a[href^="browse.php"]`).GetAttribute("href")
		cat := catHref[strings.LastIndex(catHref, "?")+1:]

		publishDate, err := datetimeutil.FromTimeAgo(dateAdded, ix.Clock)
		if err != nil {
			return nil, err
		}
		size, err := parseutil.GetBytes(row.QuerySelector("td:nth-of-type(8)").TextContent())
		if err != nil {
			return nil, err
		}
		files, err := parseutil.CoerceInt(row.QuerySelector("td:nth-of-type(4)").TextContent())
		if err != nil {
			return nil, err
		}
		grabs, err := parseutil.CoerceInt(row.QuerySelector("td:nth-of-type(9)").TextContent())
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
			Files:                files,
			Grabs:                grabs,
			Seeders:              seeders,
			Peers:                leechers + seeders,
			DownloadVolumeFactor: 0, // ratioless
			UploadVolumeFactor:   1,
			MinimumRatio:         0.75,
			MinimumSeedTime:      216000, // 60 hours
		}

		releaseInfos = append(releaseInfos, release)
	}

	return releaseInfos, nil
}

type PreToMeSettings struct {
	port.UserPass
	port.PINField `help:"The PIN you type on the PreToMe sign-in page."`
}
