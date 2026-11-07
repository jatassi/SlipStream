// Ported from Prowlarr's src/NzbDrone.Core/Indexers/Definitions/SpeedApp/SpeedAppBase.cs
// at ef9c182466ed04f34e6bcfedebb02c3c4af7eaf9. Copyright the Prowlarr
// contributors, GPL-3.0. Modified for SlipStream: rewritten in Go.

// Package speedapp is SpeedAppBase, the family of SpeedApp and RetroFlix:
// a sign-in with email and password to a bearer token, one search API
// and the factors the API states. A site's package declares its Entry
// and the members its C# class overrides, and Define does the rest.
package speedapp

import (
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"strconv"
	"strings"

	"github.com/jatassi/slipstream/internal/ports/port"
	"github.com/jatassi/slipstream/internal/prowlarr/datetimeutil"
	"github.com/jatassi/slipstream/internal/prowlarr/parseutil"
	"github.com/jatassi/slipstream/internal/prowlarr/stringutil"
)

const (
	loginURL = "api/login"
	PageSize = 100
	// minimumSeedTime is 48 hours.
	minimumSeedTime = 172800
)

// Site is what a site's C# class overrides.
type Site struct {
	// MinimumSeedTime is RetroFlix's, in seconds. Zero is the base's 48
	// hours.
	MinimumSeedTime int64
}

// Define is the site's Port: e with the base's PageSize, and the modes
// its Capabilities declare.
func Define(e port.Entry, s Site) port.Port {
	e.PageSize = PageSize
	if s.MinimumSeedTime == 0 {
		s.MinimumSeedTime = minimumSeedTime
	}
	g := speedAppRequestGenerator{pageSize: PageSize}
	impl := port.Impl[Settings]{
		Login:          port.TokenLogin[Settings]{DoLogin: doLogin, CheckIfLoginNeeded: checkIfLoginNeeded},
		Basic:          g.basic,
		Parse:          speedAppParser{minimumSeedTime: s.MinimumSeedTime}.parseResponse,
		FilterReleases: cleanupReleases,
	}
	if e.Capabilities.MovieSearchParams != nil {
		impl.Movie = g.movie
	}
	if e.Capabilities.TvSearchParams != nil {
		impl.TV = g.tv
	}
	return port.Define(e, impl)
}

func cleanupReleases(_ *port.Indexer[Settings], searchCriteria port.SearchCriteria, cleanReleases []port.TorrentInfo) []port.TorrentInfo {
	return port.FilterReleasesByQuery(cleanReleases, searchCriteria)
}

// SlipStream: an empty ApiKey is a session with no token, which the
// framework signs in for before the request.
func checkIfLoginNeeded(httpResponse *port.IndexerResponse) bool {
	return httpResponse.StatusCode == http.StatusUnauthorized
}

// doLogin is DoLogin. The token it returns is the session the framework
// keeps and sends on every request to the Indexer's hosts, which is
// ModifyRequest and GetDownloadRequest.
func doLogin(in *port.SignIn[Settings]) (string, error) {
	request := port.Post(in.BaseURL.JoinPath(loginURL)).
		SetJSON(speedAppAuthenticationRequest{
			Email:    in.Settings.Email,
			Password: string(in.Settings.Password),
		})

	response, err := in.Do(request)
	if err != nil {
		return "", err
	}

	statusCode := response.StatusCode

	// SlipStream: upstream fails on any status outside 2xx. A 401 is the
	// API refusing the email or password, quoted when it says why.
	if statusCode == http.StatusUnauthorized {
		var errorResponse speedAppErrorResponse
		_ = json.Unmarshal(response.Body, &errorResponse)
		return "", port.IndexerAuthError(errorResponse.Message)
	}
	if statusCode < 200 || statusCode > 299 {
		return "", port.IndexerError("the Indexer responded to the sign-in with status %d", statusCode)
	}

	var parsedResponse speedAppAuthenticationResponse
	if err := json.Unmarshal(response.Body, &parsedResponse); err != nil {
		return "", port.IndexerError("the Indexer's sign-in response isn't JSON")
	}

	return parsedResponse.Token, nil
}

type speedAppRequestGenerator struct{ pageSize int }

func (g speedAppRequestGenerator) movie(ix *port.Indexer[Settings], searchCriteria port.SearchCriteria) (port.Chain, error) {
	return g.getSearch(ix, searchCriteria, searchCriteria.FullImdbID(), 0, "")
}

// SlipStream: no music or book search, which the framework doesn't run.

func (g speedAppRequestGenerator) tv(ix *port.Indexer[Settings], searchCriteria port.SearchCriteria) (port.Chain, error) {
	return g.getSearch(ix, searchCriteria, searchCriteria.FullImdbID(), searchCriteria.Season, searchCriteria.Episode)
}

func (g speedAppRequestGenerator) basic(ix *port.Indexer[Settings], searchCriteria port.SearchCriteria) (port.Chain, error) {
	return g.getSearch(ix, searchCriteria, "", 0, "")
}

// SlipStream: a season of 0 and an episode of "" are C#'s nulls.
func (g speedAppRequestGenerator) getSearch(ix *port.Indexer[Settings], searchCriteria port.SearchCriteria, imdbID string, season int, episode string) (port.Chain, error) {
	return port.One(g.getPagedRequests(ix, searchCriteria.SanitizedSearchTerm(), searchCriteria, imdbID, season, episode))
}

