// Ported from Prowlarr's src/NzbDrone.Core/Indexers/Definitions/BeyondHD.cs
// at ef9c182466ed04f34e6bcfedebb02c3c4af7eaf9. Copyright the Prowlarr
// contributors, GPL-3.0. Modified for SlipStream: rewritten in Go.

package beyondhd

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"net/http"
	"slices"
	"strconv"
	"strings"

	"github.com/jatassi/slipstream/internal/newznab"
	"github.com/jatassi/slipstream/internal/ports/port"
	"github.com/jatassi/slipstream/internal/prowlarr/datetimeutil"
	"github.com/jatassi/slipstream/internal/prowlarr/parseutil"
	"github.com/jatassi/slipstream/internal/prowlarr/serializer"
)

//go:embed upstream.json
var upstream []byte

// Port is BeyondHD.
//
// SlipStream: the id is the one Prowlarr's hidden list names, and the
// package drops its hyphen.
var Port = port.Define(port.Entry{
	ID:          "beyond-hd",
	Name:        "BeyondHD",
	Description: "BeyondHD is a private tracker for HD movies and TV.",
	Privacy:     port.Private,
	IndexerURLs: []string{"https://beyond-hd.me/"},
	PageSize:    100,
	Capabilities: port.Capabilities{
		TvSearchParams:    []port.Param{port.Q, port.Season, port.Ep, port.ImdbID},
		MovieSearchParams: []port.Param{port.Q, port.ImdbID, port.TmdbID},
		Categories: port.Categories(
			port.Map(1, newznab.Movies, "Movies"),
			port.Map(2, newznab.TV, "TV"),
		),
	},
	Replaces: []string{"beyond-hd-oneurl"},
	Aliases:  []string{"beyond-hd-api"},
	Upstream: port.ReadUpstream(upstream),
}, port.Impl[beyondHDSettings]{
	Login:          port.NoLogin{},
	Movie:          getMovieSearchRequests,
	TV:             getTvSearchRequests,
	Basic:          getBasicSearchRequests,
	Parse:          parseResponse,
	FilterReleases: cleanupReleases,
})

type indexer = port.Indexer[beyondHDSettings]

func cleanupReleases(_ *indexer, searchCriteria port.SearchCriteria, releases []port.TorrentInfo) []port.TorrentInfo {
	return port.FilterReleasesByQuery(releases, searchCriteria)
}

func getPagedRequests(ix *indexer, searchCriteria port.SearchCriteria, searchTerm, imdbID string, tmdbID int) *port.Request {
	body := map[string]any{
		"action": "search",
		"rsskey": ix.Settings.RSSKey,
	}

	if ix.Settings.FreeleechOnly {
		body["freeleech"] = 1
	}

	if ix.Settings.LimitedOnly {
		body["limited"] = 1
	}

	if ix.Settings.RefundOnly {
		body["refund"] = 1
	}

	if ix.Settings.RewindOnly {
		body["rewind"] = 1
	}

	if strings.TrimSpace(imdbID) != "" {
		body["imdb_id"] = imdbID
	} else if tmdbID > 0 {
		body["tmdb_id"] = fmt.Sprintf("movie/%d", tmdbID)
	}

	if strings.TrimSpace(searchTerm) != "" {
		body["search"] = strings.TrimSpace(searchTerm)
	}

	cats := ix.Categories.TrackerIDs(searchCriteria.Categories)

	if len(cats) > 0 {
		ids := make([]int, len(cats))
		for i, cat := range cats {
			ids[i], _ = strconv.Atoi(cat)
		}
		body["categories"] = ids
	}

	if len(ix.Settings.SearchTypes) > 0 {
		var searchTypes []string
		for _, t := range ix.Settings.SearchTypes {
			if label := t.label(); label != "" && !slices.Contains(searchTypes, label) {
				searchTypes = append(searchTypes, label)
			}
		}

		if len(searchTypes) > 0 {
			body["types"] = searchTypes
		}
	}

	if searchCriteria.Limit > 0 && searchCriteria.Offset > 0 {
		page := searchCriteria.Offset/searchCriteria.Limit + 1
		body["page"] = page
	}

	searchURL := ix.URL("api/torrents/" + ix.Settings.APIKey)

	return port.Post(searchURL).Accept("application/json").SetJSON(body)
}

