// Package subdl is the conformance suite for SubDL: it drives SlipStream's
// SubDL client against fake-subdl on every PR and against api.subdl.com in
// the online job, so a fake that drifts from the real service fails. It
// follows the recorded run of 2026-11-17
// (subtitles/online/run-2026-11-17.md).
package subdl

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/jatassi/slipstream/conformance/internal/shape"
	"github.com/jatassi/slipstream/internal/calls"
	"github.com/jatassi/slipstream/internal/language"
	provider "github.com/jatassi/slipstream/internal/providers/subdl"
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
	// Film is a film's IMDb and TMDB ids with English subtitles.
	FilmIMDb, FilmTMDB string
	// Series is a series' IMDb id whose Season and Episode have English
	// subtitles.
	Series          string
	Season, Episode int
	// Packs: the episode's search lists a season pack.
	Packs bool
	// AI: the film's search lists a machine translation.
	AI bool
	// Lines builds a call layer over a client, as a Node does.
	Lines   func(*http.Client) *calls.Lines
	Timings provider.Timings
	// Cue plays one of the fake's cues; nil online, where none play.
	Cue func(name string, args any)
	// Record, when set, takes every body in place of the check against
	// shape.json; the online suite's -update sets it to rewrite the file.
	Record func(key string, body []byte)
}

// Run drives the SubDL client through every call SlipStream makes and
// asserts what holds on both sides: /me answers the account, plan and
// allowance with ISO 8601 resets; a missing and a wrong key are each a 401
// refusal with no rate headers; a film searches by IMDb and TMDB id and an
// episode by season and episode, with rate headers on each 200 search; a
// pack's members carry their episode and the offer names them; "SUBDL AI"
// reads as AI; and a download goes through the v2 route on the api host
// and counts against the key. No request ever leaves for another host. Against the fake it then plays each cue.
func Run(t *testing.T, tgt Target) {
	t.Helper()
	ctx := t.Context()
	next := tgt.Client.Transport
	if next == nil {
		next = http.DefaultTransport
	}
	w := &wire{next: next, api: tgt.Endpoints.API.JoinPath().Host}
	rec := shape.NewRecorder(w, route)
	client := func() *provider.Client {
		c := *tgt.Client
		c.Transport = rec
		return provider.New(tgt.Lines(calls.NoRedirects(&c)), tgt.Endpoints, tgt.Timings)
	}
	c := client()
	with := func(key string) calls.Credentials {
		return calls.Credentials{Keys: map[calls.ID]string{provider.ID: key}, Tally: &calls.Tally{}}
	}

	cr := with(tgt.Key)
	account, err := c.CheckKey(ctx, cr)
	if err != nil {
		t.Fatalf("CheckKey: %v", err)
	}
	if account.Name == "" || account.Plan == "" {
		t.Errorf("CheckKey = %+v, want an account name and a plan", account)
	}
	before := allowance(t, cr, "CheckKey")
	refusals(t, client, with, tgt.Wrong)

	en, _ := language.Parse("en")
	film := subtitling.Query{Subject: subtitling.Subject{Kind: subtitling.Feature, IMDb: tgt.FilmIMDb}, Language: en}
	byIMDb := search(t, c, cr, film)
	film.Subject = subtitling.Subject{Kind: subtitling.Feature, TMDB: tgt.FilmTMDB}
	byTMDB := search(t, c, cr, film)
	if !slices.ContainsFunc(byIMDb, func(o subtitling.Offer) bool { return o.Key == byTMDB[0].Key }) {
		t.Errorf("a search by TMDB id first offers %s, which the search by IMDb id lacks", byTMDB[0].Key)
	}
	if tgt.AI && !slices.ContainsFunc(byIMDb, func(o subtitling.Offer) bool { return o.AI }) {
		t.Errorf("the film's offers %+v have no machine translation", byIMDb)
	}

	offers := episodeOffers(t, c, cr, tgt, en)
	authors(t, rec.Bodies(), slices.Concat(byIMDb, offers))

	i := slices.IndexFunc(byIMDb, func(o subtitling.Offer) bool { return !o.Image && !o.AI })
	if i < 0 {
		t.Fatalf("the film's offers %+v hold no text subtitle a person made", byIMDb)
	}
	dl, err := c.Download(ctx, cr, byIMDb[i])
	if err != nil {
		t.Fatalf("Download(%s): %v", byIMDb[i].Key, err)
	}
	if !bytes.HasPrefix(dl.Bytes, []byte("PK\x03\x04")) || dl.Name == "" {
		t.Errorf("Download(%s) = %q, %d bytes starting %q, want a named zip", byIMDb[i].Key, dl.Name, len(dl.Bytes), dl.Bytes[:min(len(dl.Bytes), 4)])
	}
	after := with(tgt.Key)
	if _, err := c.CheckKey(ctx, after); err != nil {
		t.Fatal(err)
	}
	if got := allowance(t, after, "CheckKey after a download").Downloads; got.Used != before.Downloads.Used+1 {
		t.Errorf("downloads used = %d after one download from %d, want it counted exactly", got.Used, before.Downloads.Used)
	}

	var bodies []shape.Body
	for _, b := range rec.Bodies() {
		if !strings.HasPrefix(b.Key, "download 200") {
			bodies = append(bodies, b)
		}
	}
	shape.Verify(t, bodies, shapeJSON, tgt.Record)

	if tgt.Cue != nil {
		cues(t, tgt, func() *provider.Client {
			cl := *tgt.Client
			cl.Transport = w
			return provider.New(tgt.Lines(calls.NoRedirects(&cl)), tgt.Endpoints, tgt.Timings)
		}, with, byIMDb[i], film)
	}
	w.check(t)
}

