// Package tvdb is the conformance suite for TVDB: it drives SlipStream's
// TVDB client against fake-tvdb on every PR and against api4.thetvdb.com
// in the online job, so a fake that drifts from the real service fails.
// Until the online job first runs with a key, shape.json holds the shapes
// TVDB's published OpenAPI document gives each answer.
package tvdb

import (
	_ "embed"
	"errors"
	"net/http"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/jatassi/slipstream/conformance/internal/shape"
	"github.com/jatassi/slipstream/internal/metadata"
	provider "github.com/jatassi/slipstream/internal/providers/tvdb"
	"github.com/jatassi/slipstream/internal/works"
)

//go:embed shape.json
var shapeJSON []byte

// unknownID is a series id neither TVDB nor any Scenario holds.
const unknownID provider.SeriesID = 999_999_999

// Known is a series the target lists with specials and a DVD order, found
// by searching for Name.
type Known struct {
	ID   provider.SeriesID
	Name string
}

type Target struct {
	// Endpoints is the fake's front, or the real default.
	Endpoints provider.Endpoints
	// Client trusts the run CA for the fake and the system roots online.
	Client *http.Client
	// Key logs in. The fake takes any key but none.
	Key string
	// Refused is a key the target refuses: any wrong key online, one a
	// cue rejected on the fake.
	Refused string
	Known   Known
	// Lines builds a call layer over a client, as a Node does.
	Lines   func(*http.Client) *metadata.Lines
	Timings provider.Timings
	// Record, when set, takes every body in place of the check against
	// shape.json; the online suite's -update sets it to rewrite the file.
	Record func(key string, body []byte)
}

// Run drives the TVDB client through every call SlipStream makes and
// asserts what holds on both sides: the login and the languages test
// answer, search finds Known and filters to the year it first aired, its record lists specials and the Aired and
// DVD orders, its episodes read in both orders with specials in Aired, an
// unknown id is works.ErrNotFound, the updates list reads the last day,
// and a wrong key is refused. It then checks every recorded body against
// shape.json: keys and types, no values.
func Run(t *testing.T, tgt Target) {
	t.Helper()
	ctx := t.Context()
	rec := shape.NewRecorder(tgt.Client.Transport, route)
	client := func() *provider.Client {
		c := *tgt.Client
		c.Transport = rec
		return provider.New(tgt.Lines(&c), tgt.Endpoints, tgt.Timings)
	}
	ask := func(key string) metadata.Ask {
		return metadata.NewAsk(metadata.AskOptions{Language: "en-US", Region: "US", Keys: map[metadata.ProviderID]string{provider.ID: key}})
	}
	c, known := client(), tgt.Known

	if err := c.Test(ctx, ask(tgt.Key)); err != nil {
		t.Fatalf("Test: %v", err)
	}

	found, err := c.Search(ctx, ask(tgt.Key), known.Name, 0)
	if err != nil {
		t.Fatalf("Search(%q): %v", known.Name, err)
	}
	at := slices.IndexFunc(found, func(f provider.Found) bool { return f.ID == known.ID })
	if at < 0 || found[at].Year == 0 {
		t.Fatalf("Search(%q) = %+v, want %d among them with a year", known.Name, found, known.ID)
	}
	for year, want := range map[int]bool{found[at].Year: true, found[at].Year - 1: false} {
		inYear, err := c.Search(ctx, ask(tgt.Key), known.Name, year)
		if got := slices.ContainsFunc(inYear, func(f provider.Found) bool { return f.ID == known.ID }); err != nil || got != want {
			t.Errorf("Search(%q, %d) holds %d: %t, %v; want %t", known.Name, year, known.ID, got, err, want)
		}
	}

	series, err := c.Series(ctx, ask(tgt.Key), known.ID)
	if err != nil {
		t.Fatalf("Series(%d): %v", known.ID, err)
	}
	if series.Name == "" || series.Poster == "" || !slices.Contains(series.Orders, provider.OrderAired) || !slices.Contains(series.Orders, provider.OrderDVD) ||
		!slices.ContainsFunc(series.Seasons, func(s provider.Season) bool { return s.Order == provider.OrderAired && s.Number == 0 }) {
		t.Errorf("Series(%d) = %+v, want a name, a poster, Aired and DVD orders and an Aired season 0", known.ID, series)
	}

	for _, o := range []provider.Order{provider.OrderAired, provider.OrderDVD} {
		checkEpisodes(t, c, ask(tgt.Key), known.ID, o)
	}

	_, err = c.Series(ctx, ask(tgt.Key), unknownID)
	if want := "GET /series/999999999/extended: TVDB: has no such record"; err == nil || err.Error() != want || !errors.Is(err, works.ErrNotFound) {
		t.Errorf("Series(%d) = %v, want %s", unknownID, err, want)
	}

	if _, err := c.Updates(ctx, ask(tgt.Key), time.Now().Add(-24*time.Hour), 1); err != nil {
		t.Errorf("Updates for the last day: %v", err)
	}

	if err := client().Test(ctx, ask(tgt.Refused)); !errors.Is(err, works.ErrRefused) {
		t.Errorf("Test with a wrong key = %v, want works.ErrRefused", err)
	}

	shape.Verify(t, rec.Bodies(), shapeJSON, tgt.Record)
}

// checkEpisodes reads id's episodes in o: each has an id and a number, and
// Aired order lists specials.
func checkEpisodes(t *testing.T, c *provider.Client, ask metadata.Ask, id provider.SeriesID, o provider.Order) {
	t.Helper()
	episodes, err := c.Episodes(t.Context(), ask, id, o)
	if err != nil {
		t.Fatalf("Episodes(%d, %s): %v", id, o, err)
	}
	if len(episodes) == 0 || slices.ContainsFunc(episodes, func(e provider.Episode) bool { return e.ID == 0 || e.Number < 1 }) {
		t.Errorf("Episodes(%d, %s) = %+v, want episodes each with an id and a number", id, o, episodes)
	}
	if o == provider.OrderAired && !slices.ContainsFunc(episodes, func(e provider.Episode) bool { return e.Season == 0 }) {
		t.Errorf("Episodes(%d, %s) lists no special", id, o)
	}
}

// route names each call by its path: /v4/series/1/extended is series.
func route(path string) string {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	switch {
	case slices.Contains(parts, "episodes"):
		return "episodes"
	case slices.Contains(parts, "series"):
		return "series"
	}
	return parts[len(parts)-1]
}