func getMovieSearchRequests(ix *indexer, searchCriteria port.SearchCriteria) (port.Chain, error) {
	return port.One(getPagedRequests(ix, searchCriteria, searchCriteria.SanitizedSearchTerm(), searchCriteria.FullImdbID(), searchCriteria.TmdbID))
}

// SlipStream: the Music and Book overloads are gone. The capabilities
// declare neither, so Prowlarr never calls them.
func getTvSearchRequests(ix *indexer, searchCriteria port.SearchCriteria) (port.Chain, error) {
	searchTerm := searchCriteria.SanitizedTvSearchString()

	if searchCriteria.Season != nil && *searchCriteria.Season > 0 && strings.TrimSpace(searchCriteria.Episode) != "" {
		if showDate, ok := datetimeutil.TryParseExact(port.Interpolated(searchCriteria.Season)+" "+searchCriteria.Episode, "yyyy MM/dd", datetimeutil.None, ix.Clock); ok {
			searchTerm = searchCriteria.SanitizedSearchTerm() + " " + showDate.Format("2006-01-02")
		}
	}

	return port.One(getPagedRequests(ix, searchCriteria, searchTerm, searchCriteria.FullImdbID(), 0))
}

func getBasicSearchRequests(ix *indexer, searchCriteria port.SearchCriteria) (port.Chain, error) {
	return port.One(getPagedRequests(ix, searchCriteria, searchCriteria.SanitizedSearchTerm(), "", 0))
}

// SlipStream: the messages are SlipStream's own, per WRITING.md. "Invalid
// API Key" is a key the site refused, named account_refused with
// BeyondHD's status message.
func parseResponse(ix *indexer, indexerResponse *port.IndexerResponse) ([]port.TorrentInfo, error) {
	if indexerResponse.StatusCode != http.StatusOK {
		return nil, port.IndexerError("the Indexer responded with status %d", indexerResponse.StatusCode)
	}

	contentType := indexerResponse.Headers.Get("Content-Type")

	if strings.Contains(contentType, "text/html") {
		if strings.Contains(strings.ToLower(indexerResponse.Content), "site maintenance") {
			return nil, port.IndexerError("the Indexer responded with its maintenance page")
		}
		return nil, port.IndexerError("the Indexer responded with HTML where JSON was expected")
	}

	if !strings.Contains(contentType, "application/json") {
		return nil, port.IndexerError("the Indexer responded with %q where application/json was expected", contentType)
	}

	var jsonResponse beyondHDResponse
	err := json.Unmarshal(indexerResponse.Body, &jsonResponse)

	if strings.Contains(strings.ToLower(indexerResponse.Content), "invalid api key") {
		return nil, port.IndexerAuthError(jsonResponse.StatusMessage)
	}

	if err != nil {
		return nil, port.IndexerError("read the Indexer's search response: %v", err)
	}

	if jsonResponse.StatusCode == 0 {
		return nil, port.IndexerError("the Indexer's API responded with the error %q", jsonResponse.StatusMessage)
	}

	var releaseInfos []port.TorrentInfo

	for _, row := range jsonResponse.Results {
		// Skip invalid results when freeleech or limited filtering is set
		if (ix.Settings.FreeleechOnly && !bool(row.Freeleech)) || (ix.Settings.LimitedOnly && !bool(row.Limited)) {
			continue
		}

		details := row.InfoURL
		link := row.DownloadLink

		publishDate, err := datetimeutil.Parse(row.CreatedAt, datetimeutil.AssumeUniversal, ix.Clock)
		if err != nil {
			return nil, port.IndexerError("read torrent %q of the Indexer's search response: %v", row.Name, err)
		}

		imdbID, _ := parseutil.GetImdbID(row.ImdbID)

		var downloadVolumeFactor float64
		switch {
		case bool(row.Freeleech || row.Limited):
			downloadVolumeFactor = 0
		case bool(row.Promo75):
			downloadVolumeFactor = 0.25
		case bool(row.Promo50):
			downloadVolumeFactor = 0.5
		case bool(row.Promo25):
			downloadVolumeFactor = 0.75
		default:
			downloadVolumeFactor = 1
		}

		// SlipStream: TorrentInfo has no Languages or Subs, because a
		// Release carries neither.
		release := port.TorrentInfo{
			Title:                row.Name,
			DownloadURL:          link,
			InfoHash:             row.InfoHash,
			InfoURL:              details,
			GUID:                 details,
			Categories:           ix.Categories.FromDesc(row.Category),
			PublishDate:          publishDate,
			IndexerFlags:         getIndexerFlags(row),
			Size:                 row.Size,
			Grabs:                row.Grabs,
			Seeders:              row.Seeders,
			ImdbID:               imdbID,
			Peers:                row.Leechers + row.Seeders,
			DownloadVolumeFactor: downloadVolumeFactor,
			UploadVolumeFactor:   1,
			MinimumRatio:         1,
			MinimumSeedTime:      172800, // 120 hours
		}

		// BHD can return crazy values for tmdb
		if strings.TrimSpace(row.TmdbID) != "" {
			if parts := strings.Split(row.TmdbID, "/"); len(parts) > 1 {
				release.TmdbID, _ = parseutil.TryCoerceInt(parts[1])
			}
		}

		releaseInfos = append(releaseInfos, release)
	}

	// order by date
	slices.SortStableFunc(releaseInfos, func(a, b port.TorrentInfo) int { return b.PublishDate.Compare(a.PublishDate) })

	return releaseInfos, nil
}

