// Package tmdb is the conformance suite for TMDB: it drives SlipStream's
// TMDB client against fake-tmdb on every PR and against api.themoviedb.org
// in the online job, so a fake that drifts from the real service fails.
package tmdb

import (
	_ "embed"
	"errors"
	"net/http"
	"path"
	"slices"
	"strconv"
	"testing"

	"github.com/jatassi/slipstream/conformance/internal/shape"
	provider "github.com/jatassi/slipstream/internal/providers/tmdb"
	"github.com/jatassi/slipstream/internal/works"
)

//go:embed shape.json
var shapeJSON []byte

const userAgent = "SlipStream/conformance ( github.com/jatassi/SlipStream )"

// unknownID is an id neither TMDB nor any Scenario holds.
const unknownID = "999999999"

// Known is a movie the target holds.
type Known struct {
	ID     int64
	Title  string
	IMDbID string
}

type Target struct {
	// Endpoints is the fake's front, or the real default.
	Endpoints provider.Endpoints
	// Client trusts the run CA for the fake and the system roots online.
	Client *http.Client
	// Key is sent as api_key. It is "" against the fake, which takes any
	// key or none.
	Key   string
	Known Known
	// Record, when set, takes every body in place of the check against
	// shape.json; the online suite's -update sets it to rewrite the file.
	Record func(key string, body []byte)
}

// Run drives the TMDB client through every call SlipStream makes and
// asserts what holds on both sides: search finds Known, detail carries
// external ids and release dates, and an unknown id is works.ErrNotFound.
// A target with a key also gets a wrong key refused. It then checks every
// recorded body against shape.json: keys and types, no values.
func Run(t *testing.T, tgt Target) {
	t.Helper()
	ctx := t.Context()
	rec := shape.NewRecorder(tgt.Client.Transport, route)
	client := func(key string) *provider.Client {
		c := *tgt.Client
		c.Transport = keyed{next: rec, key: key}
		return provider.New(&c, tgt.Endpoints, userAgent)
	}
	c := client(tgt.Key)

	found, err := c.SearchMovies(ctx, tgt.Known.Title)
	if err != nil {
		t.Fatalf("SearchMovies(%q): %v", tgt.Known.Title, err)
	}
	if !slices.ContainsFunc(found, func(m provider.MovieResult) bool { return m.ID == tgt.Known.ID && m.Title == tgt.Known.Title }) {
		t.Errorf("SearchMovies(%q) = %+v, want %d %q among them", tgt.Known.Title, found, tgt.Known.ID, tgt.Known.Title)
	}

	m, err := c.Movie(ctx, strconv.FormatInt(tgt.Known.ID, 10))
	if err != nil {
		t.Fatalf("Movie(%d): %v", tgt.Known.ID, err)
	}
	if m.ID != tgt.Known.ID || m.Title != tgt.Known.Title || m.ExternalIDs.IMDbID != tgt.Known.IMDbID {
		t.Errorf("Movie(%d) = %d %q %q, want %d %q %q", tgt.Known.ID, m.ID, m.Title, m.ExternalIDs.IMDbID, tgt.Known.ID, tgt.Known.Title, tgt.Known.IMDbID)
	}
	if m.Released() == "" {
		t.Errorf("Movie(%d) lists no theatrical or digital release in %+v", tgt.Known.ID, m.ReleaseDates)
	}

	_, err = c.Movie(ctx, unknownID)
	if want := "GET /movie/" + unknownID + ": 404 Not Found: the metadata source has no record with that key"; err == nil || err.Error() != want || !errors.Is(err, works.ErrNotFound) {
		t.Errorf("Movie(%s) = %v, want %s", unknownID, err, want)
	}

	if tgt.Key != "" {
		_, err := client("0").SearchMovies(ctx, tgt.Known.Title)
		if want := "GET /search/movie: 401 Unauthorized"; err == nil || err.Error() != want {
			t.Errorf("SearchMovies with a wrong key = %v, want %s", err, want)
		}
	}

	shape.Verify(t, rec.Bodies(), shapeJSON, tgt.Record)
}

// route names an answer in shape.json by the call that asked for it.
func route(p string) string {
	switch path.Dir(p) {
	case "/3/search":
		return "search"
	case "/3/movie":
		return "movie"
	}
	return p
}

type keyed struct {
	next http.RoundTripper
	key  string
}

func (k keyed) RoundTrip(req *http.Request) (*http.Response, error) {
	if k.key == "" {
		return k.next.RoundTrip(req) //nolint:wrapcheck // a RoundTripper hands its transport's error on unchanged
	}
	req = req.Clone(req.Context())
	q := req.URL.Query()
	q.Set("api_key", k.key)
	req.URL.RawQuery = q.Encode()
	return k.next.RoundTrip(req) //nolint:wrapcheck // a RoundTripper hands its transport's error on unchanged
}
