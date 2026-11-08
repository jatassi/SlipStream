// Ported from Prowlarr's src/NzbDrone.Core/Indexers/Definitions/Anidub.cs
// at ef9c182466ed04f34e6bcfedebb02c3c4af7eaf9. Copyright the Prowlarr
// contributors, GPL-3.0. Modified for SlipStream: rewritten in Go.

package anidub

import (
	_ "embed"
	"fmt"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/jatassi/slipstream/internal/newznab"
	"github.com/jatassi/slipstream/internal/ports/port"
	"github.com/jatassi/slipstream/internal/prowlarr/datetimeutil"
	"github.com/jatassi/slipstream/internal/prowlarr/dom"
	"github.com/jatassi/slipstream/internal/prowlarr/parseutil"
)

//go:embed upstream.json
var upstream []byte

type settings struct{ port.UserPass }

type ix = port.Indexer[settings]

var Port = port.Define(port.Entry{
	ID:          "anidub",
	Name:        "Anidub",
	IndexerURLs: []string{"https://tr.anidub.com/"},
	// SlipStream: SlipStream's own sentence, per WRITING.md.
	Description:  "Anidub is a semi-private Russian Tracker for anime, run by the Anidub voiceover group.",
	Language:     "ru-RU",
	Privacy:      port.SemiPrivate,
	Capabilities: SetCapabilities(),
	Upstream:     port.ReadUpstream(upstream),
}, port.Impl[settings]{
	Login: port.FormLogin[settings]{DoLogin: DoLogin, CheckIfLoginNeeded: CheckIfLoginNeeded},
	Movie: GetSearchRequests,
	TV:    GetTvSearchRequests,
	Basic: GetSearchRequests,
	Parse: ParseResponse,
})

func DoLogin(in *port.SignIn[settings]) error {
	// SlipStream: a sign-in's requests carry only the cookies it earns, so
	// UpdateCookies(null, null) needs no line, and the main page's cookies
	// reach the form through the sign-in's own jar.
	if _, err := in.Do(port.Get(in.BaseURL)); err != nil {
		return err
	}

	response, err := in.Do(port.Post(in.BaseURL.JoinPath("index.php")).
		AddFormParameter("login_name", in.Settings.Username).
		AddFormParameter("login_password", string(in.Settings.Password)).
		AddFormParameter("login", "submit").
		SetHeader("Content-Type", "application/x-www-form-urlencoded"))
	if err != nil {
		return err
	}

	if response.Content != "" && !CheckIfLoginNeeded(response) {
		return nil
	}

	document := dom.Parse(response.Content)
	// SlipStream: no fallback message, since the stop sentence names the
	// sign-in fields and Prowlarr's fallback isn't the site's words.
	errorMessage := strings.TrimSpace(document.QuerySelector("#content .berror .berror_c").TextContent())

	return port.IndexerAuthError(errorMessage)
}

func CheckIfLoginNeeded(httpResponse *port.IndexerResponse) bool {
	return !strings.Contains(httpResponse.Content, "index.php?action=logout")
}

func SetCapabilities() port.Capabilities {
	// SlipStream: BookSearchParams and MusicSearchParams have no mode in
	// SlipStream. Their searches go through Basic.
	return port.Capabilities{
		TvSearchParams:    []port.Param{port.Q, port.Season, port.Ep},
		MovieSearchParams: []port.Param{port.Q},
		Categories: port.Categories(
			port.Map(2, newznab.TVAnime, "Аниме TV"),
			port.Map(3, newznab.Movies, "Аниме Фильмы"),
			port.Map(4, newznab.TVAnime, "Аниме OVA"),
			port.Map(5, newznab.TVAnime, "Аниме OVA |- Аниме ONA"),
			port.Map(6, newznab.TV, "Дорамы / Японские Сериалы и Фильмы"),
			port.Map(7, newznab.TV, "Дорамы / Корейские Сериалы и Фильмы"),
			port.Map(8, newznab.TV, "Дорамы / Китайские Сериалы и Фильмы"),
			port.Map(9, newznab.TV, "Дорамы"),
			port.Map(10, newznab.TVAnime, "Аниме TV / Аниме Ongoing"),
			port.Map(11, newznab.TVAnime, "Аниме TV / Многосерийный сёнэн"),
			port.Map(12, newznab.Other, "Аниме Ongoing Анонсы"),
			port.Map(13, newznab.XXX, "18+"),
			port.Map(14, newznab.TVAnime, "Аниме TV / Законченные сериалы"),
			port.Map(15, newznab.BooksComics, "Манга"),
			port.Map(16, newznab.Audio, "OST"),
			port.Map(17, newznab.Audio, "Подкасты"),
		),
	}
}

