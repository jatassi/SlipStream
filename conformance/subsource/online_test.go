//go:build online

package subsource_test

import (
	"math/rand/v2"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/jatassi/slipstream/conformance/internal/shape"
	conformance "github.com/jatassi/slipstream/conformance/subsource"
	"github.com/jatassi/slipstream/internal/calls"
	provider "github.com/jatassi/slipstream/internal/providers/subsource"
	"github.com/jatassi/slipstream/internal/subtitling"
)

const (
	// heat is the recorded run's film.
	heat = "tt0113277"
	// breakingBad's first season lists S01E01 and a season pack.
	breakingBad = "tt0903747"
)

func TestSubSourceConforms(t *testing.T) {
	key := os.Getenv("TEST_ONLINE_SUBSOURCE_KEY")
	if key == "" {
		t.Skip("TEST_ONLINE_SUBSOURCE_KEY is not set until CI holds a SubSource key as the SUBSOURCE_API_KEY repository secret: https://github.com/jatassi/SlipStream/issues/642")
	}
	var bodies []shape.Body
	tgt := conformance.Target{
		Endpoints: provider.DefaultEndpoints(), Client: &http.Client{Timeout: time.Minute}, Key: key, Wrong: "0",
		FilmIMDb: heat, Series: breakingBad, Season: 1, Episode: 1, Lines: lines, Timings: provider.DefaultTimings(),
	}
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

func lines(c *http.Client) *calls.Lines {
	return calls.NewLines(c, "SlipStream/conformance ( github.com/jatassi/SlipStream )", subtitling.DefaultTimings().Calls, rand.New(rand.NewPCG(1, 2))) //nolint:gosec // jitter needs no secrecy
}
