// Ported from Prowlarr's src/NzbDrone.Core/Indexers/Definitions/HDTorrents.cs
// at ef9c182466ed04f34e6bcfedebb02c3c4af7eaf9. Copyright the Prowlarr
// contributors, GPL-3.0. Modified for SlipStream: rewritten in Go.

package hdtorrents

import (
	_ "embed"
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

type settings struct{ port.UserPass }

type ix = port.Indexer[settings]

var Port = port.Define(port.Entry{
	ID:          "hdtorrents",
	Name:        "HD-Torrents",
	Description: "HD-Torrents is a private Tracker for HD movies, TV and music, with strict rules for uploads.",
	Language:    "en-US",
	Privacy:     port.Private,
	IndexerURLs: []string{
		"https://hdts.ru/",
		"https://hd-torrents.org/",
		"https://hd-torrents.net/",
		"https://hd-torrents.me/",
	},
	Capabilities: setCapabilities(),
	Upstream:     port.ReadUpstream(upstream),
}, port.Impl[settings]{
	Login: port.FormLogin[settings]{DoLogin: doLogin, CheckIfLoginNeeded: checkIfLoginNeeded},
	Movie: func(ix *ix, q port.SearchCriteria) (port.Chain, error) {
		return port.One(getPagedRequests(ix, q.SearchTerm, q.Categories, q.FullImdbID()))
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
	loginURL := in.BaseURL.JoinPath("login.php")
	response, err := in.Do(port.Post(loginURL).
		AddFormParameter("uid", in.Settings.Username).
		AddFormParameter("pwd", string(in.Settings.Password)).
		SetHeader("Referer", loginURL.String()))
	if err != nil {
		return err
	}

	if response.Content != "" && !strings.Contains(strings.ToLower(response.Content), "if your browser doesn't have javascript enabled") {
		doc := dom.Parse(response.Content)
		// SlipStream: no fallback message, since the stop sentence names
		// the sign-in fields and Prowlarr's fallback isn't the site's words.
		errorMessage := strings.TrimSpace(doc.QuerySelector(`div > font[color="#FF0000"]`).TextContent())

		return port.IndexerAuthError(errorMessage)
	}

	return nil
}

func checkIfLoginNeeded(httpResponse *port.IndexerResponse) bool {
	return strings.Contains(httpResponse.Content, "Error:You're not authorized")
}

func setCapabilities() port.Capabilities {
	// SlipStream: no MusicSearchParams, since SlipStream has no music
	// search mode. Music releases still come back from Basic.
	return port.Capabilities{
		TvSearchParams:    []port.Param{port.Q, port.Season, port.Ep, port.ImdbID},
		MovieSearchParams: []port.Param{port.Q, port.ImdbID},
		Categories: port.Categories(
			port.Map("70", newznab.MoviesBluRay, "Movie/UHD/Blu-Ray"),
			port.Map("1", newznab.MoviesBluRay, "Movie/Blu-Ray"),
			port.Map("71", newznab.MoviesUHD, "Movie/UHD/Remux"),
			port.Map("2", newznab.MoviesHD, "Movie/Remux"),
			port.Map("5", newznab.MoviesHD, "Movie/1080p/i"),
			port.Map("3", newznab.MoviesHD, "Movie/720p"),
			port.Map("64", newznab.MoviesUHD, "Movie/2160p"),
			port.Map("63", newznab.Audio, "Movie/Audio Track"),

			// TV Show
			port.Map("72", newznab.TVUHD, "TV Show/UHD/Blu-ray"),
			port.Map("59", newznab.TVHD, "TV Show/Blu-ray"),
			port.Map("73", newznab.TVUHD, "TV Show/UHD/Remux"),
			port.Map("60", newznab.TVHD, "TV Show/Remux"),
			port.Map("30", newznab.TVHD, "TV Show/1080p/i"),
			port.Map("38", newznab.TVHD, "TV Show/720p"),
			port.Map("65", newznab.TVUHD, "TV Show/2160p"),

			// Music
			port.Map("44", newznab.Audio, "Music/Album"),
			port.Map("61", newznab.AudioVideo, "Music/Blu-Ray"),
			port.Map("62", newznab.AudioVideo, "Music/Remux"),
			port.Map("57", newznab.AudioVideo, "Music/1080p/i"),
			port.Map("45", newznab.AudioVideo, "Music/720p"),
			port.Map("66", newznab.AudioVideo, "Music/2160p"),

			// XXX
			port.Map("58", newznab.XXX, "XXX/Blu-ray"),
			port.Map("74", newznab.XXX, "XXX/UHD/Blu-ray"),
			port.Map("48", newznab.XXX, "XXX/1080p/i"),
			port.Map("47", newznab.XXX, "XXX/720p"),
			port.Map("67", newznab.XXX, "XXX/2160p"),
		),
	}
}

func getPagedRequests(ix *ix, term string, categories []newznab.ID, imdbID string) *port.Request {
	var categoryParams []string
	for _, cat := range ix.Categories.TrackerIDs(categories) {
		categoryParams = append(categoryParams, "category[]="+cat+"&")
	}
	searchURL := "torrents.php?" + strings.Join(categoryParams, "")

	var search []string
	for _, s := range []string{imdbID, term} {
		if s != "" {
			search = append(search, s)
		}
	}

	var queryCollection stringutil.NameValueCollection
	queryCollection.Add("search", strings.Join(search, " "))
	queryCollection.Add("active", "0")
	queryCollection.Add("options", "0")

	// manually url encode parenthesis to prevent "hacking" detection
	// SlipStream: a dot becomes %20 where C# writes a space for Uri to
	// escape, since a URL here carries no raw space.
	searchURL += strings.NewReplacer("(", "%28", ")", "%29", ".", "%20").Replace(queryCollection.GetQueryString())

	return port.Get(ix.URL(searchURL)).Accept("text/html")
}

var freeleechRanks = []string{
	"VIP",
	"Uploader",
	"HD Internal",
	"Moderator",
	"Administrator",
	"Owner",
}

func parseResponse(ix *ix, indexerResponse *port.IndexerResponse) ([]port.TorrentInfo, error) {
	var releaseInfos []port.TorrentInfo

	doc := dom.Parse(indexerResponse.Content)

	userInfo := doc.QuerySelector("table.navus tr")
	var userRank string
	if userInfo != nil {
		userRank = strings.TrimSpace(strings.ReplaceAll(userInfo.Children()[1].TextContent(), "Rank:", ""))
	}
	hasFreeleech := slices.ContainsFunc(freeleechRanks, func(rank string) bool { return strings.EqualFold(rank, userRank) })

	rows := doc.QuerySelectorAll("table.mainblockcontenttt tr:has(td.mainblockcontent)")
	for _, row := range rows[min(1, len(rows)):] {
		children := row.Children()
		mainLink := children[2].QuerySelector("a")
		title := mainLink.TextContent()
		details := ix.BaseURL.String() + mainLink.GetAttribute("href")

		// SlipStream: no poster or description, which a release here
		// doesn't carry.
		link := ix.BaseURL.String() + children[4].FirstElementChild().GetAttribute("href")
		size, err := parseutil.GetBytes(children[7].TextContent())
		if err != nil {
			return nil, err
		}

		var dateParts []string
		for _, a := range children[6].FirstElementChild().Attributes() {
			dateParts = append(dateParts, a.Name)
		}
		dateAdded := strings.Join(dateParts[:min(4, len(dateParts))], " ")
		publishDate, err := datetimeutil.ParseExact(dateAdded, "dd MMM yyyy HH:mm:ss", datetimeutil.None, ix.Clock)
		if err != nil {
			return nil, err
		}

		categoryLink := row.FirstElementChild().FirstElementChild().GetAttribute("href")
		cat, err := parseutil.GetArgumentFromQueryString(categoryLink, "category")
		if err != nil {
			return nil, err
		}

		// Sometimes the uploader column is missing, so seeders, leechers, and grabs may be at a different index.
		// There's room for improvement, but this works for now.
		endIndex := len(children)

		if children[endIndex-1].TextContent() == "Edit" {
			// Maybe use row.Children.Index(Node) after searching for an element instead?
			endIndex--
		} else if children[endIndex-4].TextContent() == "Edit" {
			// moderators get additional delete, recommend and like links
			endIndex -= 4
		}

		var seeders, peers int
		if rSeeders, ok := parseutil.TryCoerceInt(children[endIndex-3].TextContent()); ok {
			seeders = rSeeders
			if rLeechers, ok := parseutil.TryCoerceInt(children[endIndex-2].TextContent()); ok {
				peers = rLeechers + rSeeders
			}
		}

		grabs, _ := parseutil.TryCoerceInt(children[endIndex-1].TextContent())

		dlVolumeFactor := 1.0
		upVolumeFactor := 1.0
		switch {
		case row.QuerySelector(`img[src$="no_ratio.png"]`) != nil:
			dlVolumeFactor = 0
			upVolumeFactor = 0
		case hasFreeleech || row.QuerySelector(`img[src$="free.png"]`) != nil:
			dlVolumeFactor = 0
		case row.QuerySelector(`img[src$="50.png"]`) != nil:
			dlVolumeFactor = 0.5
		case row.QuerySelector(`img[src$="25.png"]`) != nil:
			dlVolumeFactor = 0.75
		case row.QuerySelector(`img[src$="75.png"]`) != nil:
			dlVolumeFactor = 0.25
		}

		imdbHref := strings.TrimRight(row.QuerySelector(`a[href*="www.imdb.com/title/"]`).GetAttribute("href"), "/")
		imdbID, _ := parseutil.GetImdbID(imdbHref[strings.LastIndex(imdbHref, "/")+1:])

		var flags []port.SiteFlag

		if row.QuerySelector(`img[src$="internal.png"]`) != nil {
			flags = append(flags, port.Internal)
		}

		release := port.TorrentInfo{
			Title:                title,
			GUID:                 details,
			DownloadURL:          link,
			InfoURL:              details,
			PublishDate:          publishDate,
			Categories:           ix.Categories.FromTracker(cat),
			ImdbID:               imdbID,
			Size:                 size,
			Grabs:                grabs,
			Seeders:              seeders,
			Peers:                peers,
			IndexerFlags:         flags,
			DownloadVolumeFactor: dlVolumeFactor,
			UploadVolumeFactor:   upVolumeFactor,
			MinimumRatio:         1,
			MinimumSeedTime:      172800, // 48 hours
		}

		releaseInfos = append(releaseInfos, release)
	}

	return releaseInfos, nil
}
