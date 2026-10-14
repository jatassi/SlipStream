//go:build online

package tmdb_test

import (
	"flag"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/jatassi/slipstream/conformance/internal/shape"
	conformance "github.com/jatassi/slipstream/conformance/tmdb"
	provider "github.com/jatassi/slipstream/internal/providers/tmdb"
)

var update = flag.Bool("update", false, "rewrite shape.json from api.themoviedb.org's answers")

// fightClub is a pinned public movie.
var fightClub = conformance.Known{ID: 550, Title: "Fight Club", IMDbID: "tt0137523"}

func TestTMDBConforms(t *testing.T) {
	key := os.Getenv("TEST_ONLINE_TMDB_KEY")
	if key == "" {
		t.Fatal("TEST_ONLINE_TMDB_KEY is not set")
	}
	var bodies []shape.Body
	tgt := conformance.Target{Endpoints: provider.DefaultEndpoints(), Client: &http.Client{Timeout: time.Minute}, Key: key, Known: fightClub}
	if *update {
		tgt.Record = func(key string, body []byte) { bodies = append(bodies, shape.Body{Key: key, JSON: body}) }
	}
	conformance.Run(t, tgt)
	if *update && !t.Failed() {
		if err := shape.Write("shape.json", bodies); err != nil {
			t.Fatal(err)
		}
	}
}
