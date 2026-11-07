//go:build online

package tvdb_test

import (
	"math/rand/v2"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/jatassi/slipstream/conformance/internal/shape"
	conformance "github.com/jatassi/slipstream/conformance/tvdb"
	"github.com/jatassi/slipstream/internal/metadata"
	provider "github.com/jatassi/slipstream/internal/providers/tvdb"
)

// simpsons has specials, a DVD order and more than one page of episodes.
var simpsons = conformance.Known{ID: 71663, Name: "The Simpsons"}

func TestTVDBConforms(t *testing.T) {
	key := os.Getenv("TEST_ONLINE_TVDB_KEY")
	if key == "" {
		t.Skip("TEST_ONLINE_TVDB_KEY is not set")
	}
	var bodies []shape.Body
	tgt := conformance.Target{Endpoints: provider.DefaultEndpoints(), Client: &http.Client{Timeout: time.Minute}, Key: key, Refused: "0", Known: simpsons, Lines: lines, Timings: provider.DefaultTimings()}
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
