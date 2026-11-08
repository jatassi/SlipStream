// Package tmdb is the conformance suite for TMDB: it drives SlipStream's
// TMDB client against fake-tmdb on every PR and against api.themoviedb.org
// in the online job, so a fake that drifts from the real service fails.
package tmdb

import (
	_ "embed"
	"encoding/json"
	"errors"
	"net/http"
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/jatassi/slipstream/conformance/internal/shape"
	"github.com/jatassi/slipstream/internal/metadata"
	"github.com/jatassi/slipstream/internal/metadata/artwork"
	provider "github.com/jatassi/slipstream/internal/providers/tmdb"
	"github.com/jatassi/slipstream/internal/works"
)

//go:embed shape.json
var shapeJSON []byte

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

// KnownSeries is a series the target holds under its TVDB and IMDb ids,
// with cast, creators, a US content rating, a trailer, a network logo, a
// poster, a backdrop and a title treatment.
type KnownSeries struct {
	ID     int64
	Name   string
	TVDB   int64
	IMDbID string
}

type Target struct {
	// Endpoints is the fake's front, or the real default.
	Endpoints provider.Endpoints
	// Client trusts the run CA for the fake and the system roots online.
	Client *http.Client
	// Key is sent as api_key. It is "" against the fake, which takes any
	// key or none.
	Key    string
	Known  Known
	Series KnownSeries
	// Lines builds a call layer over a client, as a Node does.
	Lines   func(*http.Client) *metadata.Lines
	Timings provider.Timings
	// Record, when set, takes every body in place of the check against
	// shape.json; the online suite's -update sets it to rewrite the file.
	Record func(key string, body []byte)
}

// Run drives the TMDB client through every call SlipStream makes and
// asserts what holds on both sides: the configuration test call answers,
// search finds Known by title, by primary release year and page by page,
// the movie call decodes every appended block, an unknown id is
// works.ErrNotFound and the changes list reads a day and refuses more than
// TMDB keeps. Search, the series call, its external ids, the lookup by
// TVDB id and the series changes list do the same for Series. A target
// with a key also gets a wrong key refused. It then
// checks every recorded body against shape.json: keys and types, no
// values.
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
		t.Errorf("Test: %v", err)
	}
	search(t, c, ask(tgt.Key), known)
	movie(t, c, ask(tgt.Key), known)
	series(t, c, ask(tgt.Key), tgt.Series)

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
	tvDay, err := c.SeriesChanges(ctx, ask(tgt.Key), now.Add(-24*time.Hour), now, 1)
	if err != nil || tvDay.Page != 1 {
		t.Errorf("SeriesChanges over a day = page %d, %v; want page 1", tvDay.Page, err)
	}
	t.Logf("a day of TMDB series changes is %d pages", tvDay.TotalPages)
	if _, err := c.SeriesChanges(ctx, ask(tgt.Key), now.Add(-provider.ChangesWindow-time.Hour), now, 1); !errors.Is(err, metadata.ErrNoChanges) {
		t.Errorf("SeriesChanges past %s = %v, want metadata.ErrNoChanges", provider.ChangesWindow, err)
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
	english := slices.IndexFunc(m.Translations.Translations, func(tr provider.Translation) bool { return tr.Language == "en" && tr.Country == "US" })
	if english >= 0 {
		t.Logf("the en-US translation of %d is %+v", known.ID, m.Translations.Translations[english].Data)
	}
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
			t.Errorf("Movie(%d) has no %s; translations %+v", known.ID, block, m.Translations.Translations)
		}
	}
	for _, img := range m.Images.Posters {
		if _, err := c.Images().URL("poster", img.FilePath, "medium"); err != nil {
			t.Errorf("poster %q: %v", img.FilePath, err)
		}
	}
}

