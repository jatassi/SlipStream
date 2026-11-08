// Ported from Prowlarr's src/NzbDrone.Core/Indexers/Definitions/GreatPosterWall.cs
// at ef9c182466ed04f34e6bcfedebb02c3c4af7eaf9. Copyright the Prowlarr
// contributors, GPL-3.0. Modified for SlipStream: rewritten in Go.

// Package greatposterwall is GreatPosterWall, on the Gazelle family with
// its own movie search, parser and response types.
package greatposterwall

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"net/http"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/jatassi/slipstream/internal/newznab"
	"github.com/jatassi/slipstream/internal/ports/internal/gazelle"
	"github.com/jatassi/slipstream/internal/ports/port"
	"github.com/jatassi/slipstream/internal/prowlarr/datetimeutil"
	"github.com/jatassi/slipstream/internal/prowlarr/encoding"
	"github.com/jatassi/slipstream/internal/prowlarr/parseutil"
	"github.com/jatassi/slipstream/internal/prowlarr/serializer"
	"github.com/jatassi/slipstream/internal/prowlarr/stringutil"
)

//go:embed upstream.json
var upstream []byte

type ix = port.Indexer[greatPosterWallSettings]

var Port = gazelle.Define(port.Entry{
	ID:           "greatposterwall",
	Name:         "GreatPosterWall",
	IndexerURLs:  []string{"https://greatposterwall.com/"},
	Description:  "GreatPosterWall is a private Chinese Tracker for movies.",
	Privacy:      port.Private,
	Capabilities: setCapabilities(),
	Upstream:     port.ReadUpstream(upstream),
}, gazelle.Site[greatPosterWallSettings]{
	GetMovieSearchRequests:   getSearchRequests,
	GetBasicSearchParameters: getBasicSearchParameters,
	ParseResponse:            parseResponse,
	GetDownloadURL:           getDownloadURL,
	CleanupReleases:          cleanupReleases,
})

func cleanupReleases(_ *ix, searchCriteria port.SearchCriteria, cleanReleases []port.TorrentInfo) []port.TorrentInfo {
	if searchCriteria.IsRssSearch() {
		cleanReleases = cleanReleases[:min(50, len(cleanReleases))]
	}

	return cleanReleases
}

func setCapabilities() port.Capabilities {
	return port.Capabilities{
		MovieSearchParams: []port.Param{port.Q, port.ImdbID},
		Categories: port.Categories(
			port.Map(1, newznab.Movies, "Movies 电影"),
		),
	}
}

// getSearchRequests is GreatPosterWallRequestGenerator's movie
// GetSearchRequests.
func getSearchRequests(ix *ix, searchCriteria port.SearchCriteria) (port.Chain, error) {
	parameters := getBasicSearchParameters(ix, searchCriteria, searchCriteria.SearchTerm)

	if searchCriteria.ImdbID != "" {
		parameters.Set("searchstr", searchCriteria.FullImdbID())
	}

	return gazelle.GetRequest(ix, parameters)
}

// getBasicSearchParameters is GreatPosterWallRequestGenerator's.
func getBasicSearchParameters(ix *ix, searchCriteria port.SearchCriteria, term string) stringutil.NameValueCollection {
	parameters := gazelle.GetBasicSearchParameters(ix, searchCriteria, term)

	if ix.Settings.FreeleechOnly {
		parameters.Set("freetorrent", "1")
	}

	return parameters
}

var hdResolutions = []string{"1080p", "1080i", "720p"}

