// Ported from Prowlarr's src/NzbDrone.Core/Indexers/Definitions/NorBits.cs
// at ef9c182466ed04f34e6bcfedebb02c3c4af7eaf9. Copyright the Prowlarr
// contributors, GPL-3.0. Modified for SlipStream: rewritten in Go.

package norbits

import (
	_ "embed"
	"regexp"
	"strings"
	"time"

	"github.com/jatassi/slipstream/internal/newznab"
	"github.com/jatassi/slipstream/internal/ports/port"
	"github.com/jatassi/slipstream/internal/prowlarr/datetimeutil"
	"github.com/jatassi/slipstream/internal/prowlarr/dom"
	"github.com/jatassi/slipstream/internal/prowlarr/encoding"
	"github.com/jatassi/slipstream/internal/prowlarr/parseutil"
)

//go:embed upstream.json
var upstream []byte

var Port = port.Define(port.Entry{
	ID:          "norbits",
	Name:        "NorBits",
	IndexerURLs: []string{"https://norbits.net/"},
	// SlipStream: SlipStream's own sentence, per WRITING.md.
	Description:  "NorBits is a private Norwegian Tracker for movies, TV, software, games, music and books.",
	Language:     "nb-NO",
	Privacy:      port.Private,
	Capabilities: SetCapabilities(),
	Upstream:     port.ReadUpstream(upstream),
}, port.Impl[NorBitsSettings]{
	Login: port.CodeLogin[NorBitsSettings]{DoLogin: DoLogin, CheckIfLoginNeeded: CheckIfLoginNeeded},
	Movie: MovieSearchRequests,
	TV:    TvSearchRequests,
	Basic: BasicSearchRequests,
	Parse: ParseResponse,
})

func DoLogin(in *port.SignIn[NorBitsSettings]) error {
	if _, err := in.Do(port.Get(in.BaseURL)); err != nil {
		return err
	}

	loginURL := in.BaseURL.JoinPath("login.php")

	// SlipStream: the sign-in's own jar carries the index page's cookies to
	// each request after it, as SetCookies(indexPage.GetCookies()) does.
	// Get login page -- (not used, but simulation needed by tracker security's checks)
	if _, err := in.Do(port.Get(loginURL)); err != nil {
		return err
	}

	// SlipStream: the code is the authenticator key's or the one typed for a
	// Test, never a stored one. Upstream's TwoFactorAuthCode isn't ported.
	code, hasCode := in.TwoFactorCode()

	authLoginCheckRequest := port.Post(in.BaseURL.JoinPath("takelogin.php")).
		AddFormParameter("username", in.Settings.Username).
		AddFormParameter("password", string(in.Settings.Password)).
		AddFormParameter("code", code).
		AddFormParameter("logout", "no").
		AddFormParameter("returnto", "/").
		SetHeader("Referer", loginURL.String())

	loginResponse, err := in.Do(authLoginCheckRequest)
	if err != nil {
		return err
	}

	if _, ok := loginResponse.Cookies()["uid"]; !ok {
		// SlipStream: an account with two-factor sign-in that signed in with
		// no code stops for one. Upstream's "Login failed" is Prowlarr's
		// words, not NorBits', so no message goes with the refusal.
		if !hasCode && asksForCode(loginResponse) {
			return port.ErrTwoFactorCodeNeeded
		}
		return port.IndexerAuthError("")
	}

	return nil
}

// SlipStream: no NorBits page was recorded, so the words a refused
// sign-in uses to ask for the two-factor code are a guess. A capture of a
// refused sign-in replaces it.
var codeWords = regexp.MustCompile(`(?i)\b2fa\b|tofaktor|two-factor`)

func asksForCode(r *port.IndexerResponse) bool {
	return codeWords.MatchString(dom.Parse(r.Content).QuerySelector("body").TextContent())
}

func CheckIfLoginNeeded(r *port.IndexerResponse) bool {
	return !strings.Contains(r.Content, "logout.php")
}

