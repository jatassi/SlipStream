// Ported from Prowlarr's src/NzbDrone.Core/Indexers/Definitions/MTeamTp.cs
// at ef9c182466ed04f34e6bcfedebb02c3c4af7eaf9. Copyright the Prowlarr
// contributors, GPL-3.0. Modified for SlipStream: rewritten in Go.

package mteamtp

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/jatassi/slipstream/internal/newznab"
	"github.com/jatassi/slipstream/internal/ports/port"
	"github.com/jatassi/slipstream/internal/prowlarr/datetimeutil"
)

//go:embed upstream.json
var upstream []byte

const userAgent = "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/87.0.4280.88 Safari/537.35"

// Port is M-Team - TP.
//
// SlipStream: the Music and Book search params are gone, because
// SlipStream has neither mode. CheckIfLoginNeeded's false is NoLogin's
// own behavior.
var Port = port.Define(port.Entry{
	ID:          "mteamtp",
	Name:        "M-Team - TP",
	Description: "M-Team TP is a private Chinese Tracker for HD movies, TV and adult video.",
	Language:    "zh-CN",
	Privacy:     port.Private,
	IndexerURLs: []string{
		"https://kp.m-team.cc/",
		"https://tp.m-team.cc/",
		"https://pt.m-team.cc/",
	},
	Hosts:     apiHosts,
	UserAgent: userAgent,
	RateLimit: 5 * time.Second,
	PageSize:  100,
	Capabilities: port.Capabilities{
		TvSearchParams:    []port.Param{port.Q, port.Season, port.Ep, port.ImdbID},
		MovieSearchParams: []port.Param{port.Q, port.ImdbID},
		Categories: port.Categories(
			port.Map(401, newznab.MoviesSD, "Movie(電影)/SD"),
			port.Map(419, newznab.MoviesHD, "Movie(電影)/HD"),
			port.Map(420, newznab.MoviesDVD, "Movie(電影)/DVDiSo"),
			port.Map(421, newznab.MoviesBluRay, "Movie(電影)/Blu-Ray"),
			port.Map(439, newznab.MoviesHD, "Movie(電影)/Remux"),
			port.Map(403, newznab.TVSD, "TV Series(影劇/綜藝)/SD"),
			port.Map(402, newznab.TVHD, "TV Series(影劇/綜藝)/HD"),
			port.Map(438, newznab.TVHD, "TV Series(影劇/綜藝)/BD"),
			port.Map(435, newznab.TVSD, "TV Series(影劇/綜藝)/DVDiSo"),
			port.Map(404, newznab.TVDocumentary, "紀錄教育"),
			port.Map(434, newznab.Audio, "Music(無損)"),
			port.Map(406, newznab.AudioVideo, "MV(演唱)"),
			port.Map(423, newznab.PCGames, "PCGame(PC遊戲)"),
			port.Map(448, newznab.ConsoleOther, "TvGame(TV遊戲)"),
			port.Map(405, newznab.TVAnime, "Anime(動畫)"),
			port.Map(453, newznab.TVAnime, "Anime(動畫)/BluRay"),
			port.Map(407, newznab.TVSport, "Sports(運動)"),
			port.Map(427, newznab.BooksEBook, "Study/Edu ebook(教育書面)"),
			port.Map(422, newznab.PC0day, "Software(軟體)"),
			port.Map(442, newznab.AudioAudiobook, "Study/Edu audio(教育音檔)"),
			port.Map(451, newznab.MoviesOther, "教育影片"),
			port.Map(409, newznab.Other, "Misc(其他)"),

			// adult
			port.Map(410, newznab.XXX, "AV(有碼)/HD Censored"),
			port.Map(424, newznab.XXXSD, "AV(有碼)/SD Censored"),
			port.Map(437, newznab.XXXDVD, "AV(有碼)/DVDiSo Censored"),
			port.Map(431, newznab.XXX, "AV(有碼)/Blu-Ray Censored"),
			port.Map(429, newznab.XXX, "AV(無碼)/HD Uncensored"),
			port.Map(430, newznab.XXXSD, "AV(無碼)/SD Uncensored"),
			port.Map(426, newznab.XXXDVD, "AV(無碼)/DVDiSo Uncensored"),
			port.Map(432, newznab.XXX, "AV(無碼)/Blu-Ray Uncensored"),
			port.Map(436, newznab.XXX, "AV(網站)/0Day"),
			port.Map(440, newznab.XXX, "AV(Gay)/HD"),
			port.Map(425, newznab.XXX, "IV(寫真影集)/Video Collection"),
			port.Map(433, newznab.XXXImageSet, "IV(寫真圖集)/Picture Collection"),
			port.Map(411, newznab.XXX, "H-Game(遊戲)"),
			port.Map(412, newznab.XXX, "H-Anime(動畫)"),
			port.Map(413, newznab.XXX, "H-Comic(漫畫)"),
		),
	},
	Replaces: []string{"mteamtp2fa"},
	Upstream: port.ReadUpstream(upstream),
}, port.Impl[mTeamTpSettings]{
	Login:    port.NoLogin{},
	Movie:    getMovieSearchRequests,
	TV:       getTvSearchRequests,
	Basic:    getBasicSearchRequests,
	Parse:    parseResponse,
	Download: download,
})

