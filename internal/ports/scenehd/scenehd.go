// Ported from Prowlarr's src/NzbDrone.Core/Indexers/Definitions/SceneHD.cs
// at ef9c182466ed04f34e6bcfedebb02c3c4af7eaf9. Copyright the Prowlarr
// contributors, GPL-3.0. Modified for SlipStream: rewritten in Go.

package scenehd

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/jatassi/slipstream/internal/newznab"
	"github.com/jatassi/slipstream/internal/ports/port"
	"github.com/jatassi/slipstream/internal/prowlarr/datetimeutil"
	"github.com/jatassi/slipstream/internal/prowlarr/parseutil"
	"github.com/jatassi/slipstream/internal/prowlarr/stringutil"
)

//go:embed upstream.json
var upstream []byte

// Port is SceneHD.
//
// SlipStream: the Music search params are gone, because SlipStream has no
// music mode. CheckIfLoginNeeded's false is NoLogin's own behavior.
var Port = port.Define(port.Entry{
	ID:          "scenehd",
	Name:        "SceneHD",
	Description: "SceneHD is a private Tracker for HD movies and TV.",
	Language:    "en-US",
	Privacy:     port.Private,
	IndexerURLs: []string{"https://scenehd.org/"},
	Capabilities: port.Capabilities{
		TvSearchParams:    []port.Param{port.Q, port.Season, port.Ep},
		MovieSearchParams: []port.Param{port.Q, port.ImdbID},
		Categories: port.Categories(
			port.Map(2, newznab.MoviesUHD, "Movie/2160"),
			port.Map(1, newznab.MoviesHD, "Movie/1080"),
			port.Map(4, newznab.MoviesHD, "Movie/720"),
			port.Map(8, newznab.MoviesBluRay, "Movie/BD5/9"),
			port.Map(6, newznab.TVUHD, "TV/2160"),
			port.Map(5, newznab.TVHD, "TV/1080"),
			port.Map(7, newznab.TVHD, "TV/720"),
			port.Map(22, newznab.MoviesBluRay, "Bluray/Complete"),
			port.Map(10, newznab.XXX, "XXX"),
			port.Map(16, newznab.MoviesOther, "Subpacks"),
			port.Map(13, newznab.AudioVideo, "MVID"),
			port.Map(9, newznab.Other, "Other"),
		),
	},
	Upstream: port.ReadUpstream(upstream),
}, port.Impl[sceneHDSettings]{
	Login: port.NoLogin{},
	Movie: getMovieSearchRequests,
	TV:    getTvSearchRequests,
	Basic: getBasicSearchRequests,
	Parse: parseResponse,
})

type indexer = port.Indexer[sceneHDSettings]

func getPagedRequests(ix *indexer, term string, categories []newznab.ID, imdbID string) *port.Request {
	var search []string
	for _, s := range []string{imdbID, term} {
		if strings.TrimSpace(s) != "" {
			search = append(search, s)
		}
	}

	var parameters stringutil.NameValueCollection
	parameters.Add("api", "")
	parameters.Add("passkey", ix.Settings.Passkey)
	parameters.Add("search", strings.Join(search, " "))

	if len(categories) > 0 {
		parameters.Add("cat", strings.Join(ix.Categories.TrackerIDs(categories), ","))
	}

	searchURL := ix.URL("browse.php?" + parameters.GetQueryString())

	return port.Get(searchURL).Accept("application/json")
}

func getMovieSearchRequests(ix *indexer, searchCriteria port.SearchCriteria) (port.Chain, error) {
	return port.One(getPagedRequests(ix, searchCriteria.SanitizedSearchTerm(), searchCriteria.Categories, searchCriteria.FullImdbID()))
}

// SlipStream: the Music and Book overloads are gone. SlipStream has
// neither mode.
func getTvSearchRequests(ix *indexer, searchCriteria port.SearchCriteria) (port.Chain, error) {
	return port.One(getPagedRequests(ix, searchCriteria.SanitizedTvSearchString(), searchCriteria.Categories, searchCriteria.FullImdbID()))
}

func getBasicSearchRequests(ix *indexer, searchCriteria port.SearchCriteria) (port.Chain, error) {
	return port.One(getPagedRequests(ix, searchCriteria.SanitizedSearchTerm(), searchCriteria.Categories, ""))
}

// SlipStream: the messages are SlipStream's own, per WRITING.md. "User not
// found or passkey not set" is a key the site refused, which SlipStream
// names account_refused.
func parseResponse(ix *indexer, indexerResponse *port.IndexerResponse) ([]port.TorrentInfo, error) {
	var torrentInfos []port.TorrentInfo

	detailsURL := ix.BaseURL.String() + "details.php?"
	downloadURL := ix.BaseURL.String() + "download.php?"

	if strings.Contains(indexerResponse.Content, "User not found or passkey not set") {
		return nil, port.IndexerAuthError("")
	}

	var jsonContent []jToken
	if err := json.Unmarshal(indexerResponse.Body, &jsonContent); err != nil {
		return nil, port.IndexerError("read the Indexer's search response: %v", err)
	}

	for _, item := range jsonContent {
		var f fields
		title := item.str("name")

		id := f.long(item, "id")
		details := detailsURL + "id=" + strconv.FormatInt(id, 10)
		link := downloadURL + "id=" + strconv.FormatInt(id, 10) + "&passkey=" + ix.Settings.Passkey
		publishDate, err := datetimeutil.ParseExact(item.str("added"), "yyyy-MM-dd HH:mm:ss", datetimeutil.None, ix.Clock)
		if err != nil && f.err == nil {
			f.err = err
		}
		dlVolumeFactor := 1.0
		if f.long(item, "is_freeleech") == 1 {
			dlVolumeFactor = 0
		}

		imdbID, _ := parseutil.GetImdbID(item.str("imdbid"))

		release := port.TorrentInfo{
			Title:                title,
			DownloadURL:          link,
			InfoURL:              details,
			GUID:                 details,
			Categories:           ix.Categories.FromTracker(item.str("category")),
			PublishDate:          publishDate,
			Size:                 f.long(item, "size"),
			Grabs:                f.integer(item, "times_completed"),
			Files:                f.integer(item, "numfiles"),
			Seeders:              f.integer(item, "seeders"),
			Peers:                f.integer(item, "leechers") + f.integer(item, "seeders"),
			ImdbID:               imdbID,
			MinimumRatio:         1,
			MinimumSeedTime:      0,
			DownloadVolumeFactor: dlVolumeFactor,
			UploadVolumeFactor:   1,
		}
		if f.err != nil {
			return nil, port.IndexerError("read torrent %q of the Indexer's search response: %v", title, f.err)
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

// fields is Value<long> and Value<int>, keeping the first failure.
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

func (f *fields) integer(t jToken, key string) int {
	s := t.str(key)
	if s == "" {
		return 0
	}
	v, err := strconv.ParseInt(s, 10, 32)
	if err != nil && f.err == nil {
		f.err = fmt.Errorf("%s %q isn't a whole number", key, s)
	}
	return int(v)
}

// SlipStream: Prowlarr's validator is the required flag on the passkey.
// Its check that the passkey is 32 characters long is gone, because a
// field has no length rule.
type sceneHDSettings struct {
	port.PasskeyField
}
