//go:build online

package omdb_test

import (
	"math/rand/v2"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/jatassi/slipstream/conformance/internal/shape"
	conformance "github.com/jatassi/slipstream/conformance/omdb"
	"github.com/jatassi/slipstream/internal/metadata"
	provider "github.com/jatassi/slipstream/internal/providers/omdb"
)

// fightClub is rated on IMDb, Rotten Tomatoes and Metacritic and has an
// awards line.
const fightClub = "tt0137523"

func TestOMDbConforms(t *testing.T) {
	key := os.Getenv("TEST_ONLINE_OMDB_KEY")
	if key == "" {
		t.Skip("TEST_ONLINE_OMDB_KEY is not set until CI holds an OMDb key: https://github.com/jatassi/SlipStream/issues/310")
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
