// Package gestdown is the conformance suite for Gestdown: it drives
// SlipStream's Gestdown client against fake-gestdown on every PR and
// against api.gestdown.info in the online job, so a fake that drifts from
// the real service fails. It follows the recorded run of 2026-11-17
// (subtitles/online/run-2026-11-17.md).
package gestdown

import (
	"bytes"
	_ "embed"
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/jatassi/slipstream/conformance/internal/shape"
	"github.com/jatassi/slipstream/internal/calls"
	"github.com/jatassi/slipstream/internal/language"
	provider "github.com/jatassi/slipstream/internal/providers/gestdown"
	"github.com/jatassi/slipstream/internal/subtitling"
)

//go:embed shape.json
var shapeJSON []byte

type Target struct {
	// Endpoints is the fake's front, or the real default.
	Endpoints provider.Endpoints
	// Client trusts the run CA for the fake and the system roots online.
	Client *http.Client
	// Series is a series' TVDB id whose Season and Episode have English
	// subtitles, and Unknown a TVDB id Gestdown doesn't know.
	Series, Unknown string
	Season, Episode int
	// Lines builds a call layer over a client, as a Node does.
	Lines   func(*http.Client) *calls.Lines
	Timings provider.Timings
	// Cue plays one of the fake's cues; nil online, where none play.
	Cue func(name string, args any)
	// Unaired is the Season episode the fake lists once the gestdown-air
	// cue airs AirID.
	Unaired int
	AirID   string
	// Elapse lets d pass, nil where time can't be hurried.
	Elapse func(d time.Duration)
	// Record, when set, takes every body in place of the check against
	// shape.json; the online suite's -update sets it to rewrite the file.
	Record func(key string, body []byte)
}

var (
	id   = regexp.MustCompile(`^[0-9A-Za-z]{22}$`)
	pack = regexp.MustCompile(`^sp_[0-9A-Za-z]{22}(_ep_\d+|_entry_[0-9A-Za-z]+)?$`)
)

// Run drives the Gestdown client through every call SlipStream makes and
// asserts what holds on both sides: it searches TV only and sends no key;
// an episode lists completed subtitles whose ids are 22 base62 characters
// or a season pack's; a download is a raw SRT with CRLF line ends; and an
// unknown show is a 404 whose body is a bare string, which finds nothing.
// No request leaves for another host. Against the fake it then plays a
// 423 refresh, the 12 h not-found cache, its upstream accounts out, and
// a spent address bucket.
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
	cr := func() calls.Credentials { return calls.Credentials{Tally: &calls.Tally{}} }
	if spec := c.Spec(); spec.Keyed || !slices.Equal(spec.Kinds, []subtitling.Kind{subtitling.Episode}) {
		t.Errorf("Spec = %+v, want keyless and TV only", spec)
	}

	en, _ := language.Parse("en")
	episode := func(tvdb string, e int) subtitling.Query {
		return subtitling.Query{Subject: subtitling.Subject{Kind: subtitling.Episode, TVDB: tvdb, Season: tgt.Season, Episodes: []int{e}}, Language: en}
	}
	film := subtitling.Query{Subject: subtitling.Subject{Kind: subtitling.Feature, IMDb: "tt0113277", TVDB: tgt.Series}, Language: en}
	if offers, err := c.Search(ctx, cr(), film); err != nil || len(offers) != 0 || w.sent() != 0 {
		t.Errorf("a film's search = %+v, %v after %d requests, want nothing asked", offers, err, w.sent())
	}
	offers := search(t, c, cr(), episode(tgt.Series, tgt.Episode))
	i := slices.IndexFunc(offers, func(o subtitling.Offer) bool { return !o.Pack })
	if i < 0 {
		t.Fatalf("episode %d's offers %+v hold no single episode", tgt.Episode, offers)
	}
	dl, err := c.Download(ctx, cr(), offers[i])
	if err != nil {
		t.Fatalf("Download(%s): %v", offers[i].Key, err)
	}
	if bytes.HasPrefix(dl.Bytes, []byte("PK")) || !bytes.Contains(dl.Bytes, []byte("\r\n")) || !strings.HasSuffix(dl.Name, ".srt") {
		t.Errorf("Download(%s) = %q, %d bytes, want a raw SRT with CRLF line ends", offers[i].Key, dl.Name, len(dl.Bytes))
	}
	unknown := cr()
	if offers, err := c.Search(ctx, unknown, episode(tgt.Unknown, 1)); err != nil || len(offers) != 0 {
		t.Errorf("an unknown show's search = %+v, %v, want nothing", offers, err)
	}
	if os := unknown.Tally.Outcomes(); len(os) != 1 || os[0].Kind != calls.NotFound {
		t.Errorf("an unknown show's outcomes = %+v, want one not found", os)
	}

	var bodies []shape.Body
	for _, b := range rec.Bodies() {
		if !strings.HasPrefix(b.Key, "download 200") {
			bodies = append(bodies, b)
		}
	}
	shape.Verify(t, bodies, shapeJSON, tgt.Record)

	if tgt.Cue != nil {
		cues(t, tgt, func() *provider.Client { return client(w) }, w, episode)
	}
	w.check(t)
}

