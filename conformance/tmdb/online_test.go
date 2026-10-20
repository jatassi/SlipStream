//go:build online

package tmdb_test

import (
	"encoding/json"
	"math/rand/v2"
	"net/http"
	"os"
	"slices"
	"testing"
	"time"

	"github.com/jatassi/slipstream/conformance/internal/shape"
	conformance "github.com/jatassi/slipstream/conformance/tmdb"
	"github.com/jatassi/slipstream/internal/metadata"
	provider "github.com/jatassi/slipstream/internal/providers/tmdb"
	"github.com/jatassi/slipstream/internal/setting"
)

var fightClub = conformance.Known{ID: 550, Title: "Fight Club", IMDbID: "tt0137523", Year: 1999}

func TestTMDBConforms(t *testing.T) {
	key := os.Getenv("TEST_ONLINE_TMDB_KEY")
	if key == "" {
		t.Fatal("TEST_ONLINE_TMDB_KEY is not set")
	}
	var bodies []shape.Body
	tgt := conformance.Target{Endpoints: provider.DefaultEndpoints(), Client: &http.Client{Timeout: time.Minute}, Key: key, Known: fightClub, Lines: lines, Timings: provider.DefaultTimings()}
	if *shape.Update {
		tgt.Record = func(key string, body []byte) { bodies = append(bodies, shape.Body{Key: key, JSON: body}) }
	}
	conformance.Run(t, tgt)
	if *shape.Update && !t.Failed() {
		if err := shape.Write("shape.json", bodies); err != nil {
			t.Fatal(err)
		}
	}
}

func TestTheLanguageAndRegionListsAreTMDBs(t *testing.T) {
	key := os.Getenv("TEST_ONLINE_TMDB_KEY")
	if key == "" {
		t.Fatal("TEST_ONLINE_TMDB_KEY is not set")
	}
	var translations []string
	get(t, key, "primary_translations", &translations)
	var countries []struct {
		ISO string `json:"iso_3166_1"`
	}
	get(t, key, "countries", &countries)
	codes := make([]string, len(countries))
	for i, c := range countries {
		codes[i] = c.ISO
	}
	same(t, "language", values(metadata.LanguageChoice.Options), translations)
	same(t, "region", values(metadata.RegionChoice.Options), codes)
}

func get(t *testing.T, key, list string, v any) {
	t.Helper()
	u := provider.DefaultEndpoints().API.JoinPath("configuration", list)
	u.RawQuery = "api_key=" + key
	req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, u.String(), http.NoBody)
	if err != nil {
		t.Fatal(err)
	}
	res, err := (&http.Client{Timeout: time.Minute}).Do(req)
	if err != nil {
		t.Fatalf("GET /configuration/%s: %v", list, err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("GET /configuration/%s: %s", list, res.Status)
	}
	if err := json.NewDecoder(res.Body).Decode(v); err != nil {
		t.Fatalf("GET /configuration/%s: %v", list, err)
	}
}

func values(options []setting.Option[string]) []string {
	out := make([]string, len(options))
	for i, o := range options {
		out[i] = o.Value
	}
	return out
}

func same(t *testing.T, what string, ours, tmdbs []string) {
	t.Helper()
	for _, v := range ours {
		if !slices.Contains(tmdbs, v) {
			t.Errorf("the %s option %s is not on TMDB's list", what, v)
		}
	}
	for _, v := range tmdbs {
		if !slices.Contains(ours, v) {
			t.Errorf("TMDB lists %s %s, which the %s setting does not offer", what, v, what)
		}
	}
}

func lines(c *http.Client) *metadata.Lines {
	return metadata.NewLines(c, "SlipStream/conformance ( github.com/jatassi/SlipStream )", metadata.DefaultTimings(), rand.New(rand.NewPCG(1, 2))) //nolint:gosec // jitter needs no secrecy
}
