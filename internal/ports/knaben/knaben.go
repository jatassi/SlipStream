// Ported from Prowlarr's src/NzbDrone.Core/Indexers/Definitions/Knaben.cs
// at ef9c182466ed04f34e6bcfedebb02c3c4af7eaf9. Copyright the Prowlarr
// contributors, GPL-3.0. Modified for SlipStream: rewritten in Go.

package knaben

import (
	_ "embed"
	"encoding/json"
	"net/http"
	"regexp"
	"slices"
	"strconv"
	"strings"

	"github.com/jatassi/slipstream/internal/newznab"
	"github.com/jatassi/slipstream/internal/ports/port"
	"github.com/jatassi/slipstream/internal/prowlarr/datetimeutil"
)

//go:embed upstream.json
var upstream []byte

type ix = port.Indexer[struct{}]

var Port = port.Define(port.Entry{
	ID:   "knaben",
	Name: "Knaben",
	// SlipStream: the address is the API's host, where every request goes,
	// so a typed mirror moves the API too. Prowlarr's IndexerUrls is the
	// site's front page, which the port never reads, and its API address
	// is a constant.
	IndexerURLs:  []string{"https://api.knaben.org/"},
	LegacyURLs:   []string{"https://knaben.eu/", "https://knaben.org/"},
	Description:  "Knaben is a public search engine over the Releases of many other sites.",
	Privacy:      port.Public,
	Capabilities: setCapabilities(),
	Upstream:     port.ReadUpstream(upstream),
}, port.Impl[struct{}]{
	Login: port.NoLogin{},
	Movie: movieSearch, TV: tvSearch, Basic: basicSearch,
	Parse: parseResponse,
})

