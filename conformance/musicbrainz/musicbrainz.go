// Package musicbrainz is the conformance suite for the music test type's
// metadata source: it drives the type against fake-music-meta on every PR
// and against musicbrainz.org in the online job, so a fake that drifts
// from the real service fails.
package musicbrainz

import (
	_ "embed"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"path"
	"slices"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/jatassi/slipstream/conformance/internal/shape"
	"github.com/jatassi/slipstream/fakes/contenttypes/music"
	"github.com/jatassi/slipstream/internal/contenttype"
	"github.com/jatassi/slipstream/internal/works"
)

//go:embed shape.json
var shapeJSON []byte

// userAgent names the application and a contact, as MusicBrainz asks of
// every client.
const userAgent = "SlipStream/conformance ( github.com/jatassi/SlipStream )"

// unknownArtist is an MBID neither MusicBrainz nor any Scenario holds.
const unknownArtist = "ffffffff-0000-4000-8000-000000000000"

// Known is an artist the target holds.
type Known struct {
	MBID string
	Name string
}

type Target struct {
	// Endpoints is the fake's front, or musicbrainz.org.
	Endpoints music.Endpoints
	// Client trusts the run CA for the fake and the system roots online.
	Client *http.Client
	// Spacing is the least time between two requests. MusicBrainz asks for
	// a second; the fake needs none.
	Spacing time.Duration
	Known   Known
	// Record, when set, takes every body in place of the check against
	// shape.json; the online suite's -update sets it to rewrite the file.
	Record func(key string, body []byte)
}

// Run drives the music type through every call SlipStream makes of it and
// asserts what holds on both sides: search finds Known, a fetch of Known
// names it and lists an album with tracks, and a fetch of an unknown MBID
// is works.ErrNotFound. It then checks every recorded body against
// shape.json: keys and types, no values.
func Run(t *testing.T, tgt Target) {
	t.Helper()
	ctx := t.Context()
	rec := shape.NewRecorder(tgt.Client.Transport, route)
	client := *tgt.Client
	client.Transport = &spaced{next: rec, every: tgt.Spacing}
	typ := music.Def.Open(contenttype.Deps{HTTP: &client, UserAgent: userAgent, Log: slog.New(slog.DiscardHandler)}, &tgt.Endpoints)

	found, err := typ.Search(ctx, tgt.Known.Name)
	if err != nil {
		t.Fatalf("Search(%q): %v", tgt.Known.Name, err)
	}
	if !slices.ContainsFunc(found, func(c works.Candidate) bool { return c.Key == works.Key(tgt.Known.MBID) && c.Label == tgt.Known.Name }) {
		t.Errorf("Search(%q) = %+v, want %s %q among them", tgt.Known.Name, found, tgt.Known.MBID, tgt.Known.Name)
	}

	f, err := typ.Fetch(ctx, works.Key(tgt.Known.MBID))
	if err != nil {
		t.Fatalf("Fetch(%s): %v", tgt.Known.MBID, err)
	}
	if f.Label != tgt.Known.Name {
		t.Errorf("Fetch(%s) is named %q, want %q", tgt.Known.MBID, f.Label, tgt.Known.Name)
	}
	if !slices.ContainsFunc(f.Children, func(album works.Entry) bool { return len(album.Children) > 0 }) {
		t.Errorf("Fetch(%s) lists no album with tracks: %+v", tgt.Known.MBID, f.Children)
	}

	_, err = typ.Fetch(ctx, unknownArtist)
	if want := "GET /artist/" + unknownArtist + ": 404 Not Found: the metadata source has no record with that key"; err == nil || err.Error() != want || !errors.Is(err, works.ErrNotFound) {
		t.Errorf("Fetch(%s) = %v, want %s", unknownArtist, err, want)
	}

	shape.Verify(t, rec.Bodies(), shapeJSON, tgt.Record)
}

func route(p string) string {
	if path.Base(path.Dir(p)) == "artist" {
		return "artist"
	}
	switch path.Base(p) {
	case "artist":
		return "search"
	case "release-group":
		return "release-groups"
	case "release":
		return "releases"
	}
	return p
}

// spaced starts each request no sooner than every after the last one ended.
type spaced struct {
	next  http.RoundTripper
	every time.Duration

	mu   sync.Mutex
	last time.Time
}

func (s *spaced) RoundTrip(req *http.Request) (*http.Response, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if wait := time.Until(s.last.Add(s.every)); wait > 0 {
		timer := time.NewTimer(wait)
		defer timer.Stop()
		select {
		case <-req.Context().Done():
			return nil, req.Context().Err() //nolint:wrapcheck // a RoundTripper hands its request's context error on unchanged
		case <-timer.C:
		}
	}
	res, err := s.next.RoundTrip(req)
	s.last = time.Now()
	return res, err //nolint:wrapcheck // a RoundTripper hands its transport's error on unchanged
}

// RetryRateLimited retries the 503 MusicBrainz answers when a client's address
// is over its rate limit, which shared CI runner addresses often are.
func RetryRateLimited(next http.RoundTripper) http.RoundTripper { return rateLimited{next} }

type rateLimited struct{ next http.RoundTripper }

func (r rateLimited) RoundTrip(req *http.Request) (*http.Response, error) {
	wait := 2 * time.Second
	for attempt := 1; ; attempt++ {
		res, err := r.next.RoundTrip(req)
		if err != nil {
			return nil, fmt.Errorf("musicbrainz: %w", err)
		}
		if res.StatusCode != http.StatusServiceUnavailable || attempt == 6 {
			return res, nil
		}
		if s, err := strconv.Atoi(res.Header.Get("Retry-After")); err == nil && s > 0 {
			wait = time.Duration(s) * time.Second
		}
		_ = res.Body.Close()
		select {
		case <-req.Context().Done():
			return nil, fmt.Errorf("musicbrainz: waiting out a 503: %w", req.Context().Err())
		case <-time.After(wait):
		}
		wait *= 2
	}
}