func getIndexerFlags(item beyondHDTorrent) []port.SiteFlag {
	var flags []port.SiteFlag

	if bool(item.Internal) {
		flags = append(flags, port.Internal)
	}

	if bool(item.Exclusive) {
		flags = append(flags, port.Exclusive)
	}

	return flags
}

// SlipStream: Prowlarr's validator is the required flag on the keys. Its
// check that each is 32 characters long is gone, because a field has no
// length rule.
type beyondHDSettings struct {
	port.APIKeyField `help:"Find your API key on BeyondHD under My Security > API Key."`
	port.RSSKeyField `help:"Find your RSS key on BeyondHD under My Security > RSS Key."`
	FreeleechOnly    bool                 `port:"freeleechOnly" label:"Freeleech only" help:"Search only freeleech Releases."`
	LimitedOnly      bool                 `port:"limitedOnly" label:"Limited only" help:"Search only limited freeleech Releases."`
	RefundOnly       bool                 `port:"refundOnly" label:"Refund only" help:"Search only refund Releases."`
	RewindOnly       bool                 `port:"rewindOnly" label:"Rewind only" help:"Search only rewind Releases."`
	SearchTypes      []beyondHDSearchType `port:"searchTypes" label:"Search types" help:"Choose the types of Release to search. Leave empty to search every type."`
}

type beyondHDSearchType int

var beyondHDSearchTypes = []string{
	"UHD 100", "UHD 66", "UHD 50", "UHD Remux", "BD 50", "BD 25", "BD Remux",
	"2160p", "1080p", "1080i", "720p", "576p", "540p",
	"DVD 9", "DVD 5", "DVD Remux", "480p", "Other",
}

func (beyondHDSearchType) Options() []port.Option {
	opts := make([]port.Option, len(beyondHDSearchTypes))
	for i, label := range beyondHDSearchTypes {
		opts[i] = port.Option{Value: strconv.Itoa(i), Label: label}
	}
	return opts
}

// label is the type's FieldOption label, "" for a value with none.
func (t beyondHDSearchType) label() string {
	if t < 0 || int(t) >= len(beyondHDSearchTypes) {
		return ""
	}
	return beyondHDSearchTypes[t]
}

type beyondHDResponse struct {
	StatusCode    int               `json:"status_code"`
	StatusMessage string            `json:"status_message"`
	Results       []beyondHDTorrent `json:"results"`
}

type beyondHDTorrent struct {
	Name         string             `json:"name"`
	InfoHash     string             `json:"info_hash"`
	Category     string             `json:"category"`
	Type         string             `json:"type"`
	Size         int64              `json:"size"`
	Grabs        int                `json:"times_completed"`
	Seeders      int                `json:"seeders"`
	Leechers     int                `json:"leechers"`
	Audios       string             `json:"audios"`
	Subtitles    string             `json:"subtitles"`
	CreatedAt    string             `json:"created_at"`
	DownloadLink string             `json:"download_url"`
	InfoURL      string             `json:"url"`
	ImdbID       string             `json:"imdb_id"`
	TmdbID       string             `json:"tmdb_id"`
	Freeleech    serializer.STJBool `json:"freeleech"`
	Promo25      serializer.STJBool `json:"promo25"`
	Promo50      serializer.STJBool `json:"promo50"`
	Promo75      serializer.STJBool `json:"promo75"`
	Limited      serializer.STJBool `json:"limited"`
	Exclusive    serializer.STJBool `json:"exclusive"`
	Internal     serializer.STJBool `json:"internal"`
}
