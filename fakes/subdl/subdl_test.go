package subdl_test

import (
	"encoding/json"
	"fmt"
	"io"
	"math/rand/v2"
	"net/http"
	"strings"
	"testing"
	"testing/synctest"
	"time"

	conformance "github.com/jatassi/slipstream/conformance/subdl"
	"github.com/jatassi/slipstream/fakes/fake"
	"github.com/jatassi/slipstream/fakes/fake/faketest"
	"github.com/jatassi/slipstream/fakes/fault"
	"github.com/jatassi/slipstream/fakes/scenario"
	"github.com/jatassi/slipstream/fakes/subdl"
	"github.com/jatassi/slipstream/internal/calls"
	"github.com/jatassi/slipstream/internal/node/endpoint"
	provider "github.com/jatassi/slipstream/internal/providers/subdl"
	"github.com/jatassi/slipstream/internal/subtitling"
)

func inBothTiers(t *testing.T, s scenario.Scenario, test func(t *testing.T, f faketest.Served), opts ...faketest.Option) {
	t.Run("loopback", func(t *testing.T) { test(t, faketest.Loopback(t, subdl.Fake, s, opts...)) })
	t.Run("in memory", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) { test(t, faketest.InMemory(t, subdl.Fake, s, opts...)) })
	})
}

// titled is single-machine with a film and a series of three episodes
// in its first season.
func titled(t *testing.T) scenario.Scenario {
	t.Helper()
	s, err := scenario.Load("single-machine")
	if err != nil {
		t.Fatal(err)
	}
	s.TMDB = &scenario.TMDB{Movies: []scenario.TMDBMovie{{ID: 9001, IMDbID: "tt9000001", Title: "The Quiet Orchard", Runtime: 2 * time.Hour}}}
	s.TVDB = &scenario.TVDB{Series: []scenario.TVDBSeries{{ID: 70001, Name: "The Lantern Keepers", Status: "continuing", IMDb: "tt9100001", Seasons: []scenario.TVDBSeason{{Number: 1, Episodes: []scenario.TVDBEpisode{
		{ID: 71001, Number: 1, Name: "The First Light"}, {ID: 71002, Number: 2, Name: "Low Tide"}, {ID: 71003, Number: 3, Name: "Ashore"},
	}}}}}}
	film := scenario.CatalogSubtitle{Provider: "subdl", Work: "The Quiet Orchard", Language: "en", Release: "The.Quiet.Orchard.2026.1080p.WEB-DL-SLIP"}
	series := scenario.CatalogSubtitle{Provider: "subdl", Work: "The Lantern Keepers", Season: 1, Language: "en"}
	add := func(c scenario.CatalogSubtitle, id string, set func(*scenario.CatalogSubtitle)) {
		c.ID = id
		set(&c)
		s.SubtitleProviders.Catalog = append(s.SubtitleProviders.Catalog, c)
	}
	add(film, "film", func(*scenario.CatalogSubtitle) {})
	add(film, "film-ai", func(c *scenario.CatalogSubtitle) { c.AI = true })
	add(film, "film-image", func(c *scenario.CatalogSubtitle) { c.Format = "sup" })
	for e := 1; e <= 3; e++ {
		add(series, fmt.Sprintf("s01e%02d", e), func(c *scenario.CatalogSubtitle) {
			c.Episodes, c.Release = []int{e}, fmt.Sprintf("The.Lantern.Keepers.S01E%02d.1080p.WEB-DL-SLIP", e)
		})
	}
	add(series, "s01", func(c *scenario.CatalogSubtitle) {
		c.Episodes, c.Release = []int{1, 2, 3}, "The.Lantern.Keepers.S01.1080p.WEB-DL-SLIP"
	})
	return s
}

const wrongKey = "wrong"

func TestSubDLConforms(t *testing.T) {
	inBothTiers(t, titled(t), func(t *testing.T, f faketest.Served) {
		f.Cue(t, subdl.RevokeKey.Name, map[string]string{"key": wrongKey})
		timings := provider.DefaultTimings()
		timings.Interval, timings.Busy.Every = time.Millisecond, 10*time.Millisecond
		conformance.Run(t, conformance.Target{
			Endpoints: provider.Endpoints{API: endpoint.MustParseURL(f.Address.URL("")), KeyPage: endpoint.MustParseURL("https://subdl.invalid/panel/api")},
			Client:    f.Client, Key: "fake-key", Wrong: wrongKey,
			FilmIMDb: "tt9000001", FilmTMDB: "9001", Series: "tt9100001", Season: 1, Episode: 2, Packs: true, AI: true,
			Lines:   lines,
			Timings: timings,
			Cue:     func(name string, args any) { f.Cue(t, name, args) },
		})
	})
}

func lines(c *http.Client) *calls.Lines {
	t := subtitling.DefaultTimings().Calls
	t.RetryMin, t.RetryMax = time.Millisecond, time.Millisecond
	return calls.NewLines(c, "SlipStream/test", t, rand.New(rand.NewPCG(1, 2))) //nolint:gosec // jitter needs no secrecy
}

func get(t *testing.T, c *http.Client, url, key string) (status int, body []byte) {
	t.Helper()
	req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, url, http.NoBody)
	if err != nil {
		t.Fatal(err)
	}
	if key != "" {
		req.Header.Set("Authorization", "Bearer "+key)
	}
	res, err := c.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = res.Body.Close() }()
	body, err = io.ReadAll(res.Body)
	if err != nil {
		t.Fatal(err)
	}
	return res.StatusCode, body
}

func TestKeyedURLsNameTheDLHost(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		dl := faketest.InMemory(t, subdl.DL, titled(t))
		f := faketest.InMemory(t, subdl.Fake, titled(t), faketest.WithFronts(fake.Fronts{{Target: fault.Target(subdl.DL.Name), Node: "", Address: dl.Address}}))
		status, body := get(t, f.Client, f.Address.URL("/api/v2/subtitles/search?imdb_id=tt9000001&type=movie&languages=EN"), "fake-key")
		var found struct {
			Subtitles []struct {
				URL string `json:"url"`
			} `json:"subtitles"`
		}
		if err := json.Unmarshal(body, &found); err != nil || status != http.StatusOK || len(found.Subtitles) == 0 {
			t.Fatalf("search = %d %s", status, body)
		}
		u := found.Subtitles[0].URL
		if !strings.Contains(u, dl.Address.AddrPort.String()) || !strings.Contains(u, "api_key=fake-key") {
			t.Errorf("url = %s, want the key on fake-subdl-dl's front %s", u, dl.Address.AddrPort)
		}
		if status, _ := get(t, dl.Client, strings.Replace(u, "https://"+dl.Address.AddrPort.String(), dl.Address.URL(""), 1), ""); status != http.StatusGone {
			t.Errorf("a keyed url answers %d, want 410", status)
		}
	})
}