func SetCapabilities() port.Capabilities {
	// SlipStream: MusicSearchParams and BookSearchParams have no mode in
	// SlipStream. Their searches go through Basic.
	return port.Capabilities{
		TvSearchParams:    []port.Param{port.Q, port.Season, port.Ep},
		MovieSearchParams: []port.Param{port.Q, port.ImdbID},
		Categories: port.Categories(
			port.Map("main_cat[]=1", newznab.Movies, "Filmer"),
			port.Map("main_cat[]=2", newznab.TV, "TV"),
			port.Map("main_cat[]=3", newznab.PC, "Programmer"),
			port.Map("main_cat[]=4", newznab.Console, "Spill"),
			port.Map("main_cat[]=5", newznab.Audio, "Musikk"),
			port.Map("main_cat[]=6", newznab.Books, "Tidsskrift"),
			port.Map("main_cat[]=7", newznab.AudioAudiobook, "Lydbøker"),
			port.Map("main_cat[]=8", newznab.AudioVideo, "Musikkvideoer"),
			port.Map("main_cat[]=40", newznab.AudioOther, "Podcasts"),
		),
	}
}

func GetPagedRequests(ix *port.Indexer[NorBitsSettings], term string, categories []newznab.ID, imdbID string) *port.Request {
	searchURL := ix.URL("browse.php")

	fullsearch := "0"
	if ix.Settings.UseFullSearch {
		fullsearch = "1"
	}
	parameters := [][2]string{
		{"incldead", "1"},
		{"fullsearch", fullsearch},
		{"scenerelease", "0"},
	}

	if ix.Settings.FreeLeechOnly {
		parameters = append(parameters, [2]string{"FL", "1"})
	}

	searchTerm := "search="

	if strings.TrimSpace(imdbID) != "" {
		searchTerm = "imdbsearch=" + imdbID
	} else if strings.TrimSpace(term) != "" {
		searchTerm = "search=" + encoding.UTF8.URLEncode(term)
	}

	searchURL.RawQuery = searchTerm + "&" + getQueryString(parameters)

	if categoriesList := ix.Categories.TrackerIDs(categories); len(categoriesList) > 0 {
		searchURL.RawQuery += "&" + strings.Join(categoriesList, "&")
	}

	return port.Get(searchURL).Accept("text/html")
}

// getQueryString is StringUtil.GetQueryString over a NameValueCollection.
func getQueryString(parameters [][2]string) string {
	pairs := make([]string, len(parameters))
	for i, kv := range parameters {
		pairs[i] = kv[0] + "=" + encoding.UTF8.URLEncode(kv[1])
	}
	return strings.Join(pairs, "&")
}

func MovieSearchRequests(ix *port.Indexer[NorBitsSettings], q port.SearchCriteria) (port.Chain, error) {
	return port.One(GetPagedRequests(ix, q.SanitizedSearchTerm(), q.Categories, q.FullImdbID()))
}

func TvSearchRequests(ix *port.Indexer[NorBitsSettings], q port.SearchCriteria) (port.Chain, error) {
	return port.One(GetPagedRequests(ix, q.SanitizedTvSearchString(), q.Categories, q.FullImdbID()))
}

func BasicSearchRequests(ix *port.Indexer[NorBitsSettings], q port.SearchCriteria) (port.Chain, error) {
	return port.One(GetPagedRequests(ix, q.SanitizedSearchTerm(), q.Categories, ""))
}

// SlipStream: NorBits lists times in Norway's zone. Upstream reads them in
// the zone of Prowlarr's host, which is Norway's for most of its users.
var oslo = mustZone("Europe/Oslo")

func mustZone(name string) *time.Location {
	loc, err := time.LoadLocation(name)
	if err != nil {
		panic(err)
	}
	return loc
}

func ParseResponse(ix *port.Indexer[NorBitsSettings], indexerResponse *port.IndexerResponse) ([]port.TorrentInfo, error) {
	var releaseInfos []port.TorrentInfo

	doc := dom.Parse(indexerResponse.Content)

	rows := doc.QuerySelectorAll("#torrentTable > tbody > tr")
	if len(rows) > 0 {
		rows = rows[1:]
	}

	for _, row := range rows {
		release, err := parseRow(ix, row)
		if err != nil {
			return nil, err
		}
		releaseInfos = append(releaseInfos, release)
	}

	return releaseInfos, nil
}