type indexer = port.Indexer[mTeamTpSettings]

// SlipStream: the token request goes through ix.Fetch, so it is spaced
// like a search. GetDownloadRequest's User-Agent is Entry.UserAgent.
func download(ix *indexer, link *url.URL) (port.Got, error) {
	res, err := ix.Fetch(port.Post(link).
		SetHeader("x-api-key", ix.Settings.APIKey).
		Accept("application/json"))
	if err != nil {
		return nil, err
	}

	var jsonResponse mTeamTpAPIDownloadTokenResponse
	if json.Unmarshal(res.Body, &jsonResponse) != nil {
		return nil, port.IndexerError("the Indexer responded to the download token request with something other than JSON")
	}

	if strings.TrimSpace(jsonResponse.Data) == "" {
		return nil, port.IndexerError("the Indexer's download token response has no download link")
	}

	torrentURL, err := url.Parse(jsonResponse.Data)
	if err != nil {
		return nil, port.IndexerError("the Indexer's download link %q isn't a URL", jsonResponse.Data)
	}

	return ix.Download(torrentURL)
}

// apiHosts is BuildApiUrl's host: "api." and the address's host after
// its first label.
func apiHosts(base *url.URL) []string {
	return []string{apiURL(base).Host}
}

func apiURL(base *url.URL) *url.URL {
	_, domain, _ := strings.Cut(base.Host, ".")
	return &url.URL{Scheme: "https", Host: "api." + domain, Path: "/"}
}

var trackerAdultCategories = []int{410, 424, 437, 431, 429, 430, 426, 432, 436, 440, 425, 433, 411, 412, 413}

func getMovieSearchRequests(ix *indexer, searchCriteria port.SearchCriteria) (port.Chain, error) {
	var pageableRequests port.Chain

	pageableRequests.Add(getPagedRequests(ix, requestTypeNormal, searchCriteria, searchCriteria.SanitizedSearchTerm(), searchCriteria.FullImdbID())...)

	return pageableRequests, nil
}

// SlipStream: the Music and Book overloads are gone. SlipStream has
// neither mode.
func getTvSearchRequests(ix *indexer, searchCriteria port.SearchCriteria) (port.Chain, error) {
	var pageableRequests port.Chain

	pageableRequests.Add(getPagedRequests(ix, requestTypeNormal, searchCriteria, searchCriteria.SanitizedTvSearchString(), searchCriteria.FullImdbID())...)

	return pageableRequests, nil
}

func getBasicSearchRequests(ix *indexer, searchCriteria port.SearchCriteria) (port.Chain, error) {
	var pageableRequests port.Chain

	pageableRequests.Add(getPagedRequests(ix, requestTypeNormal, searchCriteria, searchCriteria.SanitizedSearchTerm(), "")...)
	pageableRequests.Add(getPagedRequests(ix, requestTypeAdult, searchCriteria, searchCriteria.SanitizedSearchTerm(), "")...)

	return pageableRequests, nil
}

func getPagedRequests(ix *indexer, requestType mTeamTpRequestType, searchCriteria port.SearchCriteria, searchTerm, imdbID string) []*port.Request {
	var categoryMapping []int
	for _, cat := range ix.Categories.TrackerIDs(searchCriteria.Categories) {
		id, _ := strconv.Atoi(cat)
		if !slices.Contains(categoryMapping, id) {
			categoryMapping = append(categoryMapping, id)
		}
	}

	var adultCategories, normalCategories []int
	for _, c := range categoryMapping {
		if slices.Contains(trackerAdultCategories, c) {
			adultCategories = append(adultCategories, c)
		} else {
			normalCategories = append(normalCategories, c)
		}
	}

	switch {
	case requestType == requestTypeAdult && len(adultCategories) > 0:
		return []*port.Request{buildSearchRequest(ix, requestType, adultCategories, searchTerm, imdbID)}
	case requestType == requestTypeNormal && (len(categoryMapping) == 0 || len(normalCategories) > 0):
		return []*port.Request{buildSearchRequest(ix, requestType, normalCategories, searchTerm, imdbID)}
	}
	return nil
}

