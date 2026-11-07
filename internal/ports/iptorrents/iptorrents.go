// Ported from Prowlarr's src/NzbDrone.Core/Indexers/Definitions/IPTorrents.cs
// at ef9c182466ed04f34e6bcfedebb02c3c4af7eaf9. Copyright the Prowlarr
// contributors, GPL-3.0. Modified for SlipStream: rewritten in Go.

package iptorrents

import (
	_ "embed"
	"net/url"
	"regexp"
	"slices"
	"strconv"
	"strings"

	"github.com/jatassi/slipstream/internal/newznab"
	"github.com/jatassi/slipstream/internal/ports/port"
	"github.com/jatassi/slipstream/internal/prowlarr/datetimeutil"
	"github.com/jatassi/slipstream/internal/prowlarr/dom"
	"github.com/jatassi/slipstream/internal/prowlarr/parseutil"
)

//go:embed upstream.json
var upstream []byte

type ix = port.Indexer[settings]

var Port = port.Define(port.Entry{
	ID:   "iptorrents",
	Name: "IPTorrents",
	IndexerURLs: []string{
		"https://iptorrents.com/",
		"https://iptorrents.me/",
		"https://nemo.iptorrents.com/",
		"https://ipt.getcrazy.me/",
		"https://ipt.findnemo.net/",
		"https://ipt.beelyrics.net/",
		"https://ipt.venom.global/",
		"https://ipt.workisboring.net/",
		"https://ipt.lol/",
		"https://ipt.cool/",
		"https://ipt.world/",
		"https://ipt.octopus.town/",
	},
	Description:  "IPTorrents is a private general tracker with movies, TV, music, games, apps and books.",
	Privacy:      port.Private,
	Upstream:     port.ReadUpstream(upstream),
	Capabilities: setCapabilities(),
}, port.Impl[settings]{
	Login: port.CookieLogin{UserAgent: true, CheckIfLoginNeeded: checkIfLoginNeeded},
	Movie: movieSearch, TV: tvSearch, Basic: basicSearch,
	Parse:          parseResponse,
	FilterReleases: cleanupReleases,
})

// SlipStream: logged out is cookie_expired, which the framework words, so
// the C#'s exception text isn't kept. The framework also reads downloads,
// and a .torrent has no sign-out link, so only an HTML page is read.
func checkIfLoginNeeded(r *port.IndexerResponse) bool {
	return strings.Contains(r.Headers.Get("Content-Type"), "html") && !strings.Contains(r.Content, "lout.php")
}

func cleanupReleases(_ *ix, q port.SearchCriteria, rs []port.TorrentInfo) []port.TorrentInfo {
	return port.FilterReleasesByQuery(rs, q)
}

