// Package subsource is the conformance suite for SubSource: it drives
// SlipStream's SubSource client against fake-subsource on every PR and
// against api.subsource.net in the online job, so a fake that drifts from
// the real service fails. It follows the recorded run of 2026-11-17
// (subtitles/online/run-2026-11-17.md).
package subsource

import (
	"bytes"
	_ "embed"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/jatassi/slipstream/conformance/internal/shape"
	"github.com/jatassi/slipstream/internal/calls"
	"github.com/jatassi/slipstream/internal/language"
	provider "github.com/jatassi/slipstream/internal/providers/subsource"
	"github.com/jatassi/slipstream/internal/subtitling"
)

//go:embed shape.json
var shapeJSON []byte

type Target struct {
	// Endpoints is the fake's front, or the real default.
	Endpoints provider.Endpoints
	// Client trusts the run CA for the fake and the system roots online.
	Client *http.Client
	// Key is a key the target takes, and Wrong one it refuses.
	Key, Wrong string
	// FilmIMDb is a film's IMDb id with English subtitles.
	FilmIMDb string
	// Series is a series' IMDb id whose Season and Episode have English
	// subtitles, a season pack among them.
	Series          string
	Season, Episode int
	// Lines builds a call layer over a client, as a Node does.
	Lines   func(*http.Client) *calls.Lines
	Timings provider.Timings
	// Cue plays one of the fake's cues; nil online, where none play.
	Cue func(name string, args any)
	// Record, when set, takes every body in place of the check against
	// shape.json; the online suite's -update sets it to rewrite the file.
	Record func(key string, body []byte)
}

// Run drives the SubSource client through every call SlipStream makes and
// asserts what holds on both sides: the key goes as X-API-Key and never
// in a URL; a missing and a wrong key are each a 401 refusal quoting its
// message, with no rate headers; every 200 carries the minute, hour and
// day windows and one ISO 8601 reset; the key check reports the day's
// requests, exact, resetting at UTC midnight; a film is found by IMDb id
// and an episode's season by IMDb id and season, whose list keeps the
// episode's subtitles and the season pack; and a download is a named zip.
// No request leaves for another host. Against the fake it then spends a
// window by cue.
func Run(t *testing.T, tgt Target) {
	t.Helper()
	ctx := t.Context()
	next := tgt.Client.Transport
	if next == nil {
		next = http.DefaultTransport
	}
	w := &wire{next: next, api: tgt.Endpoints.API.JoinPath().Host}
	rec := shape.NewRecorder(w, route)
	client := func(rt http.RoundTripper) *provider.Client {
		c := *tgt.Client
		c.Transport = rt
		return provider.New(tgt.Lines(calls.NoRedirects(&c)), tgt.Endpoints, tgt.Timings)
	}
	c := client(rec)
	with := func(key string) calls.Credentials {
		return calls.Credentials{Keys: map[calls.ID]string{provider.ID: key}, Tally: &calls.Tally{}}
	}

	cr := with(tgt.Key)
	if _, err := c.CheckKey(ctx, cr); err != nil {
		t.Fatalf("CheckKey: %v", err)
	}
	day(t, cr)
	for _, row := range []struct{ name, key, words string }{
		{"a missing key", "", "provide an API key in the X-API-Key header"},
		{"a wrong key", tgt.Wrong, "The provided API key is invalid or inactive"},
	} {
		_, err := client(rec).CheckKey(ctx, with(row.key))
		if f, ok := errors.AsType[*calls.Failed](err); !ok || f.Outcome.Kind != calls.Refused || !strings.HasPrefix(f.Outcome.Words, "refused the key: ") || !strings.Contains(f.Outcome.Words, row.words) {
			t.Errorf("CheckKey with %s = %v, want refused with %q", row.name, err, row.words)
		}
	}

	en, _ := language.Parse("en")
	film := subtitling.Query{Subject: subtitling.Subject{Kind: subtitling.Feature, IMDb: tgt.FilmIMDb}, Language: en}
	films := search(t, c, cr, film)
	episode := subtitling.Query{Subject: subtitling.Subject{Kind: subtitling.Episode, IMDb: tgt.Series, Season: tgt.Season, Episodes: []int{tgt.Episode}}, Language: en}
	packs := 0
	for _, o := range search(t, c, cr, episode) {
		switch {
		case o.Pack:
			packs++
		case len(o.Episodes) != 1 || o.Episodes[0] != tgt.Episode || o.Season != tgt.Season:
			t.Errorf("episode %d's offer %s names season %d episodes %v", tgt.Episode, o.Key, o.Season, o.Episodes)
		}
	}
	if packs == 0 {
		t.Errorf("episode %d's season list holds no pack", tgt.Episode)
	}

	dl, err := c.Download(ctx, cr, films[0])
	if err != nil {
		t.Fatalf("Download(%s): %v", films[0].Key, err)
	}
	if !bytes.HasPrefix(dl.Bytes, []byte("PK\x03\x04")) || !strings.HasSuffix(dl.Name, ".zip") {
		t.Errorf("Download(%s) = %q, %d bytes starting %q, want a named zip", films[0].Key, dl.Name, len(dl.Bytes), dl.Bytes[:min(len(dl.Bytes), 4)])
	}

	var bodies []shape.Body
	for _, b := range rec.Bodies() {
		if !strings.HasPrefix(b.Key, "download 200") {
			bodies = append(bodies, b)
		}
	}
	shape.Verify(t, bodies, shapeJSON, tgt.Record)

	if tgt.Cue != nil {
		cues(t, tgt, func() *provider.Client { return client(w) }, with, film)
	}
	w.check(t)
}