func route(path string) string {
	switch {
	case strings.HasPrefix(path, "/shows/"):
		return "show"
	case strings.HasPrefix(path, "/subtitles/get/"):
		return "episode"
	}
	return "download"
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
		single := id.MatchString(o.Key) && !o.Pack && slices.Equal(o.Episodes, q.Subject.Episodes)
		packed := pack.MatchString(o.Key) && o.Pack
		if !single && !packed || o.Language.String() != "en" || len(o.Releases) == 0 {
			t.Errorf("Search(%+v) offers %+v", q.Subject, o)
		}
	}
	return offers
}

func cues(t *testing.T, tgt Target, client func() *provider.Client, w *wire, episode func(string, int) subtitling.Query) {
	t.Helper()
	ctx := t.Context()
	cr := func() calls.Credentials { return calls.Credentials{Tally: &calls.Tally{}} }

	tgt.Cue("gestdown-refreshing", map[string]int{"calls": 2})
	before := w.answered(http.StatusLocked)
	search(t, client(), cr(), episode(tgt.Series, tgt.Episode))
	if got := w.answered(http.StatusLocked) - before; got != 2 {
		t.Errorf("a search behind a refresh met %d 423s, want 2 then a pass", got)
	}

	missed := cr()
	if offers, err := client().Search(ctx, missed, episode(tgt.Series, tgt.Unaired)); err != nil || len(offers) != 0 {
		t.Errorf("the unaired episode's search = %+v, %v, want nothing", offers, err)
	}
	tgt.Cue("gestdown-air", map[string]string{"id": tgt.AirID})
	if offers, err := client().Search(ctx, cr(), episode(tgt.Series, tgt.Unaired)); err != nil || len(offers) != 0 {
		t.Errorf("the aired episode's search inside the cache = %+v, %v, want the cached not found", offers, err)
	}
	if tgt.Elapse != nil {
		tgt.Elapse(12*time.Hour + time.Minute)
		search(t, client(), cr(), episode(tgt.Series, tgt.Unaired))
	}

	offer := search(t, client(), cr(), episode(tgt.Series, tgt.Episode))[0]
	tgt.Cue("gestdown-upstream-out", nil)
	_, err := client().Download(ctx, cr(), offer)
	if f, ok := errors.AsType[*calls.Failed](err); !ok || f.Outcome.Kind != calls.Capped {
		t.Errorf("a download with the upstream accounts out = %v, want capped", err)
	}

	tgt.Cue("gestdown-bucket", nil)
	limited := w.answered(http.StatusTooManyRequests)
	search(t, client(), cr(), episode(tgt.Series, tgt.Episode))
	if w.answered(http.StatusTooManyRequests) == limited || !w.retryAfter() {
		t.Errorf("a search with the address bucket spent met no 429 with Retry-After it waited out")
	}
}

// wire sees every request the client sends and every answer's status.
type wire struct {
	next http.RoundTripper
	api  string

	mu       sync.Mutex
	requests int
	statuses map[int]int
	waited   bool
	strays   []string
}

func (w *wire) RoundTrip(req *http.Request) (*http.Response, error) {
	res, err := w.next.RoundTrip(req)
	if err != nil {
		return nil, fmt.Errorf("send %s: %w", req.URL.Redacted(), err)
	}
	w.mu.Lock()
	defer w.mu.Unlock()
	w.requests++
	if w.statuses == nil {
		w.statuses = map[int]int{}
	}
	w.statuses[res.StatusCode]++
	w.waited = w.waited || res.StatusCode == http.StatusTooManyRequests && res.Header.Get("Retry-After") != ""
	if req.URL.Host != w.api || req.Header.Get("Authorization") != "" || req.Header.Get("X-API-Key") != "" {
		w.strays = append(w.strays, req.URL.Redacted())
	}
	return res, nil
}

func (w *wire) sent() int {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.requests
}

func (w *wire) answered(status int) int {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.statuses[status]
}

func (w *wire) retryAfter() bool {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.waited
}

// check fails t for a request to another host than api or one carrying a
// key.
func (w *wire) check(t *testing.T) {
	t.Helper()
	w.mu.Lock()
	defer w.mu.Unlock()
	if len(w.strays) > 0 {
		t.Errorf("requests left for hosts other than %s or carried a key: %v", w.api, w.strays)
	}
}