// SlipStream: SlipStream has no music or book search, so the Music and
// Book search params aren't declared.
func setCapabilities() port.Capabilities {
	return port.Capabilities{
		TvSearchParams:    []port.Param{port.Q, port.Season, port.Ep, port.ImdbID},
		MovieSearchParams: []port.Param{port.Q, port.ImdbID},
		Categories: port.Categories(
			port.Map(72, newznab.Movies, "Movies"),
			port.Map(87, newznab.Movies3D, "Movie/3D"),
			port.Map(77, newznab.MoviesSD, "Movie/480p"),
			port.Map(101, newznab.MoviesUHD, "Movie/4K"),
			port.Map(89, newznab.MoviesHD, "Movie/BD-R"),
			port.Map(90, newznab.MoviesSD, "Movie/BD-Rip"),
			port.Map(96, newznab.MoviesSD, "Movie/Cam"),
			port.Map(6, newznab.MoviesDVD, "Movie/DVD-R"),
			port.Map(48, newznab.MoviesBluRay, "Movie/HD/Bluray"),
			port.Map(54, newznab.Movies, "Movie/Kids"),
			port.Map(62, newznab.MoviesSD, "Movie/MP4"),
			port.Map(38, newznab.MoviesForeign, "Movie/Non-English"),
			port.Map(68, newznab.Movies, "Movie/Packs"),
			port.Map(20, newznab.MoviesWEBDL, "Movie/Web-DL"),
			port.Map(7, newznab.MoviesSD, "Movie/Xvid"),
			port.Map(100, newznab.Movies, "Movie/x265"),

			port.Map(73, newznab.TV, "TV"),
			port.Map(26, newznab.TVDocumentary, "TV/Documentaries"),
			port.Map(55, newznab.TVSport, "Sports"),
			port.Map(78, newznab.TVSD, "TV/480p"),
			port.Map(23, newznab.TVHD, "TV/BD"),
			port.Map(24, newznab.TVSD, "TV/DVD-R"),
			port.Map(25, newznab.TVSD, "TV/DVD-Rip"),
			port.Map(66, newznab.TVSD, "TV/Mobile"),
			port.Map(82, newznab.TVForeign, "TV/Non-English"),
			port.Map(65, newznab.TV, "TV/Packs"),
			port.Map(83, newznab.TVForeign, "TV/Packs/Non-English"),
			port.Map(79, newznab.TVSD, "TV/SD/x264"),
			port.Map(22, newznab.TVWEBDL, "TV/Web-DL"),
			port.Map(5, newznab.TVHD, "TV/x264"),
			port.Map(99, newznab.TVHD, "TV/x265"),
			port.Map(4, newznab.TVSD, "TV/Xvid"),

			port.Map(74, newznab.Console, "Games"),
			port.Map(2, newznab.ConsoleOther, "Games/Mixed"),
			port.Map(47, newznab.ConsoleNDS, "Games/Nintendo DS"),
			port.Map(43, newznab.PCISO, "Games/PC-ISO"),
			port.Map(45, newznab.PCGames, "Games/PC-Rip"),
			port.Map(71, newznab.ConsolePS3, "Games/PS3"),
			port.Map(50, newznab.ConsoleWii, "Games/Wii"),
			port.Map(44, newznab.ConsoleXBox360, "Games/Xbox-360"),

			port.Map(75, newznab.Audio, "Music"),
			port.Map(3, newznab.AudioMP3, "Music/Audio"),
			port.Map(80, newznab.AudioLossless, "Music/Flac"),
			port.Map(93, newznab.Audio, "Music/Packs"),
			port.Map(37, newznab.AudioVideo, "Music/Video"),
			port.Map(21, newznab.AudioVideo, "Podcast"),

			port.Map(76, newznab.Other, "Other/Miscellaneous"),
			port.Map(60, newznab.TVAnime, "Anime"),
			port.Map(1, newznab.PC0day, "Appz"),
			port.Map(86, newznab.PC0day, "Appz/Non-English"),
			port.Map(64, newznab.AudioAudiobook, "AudioBook"),
			port.Map(35, newznab.Books, "Books"),
			port.Map(102, newznab.Books, "Books/Non-English"),
			port.Map(94, newznab.BooksComics, "Books/Comics"),
			port.Map(95, newznab.BooksOther, "Books/Educational"),
			port.Map(98, newznab.Other, "Other/Fonts"),
			port.Map(69, newznab.PCMac, "Appz/Mac"),
			port.Map(92, newznab.BooksMags, "Books/Magazines & Newspapers"),
			port.Map(58, newznab.PCMobileOther, "Appz/Mobile"),
			port.Map(36, newznab.Other, "Other/Pics/Wallpapers"),

			port.Map(88, newznab.XXX, "XXX"),
			port.Map(85, newznab.XXXOther, "XXX/Magazines"),
			port.Map(8, newznab.XXX, "XXX/Movie"),
			port.Map(81, newznab.XXX, "XXX/Movie/0Day"),
			port.Map(91, newznab.XXXPack, "XXX/Packs"),
			port.Map(84, newznab.XXXImageSet, "XXX/Pics/Wallpapers"),
		),
	}
}

func getPagedRequests(c *ix, term string, q port.SearchCriteria, imdbID string) (port.Chain, error) {
	searchURL := c.URL("t")

	var qc nameValueCollection

	for _, cat := range c.Categories.TrackerIDs(q.Categories) {
		qc.Add(cat, "")
	}

	if c.Settings.FreeLeechOnly {
		qc.Add("free", "on")
	}

	if !q.IsRssSearch() {
		switch c.Settings.SearchSortBy {
		case sortSeeders:
			qc.Add("o", "seeders")
		case sortLeechers:
			qc.Add("o", "leechers")
		case sortSnatches:
			qc.Add("o", "completed")
		case sortSize:
			qc.Add("o", "size")
		case sortName:
			qc.Add("o", "name")
		case sortDefault:
		}
	}

	if strings.TrimSpace(imdbID) != "" {
		qc.Add("q", "+("+imdbID+")")
		qc.Add("qf", "all")
	}

	if strings.TrimSpace(term) != "" {
		qc.Add("q", "+("+term+")")
	}

	if q.Limit > 0 && q.Offset > 0 {
		page := q.Offset/q.Limit + 1
		qc.Add("p", strconv.Itoa(page))
	}

	if qc.Count() > 0 {
		searchURL.RawQuery = qc.GetQueryString()
	}

	// SlipStream: the framework sends the pasted cookie's User-Agent on
	// every request, so the generator doesn't set it.
	return port.One(port.Get(searchURL).Accept("text/html"))
}

