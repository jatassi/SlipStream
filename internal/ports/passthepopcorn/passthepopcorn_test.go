package passthepopcorn

import (
	"net/http"
	"net/url"
	"slices"
	"testing"
	"time"

	"github.com/jatassi/slipstream/internal/ports/port"
	"github.com/jatassi/slipstream/internal/prowlarr/datetimeutil"
)

func freeleechOnly() *indexer {
	base, _ := url.Parse("https://passthepopcorn.fixture-site.fake.localhost/")
	return &indexer{
		Settings:   passThePopcornSettings{FreeleechOnly: true, GoldenPopcornOnly: true},
		BaseURL:    base,
		Categories: Port.Entry().Capabilities.Categories,
		Clock:      datetimeutil.Clock{Now: time.Now(), Location: time.UTC},
	}
}

func TestFreeleechAndGoldenPopcornOnlyNarrowTheSearch(t *testing.T) {
	q := getRequest(freeleechOnly(), "", port.SearchCriteria{}).URL.Query()
	if q.Get("freetorrent") != "1" || q.Get("scene") != "2" {
		t.Errorf("query = %v, want freetorrent=1 and scene=2", q)
	}
}

func TestFreeleechOnlyKeepsOnlyFreeleechTorrents(t *testing.T) {
	body := `{"TotalResults":"1","Movies":[{"GroupId":"1","CategoryId":"1","Year":"2021","Torrents":[
		{"Id":1,"Size":"1","UploadTime":"2026-06-01 12:00:00","Snatched":"0","Seeders":"1","Leechers":"0","ReleaseName":"The.Quiet.Orchard.2021.1080p","FreeleechType":"Half Leech"},
		{"Id":2,"Size":"1","UploadTime":"2026-06-01 12:00:00","Snatched":"0","Seeders":"1","Leechers":"0","ReleaseName":"The.Quiet.Orchard.2021.720p","FreeleechType":"Neutral Leech"},
		{"Id":3,"Size":"1","UploadTime":"2026-06-01 12:00:00","Snatched":"0","Seeders":"1","Leechers":"0","ReleaseName":"The.Quiet.Orchard.2021.480p"}]}]}`
	res := &port.IndexerResponse{StatusCode: http.StatusOK, Headers: http.Header{"Content-Type": {"application/json"}}, Body: []byte(body)}
	infos, err := parseResponse(freeleechOnly(), res)
	if err != nil {
		t.Fatal(err)
	}
	var titles []string
	for _, i := range infos {
		titles = append(titles, i.Title)
	}
	if want := []string{"The.Quiet.Orchard.2021.720p"}; !slices.Equal(titles, want) {
		t.Errorf("titles = %v, want %v", titles, want)
	}
}
