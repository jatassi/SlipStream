// Ported from Prowlarr's src/NzbDrone.Core/Indexers/Definitions/TorrentDay.cs
// at ef9c182466ed04f34e6bcfedebb02c3c4af7eaf9. Copyright the Prowlarr
// contributors, GPL-3.0. Modified for SlipStream: rewritten in Go.

package torrentday

import (
	_ "embed"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/jatassi/slipstream/internal/newznab"
	"github.com/jatassi/slipstream/internal/ports/port"
	"github.com/jatassi/slipstream/internal/prowlarr/datetimeutil"
	"github.com/jatassi/slipstream/internal/prowlarr/encoding"
	"github.com/jatassi/slipstream/internal/prowlarr/parseutil"
)

//go:embed upstream.json
var upstream []byte

type ix = port.Indexer[torrentDaySettings]

var Port = port.Define(port.Entry{
	ID:   "torrentday",
	Name: "TorrentDay",
	IndexerURLs: []string{
		"https://tday.love/",
		"https://torrentday.cool/",
		"https://secure.torrentday.com/",
		"https://classic.torrentday.com/",
		"https://www.torrentday.com/",
		"https://www.torrentday.me/",
		"https://torrentday.it/",
		"https://td.findnemo.net/",
		"https://td.getcrazy.me/",
		"https://td.venom.global/",
		"https://td.workisboring.net/",
		"https://tday.findnemo.net/",
		"https://tday.getcrazy.me/",
		"https://tday.venom.global/",
		"https://tday.workisboring.net/",
	},
	Description:  "TorrentDay is a private general Tracker with TV, movies, games, music, apps and books.",
	Privacy:      port.Private,
	Capabilities: setCapabilities(),
	Upstream:     port.ReadUpstream(upstream),
}, port.Impl[torrentDaySettings]{
	Login: port.CookieLogin{CheckIfLoginNeeded: checkIfLoginNeeded},
	Movie: movieSearch, TV: tvSearch, Basic: basicSearch,
	Parse: parseResponse,
})

// SlipStream: a search follows redirects, so the parser never sees the
// redirect to the sign-in page it reads below. The followed redirect
// landing on login.php is the expired cookie, cookie_expired, as is
// Prowlarr's default 401.
func checkIfLoginNeeded(r *port.IndexerResponse) bool {
	return r.StatusCode == http.StatusUnauthorized || strings.Contains(strings.ToLower(r.URL.Path), "/login.php")
}

// SlipStream: SlipStream has no music or book search, so the Music and
// Book search params aren't declared.
func setCapabilities() port.Capabilities {
	return port.Capabilities{
		TvSearchParams:    []port.Param{port.Q, port.Season, port.Ep, port.ImdbID},
		MovieSearchParams: []port.Param{port.Q, port.ImdbID},
		Categories: port.Categories(
			port.Map(25, newznab.MoviesSD, "Movies/480p"),
			port.Map(96, newznab.MoviesUHD, "Movie/4K"),
			port.Map(11, newznab.MoviesBluRay, "Movies/Bluray"),
			port.Map(5, newznab.MoviesBluRay, "Movies/Bluray-Full"),
			port.Map(103, newznab.MoviesSD, "Movies/Cam"),
			port.Map(3, newznab.MoviesDVD, "Movies/DVD-R"),
			port.Map(21, newznab.MoviesSD, "Movies/MP4"),
			port.Map(22, newznab.MoviesForeign, "Movies/Non-English"),
			port.Map(13, newznab.Movies, "Movies/Packs"),
			port.Map(44, newznab.MoviesSD, "Movies/SD/x264"),
			port.Map(48, newznab.Movies, "Movies/x265"),
			port.Map(1, newznab.MoviesSD, "Movies/XviD"),

			port.Map(24, newznab.TVSD, "TV/480p"),
			port.Map(104, newznab.TVUHD, "TV/4K"),
			port.Map(32, newznab.TVHD, "TV/Bluray"),
			port.Map(31, newznab.TVSD, "TV/DVD-R"),
			port.Map(33, newznab.TVSD, "TV/DVD-Rip"),
			port.Map(46, newznab.TVSD, "TV/Mobile"),
			port.Map(82, newznab.TVForeign, "TV/Non-English"),
			port.Map(14, newznab.TV, "TV/Packs"),
			port.Map(26, newznab.TVSD, "TV/SD/x264"),
			port.Map(7, newznab.TVHD, "TV/x264"),
			port.Map(34, newznab.TVx265, "TV/x265"),
			port.Map(2, newznab.TVSD, "TV/XviD"),

			port.Map(10, newznab.ConsoleNDS, "Nintendo"),
			port.Map(4, newznab.PCGames, "PC/Games"),
			port.Map(18, newznab.ConsolePS3, "PS"),
			port.Map(8, newznab.ConsolePSP, "PSP"),
			port.Map(9, newznab.ConsoleXBox, "Xbox"),

			port.Map(17, newznab.AudioMP3, "Music/Audio"),
			port.Map(27, newznab.Audio, "Music/Flac"),
			port.Map(23, newznab.AudioForeign, "Music/Non-English"),
			port.Map(41, newznab.Audio, "Music/Packs"),
			port.Map(16, newznab.AudioVideo, "Music/Video"),

			port.Map(29, newznab.TVAnime, "Anime"),
			port.Map(42, newznab.AudioAudiobook, "Audio Books"),
			port.Map(20, newznab.Books, "Books"),
			port.Map(102, newznab.BooksForeign, "Books/Non-English"),
			port.Map(30, newznab.TVDocumentary, "Documentary"),
			port.Map(95, newznab.TVDocumentary, "Educational"),
			port.Map(47, newznab.Other, "Fonts"),
			port.Map(43, newznab.PCMac, "Mac"),
			port.Map(45, newznab.AudioOther, "Podcast"),
			port.Map(28, newznab.PC, "Softwa/Packs"),
			port.Map(12, newznab.PC, "Software"),

			port.Map(19, newznab.XXX, "XXX/0Day"),
			port.Map(6, newznab.XXX, "XXX/Movies"),
			port.Map(15, newznab.XXXPack, "XXX/Packs"),
		),
	}
}