func movieSearch(c *ix, q port.SearchCriteria) (port.Chain, error) {
	return getPagedRequests(c, q.SearchTerm, q, q.FullImdbID())
}

func tvSearch(c *ix, q port.SearchCriteria) (port.Chain, error) {
	searchTerm := q.SanitizedTvSearchString()

	if q.Season > 0 && strings.TrimSpace(q.Episode) == "" {
		searchTerm += "*"
	}

	return getPagedRequests(c, strings.TrimSpace(searchTerm), q, q.FullImdbID())
}

func basicSearch(c *ix, q port.SearchCriteria) (port.Chain, error) {
	return getPagedRequests(c, q.SanitizedSearchTerm(), q, "")
}

// nameValueCollection is NameValueCollection with NzbDrone's
// GetQueryString: keys in the order first added, a repeated key's values
// joined by commas.
type nameValueCollection struct {
	keys   []string
	values map[string][]string
}

func (n *nameValueCollection) Add(k, v string) {
	if n.values == nil {
		n.values = map[string][]string{}
	}
	if _, ok := n.values[k]; !ok {
		n.keys = append(n.keys, k)
	}
	n.values[k] = append(n.values[k], v)
}

func (n *nameValueCollection) Count() int { return len(n.keys) }

func (n *nameValueCollection) GetQueryString() string {
	pairs := make([]string, len(n.keys))
	for i, k := range n.keys {
		pairs[i] = k + "=" + url.QueryEscape(strings.Join(n.values[k], ","))
	}
	return strings.Join(pairs, "&")
}

func parseResponse(c *ix, r *port.IndexerResponse) ([]port.TorrentInfo, error) {
	var torrentInfos []port.TorrentInfo

	doc := dom.Parse(r.Content)

	var headerColumns []string
	for _, x := range doc.QuerySelectorAll(`table[id="torrents"] > thead > tr > th`) {
		headerColumns = append(headerColumns, strings.TrimSpace(x.TextContent()))
	}
	sizeIndex := findColumnIndexOrDefault(headerColumns, "Sort by size", 5)
	filesIndex := findColumnIndexOrDefault(headerColumns, "Sort by files", -1)

	rows := doc.QuerySelectorAll(`table[id="torrents"] > tbody > tr`)
	for _, row := range rows {
		qTitleLink := row.QuerySelector("a.hv")

		if qTitleLink == nil {
			continue
		}

		release, err := parseRow(c, row, qTitleLink, headerColumns, sizeIndex, filesIndex)
		if err != nil {
			return nil, err
		}

		torrentInfos = append(torrentInfos, release)
	}

	return torrentInfos, nil
}

