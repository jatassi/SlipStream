//go:build online

package tmdb_test

import (
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
	conformance.Configuration(t, provider.DefaultEndpoints(), key, "primary_translations", &translations)
	var countries []struct {
		ISO string `json:"iso_3166_1"`
	}
	conformance.Configuration(t, provider.DefaultEndpoints(), key, "countries", &countries)
	codes := make([]string, len(countries))
	for i, c := range countries {
		codes[i] = c.ISO
	}
	languages := values(metadata.LanguageChoice.Options)
	onTMDBs(t, "language", languages, translations)
	for _, v := range translations {
		if !slices.Contains(languages, v) {
			t.Errorf("TMDB lists language %s, which the language setting does not offer", v)
		}
	}
	// TMDB's countries keep retired ISO codes such as YU and ZR, which the
	// region setting rightly leaves out, so only ours are checked.
	onTMDBs(t, "region", values(metadata.RegionChoice.Options), codes)
}

func values(options []setting.Option[string]) []string {
	out := make([]string, len(options))
	for i, o := range options {
		out[i] = o.Value
	}
	return out
}

func onTMDBs(t *testing.T, what string, ours, tmdbs []string) {
	t.Helper()
	for _, v := range ours {
		if !slices.Contains(tmdbs, v) {
			t.Errorf("the %s option %s is not on TMDB's list", what, v)
		}
	}
}

func lines(c *http.Client) *metadata.Lines {
	return metadata.NewLines(c, "SlipStream/conformance ( github.com/jatassi/SlipStream )", metadata.DefaultTimings(), rand.New(rand.NewPCG(1, 2))) //nolint:gosec // jitter needs no secrecy
}