// SlipStream: the framework sends the pasted cookie on every request, so
// the generator doesn't set it.
func getPagedRequests(c *ix, term string, categories []newznab.ID, imdbID string) *port.Request {
	searchURL := c.URL("t.json")

	cats := c.Categories.TrackerIDs(categories)
	if len(cats) == 0 {
		cats = c.Categories.TrackerCategories()
	}

	query := strings.Join(cats, ";")

	if c.Settings.FreeLeechOnly {
		query += ";free"
	}

	query += ";q="

	if strings.TrimSpace(imdbID) != "" {
		query += imdbID + encoding.UTF8.URLEncode(" ")
	}

	query += encoding.UTF8.URLEncode(term)

	searchURL.RawQuery = query

	return port.Get(searchURL).Accept("application/rss+xml, text/rss+xml, application/xml, text/xml")
}

func movieSearch(c *ix, q port.SearchCriteria) (port.Chain, error) {
	return port.One(getPagedRequests(c, q.SearchTerm, q.Categories, q.FullImdbID()))
}

func tvSearch(c *ix, q port.SearchCriteria) (port.Chain, error) {
	return port.One(getPagedRequests(c, q.SanitizedTvSearchString(), q.Categories, q.FullImdbID()))
}

func basicSearch(c *ix, q port.SearchCriteria) (port.Chain, error) {
	return port.One(getPagedRequests(c, q.SearchTerm, q.Categories, ""))
}

// SlipStream: the messages are SlipStream's own, per WRITING.md.
func parseResponse(c *ix, r *port.IndexerResponse) ([]port.TorrentInfo, error) {
	if r.HasHTTPRedirect {
		if strings.Contains(strings.ToLower(r.RedirectURL), "/login.php") {
			return nil, port.IndexerError("the search was redirected to TorrentDay's sign-in page")
		}

		return nil, port.IndexerError("the search was redirected to %s", r.RedirectURL)
	}

	var torrentInfos []port.TorrentInfo

	var rows []torrentDayRow
	if err := json.Unmarshal(r.Body, &rows); err != nil {
		return nil, port.IndexerError("read the Indexer's search response: %v", err)
	}

	for _, row := range rows {
		title := row.Name

		torrentID := strconv.FormatInt(row.T, 10)
		details := c.URL("details.php?id=" + torrentID)
		seeders := row.Seeders
		imdbID := row.ImdbID
		downloadMultiplier := 1.0
		if row.DownloadMultiplier != nil {
			downloadMultiplier = *row.DownloadMultiplier
		}
		link := c.URL("download.php/" + torrentID + "/" + torrentID + ".torrent")
		publishDate := datetimeutil.UnixTimestampToDateTime(row.Ctime, c.Clock)
		imdb, _ := parseutil.GetImdbID(imdbID)

		torrentInfos = append(torrentInfos, port.TorrentInfo{
			Title:                title,
			GUID:                 details.String(),
			DownloadURL:          link.String(),
			InfoURL:              details.String(),
			PublishDate:          publishDate,
			Categories:           c.Categories.FromTracker(row.C.String()),
			Size:                 row.Size,
			Files:                row.Files,
			Grabs:                row.Completed,
			Seeders:              seeders,
			Peers:                seeders + row.Leechers,
			ImdbID:               imdb,
			DownloadVolumeFactor: downloadMultiplier,
			UploadVolumeFactor:   1,
			MinimumRatio:         1,
			MinimumSeedTime:      259200, // 72 hours
		})
	}

	return torrentInfos, nil
}

// torrentDayRow is the fields the C# reads from each dynamic row.
type torrentDayRow struct {
	Name               string      `json:"name"`
	T                  int64       `json:"t"`
	Seeders            int         `json:"seeders"`
	ImdbID             string      `json:"imdb-id"`
	DownloadMultiplier *float64    `json:"download-multiplier"`
	Ctime              int64       `json:"ctime"`
	C                  json.Number `json:"c"`
	Size               int64       `json:"size"`
	Files              int         `json:"files"`
	Completed          int         `json:"completed"`
	Leechers           int         `json:"leechers"`
}

// SlipStream: the cookie is the pasted session that CookieLogin asks
// for, so it isn't a setting here.
type torrentDaySettings struct {
	FreeLeechOnly bool `port:"freeLeechOnly" label:"Freeleech only" help:"Search only freeleech Releases."`
}
