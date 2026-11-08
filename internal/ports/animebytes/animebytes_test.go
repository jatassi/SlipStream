package animebytes

import (
	"net/http"
	"net/url"
	"slices"
	"strconv"
	"testing"
	"time"

	"github.com/jatassi/slipstream/internal/newznab"
	"github.com/jatassi/slipstream/internal/ports/port"
	"github.com/jatassi/slipstream/internal/prowlarr/datetimeutil"
)

func newIndexer(s animeBytesSettings) *indexer {
	base, _ := url.Parse("https://animebytes.fixture-site.fake.localhost/")
	return &indexer{
		Settings:   s,
		BaseURL:    base,
		Categories: Port.Entry().Capabilities.Categories,
		Clock:      datetimeutil.Clock{Now: time.Date(2026, 6, 10, 12, 0, 0, 0, time.UTC), Location: time.UTC},
	}
}

func TestCleanSearchTermDropsTheEpisodeAndTheMovie(t *testing.T) {
	for term, want := range map[string]string{
		"Harbor Lights 03":         "Harbor Lights",
		"Harbor Lights 2x03":       "Harbor Lights",
		"Harbor Lights S02E03":     "Harbor Lights",
		"Harbor Lights 112":        "Harbor Lights",
		"Harbor Lights: The Movie": "Harbor Lights:",
		"Harbor Lights the movie":  "Harbor Lights",
		"Harbor LightsThe Movie":   "Harbor LightsThe Movie",
		"The Quiet Orchard":        "The Quiet Orchard",
		"Harbor Lights Ünd03":      "Harbor Lights Ünd03",
	} {
		if got := cleanSearchTerm(term); got != want {
			t.Errorf("cleanSearchTerm(%q) = %q, want %q", term, got, want)
		}
	}
}

func TestSearchByYearSendsTheTrailingYear(t *testing.T) {
	ix := newIndexer(animeBytesSettings{SearchByYear: true})
	q := getRequest(ix, port.SearchCriteria{SearchTerm: "The Quiet Orchard 2021"}, "anime", false).URL.Query()
	if q.Get("year") != "2021" || q.Get("searchstr") != "The Quiet Orchard" {
		t.Errorf("query = %v, want year=2021 and searchstr=The Quiet Orchard", q)
	}
}

func TestASeasonSearchLeavesOutTheSpecials(t *testing.T) {
	ix := newIndexer(animeBytesSettings{FreeleechOnly: true, ExcludeHentai: true})
	q := getRequest(ix, port.SearchCriteria{SearchTerm: "Harbor Lights", Season: new(2), Categories: []newznab.ID{newznab.TVAnime.ID}}, "anime", true).URL.Query()
	for _, cat := range []string{"anime[tv_series]", "anime[ona]"} {
		if q.Get(cat) != "1" {
			t.Errorf("query = %v, want %s=1", q, cat)
		}
	}
	for _, cat := range []string{"anime[tv_special]", "anime[ova]", "anime[dvd_special]", "anime[bd_special]"} {
		if q.Has(cat) {
			t.Errorf("query = %v, want no %s", q, cat)
		}
	}
	if q.Get("freeleech") != "1" || q.Get("hentai") != "0" {
		t.Errorf("query = %v, want freeleech=1 and hentai=0", q)
	}
}

func TestParseSeasonFromTitles(t *testing.T) {
	for title, want := range map[string]int{
		"Harbor Lights 2nd Season": 2,
		"Harbor Lights Season 3":   3,
		"Harbor Lights III":        3,
		"Harbor Lights 4":          4,
		"Harbor Lights S5":         5,
		"Harbor Lights Part 2":     0,
		"Harbor Lights No. 2":      0,
		"Harbor Lights 1/2":        0,
		"Harbor Lights #2":         0,
		"Harbor Lights2":           0,
		"Harbor Lights 1":          0,
		"Harbor Lights Season 3rd": 0,
	} {
		got := 0
		if s := parseSeasonFromTitles([]string{title}); s != nil {
			got = *s
		}
		if got != want {
			t.Errorf("parseSeasonFromTitles(%q) = %d, want %d", title, got, want)
		}
	}
}

const episodeBody = `{"Matches":1,"Groups":[{"CategoryName":"Anime","FullName":"Harbor Lights","GroupName":"TV Series","SeriesName":"Harbor Lights","Year":"2024",
	"SynonymnsV2":{"Japanese":"","Romaji":"","Alternative":""},"Torrents":[
	{"ID":1,"EditionData":{"EditionTitle":"Episode 7"},"RawDownMultiplier":1,"RawUpMultiplier":1,"Link":"https://animebytes.fixture-site.fake.localhost/torrent/1/download/k","Property":"Web | MKV | h264 | 1080p","Size":1,"FileList":[{"filename":"a.mkv"},{"filename":"b.mkv"}],"UploadTime":"2026-06-01 12:00:00"},
	{"ID":2,"EditionData":{"EditionTitle":""},"RawDownMultiplier":0.5,"RawUpMultiplier":1,"Link":"https://animebytes.fixture-site.fake.localhost/torrent/2/download/k","Property":"Web | MKV | h264 | 720p","Size":1,"FileList":[{"filename":"a.mkv"},{"filename":"b.mkv"}],"UploadTime":"2026-06-02 12:00:00"}]}]}`

func parse(t *testing.T, ix *indexer, body string) []string {
	t.Helper()
	res := &port.IndexerResponse{StatusCode: http.StatusOK, Headers: http.Header{"Content-Type": {"application/json; charset=utf-8"}}, Body: []byte(body)}
	infos, err := parseResponse(ix, res)
	if err != nil {
		t.Fatal(err)
	}
	var titles []string
	for _, i := range infos {
		titles = append(titles, i.Title)
	}
	return titles
}

func TestAnEpisodeWithNoSeasonIsAnAbsoluteNumber(t *testing.T) {
	got := parse(t, newIndexer(animeBytesSettings{EnableSonarrCompatibility: true}), episodeBody)
	want := []string{"Harbor Lights S01 [Web][MKV][h264][720p]", "Harbor Lights - 07 [Web][MKV][h264][1080p]"}
	if !slices.Equal(got, want) {
		t.Errorf("titles = %q, want %q", got, want)
	}
}

func TestFreeleechOnlyKeepsOnlyFreeleechTorrents(t *testing.T) {
	got := parse(t, newIndexer(animeBytesSettings{FreeleechOnly: true}), episodeBody)
	if len(got) != 0 {
		t.Errorf("titles = %q, want none", got)
	}
}

func TestARecentFeedKeepsTheLastDayOrTheFirstTwenty(t *testing.T) {
	ix := newIndexer(animeBytesSettings{})
	var releases []port.TorrentInfo
	for i := range 25 {
		releases = append(releases, port.TorrentInfo{Title: strconv.Itoa(i), PublishDate: ix.Clock.Now.Add(-time.Duration(i) * time.Hour * 2)})
	}
	if got := cleanupReleases(ix, port.SearchCriteria{}, releases); len(got) != 20 {
		t.Errorf("kept %d releases, want 20", len(got))
	}
	if got := cleanupReleases(ix, port.SearchCriteria{SearchTerm: "Harbor Lights"}, releases); len(got) != 25 {
		t.Errorf("kept %d releases of a text search, want 25", len(got))
	}
}
