//go:build online

package musicbrainz_test

import (
	"math/rand/v2"
	"net/http"
	"testing"
	"time"

	"github.com/jatassi/slipstream/conformance/internal/shape"
	conformance "github.com/jatassi/slipstream/conformance/musicbrainz"
	"github.com/jatassi/slipstream/fakes/contenttypes/music"
	"github.com/jatassi/slipstream/internal/metadata"
	"github.com/jatassi/slipstream/internal/node/endpoint"
)

// boardsOfCanada is a pinned public artist with a short discography, which
// keeps a fetch to a few dozen requests at one a second.
var boardsOfCanada = conformance.Known{MBID: "69158f97-4c07-4c4e-baf8-4e4ab1ed666e", Name: "Boards of Canada"}

func TestMusicBrainzConforms(t *testing.T) {
	var bodies []shape.Body
	tgt := conformance.Target{
		Endpoints: music.Endpoints{Metadata: endpoint.MustParseURL("https://musicbrainz.org/ws/2")},
		Client:    &http.Client{Timeout: 5 * time.Minute, Transport: conformance.RetryRateLimited(http.DefaultTransport)},
		Spacing:   time.Second,
		Lines:     lines,
		Known:     boardsOfCanada,
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

func lines(c *http.Client) *metadata.Lines {
	return metadata.NewLines(c, "SlipStream/conformance ( github.com/jatassi/SlipStream )", metadata.DefaultTimings(), rand.New(rand.NewPCG(1, 2))) //nolint:gosec // jitter needs no secrecy
}
