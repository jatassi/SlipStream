package hdbits

import (
	"net/http"
	"net/url"
	"slices"
	"testing"
	"time"

	"github.com/jatassi/slipstream/internal/ports/port"
	"github.com/jatassi/slipstream/internal/prowlarr/datetimeutil"
)

func withSettings(s hdBitsSettings) *indexer {
	base, _ := url.Parse("https://hdbits.fixture-site.fake.localhost/")
	return &indexer{
		Settings:   s,
		BaseURL:    base,
		Categories: Port.Entry().Capabilities.Categories,
		Clock:      datetimeutil.Clock{Now: time.Now(), Location: time.UTC},
	}
}

const results = `{"status":0,"data":[
	{"id":"1","name":"The Quiet Orchard 2021 1080p","filename":"The.Quiet.Orchard.2021.1080p.torrent","freeleech":"no","type_category":1,"type_medium":3,"added":"2026-06-01T12:00:00+0000"},
	{"id":"2","name":"The Quiet Orchard 2021 720p","filename":"The.Quiet.Orchard.2021.720p.torrent","freeleech":"yes","type_category":1,"type_medium":3,"added":"2026-06-01T12:00:00+0000"}]}`

func titles(t *testing.T, ix *indexer) []string {
	t.Helper()
	infos, err := parseResponse(ix, &port.IndexerResponse{StatusCode: http.StatusOK, Body: []byte(results)})
	if err != nil {
		t.Fatal(err)
	}
	var out []string
	for _, i := range infos {
		out = append(out, i.Title)
	}
	return out
}

func TestFreeleechOnlyKeepsOnlyFreeleechTorrents(t *testing.T) {
	got := titles(t, withSettings(hdBitsSettings{FreeleechOnly: true, UseFilenames: true}))
	if want := []string{"The.Quiet.Orchard.2021.720p"}; !slices.Equal(got, want) {
		t.Errorf("titles = %v, want %v", got, want)
	}
}

func TestWithoutUseFilenamesTheTitleIsTheName(t *testing.T) {
	got := titles(t, withSettings(hdBitsSettings{}))
	if want := []string{"The Quiet Orchard 2021 1080p", "The Quiet Orchard 2021 720p"}; !slices.Equal(got, want) {
		t.Errorf("titles = %v, want %v", got, want)
	}
}