// day checks the call's OK outcome reports the day's requests, which
// searches and downloads share, counted exactly to the next UTC midnight.
func day(t *testing.T, cr calls.Credentials) {
	t.Helper()
	os := cr.Tally.Outcomes()
	last := os[len(os)-1]
	a := last.Allowance
	if last.Kind != calls.OK || a == nil || a.Searches == nil || a.Searches != a.Downloads {
		t.Fatalf("CheckKey outcome = %+v, want OK with one day window for both", last)
	}
	midnight := time.Now().UTC().Truncate(24 * time.Hour).Add(24 * time.Hour)
	if w := a.Searches; w.Limit <= 0 || w.Remaining >= w.Limit || !w.Exact || !w.Resets.Equal(midnight) {
		t.Errorf("day window = %+v, want a counted limit, exact, resetting at %v", w, midnight)
	}
}

func route(path string) string {
	switch {
	case strings.HasSuffix(path, "/movies/search"):
		return "titles"
	case strings.HasSuffix(path, "/download"):
		return "download"
	}
	return "subtitles"
}

func search(t *testing.T, c *provider.Client, cr calls.Credentials, q subtitling.Query) []subtitling.Offer {
	t.Helper()
	offers, err := c.Search(t.Context(), cr, q)
	if err != nil {
		t.Fatalf("Search(%+v): %v", q.Subject, err)
	}
	if len(offers) == 0 {
		t.Fatalf("Search(%+v) found nothing", q.Subject)
	}
	for _, o := range offers {
		if o.Key == "" || o.Language.String() != "en" || len(o.Releases) == 0 || o.Page == "" {
			t.Errorf("Search(%+v) offers %+v", q.Subject, o)
		}
	}
	return offers
}

// cues spends the minute, then the day, each on a fresh call layer, since
// a pause closes a Line's gate for the key.
func cues(t *testing.T, tgt Target, client func() *provider.Client, with func(string) calls.Credentials, film subtitling.Query) {
	t.Helper()
	ctx := t.Context()
	for _, row := range []struct {
		window string
		kind   calls.OutcomeKind
		until  func(time.Time) bool
	}{
		{"minute", calls.Limited, func(until time.Time) bool {
			return until.After(time.Now()) && !until.After(time.Now().Add(time.Minute))
		}},
		{"day", calls.Capped, func(until time.Time) bool {
			return until.Equal(time.Now().UTC().Truncate(24 * time.Hour).Add(24 * time.Hour))
		}},
	} {
		tgt.Cue("subsource-window", map[string]string{"window": row.window})
		_, err := client().Search(ctx, with(tgt.Key), film)
		f, ok := errors.AsType[*calls.Failed](err)
		if !ok || f.Outcome.Kind != row.kind || !row.until(f.Outcome.Until) {
			t.Errorf("a search with the %s spent = %v, want kind %d until the %s resets", row.window, err, row.kind, row.window)
		}
	}
}

// wire sees every request the client sends and every answer's headers.
type wire struct {
	next http.RoundTripper
	api  string

	mu       sync.Mutex
	problems []string
}

func (w *wire) RoundTrip(req *http.Request) (*http.Response, error) {
	var problems []string
	if req.URL.Host != w.api {
		problems = append(problems, "a request left for "+req.URL.Redacted())
	}
	if req.URL.Query().Has("api_key") {
		problems = append(problems, "the key rode in "+req.URL.Path+"'s query")
	}
	res, err := w.next.RoundTrip(req)
	if err != nil {
		return nil, fmt.Errorf("send %s: %w", req.URL.Redacted(), err)
	}
	rated := res.Header.Get("X-Ratelimit-Remaining-Day") != ""
	switch {
	case res.StatusCode == http.StatusOK && (!rated || res.Header.Get("X-Ratelimit-Remaining") == "" || res.Header.Get("X-Ratelimit-Remaining-Hour") == ""):
		problems = append(problems, req.URL.Path+" 200 lacks a window")
	case res.StatusCode == http.StatusUnauthorized && rated:
		problems = append(problems, req.URL.Path+" 401 carries windows, which none did as recorded")
	}
	if res.StatusCode == http.StatusOK {
		if _, err := time.Parse(time.RFC3339, res.Header.Get("X-Ratelimit-Reset")); err != nil {
			problems = append(problems, req.URL.Path+" 200 has no ISO 8601 X-Ratelimit-Reset")
		}
	}
	w.mu.Lock()
	w.problems = append(w.problems, problems...)
	w.mu.Unlock()
	return res, nil
}

func (w *wire) check(t *testing.T) {
	t.Helper()
	w.mu.Lock()
	defer w.mu.Unlock()
	for _, p := range w.problems {
		t.Error(p)
	}
}
