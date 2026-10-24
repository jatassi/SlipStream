// Package coverart is the music test type's cover-art host, shaped like the
// Cover Art Archive: a release group's front cover as a generated JPEG,
// whole or as a square thumbnail.
package coverart

import (
	"net/http"
	"slices"
	"strconv"
	"strings"

	"github.com/jatassi/slipstream/fakes/fake"
	"github.com/jatassi/slipstream/fakes/internal/images"
	"github.com/jatassi/slipstream/fakes/scenario"
	"github.com/jatassi/slipstream/internal/node"
)

var Fake = fake.Spec{
	Name:        "music-coverart",
	TLS:         fake.FrontTLS,
	ContentType: "music",
	New:         newServer,
	Endpoints: func(e *node.EndpointsBlock, at fake.Address) {
		e.ContentTypes.Set("music", "coverart", at.URL(""))
	},
	Online: fake.NoOnlineRun("the music test type's cover-art host is its own; conformance/imagecdn checks the generator it shares with fake-image-cdn"),
}

// Thumbnails are the edges, in pixels, of the square thumbnails the host
// serves beside the whole front cover.
var Thumbnails = []int{250, 500, 1200}

const original = 1600

func newServer(_ scenario.Scenario, env fake.Env) (fake.Server, error) {
	return fake.HTTP(env, fake.Routes{
		"front":     {Pattern: "GET /release-group/{mbid}/{image}", Handler: serveFront},
		"not-found": {Pattern: "/", Handler: http.NotFound},
	}), nil
}

func serveFront(w http.ResponseWriter, r *http.Request) {
	mbid, image := r.PathValue("mbid"), r.PathValue("image")
	edge, ok := edgeOf(image)
	if !ok {
		http.NotFound(w, r)
		return
	}
	images.Serve(w, images.JPEG, mbid, image, edge, edge)
}

// edgeOf reads front as the whole cover and front-<edge> as a thumbnail.
func edgeOf(image string) (int, bool) {
	if image == "front" {
		return original, true
	}
	edge, found := strings.CutPrefix(image, "front-")
	n, err := strconv.Atoi(edge)
	return n, found && err == nil && slices.Contains(Thumbnails, n)
}