// Remove season and episode info from search term cause it breaks search
var seasonEpisodeSuffix = regexp.MustCompile(`(?:[SsEe]?\d{1,4}){1,2}$`)

func GetPagedRequests(ix *ix, term string) *port.Request {
	isSearch := strings.TrimSpace(term) != ""

	if !isSearch {
		return port.Get(ix.BaseURL).Accept("text/html")
	}

	// SlipStream: the form is the same pairs SetContent(GetQueryString())
	// sends, in order.
	return port.Post(ix.URL("index.php?do=search")).Accept("text/html").
		AddFormParameter("do", "search").
		AddFormParameter("subaction", "search").
		AddFormParameter("search_start", "1").
		AddFormParameter("full_search", "1").
		AddFormParameter("result_from", "1").
		AddFormParameter("story", strings.TrimRight(seasonEpisodeSuffix.ReplaceAllString(term, ""), " \t\r\n")).
		AddFormParameter("titleonly", "3").
		AddFormParameter("searchuser", "").
		AddFormParameter("replyless", "0").
		AddFormParameter("replylimit", "0").
		AddFormParameter("searchdate", "0").
		AddFormParameter("beforeafter", "after").
		AddFormParameter("sortby", "").
		AddFormParameter("resorder", "desc").
		AddFormParameter("showposts", "1").
		AddFormParameter("catlist[]", "0").
		SetHeader("Content-Type", "application/x-www-form-urlencoded")
}

func GetSearchRequests(ix *ix, searchCriteria port.SearchCriteria) (port.Chain, error) {
	return port.One(GetPagedRequests(ix, searchCriteria.SanitizedSearchTerm()))
}

func GetTvSearchRequests(ix *ix, searchCriteria port.SearchCriteria) (port.Chain, error) {
	return port.One(GetPagedRequests(ix, searchCriteria.SanitizedTvSearchString()))
}

var CategoriesMap = []struct{ path, id string }{
	{"/anime_tv/full", "14"},
	{"/anime_tv/anime_ongoing", "10"},
	{"/anime_tv/shonen", "11"},
	{"/anime_tv", "2"},
	{"/xxx", "13"},
	{"/manga", "15"},
	{"/ost", "16"},
	{"/podcast", "17"},
	{"/anime_movie", "3"},
	{"/anime_ova/anime_ona", "5"},
	{"/anime_ova", "4"},
	{"/dorama/japan_dorama", "6"},
	{"/dorama/korea_dorama", "7"},
	{"/dorama/china_dorama", "8"},
	{"/dorama", "9"},
	{"/anons_ongoing", "12"},
}

func GetTitle(content, tabNode *dom.Node) string {
	domTitle := content.QuerySelector("#news-title")
	baseTitle := strings.TrimSpace(domTitle.TextContent())
	quality := GetQuality(tabNode.ParentElement())

	if strings.TrimSpace(quality) != "" {
		return baseTitle + " [" + quality + "]"
	}

	return baseTitle
}

func GetQuality(releaseNode *dom.Node) string {
	// For some releases there's no block with quality
	if strings.TrimSpace(releaseNode.GetAttribute("id")) == "" {
		return ""
	}

	quality := strings.TrimSpace(releaseNode.GetAttribute("id"))
	switch strings.ToLower(quality) {
	case "tv720":
		return "HDTV 720p"
	case "tv1080":
		return "HDTV 1080p"
	case "bd720":
		return "BDRip 720p"
	case "bd1080":
		return "BDRip 1080p"
	case "hwp":
		return "SDTV"
	default:
		return strings.ToUpper(quality)
	}
}

// tryParseInt is int.TryParse: 0 when s isn't an integer.
func tryParseInt(s string) int {
	n, err := strconv.Atoi(strings.TrimSpace(s))
	if err != nil {
		return 0
	}
	return n
}

func GetReleaseLeechers(tabNode *dom.Node) int {
	const leechersSelector = ".list.down > .li_swing_m"

	return tryParseInt(tabNode.QuerySelector(leechersSelector).TextContent())
}

func GetReleaseSeeders(tabNode *dom.Node) int {
	const seedersSelector = ".list.down > .li_distribute_m"

	return tryParseInt(tabNode.QuerySelector(seedersSelector).TextContent())
}

func GetReleaseGrabs(tabNode *dom.Node) int {
	const grabsSelector = ".list.down > .li_download_m"

	return tryParseInt(tabNode.QuerySelector(grabsSelector).TextContent())
}

func GetDateFromDocument(content *dom.Node) string {
	const dateSelector = ".story_inf > li:nth-child(2)"

	domDate := content.QuerySelector(dateSelector).LastChild()

	if domDate.NodeName() != "#text" {
		return ""
	}

	return strings.TrimSpace(domDate.NodeValue())
}

