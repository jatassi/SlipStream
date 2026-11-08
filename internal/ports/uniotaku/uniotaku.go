// Ported from Prowlarr's src/NzbDrone.Core/Indexers/Definitions/Uniotaku.cs
// at ef9c182466ed04f34e6bcfedebb02c3c4af7eaf9. Copyright the Prowlarr
// contributors, GPL-3.0. Modified for SlipStream: rewritten in Go.

package uniotaku

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"fmt"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/jatassi/slipstream/internal/newznab"
	"github.com/jatassi/slipstream/internal/ports/port"
	"github.com/jatassi/slipstream/internal/prowlarr/dom"
	"github.com/jatassi/slipstream/internal/prowlarr/parseutil"
	"github.com/jatassi/slipstream/internal/prowlarr/stringutil"
)

//go:embed upstream.json
var upstream []byte

type ix = port.Indexer[UniotakuSettings]

var Port = port.Define(port.Entry{
	ID:           "uniotaku",
	Name:         "UniOtaku",
	IndexerURLs:  []string{"https://tracker.uniotaku.com/"},
	Description:  "UniOtaku is a semi-private Brazilian Tracker for anime.",
	Language:     "pt-BR",
	Privacy:      port.SemiPrivate,
	Capabilities: SetCapabilities(),
	Upstream:     port.ReadUpstream(upstream),
}, port.Impl[UniotakuSettings]{
	Login:    port.FormLogin[UniotakuSettings]{DoLogin: DoLogin, CheckIfLoginNeeded: CheckIfLoginNeeded},
	Movie:    GetSearchRequests,
	TV:       GetSearchRequests,
	Basic:    GetSearchRequests,
	Parse:    ParseResponse,
	Download: Download,
})

func DoLogin(in *port.SignIn[UniotakuSettings]) error {
	// SlipStream: a sign-in's requests carry only the cookies it earns, so
	// Cookies = null needs no line, and the framework keeps the session it
	// earns in place of UpdateCookies.
	response, err := in.Do(port.Post(in.BaseURL.JoinPath("account-login.php")).
		AddFormParameter("username", in.Settings.Username).
		AddFormParameter("password", string(in.Settings.Password)).
		AddFormParameter("manter", "1").
		SetHeader("Content-Type", "application/x-www-form-urlencoded").
		SetHeader("Referer", in.BaseURL.String()))
	if err != nil {
		return err
	}

	if CheckIfLoginNeeded(response) {
		doc := dom.Parse(response.Content)
		// SlipStream: no fallback message, since the stop sentence names
		// the sign-in fields and Prowlarr's fallback isn't the site's words.
		errorMessage := strings.TrimSpace(doc.QuerySelector(".login-content span.text-red").TextContent())

		return port.IndexerAuthError(errorMessage)
	}

	return nil
}

func CheckIfLoginNeeded(httpResponse *port.IndexerResponse) bool {
	cookies := httpResponse.Cookies()
	_, uid := cookies["uid"]
	_, pass := cookies["pass"]
	return !uid || !pass
}

func Download(ix *ix, link *url.URL) (port.Got, error) {
	response, err := ix.Fetch(port.Get(link).Accept("text/html"))
	if err != nil {
		return nil, err
	}

	downloadLink := strings.TrimSpace(dom.Parse(response.Content).QuerySelector(`a[href^="download.php?id="]`).GetAttribute("href"))

	if downloadLink == "" {
		return nil, port.IndexerError("the details page at %s has no download link", link)
	}

	return ix.Download(ix.URL(downloadLink))
}

func SetCapabilities() port.Capabilities {
	// SlipStream: MusicSearchParams and BookSearchParams have no mode in
	// SlipStream. Their searches go through Basic.
	return port.Capabilities{
		TvSearchParams:    []port.Param{port.Q, port.Season, port.Ep},
		MovieSearchParams: []port.Param{port.Q},
		Categories: port.Categories(
			port.Map(28, newznab.TVAnime, "Anime"),
			port.Map(47, newznab.MoviesOther, "Filme"),
			port.Map(48, newznab.TVAnime, "OVA"),
			port.Map(49, newznab.BooksComics, "Mangá"),
			port.Map(50, newznab.TVOther, "Dorama"),
			port.Map(51, newznab.Audio, "OST"),
			port.Map(52, newznab.TVAnime, "Anime Completo"),
			port.Map(53, newznab.BooksComics, "Mangá Completo"),
			port.Map(54, newznab.TVOther, "Dorama Completo"),
			port.Map(55, newznab.XXX, "Hentai"),
			port.Map(56, newznab.XXXOther, "H Doujinshi"),
			port.Map(57, newznab.TVOther, "Tokusatsu"),
			port.Map(58, newznab.TVOther, "Live Action"),
		),
	}
}

func GetSearchRequests(ix *ix, searchCriteria port.SearchCriteria) (port.Chain, error) {
	return port.One(GetPagedRequests(ix, searchCriteria.SanitizedSearchTerm(), searchCriteria.Categories))
}

