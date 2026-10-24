// Package omdb is the conformance suite for OMDb: it drives SlipStream's
// OMDb client against fake-omdb on every PR and against www.omdbapi.com in
// the online job, so a fake that drifts from the real service fails.
package omdb

import (
	_ "embed"
	"errors"
	"net/http"
	"slices"
	"testing"

	"github.com/jatassi/slipstream/conformance/internal/shape"
	"github.com/jatassi/slipstream/internal/metadata"
	provider "github.com/jatassi/slipstream/internal/providers/omdb"
	"github.com/jatassi/slipstream/internal/works"
)

//go:embed shape.json
var shapeJSON []byte

// unknownID is an IMDb id neither OMDb nor any Scenario holds.
const unknownID = "tt99999999"

type Target struct {
	// Endpoints is the fake's front, or the real default.
	Endpoints provider.Endpoints
	// Client trusts the run CA for the fake and the system roots online.
	Client *http.Client
	// Key is sent as apikey. It is "" against the fake, which takes any key
	// or none.
	Key string
	// Known is the IMDb id of a title the target rates on IMDb (with
	// votes), Rotten Tomatoes and Metacritic, and gives an awards line.
	Known string
	// Lines builds a call layer over a client, as a Node does.
	Lines   func(*http.Client) *metadata.Lines
	Timings provider.Timings
	// Record, when set, takes every body in place of the check against
	// shape.json; the online suite's -update sets it to rewrite the file.
	Record func(key string, body []byte)
}

// Run drives the OMDb client through every call SlipStream makes and
// asserts what holds on both sides: the test call answers, Known reads
// three ratings inside their scales and an awards line, and an unknown id
// is works.ErrNotFound. A target with a key also gets a wrong key refused.
// It then checks every recorded body against shape.json: keys and types,
// no values. The daily cap is checked against the fake alone, since the
// real one would spend a day's allowance.
func Run(t *testing.T, tgt Target) {
	t.Helper()
	ctx := t.Context()
	rec := shape.NewRecorder(tgt.Client.Transport, func(string) string { return "title" })
	client := func() *provider.Client {
		c := *tgt.Client
		c.Transport = rec
		return provider.New(tgt.Lines(&c), tgt.Endpoints, tgt.Timings)
	}
	ask := func(key string) metadata.Ask {
		return metadata.NewAsk(metadata.AskOptions{Ratings: true, Keys: map[metadata.ProviderID]string{provider.ID: key}})
	}
	c := client()

	if err := c.Test(ctx, ask(tgt.Key)); err != nil {
		t.Errorf("Test: %v", err)
	}

	title, err := c.Title(ctx, ask(tgt.Key), tgt.Known)
	if err != nil {
		t.Fatalf("Title(%s): %v", tgt.Known, err)
	}
	for _, source := range []works.RatingSource{"imdb", "rottentomatoes", "metacritic"} {
		i := slices.IndexFunc(title.Ratings, func(r works.Rating) bool { return r.Source == source })
		if i < 0 {
			t.Errorf("Title(%s) = %+v, want a %s rating", tgt.Known, title, source)
			continue
		}
		r := title.Ratings[i]
		if r.Value <= 0 || r.Value > works.RatingSources[source].Scale {
			t.Errorf("Title(%s) %s rating = %v, want inside 0 to %v", tgt.Known, source, r.Value, works.RatingSources[source].Scale)
		}
		if (source == "imdb") != (r.Votes != nil) || (r.Votes != nil && *r.Votes <= 0) {
			t.Errorf("Title(%s) %s votes = %v, want a count on IMDb alone", tgt.Known, source, r.Votes)
		}
	}
	if title.Awards == "" {
		t.Errorf("Title(%s) has no awards line", tgt.Known)
	}
	t.Logf("%s reads %+v", tgt.Known, title)

	_, err = c.Title(ctx, ask(tgt.Key), unknownID)
	if want := "GET " + unknownID + ": OMDb: has no such record"; err == nil || err.Error() != want || !errors.Is(err, works.ErrNotFound) {
		t.Errorf("Title(%s) = %v, want %s", unknownID, err, want)
	}

	if tgt.Key != "" {
		if err := client().Test(ctx, ask("0")); !errors.Is(err, works.ErrRefused) {
			t.Errorf("Test with a wrong key = %v, want works.ErrRefused", err)
		}
	}

	shape.Verify(t, rec.Bodies(), shapeJSON, tgt.Record)
}
