package world_test

import (
	"bytes"
	"strings"
	"testing"

	"github.com/jatassi/slipstream/fakes/scenario"
	"github.com/jatassi/slipstream/fakes/world"
)

func TestCatalogTextIsTheSubtitleItsProviderServes(t *testing.T) {
	s, err := scenario.Load("subtitles")
	if err != nil {
		t.Fatal(err)
	}
	film, err := world.CatalogText(s, "amber-speech", 0)
	if err != nil || !strings.HasPrefix(string(film), "1\n") || !strings.Contains(string(film), " --> ") {
		t.Fatalf("amber-speech reads %.40q, %v; want a SubRip file", film, err)
	}
	episode, err := world.CatalogText(s, "lamplighters-s01", 2)
	if err != nil || len(episode) == 0 || bytes.Equal(episode, film) {
		t.Fatalf("lamplighters-s01 episode 2 reads %.40q, %v; want its own SubRip file", episode, err)
	}
	for _, c := range []struct {
		id      string
		episode int
	}{{"amber-speech", 1}, {"lamplighters-s01", 4}, {"nowhere", 0}} {
		if _, err := world.CatalogText(s, c.id, c.episode); err == nil {
			t.Errorf("%s episode %d reads a file, want an error", c.id, c.episode)
		}
	}
}