// parseResponse is GreatPosterWallParser's ParseResponse.
// SlipStream: the messages are SlipStream's own, per WRITING.md, with
// the site's words quoted. CookiesUpdater(null, null) has no Go: a
// logged-out response already brings a sign-in, and a port has no
// logger for the redirect's warning. No PosterUrl, which a Release
// doesn't carry.
func parseResponse(ix *ix, indexerResponse *port.IndexerResponse) ([]port.TorrentInfo, error) {
	var releaseInfos []port.TorrentInfo

	if indexerResponse.StatusCode != http.StatusOK {
		if indexerResponse.HasHTTPRedirect {
			if strings.Contains(strings.ToLower(indexerResponse.RedirectURL), "/login.php") {
				return nil, port.IndexerError("the Indexer redirected the search to its sign-in page. The session may have ended. Check the cookie or the username and password, then test the Indexer.")
			}

			return nil, port.IndexerError("the Indexer redirected the search to %s", indexerResponse.RedirectURL)
		}

		var errorResponse gazelle.GazelleErrorResponse
		_ = json.Unmarshal(indexerResponse.Body, &errorResponse)
		if errorResponse.Error != "" {
			return nil, port.IndexerError("the Indexer responded with status %d and the error %q", indexerResponse.StatusCode, errorResponse.Error)
		}
		return nil, port.IndexerError("the Indexer responded with status %d", indexerResponse.StatusCode)
	}

	if contentType := indexerResponse.Headers.Get("Content-Type"); !strings.Contains(contentType, "application/json") {
		return nil, port.IndexerError("the Indexer responded with %q where application/json was expected", contentType)
	}

	var jsonResponse GreatPosterWallResponse
	if err := json.Unmarshal(indexerResponse.Body, &jsonResponse); err != nil {
		return nil, port.IndexerError("read the Indexer's search response: %v", err)
	}
	if jsonResponse.Status != "success" ||
		strings.TrimSpace(string(jsonResponse.Status)) == "" ||
		jsonResponse.Response == nil {
		return releaseInfos, nil
	}

	for _, result := range jsonResponse.Response.Results {
		for _, torrent := range result.Torrents {
			isFreeLeech := torrent.IsFreeleech || torrent.IsNeutralLeech || torrent.IsPersonalFreeleech

			// skip releases that cannot be used with freeleech tokens when the option is enabled
			if ix.Settings.UseFreeleechToken == gazelle.Required && !torrent.CanUseToken && !isFreeLeech {
				continue
			}

			infoURL := gazelle.GetInfoURL(ix, strconv.Itoa(result.GroupID), torrent.TorrentID)
			// SlipStream: wallTime is C#'s time, the site's text read as a
			// wall time in its +08:00 zone.
			wallTime, err := datetimeutil.Parse(string(torrent.Time), datetimeutil.AssumeUniversal|datetimeutil.AdjustToUniversal, ix.Clock)
			if err != nil {
				return nil, fmt.Errorf("read a release's date: %w", err)
			}

			// SlipStream: Scene is a flag.
			release := port.TorrentInfo{
				GUID:                 infoURL,
				InfoURL:              infoURL,
				DownloadURL:          getDownloadURL(ix, torrent.TorrentID, torrent.CanUseToken && !isFreeLeech),
				Title:                strings.TrimSpace(encoding.HTMLDecode(string(torrent.FileName))),
				PublishDate:          wallTime.Add(-8 * time.Hour),
				Categories:           parseCategories(torrent),
				Size:                 torrent.Size,
				Seeders:              torrent.Seeders,
				Peers:                torrent.Seeders + torrent.Leechers,
				Grabs:                torrent.Snatches,
				Files:                torrent.FileCount,
				DownloadVolumeFactor: factor(isFreeLeech),
				UploadVolumeFactor:   factor(torrent.IsNeutralLeech),
				MinimumRatio:         1,
				MinimumSeedTime:      172800, // 48 hours
			}
			if torrent.Scene {
				release.IndexerFlags = []port.SiteFlag{port.Scene}
			}

			if imdbID, ok := parseutil.GetImdbID(string(result.ImdbID)); ok {
				release.ImdbID = imdbID
			}

			switch torrent.FreeType {
			case "11":
				release.DownloadVolumeFactor = 0.75
			case "12":
				release.DownloadVolumeFactor = 0.5
			case "13":
				release.DownloadVolumeFactor = 0.25
			case "1":
				release.DownloadVolumeFactor = 0
			case "2":
				release.DownloadVolumeFactor = 0
				release.UploadVolumeFactor = 0
			}

			releaseInfos = append(releaseInfos, release)
		}
	}

	slices.SortStableFunc(releaseInfos, func(a, b port.TorrentInfo) int { return b.PublishDate.Compare(a.PublishDate) })
	return releaseInfos, nil
}

// getDownloadURL is GreatPosterWallParser's GetDownloadUrl.
func getDownloadURL(ix *ix, torrentID int, canUseToken bool) string {
	url := port.NewURI(ix.BaseURL).
		CombinePath("/torrents.php").
		AddQueryParam("action", "download").
		AddQueryParam("id", strconv.Itoa(torrentID))

	if t := ix.Settings.UseFreeleechToken; (t == gazelle.Preferred || t == gazelle.Required) && canUseToken {
		url = url.AddQueryParam("usetoken", "1")
	}

	return url.FullURI()
}