func series(t *testing.T, c *provider.Client, ask metadata.Ask, known KnownSeries) {
	t.Helper()
	ctx := t.Context()
	found, err := c.SearchSeries(ctx, ask, known.Name, 0)
	at := slices.IndexFunc(found, func(f provider.SeriesFound) bool {
		return f.ID == known.ID && f.Name == known.Name && f.FirstAired != ""
	})
	if err != nil || at < 0 {
		t.Fatalf("SearchSeries(%q) = %+v, %v; want %d %q with a first air date", known.Name, found, err, known.ID, known.Name)
	}
	first, err := works.ParseDate(found[at].FirstAired)
	if err != nil {
		t.Fatal(err)
	}
	for year, want := range map[int]bool{first.Year: true, first.Year - 1: false} {
		inYear, err := c.SearchSeries(ctx, ask, known.Name, year)
		if got := slices.ContainsFunc(inYear, func(f provider.SeriesFound) bool { return f.ID == known.ID }); err != nil || got != want {
			t.Errorf("SearchSeries(%q, %d) holds %d: %t, %v; want %t", known.Name, year, known.ID, got, err, want)
		}
	}

	s, err := c.Series(ctx, ask, known.ID)
	if err != nil {
		t.Fatalf("Series(%d): %v", known.ID, err)
	}
	ids := provider.SeriesIDs{TVDB: known.TVDB, IMDb: known.IMDbID}
	if s.ID != known.ID || s.SeriesIDs != ids {
		t.Errorf("Series(%d) = %d with %+v, want %+v", known.ID, s.ID, s.SeriesIDs, ids)
	}
	for block, filled := range map[string]bool{
		"cast":            len(s.Cast) > 0 && s.Cast[0].Name != "" && s.Cast[0].Role != "",
		"creators":        len(s.Creators) > 0 && s.Creators[0].Name != "",
		"US rating":       s.Certifications["US"] != "",
		"trailer":         s.Trailer != nil && s.Trailer.Site == "YouTube" && s.Trailer.Key != "",
		"network logo":    s.NetworkLogo != "",
		"title treatment": s.TitleTreatment != "",
		"votes":           s.VoteCount > 0,
	} {
		if !filled {
			t.Errorf("Series(%d) has no %s: %+v", known.ID, block, s)
		}
	}
	for shape, path := range map[artwork.Shape]string{artwork.ShapePoster: s.Poster, artwork.ShapeBackdrop: s.Backdrop, artwork.ShapeLogo: s.NetworkLogo} {
		if _, err := c.Images().URL(shape, path, artwork.SizeMedium); err != nil {
			t.Errorf("Series(%d) %s %q: %v", known.ID, shape, path, err)
		}
	}

	seriesLookups(t, c, ask, known)
}

func seriesLookups(t *testing.T, c *provider.Client, ask metadata.Ask, known KnownSeries) {
	t.Helper()
	ctx := t.Context()
	ids := provider.SeriesIDs{TVDB: known.TVDB, IMDb: known.IMDbID}
	if got, err := c.SeriesExternalIDs(ctx, ask, known.ID); err != nil || got != ids {
		t.Errorf("SeriesExternalIDs(%d) = %+v, %v; want %+v", known.ID, got, err, ids)
	}
	if id, ok, err := c.FindTVDB(ctx, ask, known.TVDB); err != nil || !ok || id != known.ID {
		t.Errorf("FindTVDB(%d) = %d, %t, %v; want %d", known.TVDB, id, ok, err, known.ID)
	}
	if id, ok, err := c.FindTVDB(ctx, ask, unknownID); err != nil || ok {
		t.Errorf("FindTVDB(%d) = %d, %t, %v; want none", unknownID, id, ok, err)
	}
	for name, call := range map[string]func() error{
		"Series":            func() error { _, err := c.Series(ctx, ask, unknownID); return err },
		"SeriesExternalIDs": func() error { _, err := c.SeriesExternalIDs(ctx, ask, unknownID); return err },
	} {
		if err := call(); !errors.Is(err, works.ErrNotFound) {
			t.Errorf("%s(%d) = %v, want works.ErrNotFound", name, unknownID, err)
		}
	}
}

func route(p string) string {
	switch {
	case p == "/3/movie/changes":
		return "changes"
	case p == "/3/tv/changes":
		return "tv changes"
	case p == "/3/configuration":
		return "configuration"
	case p == "/3/search/tv":
		return "search tv"
	case strings.HasPrefix(p, "/3/search/"):
		return "search"
	case strings.HasPrefix(p, "/3/movie/"):
		return "movie"
	case strings.HasPrefix(p, "/3/find/"):
		return "find"
	case strings.HasPrefix(p, "/3/tv/") && strings.HasSuffix(p, "/external_ids"):
		return "tv external_ids"
	case strings.HasPrefix(p, "/3/tv/"):
		return "tv"
	}
	return p
}

// Configuration decodes /configuration/<list> at e into v, read with key.
func Configuration(t *testing.T, e provider.Endpoints, key, list string, v any) {
	t.Helper()
	u := e.API.JoinPath("configuration", list)
	u.RawQuery = "api_key=" + key
	req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, u.String(), http.NoBody)
	if err != nil {
		t.Fatal(err)
	}
	res, err := (&http.Client{Timeout: time.Minute}).Do(req)
	if err != nil {
		t.Fatalf("GET /configuration/%s: %v", list, err)
	}
	defer func() { _ = res.Body.Close() }()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("GET /configuration/%s: %s", list, res.Status)
	}
	if err := json.NewDecoder(res.Body).Decode(v); err != nil {
		t.Fatalf("GET /configuration/%s: %v", list, err)
	}
}