// SlipStream: SlipStream has no music or book search, so the Music and
// Book search params aren't declared.
func setCapabilities() port.Capabilities {
	return port.Capabilities{
		TvSearchParams:    []port.Param{port.Q, port.Season, port.Ep},
		MovieSearchParams: []port.Param{port.Q},
		Categories: port.Categories(
			port.Map(1000000, newznab.Audio, "Audio"),
			port.Map(1001000, newznab.AudioMP3, "MP3"),
			port.Map(1002000, newznab.AudioLossless, "Lossless"),
			port.Map(1003000, newznab.AudioAudiobook, "Audiobook"),
			port.Map(1004000, newznab.AudioVideo, "Audio Video"),
			port.Map(1005000, newznab.AudioOther, "Radio"),
			port.Map(1006000, newznab.AudioOther, "Audio Other"),
			port.Map(2000000, newznab.TV, "TV"),
			port.Map(2001000, newznab.TVHD, "TV HD"),
			port.Map(2002000, newznab.TVSD, "TV SD"),
			port.Map(2003000, newznab.TVUHD, "TV UHD"),
			port.Map(2004000, newznab.TVDocumentary, "Documentary"),
			port.Map(2005000, newznab.TVForeign, "TV Foreign"),
			port.Map(2006000, newznab.TVSport, "Sport"),
			port.Map(2007000, newznab.TVOther, "Cartoon"),
			port.Map(2008000, newznab.TVOther, "TV Other"),
			port.Map(3000000, newznab.Movies, "Movies"),
			port.Map(3001000, newznab.MoviesHD, "Movies HD"),
			port.Map(3002000, newznab.MoviesSD, "Movies SD"),
			port.Map(3003000, newznab.MoviesUHD, "Movies UHD"),
			port.Map(3004000, newznab.MoviesDVD, "Movies DVD"),
			port.Map(3005000, newznab.MoviesForeign, "Movies Foreign"),
			port.Map(3006000, newznab.MoviesForeign, "Movies Bollywood"),
			port.Map(3007000, newznab.Movies3D, "Movies 3D"),
			port.Map(3008000, newznab.MoviesOther, "Movies Other"),
			port.Map(4000000, newznab.PC, "PC"),
			port.Map(4001000, newznab.PCGames, "Games"),
			port.Map(4002000, newznab.PC0day, "Software"),
			port.Map(4003000, newznab.PCMac, "Mac"),
			port.Map(4004000, newznab.PCISO, "Unix"),
			port.Map(5000000, newznab.XXX, "XXX"),
			port.Map(5001000, newznab.XXXx264, "XXX Video"),
			port.Map(5002000, newznab.XXXImageSet, "XXX ImageSet"),
			port.Map(5003000, newznab.XXXOther, "XXX Games"),
			port.Map(5004000, newznab.XXXOther, "XXX Hentai"),
			port.Map(5005000, newznab.XXXOther, "XXX Other"),
			port.Map(6000000, newznab.TVAnime, "Anime"),
			port.Map(6001000, newznab.TVAnime, "Anime Subbed"),
			port.Map(6002000, newznab.TVAnime, "Anime Dubbed"),
			port.Map(6003000, newznab.TVAnime, "Anime Dual audio"),
			port.Map(6004000, newznab.TVAnime, "Anime Raw"),
			port.Map(6005000, newznab.AudioVideo, "Music Video"),
			port.Map(6006000, newznab.BooksOther, "Literature"),
			port.Map(6007000, newznab.AudioOther, "Music"),
			port.Map(6008000, newznab.TVAnime, "Anime non-english translated"),
			port.Map(7000000, newznab.Console, "Console"),
			port.Map(7001000, newznab.ConsolePS4, "PS4"),
			port.Map(7002000, newznab.ConsolePS3, "PS3"),
			port.Map(7003000, newznab.ConsolePS3, "PS2"),
			port.Map(7004000, newznab.ConsolePS3, "PS1"),
			port.Map(7005000, newznab.ConsolePSVita, "PS Vita"),
			port.Map(7006000, newznab.ConsolePSP, "PSP"),
			port.Map(7007000, newznab.ConsoleXBox360, "Xbox 360"),
			port.Map(7008000, newznab.ConsoleXBox, "Xbox"),
			port.Map(7009000, newznab.ConsoleNDS, "Switch"),
			port.Map(7010000, newznab.ConsoleNDS, "NDS"),
			port.Map(7011000, newznab.ConsoleWii, "Wii"),
			port.Map(7012000, newznab.ConsoleWiiU, "WiiU"),
			port.Map(7013000, newznab.Console3DS, "3DS"),
			port.Map(7014000, newznab.ConsoleWii, "GameCube"),
			port.Map(7015000, newznab.ConsoleOther, "Other"),
			port.Map(8000000, newznab.PCMobileOther, "Mobile"),
			port.Map(8001000, newznab.PCMobileAndroid, "Android"),
			port.Map(8002000, newznab.PCMobileiOS, "IOS"),
			port.Map(8003000, newznab.PCMobileOther, "PC Other"),
			port.Map(9000000, newznab.Books, "Books"),
			port.Map(9001000, newznab.BooksEBook, "EBooks"),
			port.Map(9002000, newznab.BooksComics, "Comics"),
			port.Map(9003000, newznab.BooksMags, "Magazines"),
			port.Map(9004000, newznab.BooksTechnical, "Technical"),
			port.Map(9005000, newznab.BooksOther, "Books Other"),
			port.Map(10000000, newznab.Other, "Other"),
			port.Map(10001000, newznab.OtherMisc, "Other Misc"),
		),
	}
}

func movieSearch(c *ix, q port.SearchCriteria) (port.Chain, error) {
	return port.One(createRequest(c, q, q.SanitizedSearchTerm()))
}

func tvSearch(c *ix, q port.SearchCriteria) (port.Chain, error) {
	return port.One(createRequest(c, q, q.SanitizedTvSearchString()))
}

func basicSearch(c *ix, q port.SearchCriteria) (port.Chain, error) {
	return port.One(createRequest(c, q, q.SanitizedSearchTerm()))
}

// knabenBody is the C# body dictionary, its keys in the order it adds
// them.
type knabenBody struct {
	OrderBy        string `json:"order_by"`
	OrderDirection string `json:"order_direction"`
	From           int    `json:"from"`
	Size           int    `json:"size"`
	HideUnsafe     bool   `json:"hide_unsafe"`
	SearchType     string `json:"search_type,omitempty"`
	SearchField    string `json:"search_field,omitempty"`
	Query          string `json:"query,omitempty"`
	Categories     []int  `json:"categories,omitempty"`
}