// SlipStream: the User-Agent is Entry.UserAgent, which the framework
// sends on every request.
func buildSearchRequest(ix *indexer, requestType mTeamTpRequestType, categoryMapping []int, searchTerm, imdbID string) *port.Request {
	pageNumber, pageSize := 1, 100
	query := mTeamTpAPISearchQuery{
		Mode:       requestType,
		Categories: []string{},
		PageNumber: &pageNumber,
		PageSize:   &pageSize,
	}
	for _, c := range categoryMapping {
		query.Categories = append(query.Categories, strconv.Itoa(c))
	}

	if strings.TrimSpace(imdbID) != "" {
		query.Imdb = strings.TrimSpace(imdbID)
	}

	if strings.TrimSpace(searchTerm) != "" {
		query.Keyword = strings.TrimSpace(searchTerm)
	}

	if ix.Settings.FreeleechOnly {
		query.Discount = "FREE"
	}

	return port.Post(apiURL(ix.BaseURL).JoinPath("api/torrent/search")).
		SetHeader("x-api-key", ix.Settings.APIKey).
		Accept("application/json").
		SetJSON(query)
}

// SlipStream: the messages are SlipStream's own, per WRITING.md. A 401 is
// a key the site refused, which SlipStream names account_refused.
func parseResponse(ix *indexer, indexerResponse *port.IndexerResponse) ([]port.TorrentInfo, error) {
	if indexerResponse.StatusCode == http.StatusUnauthorized {
		return nil, port.IndexerAuthError("")
	}

	if indexerResponse.StatusCode != http.StatusOK {
		return nil, port.IndexerError("the Indexer responded with status %d", indexerResponse.StatusCode)
	}

	if contentType := indexerResponse.Headers.Get("Content-Type"); !strings.Contains(contentType, "application/json") {
		return nil, port.IndexerError("the Indexer responded with %q where application/json was expected", contentType)
	}

	var jsonResponse mTeamTpAPIResponse
	if json.Unmarshal(indexerResponse.Body, &jsonResponse) != nil {
		return nil, port.IndexerError("the Indexer responded with something other than JSON")
	}

	var releaseInfos []port.TorrentInfo

	if jsonResponse.Data == nil || jsonResponse.Data.Torrents == nil {
		if strings.TrimSpace(jsonResponse.Message) != "" && !strings.EqualFold(jsonResponse.Message, "SUCCESS") {
			return nil, port.IndexerError("the Indexer's API responded with %q", jsonResponse.Message)
		}

		return releaseInfos, nil
	}

	for _, torrent := range jsonResponse.Data.Torrents {
		var f fields
		torrentID := f.int("id", torrent.ID)
		infoURL := fmt.Sprintf("%s/detail/%d", strings.TrimRight(ix.BaseURL.String(), "/"), torrentID)

		discount := strings.ToUpper(torrent.Status.Discount)

		var downloadVolumeFactor float64
		switch discount {
		case "FREE", "_2X_FREE":
			downloadVolumeFactor = 0
		case "PERCENT_50", "_2X_PERCENT_50":
			downloadVolumeFactor = 0.5
		case "PERCENT_70":
			downloadVolumeFactor = 0.3
		default:
			downloadVolumeFactor = 1
		}

		var uploadVolumeFactor float64
		switch discount {
		case "_2X_FREE", "_2X_PERCENT_50":
			uploadVolumeFactor = 2
		default:
			uploadVolumeFactor = 1
		}

		// SlipStream: TorrentInfo has no Description, because a Release
		// carries none.
		release := port.TorrentInfo{
			GUID:                 infoURL,
			Title:                cleanTitle(torrent.Name),
			InfoURL:              infoURL,
			DownloadURL:          getDownloadURL(ix, torrentID),
			Categories:           ix.Categories.FromTracker(torrent.Category),
			Files:                f.int("numfiles", torrent.NumFiles),
			Size:                 f.long("size", torrent.Size),
			Grabs:                f.int("timesCompleted", torrent.Status.TimesCompleted),
			Seeders:              f.int("seeders", torrent.Status.Seeders),
			Peers:                f.int("seeders", torrent.Status.Seeders) + f.int("leechers", torrent.Status.Leechers),
			DownloadVolumeFactor: downloadVolumeFactor,
			UploadVolumeFactor:   uploadVolumeFactor,
			MinimumRatio:         1,
			MinimumSeedTime:      172800, // 2 days
		}
		if f.err != nil {
			return nil, port.IndexerError("read torrent %q of the Indexer's search response: %v", torrent.ID, f.err)
		}

		if torrent.Status.CreatedDate != nil {
			if publishDate, ok := datetimeutil.TryParseExact(*torrent.Status.CreatedDate+" +08:00", "yyyy-MM-dd HH:mm:ss zzz", datetimeutil.AssumeUniversal, ix.Clock); ok {
				release.PublishDate = publishDate
			}
		}

		releaseInfos = append(releaseInfos, release)
	}

	slices.SortStableFunc(releaseInfos, func(a, b port.TorrentInfo) int { return b.PublishDate.Compare(a.PublishDate) })

	return releaseInfos, nil
}