// refusals checks a missing and a wrong key each answer a 401 refusal
// with its own message.
func refusals(t *testing.T, client func() *provider.Client, with func(string) calls.Credentials, wrong string) {
	t.Helper()
	for _, row := range []struct{ name, key, words string }{
		{"a missing key", "", "refused the key: Missing API key"},
		{"a wrong key", wrong, "refused the key: Not Authorized"},
	} {
		_, err := client().CheckKey(t.Context(), with(row.key))
		if f, ok := errors.AsType[*calls.Failed](err); !ok || f.Outcome.Kind != calls.Refused || !strings.HasPrefix(f.Outcome.Words, row.words) {
			t.Errorf("CheckKey with %s = %v, want refused with %q", row.name, err, row.words)
		}
	}
}

// episodeOffers searches the target's episode: every offer names it, and
// a pack names several.
func episodeOffers(t *testing.T, c *provider.Client, cr calls.Credentials, tgt Target, en language.Code) []subtitling.Offer {
	t.Helper()
	episode := subtitling.Query{Subject: subtitling.Subject{Kind: subtitling.Episode, IMDb: tgt.Series, Season: tgt.Season, Episodes: []int{tgt.Episode}}, Language: en}
	offers := search(t, c, cr, episode)
	for _, o := range offers {
		if !slices.Contains(o.Episodes, tgt.Episode) || o.Pack != (len(o.Episodes) > 1) {
			t.Errorf("episode %d's offer %s names episodes %v (pack %t)", tgt.Episode, o.Key, o.Episodes, o.Pack)
		}
	}
	if tgt.Packs && !slices.ContainsFunc(offers, func(o subtitling.Offer) bool { return o.Pack }) {
		t.Errorf("episode %d's offers %+v hold no pack", tgt.Episode, offers)
	}
	return offers
}

func route(path string) string {
	switch {
	case strings.HasSuffix(path, "/me"):
		return "me"
	case strings.HasSuffix(path, "/search"):
		return "search"
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
		if o.Key == "" || o.Language.String() != "en" || len(o.Releases) == 0 || strings.Contains(o.Page, "api_key") {
			t.Errorf("Search(%+v) offers %+v", q.Subject, o)
		}
	}
	return offers
}

// allowance is the allowance the call's OK outcome reported: /me's two
// counters, with downloads exact and searches lagging, each resetting at an
// ISO 8601 time.
func allowance(t *testing.T, cr calls.Credentials, call string) calls.Allowance {
	t.Helper()
	os := cr.Tally.Outcomes()
	last := os[len(os)-1]
	a := last.Allowance
	if last.Kind != calls.OK || a == nil || a.Downloads == nil || a.Searches == nil {
		t.Fatalf("%s outcome = %+v, want OK with both counters", call, last)
	}
	for _, w := range []*calls.Window{a.Downloads, a.Searches} {
		if w.Limit <= 0 || w.Resets.IsZero() || w.Resets.Location() != time.UTC {
			t.Errorf("%s window = %+v, want a limit and a UTC reset", call, w)
		}
	}
	if !a.Downloads.Exact || a.Searches.Exact {
		t.Errorf("%s: downloads exact %t, searches exact %t, want only downloads exact", call, a.Downloads.Exact, a.Searches.Exact)
	}
	return *a
}

