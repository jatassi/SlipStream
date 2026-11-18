package gestdown_test

import (
	"math/rand/v2"
	"net/http"
	"testing"
	"testing/synctest"
	"time"

	conformance "github.com/jatassi/slipstream/conformance/gestdown"
	"github.com/jatassi/slipstream/fakes/fake/faketest"
	"github.com/jatassi/slipstream/fakes/gestdown"
	"github.com/jatassi/slipstream/fakes/scenario"
	"github.com/jatassi/slipstream/internal/calls"
	"github.com/jatassi/slipstream/internal/node/endpoint"
	provider "github.com/jatassi/slipstream/internal/providers/gestdown"
	"github.com/jatassi/slipstream/internal/subtitling"
)

// titled is single-machine with a series whose first two episodes have
// English subtitles, the second hearing impaired, and whose third is
// unaired.
func titled(t *testing.T) scenario.Scenario {
	t.Helper()
	s, err := scenario.Load("single-machine")
	if err != nil {
		t.Fatal(err)
	}
	s.TVDB = &scenario.TVDB{Series: []scenario.TVDBSeries{{ID: 70001, Name: "The Lantern Keepers", Status: "continuing", Seasons: []scenario.TVDBSeason{{Number: 1, Episodes: []scenario.TVDBEpisode{
		{ID: 71001, Number: 1, Name: "The First Light"}, {ID: 71002, Number: 2, Name: "Low Tide"}, {ID: 71003, Number: 3, Name: "Ashore"},
	}}}}}}
	episode := scenario.CatalogSubtitle{Provider: "gestdown", Work: "The Lantern Keepers", Season: 1, Language: "en"}
	for _, e := range []struct {
		id      string
		number  int
		hi      bool
		unaired bool
	}{{"e01", 1, false, false}, {"e02", 2, true, false}, {"e03", 3, false, true}} {
		c := episode
		c.ID, c.Episodes, c.HI, c.Unaired, c.Release = e.id, []int{e.number}, e.hi, e.unaired, "0TV, LOL"
		s.SubtitleProviders.Catalog = append(s.SubtitleProviders.Catalog, c)
	}
	if err := s.Validate(); err != nil {
		t.Fatal(err)
	}
	return s
}

func TestGestdownConforms(t *testing.T) {
	target := func(f faketest.Served, elapse func(time.Duration)) conformance.Target {
		timings := provider.DefaultTimings()
		timings.Interval, timings.Refresh.Every = time.Millisecond, 10*time.Millisecond
		return conformance.Target{
			Endpoints: provider.Endpoints{API: endpoint.MustParseURL(f.Address.URL(""))}, Client: f.Client,
			Series: "70001", Unknown: "999999999", Season: 1, Episode: 2, Unaired: 3, AirID: "e03",
			Lines: lines, Timings: timings, Elapse: elapse,
			Cue: func(name string, args any) { f.Cue(t, name, args) },
		}
	}
	s := titled(t)
	t.Run("loopback", func(t *testing.T) { conformance.Run(t, target(faketest.Loopback(t, gestdown.Fake, s), nil)) })
	t.Run("in memory", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) { conformance.Run(t, target(faketest.InMemory(t, gestdown.Fake, s), time.Sleep)) })
	})
}

func lines(c *http.Client) *calls.Lines {
	t := subtitling.DefaultTimings().Calls
	t.RetryMin, t.RetryMax = time.Millisecond, time.Millisecond
	return calls.NewLines(c, "SlipStream/test", t, rand.New(rand.NewPCG(1, 2))) //nolint:gosec // jitter needs no secrecy
}
