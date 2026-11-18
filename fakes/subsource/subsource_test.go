package subsource_test

import (
	"fmt"
	"math/rand/v2"
	"net/http"
	"testing"
	"testing/synctest"
	"time"

	conformance "github.com/jatassi/slipstream/conformance/subsource"
	"github.com/jatassi/slipstream/fakes/fake/faketest"
	"github.com/jatassi/slipstream/fakes/scenario"
	"github.com/jatassi/slipstream/fakes/subsource"
	"github.com/jatassi/slipstream/internal/calls"
	"github.com/jatassi/slipstream/internal/node/endpoint"
	provider "github.com/jatassi/slipstream/internal/providers/subsource"
	"github.com/jatassi/slipstream/internal/subtitling"
)

func inBothTiers(t *testing.T, s scenario.Scenario, test func(t *testing.T, f faketest.Served)) {
	t.Run("loopback", func(t *testing.T) { test(t, faketest.Loopback(t, subsource.Fake, s)) })
	t.Run("in memory", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) { test(t, faketest.InMemory(t, subsource.Fake, s)) })
	})
}

// titled is single-machine with a film and three episodes of a series,
// each with an English subtitle, and the season's pack.
func titled(t *testing.T) scenario.Scenario {
	t.Helper()
	s, err := scenario.Load("single-machine")
	if err != nil {
		t.Fatal(err)
	}
	s.TVDB = &scenario.TVDB{Series: []scenario.TVDBSeries{{ID: 70001, Name: "The Lantern Keepers", Status: "continuing", IMDb: "tt9100001", Seasons: []scenario.TVDBSeason{{Number: 1, Episodes: []scenario.TVDBEpisode{
		{ID: 71001, Number: 1, Name: "The First Light"}, {ID: 71002, Number: 2, Name: "Low Tide"}, {ID: 71003, Number: 3, Name: "Ashore"},
	}}}}}}
	s.SubtitleProviders.Catalog = []scenario.CatalogSubtitle{
		{ID: "film", Provider: "subsource", Work: "The Quiet Orchard", Language: "en", Release: "The.Quiet.Orchard.2021.1080p.WEB-DL-SLIP"},
		{ID: "pack", Provider: "subsource", Work: "The Lantern Keepers", Season: 1, Episodes: []int{1, 2, 3}, Language: "en", Release: "The.Lantern.Keepers.S01.WEBRip"},
	}
	for e := 1; e <= 3; e++ {
		s.SubtitleProviders.Catalog = append(s.SubtitleProviders.Catalog, scenario.CatalogSubtitle{
			ID: fmt.Sprintf("s01e%02d", e), Provider: "subsource", Work: "The Lantern Keepers", Season: 1, Episodes: []int{e}, Language: "en",
			Release: fmt.Sprintf("The.Lantern.Keepers.S01E%02d.1080p.WEB-DL-SLIP", e),
		})
	}
	if err := s.Validate(); err != nil {
		t.Fatal(err)
	}
	return s
}

const wrongKey = "wrong"

func TestSubSourceConforms(t *testing.T) {
	inBothTiers(t, titled(t), func(t *testing.T, f faketest.Served) {
		f.Cue(t, subsource.RevokeKey.Name, map[string]string{"key": wrongKey})
		conformance.Run(t, conformance.Target{
			Endpoints: provider.Endpoints{API: endpoint.MustParseURL(f.Address.URL("")), KeyPage: endpoint.MustParseURL("https://subsource.invalid/profile")},
			Client:    f.Client, Key: "fake-key", Wrong: wrongKey,
			FilmIMDb: "tt9000001", Series: "tt9100001", Season: 1, Episode: 2,
			Lines: lines, Timings: provider.Timings{Interval: time.Millisecond, Burst: 1},
			Cue: func(name string, args any) { f.Cue(t, name, args) },
		})
	})
}

func lines(c *http.Client) *calls.Lines {
	t := subtitling.DefaultTimings().Calls
	t.RetryMin, t.RetryMax = time.Millisecond, time.Millisecond
	return calls.NewLines(c, "SlipStream/test", t, rand.New(rand.NewPCG(1, 2))) //nolint:gosec // jitter needs no secrecy
}
