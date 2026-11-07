// Ported from Prowlarr's src/NzbDrone.Core/Indexers/Definitions/Anidex.cs
// at ef9c182466ed04f34e6bcfedebb02c3c4af7eaf9. Copyright the Prowlarr
// contributors, GPL-3.0. Modified for SlipStream: rewritten in Go.

package anidex

import (
	_ "embed"
	"net/http"
	"slices"
	"strconv"
	"strings"

	"github.com/jatassi/slipstream/internal/newznab"
	"github.com/jatassi/slipstream/internal/ports/port"
	"github.com/jatassi/slipstream/internal/prowlarr/datetimeutil"
	"github.com/jatassi/slipstream/internal/prowlarr/dom"
	"github.com/jatassi/slipstream/internal/prowlarr/parseutil"
	"github.com/jatassi/slipstream/internal/prowlarr/stringutil"
)

//go:embed upstream.json
var upstream []byte

type ix = port.Indexer[AnidexSettings]

var Port = port.Define(port.Entry{
	ID:          "anidex",
	Name:        "Anidex",
	IndexerURLs: []string{"https://anidex.info/"},
	// SlipStream: SlipStream's own sentence, per WRITING.md.
	Description:  "Anidex is a public Tracker for anime, mostly from English fansub groups.",
	Language:     "en-US",
	Privacy:      port.Public,
	Capabilities: SetCapabilities(),
	Upstream:     port.ReadUpstream(upstream),
}, port.Impl[AnidexSettings]{
	Login: port.NoLogin{},
	TV: func(ix *ix, searchCriteria port.SearchCriteria) (port.Chain, error) {
		return port.One(GetPagedRequests(ix, searchCriteria.SanitizedTvSearchString(), searchCriteria.Categories))
	},
	Basic: func(ix *ix, searchCriteria port.SearchCriteria) (port.Chain, error) {
		return port.One(GetPagedRequests(ix, searchCriteria.SanitizedSearchTerm(), searchCriteria.Categories))
	},
	Parse: ParseResponse,
})

func SetCapabilities() port.Capabilities {
	// SlipStream: MusicSearchParams and BookSearchParams have no mode in
	// SlipStream. Their searches go through Basic.
	return port.Capabilities{
		TvSearchParams: []port.Param{port.Q, port.Season, port.Ep},
		Categories: port.Categories(
			port.Map("1", newznab.TVAnime, "Anime - Sub"),
			port.Map("2", newznab.TVAnime, "Anime - Raw"),
			port.Map("3", newznab.TVAnime, "Anime - Dub"),
			port.Map("4", newznab.TVAnime, "LA - Sub"),
			port.Map("5", newznab.TVAnime, "LA - Raw"),
			port.Map("6", newznab.BooksEBook, "Light Novel"),
			port.Map("7", newznab.BooksComics, "Manga - TLed"),
			port.Map("8", newznab.BooksComics, "Manga - Raw"),
			port.Map("9", newznab.AudioMP3, "♫ - Lossy"),
			port.Map("10", newznab.AudioLossless, "♫ - Lossless"),
			port.Map("11", newznab.AudioVideo, "♫ - Video"),
			port.Map("12", newznab.PCGames, "Games"),
			port.Map("13", newznab.PC0day, "Applications"),
			port.Map("14", newznab.XXXImageSet, "Pictures"),
			port.Map("15", newznab.XXX, "Adult Video"),
			port.Map("16", newznab.Other, "Other"),
		),
	}
}

func GetPagedRequests(ix *ix, term string, categories []newznab.ID) *port.Request {
	var parameters stringutil.NameValueCollection
	parameters.Add("page", "search")
	parameters.Add("s", "upload_timestamp")
	parameters.Add("o", "desc")
	parameters.Add("group_id", "0") // All groups
	parameters.Add("q", term)

	if ix.Settings.AuthorisedOnly {
		parameters.Add("a", "1")
	}

	searchURL := ix.URL("")
	searchURL.RawQuery = parameters.GetQueryString()

	queryCats := ix.Categories.TrackerIDs(categories)

	if len(queryCats) > 0 && slices.ContainsFunc(GetTrackerCategories(ix), func(c string) bool { return !slices.Contains(queryCats, c) }) {
		searchURL.RawQuery += "&id=" + strings.Join(queryCats, ",")
	}

	if len(ix.Settings.LanguagesOnly) > 0 {
		languages := make([]string, 0, len(ix.Settings.LanguagesOnly))
		for _, language := range ix.Settings.LanguagesOnly {
			languages = append(languages, strconv.Itoa(int(language)))
		}
		searchURL.RawQuery += "&lang_id=" + strings.Join(languages, ",")
	}

	return port.Get(searchURL).Accept("text/html")
}

