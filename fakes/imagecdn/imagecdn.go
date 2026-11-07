// Package imagecdn is fake-image-cdn: TMDB's image host, answering any
// image file name at any size TMDB offers with a generated image, and
// TVDB's artwork host, answering any artwork path as an original or its _t
// thumbnail.
package imagecdn

import (
	"net/http"
	"path"
	"slices"
	"strconv"
	"strings"

	"github.com/jatassi/slipstream/fakes/fake"
	"github.com/jatassi/slipstream/fakes/internal/images"
	"github.com/jatassi/slipstream/fakes/scenario"
	"github.com/jatassi/slipstream/internal/node"
	"github.com/jatassi/slipstream/internal/node/endpoint"
)

var Fake = fake.Spec{
	Name: "image-cdn",
	TLS:  fake.FrontTLS,
	New:  newServer,
	Endpoints: func(e *node.EndpointsBlock, at fake.Address) {
		e.TMDB.Images = endpoint.MustParseURL(at.URL(Root))
		e.TVDB.Artwork = endpoint.MustParseURL(at.URL(""))
	},
	Online: fake.OnlineSuite("imagecdn"),
}

// Root is the path TMDB's image host serves under.
const Root = "/t/p"

// Sizes are every size TMDB's configuration lists for any image kind.
var Sizes = []string{"w45", "w92", "w154", "w185", "w300", "w342", "w500", "w780", "w1280", "h632", "original"}

func newServer(_ scenario.Scenario, env fake.Env) (fake.Server, error) {
	return fake.HTTP(env, fake.Routes{
		"image":     {Pattern: "GET " + Root + "/{size}/{file}", Handler: serveImage},
		"artwork":   {Pattern: "GET " + ArtworkRoot + "/{path...}", Handler: serveArtwork},
		"not-found": {Pattern: "/", Handler: func(w http.ResponseWriter, _ *http.Request) { notFound(w) }},
	}), nil
}

func serveImage(w http.ResponseWriter, r *http.Request) {
	size, file := r.PathValue("size"), r.PathValue("file")
	width, height, ok := dimensions(size)
	if !ok {
		html(w, http.StatusBadRequest, "<h1>Image size not supported</h1>")
		return
	}
	f, ok := images.FormatOf(file)
	if !ok {
		notFound(w)
		return
	}
	images.Serve(w, f, "/"+file, size, width, height)
}

// dimensions are a size's pixels at a poster's 2:3: wN is N wide, hN is N
// high, and original is the 2000 by 3000 TMDB keeps.
func dimensions(size string) (width, height int, ok bool) {
	if !slices.Contains(Sizes, size) {
		return 0, 0, false
	}
	if size == "original" {
		return 2000, 3000, true
	}
	n, _ := strconv.Atoi(size[1:])
	if size[0] == 'h' {
		return n * 2 / 3, n, true
	}
	return n, n * 3 / 2, true
}

// ArtworkRoot is the path TVDB's artwork host serves under.
const ArtworkRoot = "/banners"

// serveArtwork answers a TVDB artwork path at its kind's shape: a poster
// at 2:3, a clear logo at 2.58:1, anything else at 16:9, and a _t
// thumbnail at a third of the original's width.
func serveArtwork(w http.ResponseWriter, r *http.Request) {
	p := r.PathValue("path")
	f, ok := images.FormatOf(p)
	if !ok {
		notFound(w)
		return
	}
	width, height := 1920, 1080
	switch {
	case strings.Contains(p, "/posters/"):
		width, height = 680, 1000
	case strings.Contains(p, "/clearlogo/"):
		width, height = 800, 310
	}
	size := "original"
	if base := strings.TrimSuffix(p, path.Ext(p)); strings.HasSuffix(base, "_t") {
		size, width, height = "thumbnail", width/3, height/3
	}
	images.Serve(w, f, ArtworkRoot+"/"+p, size, width, height)
}

func notFound(w http.ResponseWriter) { html(w, http.StatusNotFound, "<h1>File Not Found</h1>") }

func html(w http.ResponseWriter, status int, body string) {
	w.Header().Set("Content-Type", "text/html")
	w.WriteHeader(status)
	_, _ = w.Write([]byte(body))
}
