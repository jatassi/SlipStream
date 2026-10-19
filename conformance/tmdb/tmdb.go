// Package tmdb is the conformance suite for TMDB: it drives SlipStream's
// TMDB client against fake-tmdb on every PR and against api.themoviedb.org
// in the online job, so a fake that drifts from the real service fails.
package tmdb

import (
	_ "embed"
	"errors"
	"math/rand/v2"
	"net/http"
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/jatassi/slipstream/conformance/internal/shape"
	"github.com/jatassi/slipstream/internal/metadata"
	provider "github.com/jatassi/slipstream/internal/providers/tmdb"
	"github.com/jatassi/slipstream/internal/works"
)

//go:embed shape.json
var shapeJSON []byte

const userAgent = "SlipStream/conformance ( github.com/jatassi/SlipStream )"

// unknownID is an id neither TMDB nor any Scenario holds.
const unknownID = 999999999

// Known is a movie the target holds with every block filled: dates,
// alternative titles, genres, a studio, cast and crew, a trailer, a
// poster, a backdrop and an English translation. Year is its primary
// release year.
type Known struct {
	ID     int64
	Title  string
	IMDbID string
	Year   int
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
// asserts what holds on both sides: the configuration test call answers,
// search finds Known by title, by primary release year and page by page,
// the movie call decodes every appended block, an unknown id is
// works.ErrNotFound and the changes list reads a day and refuses more than
// TMDB keeps. A target with a key also gets a wrong key refused. It then
// checks every recorded body against shape.json: keys and types, no
// values.
func Run(t *testing.T, tgt Target) {
	t.Helper()
	ctx := t.Context()
	rec := shape.NewRecorder(tgt.Client.Transport, route)
	client := func() *provider.Client {
		c := *tgt.Client
		c.Transport = rec
		return provider.New(metadata.NewLines(&c, userAgent, metadata.DefaultTimings(), rand.New(rand.NewPCG(1, 2))), tgt.Endpoints, provider.DefaultTimings()) //nolint:gosec // jitter needs no secrecy
	}
	ask := func(key string) metadata.Ask {
		return metadata.NewAsk(metadata.AskOptions{Language: "en-US", Region: "US", Keys: map[metadata.ProviderID]string{provider.ID: key}})
	}
	c, known := client(), tgt.Known

	if err := c.Test(ctx, ask(tgt.Key)); err != nil {
		t.Errorf("Test: %v", err)
	}
	search(t, c, ask(tgt.Key), known)
	movie(t, c, ask(tgt.Key), known)

	_, err := c.Movie(ctx, ask(tgt.Key), unknownID)
	if want := "GET /movie/" + strconv.Itoa(unknownID) + ": TMDB: has no such record"; err == nil || err.Error() != want || !errors.Is(err, works.ErrNotFound) {
		t.Errorf("Movie(%d) = %v, want %s", unknownID, err, want)
	}

	now := time.Now()
	day, err := c.MovieChanges(ctx, ask(tgt.Key), now.Add(-24*time.Hour), now, 1)
	if err != nil || day.Page != 1 {
		t.Errorf("MovieChanges over a day = page %d, %v; want page 1", day.Page, err)
	}
	t.Logf("a day of TMDB changes is %d pages", day.TotalPages)
	if _, err := c.MovieChanges(ctx, ask(tgt.Key), now.Add(-provider.ChangesWindow-time.Hour), now, 1); !errors.Is(err, metadata.ErrNoChanges) {
		t.Errorf("MovieChanges past %s = %v, want metadata.ErrNoChanges", provider.ChangesWindow, err)
	}

	if tgt.Key != "" {
		if err := client().Test(ctx, ask("0")); !errors.Is(err, works.ErrRefused) {
			t.Errorf("Test with a wrong key = %v, want works.ErrRefused", err)
		}
	}

	shape.Verify(t, rec.Bodies(), shapeJSON, tgt.Record)
}

func search(t *testing.T, c *provider.Client, ask metadata.Ask, known Known) {
	t.Helper()
	ctx := t.Context()
	isKnown := func(m provider.MovieResult) bool { return m.ID == known.ID && m.Title == known.Title }

	all, err := c.SearchMovies(ctx, ask, known.Title, 0, 1)
	if err != nil {
		t.Fatalf("SearchMovies(%q): %v", known.Title, err)
	}
	if all.Page != 1 || all.TotalPages < 1 || !slices.ContainsFunc(all.Results, isKnown) {
		t.Errorf("SearchMovies(%q) = %+v, want page 1 holding %d %q", known.Title, all, known.ID, known.Title)
	}

	year := strconv.Itoa(known.Year)
	inYear, err := c.SearchMovies(ctx, ask, known.Title, known.Year, 1)
	if err != nil || !slices.ContainsFunc(inYear.Results, isKnown) {
		t.Errorf("SearchMovies(%q, %d) = %+v, %v; want %d among them", known.Title, known.Year, inYear, err, known.ID)
	}
	for _, m := range inYear.Results {
		if !strings.HasPrefix(m.ReleaseDate, year+"-") {
			t.Errorf("SearchMovies(%q, %d) holds %d released %q", known.Title, known.Year, m.ID, m.ReleaseDate)
		}
	}

	past, err := c.SearchMovies(ctx, ask, known.Title, 0, all.TotalPages+1)
	if err != nil || past.Page != all.TotalPages+1 || len(past.Results) != 0 {
		t.Errorf("SearchMovies(%q) page %d = %+v, %v; want that page, empty", known.Title, all.TotalPages+1, past, err)
	}
}

func movie(t *testing.T, c *provider.Client, ask metadata.Ask, known Known) {
	t.Helper()
	m, err := c.Movie(t.Context(), ask, known.ID)
	if err != nil {
		t.Fatalf("Movie(%d): %v", known.ID, err)
	}
	if m.ID != known.ID || m.Title != known.Title || m.IMDbID != known.IMDbID || m.ExternalIDs.IMDbID != known.IMDbID || !strings.HasPrefix(m.ReleaseDate, strconv.Itoa(known.Year)+"-") {
		t.Errorf("Movie(%d) = %d %q %q %q released %q, want %d %q %q in %d", known.ID, m.ID, m.Title, m.IMDbID, m.ExternalIDs.IMDbID, m.ReleaseDate, known.ID, known.Title, known.IMDbID, known.Year)
	}
	english := slices.IndexFunc(m.Translations.Translations, func(tr provider.Translation) bool {
		return tr.Language == "en" && tr.Country == "US" && tr.Data.Title == known.Title
	})
	trailer := slices.IndexFunc(m.Videos.Results, func(v provider.Video) bool { return v.Type == "Trailer" && v.Site == "YouTube" && v.Key != "" })
	for block, filled := range map[string]bool{
		"overview":             m.Overview != "",
		"runtime":              m.Runtime > 0,
		"status":               m.Status != "",
		"genres":               len(m.Genres) > 0,
		"production_companies": len(m.ProductionCompanies) > 0,
		"poster_path":          m.PosterPath != "",
		"release_dates":        len(m.ReleaseDates.Results) > 0 && len(m.ReleaseDates.Results[0].Releases) > 0,
		"alternative_titles":   len(m.AlternativeTitles.Titles) > 0,
		"credits cast":         len(m.Credits.Cast) > 0 && m.Credits.Cast[0].Name != "",
		"credits crew":         slices.ContainsFunc(m.Credits.Crew, func(cr provider.Crew) bool { return cr.Job == "Director" }),
		"videos trailer":       trailer >= 0,
		"images posters":       len(m.Images.Posters) > 0 && m.Images.Posters[0].FilePath != "",
		"images backdrops":     len(m.Images.Backdrops) > 0,
		"translations en-US":   english >= 0,
	} {
		if !filled {
			t.Errorf("Movie(%d) has no %s", known.ID, block)
		}
	}
	for _, img := range m.Images.Posters {
		if _, err := c.Images().URL("poster", img.FilePath, "medium"); err != nil {
			t.Errorf("poster %q: %v", img.FilePath, err)
		}
	}
}

func route(p string) string {
	switch {
	case p == "/3/movie/changes":
		return "changes"
	case p == "/3/configuration":
		return "configuration"
	case strings.HasPrefix(p, "/3/search/"):
		return "search"
	case strings.HasPrefix(p, "/3/movie/"):
		return "movie"
	}
	return p
}