// SlipStream: the loop body is its own function, so the parser stays
// under the gocognit limit.
func parseRow(c *ix, row, qTitleLink *dom.Node, headerColumns []string, sizeIndex, filesIndex int) (port.TorrentInfo, error) {
	title := cleanTitle(qTitleLink.TextContent())
	details := c.URL(qTitleLink.GetAttribute("href"))

	qLink := row.QuerySelector(`a[href^="/download.php/"]`)
	link := c.URL(qLink.GetAttribute("href"))

	descrSplit := strings.Split(row.QuerySelector("div.sub").TextContent(), "|")
	dateSplit := strings.Split(descrSplit[len(descrSplit)-1], " by ")
	publishDate, err := datetimeutil.FromTimeAgo(dateSplit[0], c.Clock)
	if err != nil {
		return port.TorrentInfo{}, err
	}
	// SlipStream: a Release has no description, so the tags before
	// the first "|" aren't kept.

	catIcon := row.QuerySelector(`td:nth-of-type(1) a[href^="?"]`)
	if catIcon == nil {
		// SlipStream: reworded per WRITING.md. The C# asks the Admin
		// to change the same setting.
		return port.TorrentInfo{}, port.IndexerError(`the IPTorrents setting "Torrents - Category column" isn't Icons, and SlipStream reads categories only from the icons`)
	}

	cat := c.Categories.FromTracker(strings.TrimPrefix(catIcon.GetAttribute("href"), "?"))

	children := row.Children()
	size, err := parseutil.GetBytes(children[sizeIndex].TextContent())
	if err != nil {
		return port.TorrentInfo{}, err
	}

	var files int

	if filesIndex != -1 {
		if files, err = parseutil.CoerceInt(strings.ReplaceAll(children[filesIndex].TextContent(), "Go to files", "")); err != nil {
			return port.TorrentInfo{}, err
		}
	}

	colIndex := 6
	if len(children) == 10 {
		colIndex = 7
	}

	grabsIndex := findColumnIndexOrDefault(headerColumns, "Sort by snatches", colIndex)
	seedersIndex := findColumnIndexOrDefault(headerColumns, "Sort by seeders", colIndex+1)
	leechersIndex := findColumnIndexOrDefault(headerColumns, "Sort by leechers", colIndex+2)

	grabs, err := parseutil.CoerceInt(children[grabsIndex].TextContent())
	if err != nil {
		return port.TorrentInfo{}, err
	}
	seeders, err := parseutil.CoerceInt(children[seedersIndex].TextContent())
	if err != nil {
		return port.TorrentInfo{}, err
	}
	leechers, err := parseutil.CoerceInt(children[leechersIndex].TextContent())
	if err != nil {
		return port.TorrentInfo{}, err
	}

	downloadVolumeFactor := 1.0
	if row.QuerySelector("span.free") != nil {
		downloadVolumeFactor = 0
	}

	return port.TorrentInfo{
		GUID:                 details.String(),
		DownloadURL:          link.String(),
		InfoURL:              details.String(),
		Title:                title,
		Categories:           cat,
		Size:                 size,
		Files:                files,
		Grabs:                grabs,
		Seeders:              seeders,
		Peers:                seeders + leechers,
		PublishDate:          publishDate,
		DownloadVolumeFactor: downloadVolumeFactor,
		UploadVolumeFactor:   1,
		MinimumRatio:         1,
		MinimumSeedTime:      1209600,
	}, nil
}

func findColumnIndexOrDefault(columns []string, name string, defaultIndex int) int {
	if index := slices.Index(columns, name); index != -1 {
		return index
	}
	return defaultIndex
}

var (
	// SlipStream: .NET matches UTF-16 code units, so its Ā-￿
	// also drops every character above U+FFFF, as its surrogates.
	invalidChars = regexp.MustCompile(`[\x{0000}-\x{0008}\x{000A}-\x{001F}\x{0100}-\x{10FFFF}]`)
	requested    = regexp.MustCompile(`(?i)[\(\[\{]REQ(UEST(ED)?)?[\)\]\}]`)
	languageTag  = regexp.MustCompile(`(?i)^\[[a-z0-9 ._-]+\][-._ ](?P<title>.*-[a-z0-9]+)$`)
)

func cleanTitle(title string) string {
	title = invalidChars.ReplaceAllString(title, "")
	title = requested.ReplaceAllString(title, "")

	title = languageTag.ReplaceAllString(title, "${title}")

	return strings.Trim(title, " -:")
}

// SlipStream: the cookie and its User-Agent are the pasted session that
// CookieLogin{UserAgent: true} asks for, which also requires the
// User-Agent as IPTorrentsValidator does, so neither is a setting here.
type settings struct {
	FreeLeechOnly bool           `port:"freeLeechOnly" label:"Freeleech only" help:"Search only freeleech Releases."`
	SearchSortBy  iptorrentsSort `port:"searchSortBy" label:"Sort searches by" default:"0" help:"IPTorrents sorts each search's Releases in this order. RSS sync always takes the newest first."`
}

type iptorrentsSort int

const (
	sortDefault iptorrentsSort = iota
	sortSeeders
	sortLeechers
	sortSnatches
	sortSize
	sortName
)

// SlipStream: an option has no hint, so each label says what the C#
// hint says.
func (iptorrentsSort) Options() []port.Option {
	return []port.Option{
		{Value: "0", Label: "Newest first"},
		{Value: "1", Label: "Most seeders first"},
		{Value: "2", Label: "Most leechers first"},
		{Value: "3", Label: "Most snatches first"},
		{Value: "4", Label: "Largest first"},
		{Value: "5", Label: "Name, A to Z"},
	}
}