func getDownloadURL(ix *indexer, torrentID int) string {
	return port.NewURI(apiURL(ix.BaseURL)).
		CombinePath("/api/torrent/genDlToken").
		AddQueryParam("id", strconv.Itoa(torrentID)).
		FullURI()
}

var whitespace = regexp.MustCompile(`\s+`)

func cleanTitle(title string) string {
	title = whitespace.ReplaceAllString(title, " ")

	return strings.TrimSpace(title)
}

// fields is C#'s int.Parse and long.Parse, keeping the first failure.
type fields struct{ err error }

func (f *fields) int(name, s string) int {
	v, err := strconv.Atoi(s)
	f.fail(name, s, err)
	return v
}

func (f *fields) long(name, s string) int64 {
	v, err := strconv.ParseInt(s, 10, 64)
	f.fail(name, s, err)
	return v
}

func (f *fields) fail(name, s string, err error) {
	if err != nil && f.err == nil {
		f.err = fmt.Errorf("%s %q isn't a whole number", name, s)
	}
}

// SlipStream: Prowlarr's validator is the required flag on the API key.
type mTeamTpSettings struct {
	port.APIKeyField `help:"Find your API key on M-Team under User Control Panel > Laboratory."`
	FreeleechOnly    bool `port:"freeleechOnly" label:"Freeleech only" help:"Search only freeleech Releases."`
}

type mTeamTpRequestType string

// SlipStream: the values are what Prowlarr's StringEnumConverter writes.
const (
	requestTypeNormal mTeamTpRequestType = "normal"
	requestTypeAdult  mTeamTpRequestType = "adult"
)

// SlipStream: the json tags are what Prowlarr's ToJson writes, camelCase
// with nulls left out.
type mTeamTpAPISearchQuery struct {
	Mode       mTeamTpRequestType `json:"mode"`
	Categories []string           `json:"categories"`
	Discount   string             `json:"discount,omitempty"`
	Imdb       string             `json:"imdb,omitempty"`
	Keyword    string             `json:"keyword,omitempty"`
	PageNumber *int               `json:"pageNumber,omitempty"`
	PageSize   *int               `json:"pageSize,omitempty"`
}

type mTeamTpAPIResponse struct {
	Data    *mTeamTpAPIData `json:"data"`
	Message string          `json:"message"`
}

type mTeamTpAPIData struct {
	Torrents []mTeamTpAPITorrent `json:"data"`
}

type mTeamTpAPITorrent struct {
	ID          string                  `json:"id"`
	Name        string                  `json:"name"`
	Description string                  `json:"smallDescr"`
	Category    string                  `json:"category"`
	NumFiles    string                  `json:"numfiles"`
	Imdb        string                  `json:"imdb"`
	Size        string                  `json:"size"`
	Status      mTeamTpAPIReleaseStatus `json:"status"`
}

type mTeamTpAPIReleaseStatus struct {
	CreatedDate    *string `json:"createdDate"`
	Discount       string  `json:"discount"`
	TimesCompleted string  `json:"timesCompleted"`
	Seeders        string  `json:"seeders"`
	Leechers       string  `json:"leechers"`
}

type mTeamTpAPIDownloadTokenResponse struct {
	Data string `json:"data"`
}