// The C#'s [ -._] is the range from space to '.', which takes in every
// ASCII punctuation mark before '.', and '_'.
var termSeparators = regexp.MustCompile(`[ -._]+`) //nolint:gocritic // the range is upstream's

func GetPagedRequests(ix *ix, term string, categories []newznab.ID) *port.Request {
	if strings.TrimSpace(term) != "" {
		term = "%" + strings.TrimSpace(termSeparators.ReplaceAllString(term, "%")) + "%"
	}

	categoryMapping := ix.Categories.TrackerIDs(categories)
	categoria := "0"
	if len(categoryMapping) == 1 {
		categoria = categoryMapping[0]
	}
	status := "0"
	if ix.Settings.FreeleechOnly {
		status = "1"
	}

	var parameters stringutil.NameValueCollection
	parameters.Add("categoria", categoria)
	parameters.Add("grupo", "0")
	parameters.Add("status", status)
	parameters.Add("ordenar", "0")
	parameters.Add("start", "0")
	parameters.Add("length", "100")
	parameters.Add("search[value]", term)
	parameters.Add("search[regex]", "false")

	searchURL := ix.URL("torrents_.php")
	searchURL.RawQuery = parameters.GetQueryString()

	return port.Get(searchURL).Accept("text/html")
}

// value is item.SelectToken(path).Value<T>() for a string or a number.
func value(item []any, i int) string {
	if i >= len(item) || item[i] == nil {
		return ""
	}
	return fmt.Sprint(item[i])
}

// intValue is a nullable int column, 0 when null as Seeders and Grabs
// read a null upstream.
func intValue(item []any, i int) (int, error) {
	v := value(item, i)
	if v == "" {
		return 0, nil
	}
	n, err := parseutil.CoerceInt(v)
	if err != nil {
		return 0, fmt.Errorf("read column %d of a row: %w", i, err)
	}
	return n, nil
}

func ParseResponse(ix *ix, indexerResponse *port.IndexerResponse) ([]port.TorrentInfo, error) {
	var releaseInfos []port.TorrentInfo

	var jsonContent struct {
		Data [][]any `json:"data"`
	}
	decoder := json.NewDecoder(bytes.NewReader(indexerResponse.Body))
	decoder.UseNumber()
	if err := decoder.Decode(&jsonContent); err != nil {
		return nil, fmt.Errorf("read the search response: %w", err)
	}

	publishDate := ix.Clock.Now
	for _, item := range jsonContent.Data {
		detailsDom := dom.Parse(value(item, 0))
		categoryDom := dom.Parse(value(item, 1))
		groupDom := dom.Parse(value(item, 7))

		qTitleLink := detailsDom.QuerySelector(`a[href^="torrents-details.php?id="]`)
		title := strings.TrimSpace(qTitleLink.TextContent())
		infoURL := ix.BaseURL.String() + qTitleLink.GetAttribute("href")

		category := "Anime"
		if alt := categoryDom.QuerySelector("img[alt]"); alt != nil {
			category = strings.TrimSpace(alt.GetAttribute("alt"))
		}

		releaseGroup := strings.TrimSpace(groupDom.QuerySelector(`a[href*="teams-view.php?id="]`).TextContent())
		if releaseGroup != "" {
			title += " [" + releaseGroup + "]"
		}

		seeders, err := intValue(item, 3)
		if err != nil {
			return nil, err
		}
		leechers, err := intValue(item, 4)
		if err != nil {
			return nil, err
		}
		grabs, err := intValue(item, 5)
		if err != nil {
			return nil, err
		}
		size, err := parseutil.GetBytes(value(item, 6))
		if err != nil {
			return nil, err
		}

		publishDate = publishDate.Add(-time.Minute)

		downloadVolumeFactor := 1.0
		switch {
		case detailsDom.QuerySelector(`img[src*="images/free.gif"]`) != nil:
			downloadVolumeFactor = 0
		case detailsDom.QuerySelector(`img[src*="images/silverdownload.gif"]`) != nil:
			downloadVolumeFactor = 0.5
		}

		release := port.TorrentInfo{
			GUID:                 infoURL,
			InfoURL:              infoURL,
			DownloadURL:          infoURL,
			Title:                title,
			Categories:           ix.Categories.FromDesc(category),
			Size:                 size,
			Grabs:                grabs,
			Seeders:              seeders,
			Peers:                seeders + leechers,
			PublishDate:          publishDate,
			DownloadVolumeFactor: downloadVolumeFactor,
			UploadVolumeFactor:   1,
			MinimumRatio:         0.7,
		}

		releaseInfos = append(releaseInfos, release)
	}

	return releaseInfos, nil
}

type UniotakuSettings struct {
	port.UserPass
	// SlipStream: help reworded per WRITING.md.
	FreeleechOnly bool `port:"freeleechOnly" label:"Freeleech only" help:"Search only freeleech Releases."`
}
