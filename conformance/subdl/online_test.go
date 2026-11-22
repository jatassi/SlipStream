//go:build online

package subdl_test

import (
	"math/rand/v2"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/jatassi/slipstream/conformance/internal/shape"
	conformance "github.com/jatassi/slipstream/conformance/subdl"
	"github.com/jatassi/slipstream/internal/calls"
	provider "github.com/jatassi/slipstream/internal/providers/subdl"
	"github.com/jatassi/slipstream/internal/subtitling"
)

const (
	// heat is the recorded run's film: tt0113277, TMDB 949.
	heatIMDb, heatTMDB = "tt0113277", "949"
	// breakingBad has English subtitles for S01E01.
	breakingBad = "tt0903747"
)

func TestSubDLConforms(t *testing.T) {
	key := os.Getenv("TEST_ONLINE_SUBDL_KEY")
	if key == "" {
		t.Skip("TEST_ONLINE_SUBDL_KEY is not set until CI holds a SubDL key as the SUBDL_API_KEY repository secret: https://github.com/jatassi/SlipStream/issues/642")
	}
	var bodies []shape.Body
	tgt := conformance.Target{
		Endpoints: provider.DefaultEndpoints(), Client: &http.Client{Timeout: time.Minute}, Key: key, Wrong: "0",
		FilmIMDb: heatIMDb, FilmTMDB: heatTMDB, Series: breakingBad, Season: 1, Episode: 1, Lines: lines, Timings: provider.DefaultTimings(),
	}
	if shape.Update {
		tgt.Record = func(key string, body []byte) { bodies = append(bodies, shape.Body{Key: key, JSON: body}) }
	}
	conformance.Run(t, tgt)
	if shape.Update && !t.Failed() {
		if err := shape.Write("shape.json", bodies); err != nil {
			t.Fatal(err)
		}
	}
}

func lines(c *http.Client) *calls.Lines {
	return calls.NewLines(c, "SlipStream/conformance ( github.com/jatassi/SlipStream )", subtitling.DefaultTimings().Calls, rand.New(rand.NewPCG(1, 2))) //nolint:gosec // jitter needs no secrecy
}
