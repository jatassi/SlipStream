// Ported from Prowlarr's src/NzbDrone.Core/Indexers/Definitions/TorrentSyndikat.cs
// at ef9c182466ed04f34e6bcfedebb02c3c4af7eaf9. Copyright the Prowlarr
// contributors, GPL-3.0. Modified for SlipStream: rewritten in Go.

package torrentsyndikat

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"strconv"
	"strings"

	"github.com/jatassi/slipstream/internal/newznab"
	"github.com/jatassi/slipstream/internal/ports/port"
	"github.com/jatassi/slipstream/internal/prowlarr/datetimeutil"
	"github.com/jatassi/slipstream/internal/prowlarr/stringutil"
)

//go:embed upstream.json
var upstream []byte

// Port is TorrentSyndikat.
//
// SlipStream: the Music and Book search params are gone, because
// SlipStream has neither mode.
var Port = port.Define(port.Entry{
	ID:          "torrentsyndikat",
	Name:        "TorrentSyndikat",
	Description: "TorrentSyndikat is a private German Tracker for movies, TV and more.",
	Language:    "de-DE",
	Privacy:     port.Private,
	IndexerURLs: []string{"https://torrent-syndikat.org/"},
	Capabilities: port.Capabilities{
		TvSearchParams:    []port.Param{port.Q, port.Season, port.Ep},
		MovieSearchParams: []port.Param{port.Q, port.ImdbID},
		Categories: port.Categories(
			port.Map(2, newznab.PC, "Apps / Windows"),
			port.Map(13, newznab.PC, "Apps / Linux"),
			port.Map(4, newznab.PCMac, "Apps / MacOS"),
			port.Map(6, newznab.PC, "Apps / Misc"),

			port.Map(50, newznab.PCGames, "Spiele / Windows"),
			port.Map(51, newznab.PCGames, "Spiele / MacOS"),
			port.Map(52, newznab.PCGames, "Spiele / Linux"),
			port.Map(8, newznab.ConsoleOther, "Spiele / Playstation"),
			port.Map(7, newznab.ConsoleOther, "Spiele / Nintendo"),
			port.Map(32, newznab.ConsoleOther, "Spiele / XBOX"),

			port.Map(42, newznab.MoviesUHD, "Filme / 2160p"),
			port.Map(9, newznab.MoviesHD, "Filme / 1080p"),
			port.Map(20, newznab.MoviesHD, "Filme / 720p"),
			port.Map(10, newznab.MoviesSD, "Filme / SD"),

			port.Map(43, newznab.TVUHD, "Serien / 2160p"),
			port.Map(53, newznab.TVHD, "Serien / 1080p"),
			port.Map(54, newznab.TVHD, "Serien / 720p"),
			port.Map(15, newznab.TVSD, "Serien / SD"),
			port.Map(30, newznab.TVSport, "Serien / Sport"),

			port.Map(44, newznab.TVUHD, "Serienpacks / 2160p"),
			port.Map(55, newznab.TVHD, "Serienpacks / 1080p"),
			port.Map(56, newznab.TVHD, "Serienpacks / 720p"),
			port.Map(27, newznab.TVSD, "Serienpacks / SD"),

			port.Map(24, newznab.AudioLossless, "Audio / Musik / FLAC"),
			port.Map(25, newznab.AudioMP3, "Audio / Musik / MP3"),
			port.Map(35, newznab.AudioOther, "Audio / Other"),
			port.Map(18, newznab.AudioAudiobook, "Audio / aBooks"),
			port.Map(33, newznab.AudioVideo, "Audio / Videos"),

			port.Map(17, newznab.Books, "Misc / eBooks"),
			port.Map(5, newznab.PCMobileOther, "Misc / Mobile"),
			port.Map(39, newznab.Other, "Misc / Bildung"),

			port.Map(36, newznab.TVForeign, "Englisch / Serien"),
			port.Map(57, newznab.TVForeign, "Englisch / Serienpacks"),
			port.Map(37, newznab.MoviesForeign, "Englisch / Filme"),
			port.Map(47, newznab.Books, "Englisch / eBooks"),
			port.Map(48, newznab.Other, "Englisch / Bildung"),
			port.Map(49, newznab.TVSport, "Englisch / Sport"),
		),
	},
	Upstream: port.ReadUpstream(upstream),
}, port.Impl[torrentSyndikatSettings]{
	Login: port.NoLogin{},
	Movie: getMovieSearchRequests,
	TV:    getTvSearchRequests,
	Basic: getBasicSearchRequests,
	Parse: parseResponse,
})

type indexer = port.Indexer[torrentSyndikatSettings]

var seasonOnly = regexp.MustCompile(`(?i)(^|\s)(s\d{2})(\s|$)`)

func getPagedRequests(ix *indexer, term string, categories []newznab.ID, imdbID string) *port.Request {
	searchString := term
	var queryCollection stringutil.NameValueCollection
	queryCollection.Add("apikey", ix.Settings.APIKey)

	queryCollection.Add("limit", "50") // Default 30
	if ix.Settings.ProductsOnly {
		queryCollection.Add("ponly", "true")
	} else {
		queryCollection.Add("ponly", "false")
	}
	for _, releaseType := range ix.Settings.ReleaseTypes {
		queryCollection.Add("release_type", strings.ReplaceAll(releaseType.String(), "_", "-"))
	}

	// SlipStream: a Go empty ImdbID is C#'s null.
	if imdbID != "" {
		queryCollection.Add("imdbId", imdbID)
	} else if strings.TrimSpace(searchString) != "" {
		// Suffix the first occurrence of `s01` surrounded by whitespace with *
		// That way we also search for single episodes in a whole season search
		queryCollection.Add("searchstring", seasonOnly.ReplaceAllString(strings.TrimSpace(searchString), "${1}${2}*${3}"))
	}

	if cats := strings.Join(ix.Categories.TrackerIDs(categories), ","); cats != "" {
		queryCollection.Add("cats", cats)
	}

	searchURL := ix.URL("api_9djWe8Tb2NE3p6opyqnh/v1/browse.php?" + queryCollection.GetQueryString())

	return port.Get(searchURL).Accept("text/html")
}