// GetTrackerCategories is Categories.GetTrackerCategories: each tracker
// category the mapping names.
func GetTrackerCategories(ix *ix) []string {
	var out []string
	for _, m := range ix.Categories.Mappings() {
		if !slices.Contains(out, m.Tracker) {
			out = append(out, m.Tracker)
		}
	}
	return out
}

func ParseResponse(ix *ix, indexerResponse *port.IndexerResponse) ([]port.TorrentInfo, error) {
	if indexerResponse.StatusCode != http.StatusOK {
		return nil, port.IndexerError("the search responded with status %d", indexerResponse.StatusCode)
	}

	var releaseInfos []port.TorrentInfo

	doc := dom.Parse(indexerResponse.Content)

	rows := doc.QuerySelectorAll("div#content table > tbody > tr")
	for _, row := range rows {
		// SlipStream: the links join the address with one slash, where
		// BaseUrl + "/dl/1" has two.
		downloadURL := ix.URL(row.QuerySelector(`a[href^="/dl/"]`).GetAttribute("href")).String()
		infoURL := ix.URL(row.QuerySelector("td:nth-child(3) a").GetAttribute("href")).String()

		title := strings.TrimSpace(row.QuerySelector("td:nth-child(3) span").GetAttribute("title"))
		language := strings.TrimSpace(row.QuerySelector("td:nth-child(1) img").GetAttribute("title"))

		if language != "" {
			title += " [" + language + "]"
		}

		categoryLink := row.QuerySelector("td:nth-child(1) a").GetAttribute("href")
		cat, err := parseutil.GetArgumentFromQueryString(categoryLink, "id")
		if err != nil {
			return nil, err
		}
		categories := ix.Categories.FromTracker(cat)

		seeders, err := parseutil.CoerceInt(row.QuerySelector("td:nth-child(9)").TextContent())
		if err != nil {
			return nil, err
		}
		leechers, err := parseutil.CoerceInt(strings.TrimSpace(row.QuerySelector("td:nth-child(10)").TextContent()))
		if err != nil {
			return nil, err
		}

		added := strings.TrimSpace(row.QuerySelector("td:nth-child(8)").GetAttribute("title"))

		size, err := parseutil.GetBytes(strings.TrimSpace(row.QuerySelector("td:nth-child(7)").TextContent()))
		if err != nil {
			return nil, err
		}
		grabs, err := parseutil.CoerceInt(row.QuerySelector("td:nth-child(11)").TextContent())
		if err != nil {
			return nil, err
		}
		publishDate, err := datetimeutil.ParseExact(added, "yyyy-MM-dd HH:mm:ss UTC", datetimeutil.AssumeUniversal, ix.Clock)
		if err != nil {
			return nil, err
		}

		release := port.TorrentInfo{
			GUID:                 infoURL,
			InfoURL:              infoURL,
			DownloadURL:          downloadURL,
			MagnetURL:            row.QuerySelector(`a[href^="magnet:?"]`).GetAttribute("href"),
			Title:                title,
			Categories:           categories,
			Seeders:              seeders,
			Peers:                seeders + leechers,
			Size:                 size,
			Grabs:                grabs,
			PublishDate:          publishDate,
			DownloadVolumeFactor: 0,
			UploadVolumeFactor:   1,
		}

		releaseInfos = append(releaseInfos, release)
	}

	return releaseInfos, nil
}

type AnidexSettings struct {
	// SlipStream: labels and help reworded per WRITING.md.
	AuthorisedOnly bool             `port:"authorisedOnly" label:"Authorized only" help:"Search only Releases from authorized uploaders."`
	LanguagesOnly  []AnidexLanguage `port:"languagesOnly" label:"Languages" help:"Search only Releases in these languages. With none selected, Anidex searches every language."`
}

type AnidexLanguage int

// Options are AnidexLanguage's values, 1 to 31.
// SlipStream: an option has no hint, so each label is the C# hint.
func (AnidexLanguage) Options() []port.Option {
	names := []string{
		"English", "Japanese", "Polish", "Serbo-Croatian", "Dutch", "Italian", "Russian", "German",
		"Hungarian", "French", "Finnish", "Vietnamese", "Greek", "Bulgarian", "Spanish (Spain)",
		"Portuguese (Brazil)", "Portuguese (Portugal)", "Swedish", "Arabic", "Danish",
		"Chinese (Simplified)", "Bengali", "Romanian", "Czech", "Mongolian", "Turkish", "Indonesian",
		"Korean", "Spanish (LATAM)", "Persian", "Malaysian",
	}
	options := make([]port.Option, len(names))
	for i, name := range names {
		options[i] = port.Option{Value: strconv.Itoa(i + 1), Label: name}
	}
	return options
}