func createRequest(c *ix, q port.SearchCriteria, searchTerm string) *port.Request {
	body := knabenBody{
		OrderBy:        "date",
		OrderDirection: "desc",
		From:           0,
		Size:           100,
		HideUnsafe:     true,
	}

	searchQuery := strings.TrimSpace(searchTerm)

	if searchQuery != "" {
		body.SearchType = "100%"
		body.SearchField = "title"
		body.Query = searchQuery
	}

	for _, cat := range c.Categories.TrackerIDs(q.Categories) {
		if id, err := strconv.Atoi(cat); err == nil && !slices.Contains(body.Categories, id) {
			body.Categories = append(body.Categories, id)
		}
	}

	return port.Post(c.URL("v1")).Accept("application/json").SetJSON(body)
}

var dateTimezoneRegex = regexp.MustCompile(`[+-]\d{2}:\d{2}$`)

// SlipStream: the messages are SlipStream's own, per WRITING.md.
func parseResponse(c *ix, r *port.IndexerResponse) ([]port.TorrentInfo, error) {
	if r.StatusCode != http.StatusOK {
		return nil, port.IndexerError("the Indexer responded with status %d", r.StatusCode)
	}

	if contentType := r.Headers.Get("Content-Type"); !strings.Contains(contentType, "application/json") {
		return nil, port.IndexerError("the Indexer responded with %q where application/json was expected", contentType)
	}

	var releaseInfos []port.TorrentInfo

	var jsonResponse knabenResponse
	if err := json.Unmarshal(r.Body, &jsonResponse); err != nil {
		return nil, port.IndexerError("read the Indexer's search response: %v", err)
	}

	if jsonResponse.Hits == nil {
		return releaseInfos, nil
	}

	for _, row := range jsonResponse.Hits {
		if row.Seeders <= 0 {
			continue
		}

		// Not all entries have the TZ in the "date" field
		publishDate := row.Date
		if strings.TrimSpace(row.Date) != "" && !dateTimezoneRegex.MatchString(row.Date) {
			publishDate = row.Date + "+01:00"
		}

		date, err := datetimeutil.Parse(publishDate, datetimeutil.AssumeUniversal, c.Clock)
		if err != nil {
			return nil, port.IndexerError("read the date of %q in the Indexer's search response: %v", row.Title, err)
		}

		var categories []newznab.Category
		for _, cat := range row.CategoryIDs {
			for _, mapped := range c.Categories.FromTracker(strconv.Itoa(cat)) {
				if !slices.Contains(categories, mapped) {
					categories = append(categories, mapped)
				}
			}
		}

		releaseInfos = append(releaseInfos, port.TorrentInfo{
			GUID:                 row.InfoURL,
			Title:                row.Title,
			InfoURL:              row.InfoURL,
			DownloadURL:          notWhiteSpace(row.DownloadURL),
			MagnetURL:            notWhiteSpace(row.MagnetURL),
			Categories:           categories,
			InfoHash:             row.InfoHash,
			Size:                 row.Size,
			Seeders:              row.Seeders,
			Peers:                row.Leechers + row.Seeders,
			PublishDate:          date,
			DownloadVolumeFactor: 0,
			UploadVolumeFactor:   1,
		})
	}

	return releaseInfos, nil
}

// notWhiteSpace is C#'s s.IsNotNullOrWhiteSpace() ? s : null.
func notWhiteSpace(s string) string {
	if strings.TrimSpace(s) == "" {
		return ""
	}
	return s
}

type knabenResponse struct {
	Hits []knabenRelease `json:"hits"`
}

type knabenRelease struct {
	Title       string `json:"title"`
	CategoryIDs []int  `json:"categoryId"`
	InfoHash    string `json:"hash"`
	InfoURL     string `json:"details"`
	DownloadURL string `json:"link"`
	MagnetURL   string `json:"magnetUrl"`
	Size        int64  `json:"bytes"`
	Seeders     int    `json:"seeders"`
	Leechers    int    `json:"peers"`
	Date        string `json:"date"`
}
