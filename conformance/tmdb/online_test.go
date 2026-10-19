//go:build online

package tmdb_test

import (
	"math/rand/v2"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/jatassi/slipstream/conformance/internal/shape"
	conformance "github.com/jatassi/slipstream/conformance/tmdb"
	"github.com/jatassi/slipstream/internal/metadata"
	provider "github.com/jatassi/slipstream/internal/providers/tmdb"
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

func lines(c *http.Client) *metadata.Lines {
	return metadata.NewLines(c, "SlipStream/conformance ( github.com/jatassi/SlipStream )", metadata.DefaultTimings(), rand.New(rand.NewPCG(1, 2))) //nolint:gosec // jitter needs no secrecy
}