// SlipStream: the framework sets the bearer token on every request to
// the Indexer's hosts, so the request doesn't.
func (g speedAppRequestGenerator) getPagedRequests(ix *port.Indexer[Settings], term string, searchCriteria port.SearchCriteria, imdbID string, season int, episode string) *port.Request {
	itemsPerPage := g.pageSize
	if searchCriteria.Limit > 0 {
		itemsPerPage = min(g.pageSize, searchCriteria.Limit)
	}

	var parameters stringutil.NameValueCollection
	parameters.Add("itemsPerPage", strconv.Itoa(itemsPerPage))
	parameters.Add("sort", "torrent.createdAt")
	parameters.Add("direction", "desc")

	if searchCriteria.Limit > 0 && searchCriteria.Offset > 0 {
		page := searchCriteria.Offset/searchCriteria.Limit + 1
		parameters.Set("page", strconv.Itoa(page))
	}

	if strings.TrimSpace(imdbID) != "" {
		parameters.Set("imdbId", imdbID)
	} else {
		parameters.Set("search", term)
	}

	if season != 0 {
		parameters.Set("season", strconv.Itoa(season))
	}

	if episode != "" {
		parameters.Set("episode", episode)
	}

	cats := ix.Categories.TrackerIDs(searchCriteria.Categories)
	for _, cat := range cats {
		parameters.Add("categories[]", cat)
	}

	searchURL := ix.URL("api/torrent?" + parameters.GetQueryStringWith(stringutil.QueryString{DuplicateKeysIfMulti: true}))

	return port.Get(searchURL).Accept("application/json")
}

type speedAppParser struct{ minimumSeedTime int64 }

// SlipStream: the messages are SlipStream's own, per WRITING.md.
func (p speedAppParser) parseResponse(ix *port.Indexer[Settings], indexerResponse *port.IndexerResponse) ([]port.TorrentInfo, error) {
	if indexerResponse.StatusCode != http.StatusOK {
		return nil, port.IndexerError("the Indexer responded with status %d", indexerResponse.StatusCode)
	}

	if contentType := indexerResponse.Headers.Get("Content-Type"); !strings.Contains(contentType, "application/json") {
		return nil, port.IndexerError("the Indexer responded with %q where application/json was expected", contentType)
	}

	var jsonResponse []speedAppTorrent
	if err := json.Unmarshal(indexerResponse.Body, &jsonResponse); err != nil {
		return nil, port.IndexerError("read the Indexer's search response: %v", err)
	}

	torrentInfos := make([]port.TorrentInfo, 0, len(jsonResponse))
	for _, torrent := range jsonResponse {
		imdbID, _ := parseutil.GetImdbID(torrent.ImdbID)

		// SlipStream: Newtonsoft reads created_at as UTC.
		publishDate, err := datetimeutil.Parse(torrent.CreatedAt, datetimeutil.AssumeUniversal|datetimeutil.AdjustToUniversal, ix.Clock)
		if err != nil {
			return nil, fmt.Errorf("read a release's date: %w", err)
		}

		// SlipStream: no Description or PosterUrl, which a Release doesn't
		// carry. InfoHash is null upstream, which is Go's "".
		torrentInfos = append(torrentInfos, port.TorrentInfo{
			GUID:                 torrent.URL,
			Title:                cleanTitle(torrent.Name),
			Size:                 torrent.Size,
			ImdbID:               imdbID,
			DownloadURL:          fmt.Sprintf("%sapi/torrent/%d/download", ix.BaseURL, torrent.ID),
			InfoURL:              torrent.URL,
			Grabs:                torrent.TimesCompleted,
			PublishDate:          publishDate,
			Categories:           ix.Categories.FromTracker(strconv.Itoa(torrent.Category.ID)),
			Seeders:              torrent.Seeders,
			Peers:                torrent.Leechers + torrent.Seeders,
			MinimumRatio:         1,
			MinimumSeedTime:      p.minimumSeedTime,
			DownloadVolumeFactor: torrent.DownloadVolumeFactor,
			UploadVolumeFactor:   torrent.UploadVolumeFactor,
		})
	}

	return torrentInfos, nil
}

var requested = regexp.MustCompile(`(?i)\[REQUEST(ED)?\]`)

func cleanTitle(title string) string {
	title = requested.ReplaceAllString(title, "")

	return strings.Trim(title, " .")
}

// Settings is SpeedAppSettings. Its ApiKey is the token the
// framework keeps as the session, so it isn't a setting.
type Settings struct {
	port.EmailPass
}

type speedAppCategory struct {
	ID int `json:"id"`
}

// speedAppTorrent holds the fields the parser reads.
// SlipStream: SpeedAppUploadedBy, SpeedAppCountry and SpeedAppTag go
// unread, so they aren't ported.
type speedAppTorrent struct {
	DownloadVolumeFactor float64          `json:"download_volume_factor"`
	UploadVolumeFactor   float64          `json:"upload_volume_factor"`
	URL                  string           `json:"url"`
	ID                   int              `json:"id"`
	Name                 string           `json:"name"`
	Category             speedAppCategory `json:"category"`
	Size                 int64            `json:"size"`
	CreatedAt            string           `json:"created_at"`
	TimesCompleted       int              `json:"times_completed"`
	Leechers             int              `json:"leechers"`
	Seeders              int              `json:"seeders"`
	ImdbID               string           `json:"imdb_id"`
}

type speedAppAuthenticationRequest struct {
	Email    string `json:"username"`
	Password string `json:"password"`
}

type speedAppAuthenticationResponse struct {
	Token string `json:"token"`
}

// speedAppErrorResponse is the API's refusal of a sign-in.
// SlipStream: upstream reads no body from a refused sign-in.
type speedAppErrorResponse struct {
	Message string `json:"message"`
}
