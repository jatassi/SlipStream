//go:build online

package gestdown_test

import (
	"math/rand/v2"
	"net/http"
	"testing"
	"time"

	conformance "github.com/jatassi/slipstream/conformance/gestdown"
	"github.com/jatassi/slipstream/conformance/internal/shape"
	"github.com/jatassi/slipstream/internal/calls"
	provider "github.com/jatassi/slipstream/internal/providers/gestdown"
	"github.com/jatassi/slipstream/internal/subtitling"
)

const (
	// breakingBad is the recorded run's series, TVDB 81189, with English
	// subtitles for S01E01.
	breakingBad = "81189"
	// unknown is the recorded run's TVDB id Gestdown doesn't know.
	unknown = "999999999"
)

func TestGestdownConforms(t *testing.T) {
	var bodies []shape.Body
	tgt := conformance.Target{
		Endpoints: provider.DefaultEndpoints(), Client: &http.Client{Timeout: time.Minute},
		Series: breakingBad, Unknown: unknown, Season: 1, Episode: 1, Lines: lines, Timings: provider.DefaultTimings(),
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