func GetDateFromShowPage(ix *ix, content *dom.Node) time.Time {
	const dateFormat = "d-MM-yyyy"
	const dateTimeFormat = dateFormat + ", HH:mm"

	dateText := GetDateFromDocument(content)

	// Russian Standard Time is +03:00, no DST
	const russianStandardTimeDiff = 3 * time.Hour
	nowLocal := ix.Clock.Now.UTC().Add(russianStandardTimeDiff)

	dateText = strings.ReplaceAll(dateText, "Вчера", nowLocal.AddDate(0, 0, -1).Format("2-01-2006"))
	dateText = strings.ReplaceAll(dateText, "Сегодня", nowLocal.Format("2-01-2006"))

	if date, ok := datetimeutil.TryParseExact(dateText, dateTimeFormat, datetimeutil.AssumeUniversal, ix.Clock); ok {
		return date.UTC().Add(-russianStandardTimeDiff)
	}

	// SlipStream: upstream logs a warning here, and the Node keeps no
	// port log, so the time falls back to the job's start silently.
	return ix.Clock.Now.UTC()
}

func GetReleaseSize(tabNode *dom.Node) (int64, error) {
	const sizeSelector = ".list.down > .red"

	return parseutil.GetBytes(tabNode.QuerySelector(sizeSelector).TextContent())
}

func GetReleaseLink(ix *ix, tabNode *dom.Node) string {
	return ix.BaseURL.String() + "engine/download.php?id=" + GetTorrentID(tabNode)
}

func GetTorrentID(tabNode *dom.Node) string {
	nodeID := tabNode.GetAttribute("id")

	// Format is "torrent_{id}_info"
	return strings.ReplaceAll(strings.ReplaceAll(nodeID, "torrent_", ""), "_info", "")
}

func ParseCategories(ix *ix, uriPath string) []newznab.Category {
	for _, categoryMap := range CategoriesMap {
		if strings.HasPrefix(uriPath, categoryMap.path) {
			return ix.Categories.FromTracker(categoryMap.id)
		}
	}

	return nil
}

func ParseRelease(ix *ix, indexerResponse *port.IndexerResponse) ([]port.TorrentInfo, error) {
	var torrentInfos []port.TorrentInfo
	doc := dom.Parse(indexerResponse.Content)
	fullURI := indexerResponse.Request.URL.String()

	for _, t := range doc.QuerySelectorAll("#tabs .torrent_c > div") {
		size, err := GetReleaseSize(t)
		if err != nil {
			return nil, err
		}

		// SlipStream: a Release has no resolution of its own, so
		// GetQuality(t) isn't kept. The title carries the quality.
		release := port.TorrentInfo{
			Title:                GetTitle(doc, t),
			InfoURL:              fullURI,
			DownloadVolumeFactor: 0,
			UploadVolumeFactor:   1,

			GUID:        fullURI + t.GetAttribute("id"),
			Seeders:     GetReleaseSeeders(t),
			Peers:       GetReleaseSeeders(t) + GetReleaseLeechers(t),
			Grabs:       GetReleaseGrabs(t),
			Categories:  ParseCategories(ix, indexerResponse.Request.URL.Path),
			PublishDate: GetDateFromShowPage(ix, doc),
			DownloadURL: GetReleaseLink(ix, t),
			Size:        size,
		}
		torrentInfos = append(torrentInfos, release)
	}

	return torrentInfos, nil
}

func ParseResponse(ix *ix, indexerResponse *port.IndexerResponse) ([]port.TorrentInfo, error) {
	var torrentInfos []port.TorrentInfo

	doc := dom.Parse(indexerResponse.Content)

	links := doc.QuerySelectorAll(".searchitem > h3 > a[href], #dle-content > .story > .story_h > .lcol > h2 > a[href]")
	for _, link := range links {
		href := link.GetAttribute("href")

		// SlipStream: the link resolves against the address, which leaves
		// the site's absolute links as they are.
		releaseURL, err := ix.BaseURL.Parse(href)
		if err != nil {
			return nil, fmt.Errorf("read the release page link %q: %w", href, err)
		}

		// SlipStream: Fetch spaces the request and reads a 429 or 5xx
		// as limited or failing, as WithRateLimit and the
		// TooManyRequestsException upstream do.
		releaseResponse, err := ix.Fetch(port.Get(releaseURL).
			SetHeader("Referer", ix.BaseURL.String()).
			Accept("text/html"))
		if err != nil {
			return nil, err
		}

		if releaseResponse.StatusCode >= http.StatusBadRequest {
			return nil, port.IndexerError("the site responded with status %d to %s", releaseResponse.StatusCode, href)
		}

		releases, err := ParseRelease(ix, releaseResponse)
		if err != nil {
			return nil, err
		}
		torrentInfos = append(torrentInfos, releases...)
	}

	return torrentInfos, nil
}