// SlipStream: parseRow and categoryOf are the C# loop's body, split out
// because gocognit caps a function's complexity at 20.
func parseRow(ix *port.Indexer[NorBitsSettings], row *dom.Node) (port.TorrentInfo, error) {
	link := ix.BaseURL.String() + strings.TrimLeft(row.QuerySelector(`td:nth-of-type(2) > a[href*="download.php?id="]`).GetAttribute("href"), "/")
	qDetails := row.QuerySelector(`td:nth-of-type(2) > a[href*="details.php?id="]`)

	title := strings.TrimSpace(qDetails.GetAttribute("title"))
	details := ix.BaseURL.String() + strings.TrimLeft(qDetails.GetAttribute("href"), "/")

	category := categoryOf(row)

	seeders, err := parseutil.CoerceInt(row.QuerySelector("td:nth-of-type(9)").TextContent())
	if err != nil {
		return port.TorrentInfo{}, err
	}
	leechers, err := parseutil.CoerceInt(row.QuerySelector("td:nth-of-type(10)").TextContent())
	if err != nil {
		return port.TorrentInfo{}, err
	}
	size, err := parseutil.GetBytes(row.QuerySelector("td:nth-of-type(7)").TextContent())
	if err != nil {
		return port.TorrentInfo{}, err
	}
	files, err := parseutil.CoerceInt(strings.TrimSpace(row.QuerySelector("td:nth-of-type(3) > a").TextContent()))
	if err != nil {
		return port.TorrentInfo{}, err
	}
	grabs, err := parseutil.CoerceInt(strings.TrimSpace(row.QuerySelector("td:nth-of-type(8)").FirstChild().TextContent()))
	if err != nil {
		return port.TorrentInfo{}, err
	}
	clock := datetimeutil.Clock{Now: ix.Clock.Now, Location: oslo}
	publishDate, err := datetimeutil.ParseExact(strings.TrimSpace(row.QuerySelector("td:nth-of-type(5)").TextContent()), "yyyy-MM-ddHH:mm:ss", datetimeutil.None, clock)
	if err != nil {
		return port.TorrentInfo{}, err
	}

	release := port.TorrentInfo{
		GUID:                 details,
		InfoURL:              details,
		DownloadURL:          link,
		Title:                title,
		Categories:           ix.Categories.FromTracker(category),
		Size:                 size,
		Files:                files,
		Grabs:                grabs,
		Seeders:              seeders,
		Peers:                seeders + leechers,
		PublishDate:          publishDate,
		DownloadVolumeFactor: 1,
		UploadVolumeFactor:   1,
		MinimumRatio:         1,
		MinimumSeedTime:      172800, // 48 hours
	}

	// SlipStream: the genres in span.genres fill Description and Genres
	// upstream, which a Release doesn't carry.

	if imdbID, ok := parseutil.GetImdbID(lastSegment(row.QuerySelector(`a[href*="imdb.com/title/tt"]`).GetAttribute("href"))); ok {
		release.ImdbID = imdbID
	}

	switch {
	case row.QuerySelector(`img[title="100% freeleech"]`) != nil:
		release.DownloadVolumeFactor = 0
	case row.QuerySelector(`img[title="Halfleech"]`) != nil:
		release.DownloadVolumeFactor = 0.5
	case row.QuerySelector(`img[title="90% Freeleech"]`) != nil:
		release.DownloadVolumeFactor = 0.1
	}

	return release, nil
}

// categoryOf is the main_cat[]= pair of the category link's query.
func categoryOf(row *dom.Node) string {
	href := row.QuerySelector(`td:nth-of-type(1) a[href*="main_cat[]"]`).GetAttribute("href")
	for part := range strings.SplitSeq(href[strings.LastIndex(href, "?")+1:], "&") {
		if part = strings.TrimSpace(part); len(part) >= len("main_cat[]=") && strings.EqualFold(part[:len("main_cat[]=")], "main_cat[]=") {
			return part
		}
	}
	return ""
}

// lastSegment is href.TrimEnd('/').Split('/').LastOrDefault().
func lastSegment(href string) string {
	href = strings.TrimRight(href, "/")
	return href[strings.LastIndex(href, "/")+1:]
}

type NorBitsSettings struct {
	port.UserPass
	// SlipStream: TwoFactorAuthCode isn't ported, because a code stored in
	// the settings fails at the next sign-in. CodeLogin adds the
	// authenticator key.
	UseFullSearch bool `port:"useFullSearch" label:"Use full search" help:"Search with the full search NorBits offers."`
	FreeLeechOnly bool `port:"freeleechOnly" label:"Freeleech only" help:"Search only freeleech torrents."`
}