func parseCategories(torrent GreatPosterWallTorrent) []newznab.Category {
	cats := []newznab.Category{newznab.Movies}

	switch res := string(torrent.Resolution); {
	case slices.Contains(hdResolutions, res):
		cats = append(cats, newznab.MoviesHD)
	case res == "2160p":
		cats = append(cats, newznab.MoviesUHD)
	default:
		cats = append(cats, newznab.MoviesSD)
	}

	return cats
}

// factor is C#'s free ? 0 : 1.
func factor(free bool) float64 {
	if free {
		return 0
	}
	return 1
}

// SlipStream: Validate is the credentials every Indexer fills, and the
// cookie is optional, as UserPassOrCookieSettings says.
type greatPosterWallSettings struct {
	gazelle.UserPassOrCookieSettings
	FreeleechOnly bool `port:"freeleechOnly" label:"Freeleech only" help:"Search only freeleech Releases."`
}

type GreatPosterWallResponse struct {
	Status   serializer.String                   `json:"status"`
	Response *GreatPosterWallResponseWithResults `json:"response"`
}

type GreatPosterWallResponseWithResults struct {
	CurrentPage int                     `json:"currentPage"`
	Pages       int                     `json:"pages"`
	Results     []GreatPosterWallResult `json:"results"`
}

// GreatPosterWallResult is a group of the browse response.
// SlipStream: MaxSize is any JSON, as C#'s object.
type GreatPosterWallResult struct {
	GroupID       int                      `json:"groupId"`
	GroupName     serializer.String        `json:"groupName"`
	GroupSubName  serializer.String        `json:"groupSubName"`
	Cover         serializer.String        `json:"cover"`
	Tags          []serializer.String      `json:"tags"`
	Bookmarked    bool                     `json:"bookmarked"`
	GroupYear     int                      `json:"groupYear"`
	ReleaseType   serializer.String        `json:"releaseType"`
	GroupTime     serializer.String        `json:"groupTime"`
	MaxSize       json.RawMessage          `json:"maxSize"`
	TotalSnatched int                      `json:"totalSnatched"`
	TotalSeeders  int                      `json:"totalSeeders"`
	TotalLeechers int                      `json:"totalLeechers"`
	ImdbID        serializer.String        `json:"imdbId"`
	ImdbRating    serializer.String        `json:"imdbRating"`
	ImdbVote      serializer.String        `json:"imdbVote"`
	DoubanID      serializer.String        `json:"doubanId"`
	DoubanRating  serializer.String        `json:"doubanRating"`
	DoubanVote    serializer.String        `json:"doubanVote"`
	RtRating      serializer.String        `json:"rtRating"`
	Region        serializer.String        `json:"region"`
	Torrents      []GreatPosterWallTorrent `json:"torrents"`
}

// GreatPosterWallTorrent is a group's torrent.
// SlipStream: Time is the site's text, read as a wall time where the
// parser uses it.
type GreatPosterWallTorrent struct {
	TorrentID           int               `json:"torrentId"`
	EditionID           int               `json:"editionId"`
	RemasterYear        int               `json:"remasterYear"`
	RemasterTitle       serializer.String `json:"remasterTitle"`
	RemasterCustomTitle serializer.String `json:"remasterCustomTitle"`
	Scene               bool              `json:"scene"`
	Jinzhuan            bool              `json:"jinzhuan"`
	FileCount           int               `json:"fileCount"`
	Time                serializer.String `json:"time"`
	Size                int64             `json:"size"`
	Snatches            int               `json:"snatches"`
	Seeders             int               `json:"seeders"`
	Leechers            int               `json:"leechers"`
	IsFreeleech         bool              `json:"isFreeleech"`
	IsNeutralLeech      bool              `json:"isNeutralLeech"`
	FreeType            serializer.String `json:"freeType"`
	IsPersonalFreeleech bool              `json:"isPersonalFreeleech"`
	CanUseToken         bool              `json:"canUseToken"`
	HasSnatched         bool              `json:"hasSnatched"`
	Resolution          serializer.String `json:"resolution"`
	Source              serializer.String `json:"source"`
	Codec               serializer.String `json:"codec"`
	Container           serializer.String `json:"container"`
	Processing          serializer.String `json:"processing"`
	ChineseDubbed       serializer.String `json:"chineseDubbed"`
	SpecialSub          serializer.String `json:"specialSub"`
	Subtitles           serializer.String `json:"subtitles"`
	FileName            serializer.String `json:"fileName"`
	ReleaseGroup        serializer.String `json:"releaseGroup"`
}