type itemJSON struct {
	Author       string `json:"author"`
	SubtitlePage string `json:"subtitlePage"`
}

// authors checks the search bodies against the offers: "SUBDL AI" reads
// as AI and every other author as not.
func authors(t *testing.T, bodies []shape.Body, offers []subtitling.Offer) {
	t.Helper()
	ai := map[string]bool{}
	for _, b := range bodies {
		if b.Key != "search 200" {
			continue
		}
		var found struct {
			Subtitles []itemJSON `json:"subtitles"`
		}
		if err := json.Unmarshal(b.JSON, &found); err != nil {
			t.Fatal(err)
		}
		for _, s := range found.Subtitles {
			ai[strings.TrimPrefix(s.SubtitlePage, "/s/info/")] = s.Author == "SUBDL AI"
		}
	}
	for _, o := range offers {
		if want, ok := ai[o.Key]; !ok || o.AI != want {
			t.Errorf("offer %s AI = %t, want %t from its author", o.Key, o.AI, want)
		}
	}
}

// cues plays each of the fake's cues and checks its signal, each on a
// fresh call layer, since a pause closes a Line's gate for the key.
func cues(t *testing.T, tgt Target, client func() *provider.Client, with func(string) calls.Credentials, offer subtitling.Offer, film subtitling.Query) {
	t.Helper()
	ctx := t.Context()
	failed := func(err error) calls.Outcome {
		f, ok := errors.AsType[*calls.Failed](err)
		if !ok {
			t.Fatalf("err = %v, want a *calls.Failed", err)
		}
		return f.Outcome
	}

	tgt.Cue("subdl-busy", map[string]int{"calls": 1})
	if _, err := client().Search(ctx, with(tgt.Key), film); err != nil {
		t.Errorf("a search after one service_busy = %v, want it tried again and answered", err)
	}

	tgt.Cue("subdl-waf", map[string]int{"calls": 1})
	if _, err := client().Search(ctx, with(tgt.Key), film); failed(err).Kind != calls.Unreachable {
		t.Errorf("a search behind a challenge page = %v, want unreachable, not a refused key", err)
	}

	tgt.Cue("subdl-daily-limit", map[string]string{"key": tgt.Key})
	_, err := client().Download(ctx, with(tgt.Key), offer)
	now := time.Now().UTC()
	if o := failed(err); o.Kind != calls.Capped || !o.Until.Equal(now.Truncate(24*time.Hour).Add(24*time.Hour)) {
		t.Errorf("a download past the daily limit = %+v, want capped until the next UTC midnight", o)
	}

	tgt.Cue("subdl-revoke-key", map[string]string{"key": tgt.Key})
	if _, err := client().CheckKey(ctx, with(tgt.Key)); failed(err).Kind != calls.Refused {
		t.Errorf("CheckKey with a revoked key = %v, want refused", err)
	}
}

// wire sees every request the client sends and every answer's headers.
type wire struct {
	next http.RoundTripper
	api  string

	mu      sync.Mutex
	strays  []string
	headers []string
}

func (w *wire) RoundTrip(req *http.Request) (*http.Response, error) {
	w.mu.Lock()
	if req.URL.Host != w.api {
		w.strays = append(w.strays, req.URL.Redacted())
	}
	w.mu.Unlock()
	res, err := w.next.RoundTrip(req)
	if err != nil {
		return nil, fmt.Errorf("send %s: %w", req.URL.Redacted(), err)
	}
	rated := res.Header.Get("X-Ratelimit-Remaining") != ""
	search := strings.HasSuffix(req.URL.Path, "/search") && res.StatusCode == http.StatusOK
	if rated != search {
		w.mu.Lock()
		w.headers = append(w.headers, req.URL.Path+" "+res.Status)
		w.mu.Unlock()
	}
	return res, nil
}

// check fails t for any request to another host than api, and for rate
// headers anywhere but on a 200 search or missing from one.
func (w *wire) check(t *testing.T) {
	t.Helper()
	w.mu.Lock()
	defer w.mu.Unlock()
	if len(w.strays) > 0 {
		t.Errorf("requests left for hosts other than %s: %v", w.api, w.strays)
	}
	if len(w.headers) > 0 {
		t.Errorf("rate headers out of place, as recorded only on 200 searches: %v", w.headers)
	}
}