func getMovieSearchRequests(ix *indexer, searchCriteria port.SearchCriteria) (port.Chain, error) {
	return port.One(getPagedRequests(ix, searchCriteria.SanitizedSearchTerm(), searchCriteria.Categories, searchCriteria.ImdbID))
}

// SlipStream: the Music and Book overloads are gone. SlipStream has
// neither mode.
func getTvSearchRequests(ix *indexer, searchCriteria port.SearchCriteria) (port.Chain, error) {
	return port.One(getPagedRequests(ix, searchCriteria.SanitizedTvSearchString(), searchCriteria.Categories, searchCriteria.ImdbID))
}

func getBasicSearchRequests(ix *indexer, searchCriteria port.SearchCriteria) (port.Chain, error) {
	return port.One(getPagedRequests(ix, searchCriteria.SanitizedSearchTerm(), searchCriteria.Categories, ""))
}

// SlipStream: the messages are SlipStream's own, per WRITING.md. A 401 is
// a key the site refused, which SlipStream names account_refused. The
// poster and the description Prowlarr builds from the title, year,
// product, genres, tags and release type are gone, because a Release
// carries neither.
func parseResponse(ix *indexer, indexerResponse *port.IndexerResponse) ([]port.TorrentInfo, error) {
	var torrentInfos []port.TorrentInfo

	if indexerResponse.StatusCode == http.StatusUnauthorized {
		return nil, port.IndexerAuthError("")
	}

	var jsonContent struct {
		Rows []jToken `json:"rows"`
	}
	if err := json.Unmarshal(indexerResponse.Body, &jsonContent); err != nil {
		return nil, port.IndexerError("read the Indexer's search response: %v", err)
	}
	if jsonContent.Rows == nil {
		return nil, port.IndexerError("the Indexer's search response has no rows")
	}

	for _, row := range jsonContent.Rows {
		var f fields

		id := row.str("id")
		details := ix.BaseURL.String() + "details.php?id=" + id
		seeders := int(f.long(row, "seeders"))

		release := port.TorrentInfo{
			MinimumRatio:         1,
			MinimumSeedTime:      96 * 60 * 60,
			DownloadVolumeFactor: 1,
			UploadVolumeFactor:   1,
			GUID:                 details,
			InfoURL:              details,
			DownloadURL:          ix.BaseURL.String() + "download.php?id=" + id + "&apikey=" + ix.Settings.APIKey,
			Title:                row.str("name"),
			Categories:           ix.Categories.FromTracker(strconv.FormatInt(f.long(row, "category"), 10)),
			PublishDate:          datetimeutil.UnixTimestampToDateTime(f.long(row, "added"), ix.Clock),
			Size:                 f.long(row, "size"),
			Files:                int(f.long(row, "numfiles")),
			Seeders:              seeders,
			Peers:                seeders + int(f.long(row, "leechers")),
			Grabs:                int(f.long(row, "snatched")),
			ImdbID:               int(f.long(row, "imdbId")),
			TvdbID:               int(f.long(row, "tvdbId")),
			TmdbID:               int(f.long(row, "tmdbId")),
		}
		if f.err != nil {
			return nil, port.IndexerError("read torrent %s of the Indexer's search response: %v", id, f.err)
		}

		torrentInfos = append(torrentInfos, release)
	}

	return torrentInfos, nil
}

// jToken is one JArray item, read as JToken.Value<T> reads it: a number
// or a string alike.
type jToken map[string]json.RawMessage

// str is Value<string>: a string's text, a number's digits, "" for null
// or a missing key.
func (t jToken) str(key string) string {
	raw := t[key]
	var s string
	if json.Unmarshal(raw, &s) == nil {
		return s
	}
	if string(raw) == "null" {
		return ""
	}
	return string(raw)
}

// fields is Value<long>, Value<int> and Value<int?>, zero for null,
// keeping the first failure.
type fields struct{ err error }

func (f *fields) long(t jToken, key string) int64 {
	s := t.str(key)
	if s == "" {
		return 0
	}
	v, err := strconv.ParseInt(s, 10, 64)
	if err != nil && f.err == nil {
		f.err = fmt.Errorf("%s %q isn't a whole number", key, s)
	}
	return v
}

// SlipStream: Prowlarr's validator is the required flag on the API key.
type torrentSyndikatSettings struct {
	port.APIKeyField
	ProductsOnly bool                         `port:"productsOnly" label:"Products only" help:"Search only torrents linked to a product."`
	ReleaseTypes []torrentSyndikatReleaseType `port:"releaseTypes" label:"Release types"`
}

type torrentSyndikatReleaseType int

const (
	releaseTypeP2P    torrentSyndikatReleaseType = 1
	releaseTypeScene  torrentSyndikatReleaseType = 2
	releaseTypeOScene torrentSyndikatReleaseType = 3
)

// String is the enum's C# name.
func (t torrentSyndikatReleaseType) String() string {
	switch t {
	case releaseTypeP2P:
		return "P2P"
	case releaseTypeScene:
		return "Scene"
	case releaseTypeOScene:
		return "O_Scene"
	}
	return strconv.Itoa(int(t))
}

func (torrentSyndikatReleaseType) Options() []port.Option {
	return []port.Option{
		{Value: "1", Label: "P2P"},
		{Value: "2", Label: "Scene"},
		{Value: "3", Label: "O-Scene"},
	}
}
