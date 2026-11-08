package bakabt

import (
	"net/url"
	"os"
	"slices"
	"testing"
	"time"

	"github.com/jatassi/slipstream/internal/ports/port"
	"github.com/jatassi/slipstream/internal/prowlarr/datetimeutil"
)

func TestRomajiTitlesAndAppendedSeasonsAddReleases(t *testing.T) {
	body, err := os.ReadFile("../../../fakes/site/ports/bakabt/bodies/search.html")
	if err != nil {
		t.Fatal(err)
	}
	base, _ := url.Parse("https://bakabt.example/")
	indexer := &ix{
		BaseURL:    base,
		Categories: setCapabilities().Categories,
		Settings:   BakaBTSettings{AddRomajiTitle: true, AppendSeason: true},
		Clock:      datetimeutil.Clock{Now: time.Date(2026, 6, 4, 0, 0, 0, 0, time.UTC), Location: time.UTC},
	}
	releases, err := parseResponse(indexer, &port.IndexerResponse{Content: string(body)})
	if err != nil {
		t.Fatal(err)
	}
	var titles []string
	for _, r := range releases {
		titles = append(titles, r.Title)
	}
	want := []string{
		"Minato no Akari Season 1 (2019) [BD 1080p]",
		"Harbor Lights  Season 1 (2019) [BD 1080p]",
		"Harbor Lights (2019) Season 2 [WEB 720p]",
	}
	if !slices.Equal(titles, want) {
		t.Errorf("titles = %q, want %q", titles, want)
	}
}
