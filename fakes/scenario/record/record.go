package main

import (
	"cmp"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/jatassi/slipstream/fakes/scenario"
	tmdbapi "github.com/jatassi/slipstream/internal/providers/tmdb"
)

type config struct {
	scenario, store              string
	tmdbAPI, tmdbImages, omdbAPI string
	tmdbKey, omdbKey             string
	ids                          []int64
	client                       *http.Client
}

// Each movie keeps its dates and alternative titles in countries, its
// translations in languages, its first maxTitles alternative titles and
// its first maxCast cast.
var (
	countries = []string{"US", "GB", "FR", "DE"}
	languages = []string{"fr-FR", "de-DE", "es-ES"}
)

const (
	maxTitles = 5
	maxCast   = 10
)

// sizes is the size each image is downloaded at: big enough for the
// sizes the app asks for, small enough to check in.
var sizes = map[string]string{"poster": "w500", "backdrop": "w1280", "logo": "w500", "profile": "w185"}

func run(ctx context.Context, c config) error {
	ids := slices.Sorted(slices.Values(c.ids))
	ids = slices.Compact(ids)
	movies := make([]scenario.TMDBMovie, 0, len(ids))
	titles := make([]scenario.OMDbTitle, 0, len(ids))
	for _, id := range ids {
		var m tmdbapi.Movie
		q := url.Values{
			"api_key":                {c.tmdbKey},
			"language":               {"en-US"},
			"append_to_response":     {"external_ids,release_dates,alternative_titles,credits,videos,images,translations"},
			"include_image_language": {"en,null"},
		}
		if err := c.getJSON(ctx, c.tmdbAPI+"/movie/"+strconv.FormatInt(id, 10), q, &m); err != nil {
			return err
		}
		rec, err := record(m)
		if err != nil {
			return err
		}
		for _, f := range files(rec) {
			if err := c.download(ctx, f); err != nil {
				return err
			}
		}
		t, err := c.omdb(ctx, rec.IMDbID)
		if err != nil {
			return err
		}
		movies = append(movies, rec)
		titles = append(titles, t)
	}
	return rewrite(c.scenario, movies, titles)
}

// record is the Scenario's record of TMDB's movie m.
func record(m tmdbapi.Movie) (scenario.TMDBMovie, error) {
	out := scenario.TMDBMovie{
		ID: m.ID, IMDbID: cmp.Or(m.ExternalIDs.IMDbID, m.IMDbID), Title: m.Title,
		OriginalTitle: m.OriginalTitle, OriginalLanguage: m.OriginalLanguage, Overview: m.Overview, Tagline: m.Tagline,
		Status: m.Status, Runtime: time.Duration(m.Runtime) * time.Minute, VoteAverage: m.VoteAverage, VoteCount: m.VoteCount,
		Trailer: trailer(m.Videos.Results), Dates: dates(m.ReleaseDates), Titles: alternatives(m.AlternativeTitles.Titles),
		Translations: translations(m.Translations.Translations), Credits: credits(m.Credits),
	}
	if out.IMDbID == "" {
		return scenario.TMDBMovie{}, fmt.Errorf("TMDB movie %d has no IMDb id", m.ID)
	}
	for _, g := range m.Genres {
		out.Genres = append(out.Genres, g.Name)
	}
	if c := m.BelongsToCollection; c != nil {
		out.Collection = &scenario.TMDBNamed{ID: c.ID, Name: c.Name}
	}
	if len(m.ProductionCompanies) > 0 {
		out.Studio = &scenario.TMDBNamed{ID: m.ProductionCompanies[0].ID, Name: m.ProductionCompanies[0].Name}
	}
	for _, pick := range []struct {
		shape, language string
		from            []tmdbapi.Image
	}{{"poster", "en", m.Images.Posters}, {"backdrop", "", m.Images.Backdrops}, {"logo", "en", m.Images.Logos}} {
		if im, ok := best(pick.from, pick.language); ok {
			out.Images = append(out.Images, scenario.TMDBImage{
				Shape: pick.shape, Language: pick.language, Votes: im.VoteAverage, Path: im.FilePath, Width: im.Width, Height: im.Height,
			})
		}
	}
	return out, nil
}

// best is the best-voted image in language, "" for no text.
func best(ims []tmdbapi.Image, language string) (tmdbapi.Image, bool) {
	ims = slices.DeleteFunc(slices.Clone(ims), func(im tmdbapi.Image) bool {
		return im.Language != language || !scenario.ArtworkFile.MatchString(im.FilePath)
	})
	if len(ims) == 0 {
		return tmdbapi.Image{}, false
	}
	return slices.MinFunc(ims, func(a, b tmdbapi.Image) int {
		return cmp.Or(cmp.Compare(b.VoteAverage, a.VoteAverage), cmp.Compare(b.VoteCount, a.VoteCount), cmp.Compare(a.FilePath, b.FilePath))
	}), true
}

// trailer is the key of the first official YouTube trailer.
func trailer(videos []tmdbapi.Video) string {
	i := slices.IndexFunc(videos, func(v tmdbapi.Video) bool { return v.Site == "YouTube" && v.Type == "Trailer" && v.Official })
	if i < 0 {
		return ""
	}
	return videos[i].Key
}

func dates(r tmdbapi.ReleaseDates) []scenario.TMDBDate {
	var out []scenario.TMDBDate
	for _, country := range countries {
		i := slices.IndexFunc(r.Results, func(c tmdbapi.CountryReleases) bool { return c.Country == country })
		if i < 0 {
			continue
		}
		for _, rel := range r.Results[i].Releases {
			if rel.Type < tmdbapi.Premiere || rel.Type > tmdbapi.TV || len(rel.Date) < len(time.DateOnly) {
				continue
			}
			out = append(out, scenario.TMDBDate{
				Country: country, Kind: scenario.TMDBDateKinds[rel.Type-1], Date: rel.Date[:len(time.DateOnly)],
				Certification: rel.Certification, Note: rel.Note,
			})
		}
	}
	return out
}

func alternatives(titles []tmdbapi.AlternativeTitle) []scenario.TMDBTitle {
	var out []scenario.TMDBTitle
	for _, t := range titles {
		if len(out) < maxTitles && slices.Contains(countries, t.Country) && t.Title != "" {
			out = append(out, scenario.TMDBTitle{Country: t.Country, Title: t.Title, Kind: t.Type})
		}
	}
	return out
}

func translations(ts []tmdbapi.Translation) []scenario.TMDBTranslation {
	var out []scenario.TMDBTranslation
	for _, language := range languages {
		i := slices.IndexFunc(ts, func(t tmdbapi.Translation) bool { return t.Language+"-"+t.Country == language })
		if i < 0 || ts[i].Data == (tmdbapi.TranslationData{}) {
			continue
		}
		d := ts[i].Data
		out = append(out, scenario.TMDBTranslation{Language: language, Title: d.Title, Overview: d.Overview, Tagline: d.Tagline})
	}
	return out
}

// credits is the top-billed cast, the directors and the writers.
func credits(c tmdbapi.Credits) scenario.TMDBCredits {
	var out scenario.TMDBCredits
	cast := slices.SortedStableFunc(slices.Values(c.Cast), func(a, b tmdbapi.Cast) int { return cmp.Compare(a.Order, b.Order) })
	for _, p := range cast[:min(len(cast), maxCast)] {
		out.Cast = append(out.Cast, scenario.TMDBCast{ID: p.ID, Name: p.Name, Character: p.Character, Profile: profile(p.ProfilePath)})
	}
	for _, p := range c.Crew {
		kept := slices.ContainsFunc(out.Crew, func(o scenario.TMDBCrew) bool { return o.ID == p.ID && o.Job == p.Job })
		if !kept && (p.Job == "Director" || p.Department == "Writing") {
			out.Crew = append(out.Crew, scenario.TMDBCrew{ID: p.ID, Name: p.Name, Department: p.Department, Job: p.Job, Profile: profile(p.ProfilePath)})
		}
	}
	return out
}

func profile(p string) string {
	if scenario.ArtworkFile.MatchString(p) {
		return p
	}
	return ""
}

// file is one image to download into the store, at TMDB size size.
type file struct {
	path, size string
}

// files are the images rec names, each once.
func files(rec scenario.TMDBMovie) []file {
	var out []file
	add := func(p, size string) {
		if strings.HasSuffix(p, ".svg") {
			size = "original"
		}
		if f := (file{p, size}); p != "" && !slices.Contains(out, f) {
			out = append(out, f)
		}
	}
	for _, im := range rec.Images {
		add(im.Path, sizes[im.Shape])
	}
	for _, p := range rec.Credits.Cast {
		add(p.Profile, sizes["profile"])
	}
	for _, p := range rec.Credits.Crew {
		add(p.Profile, sizes["profile"])
	}
	return out
}

// download saves f in the store, unless the store has it: a TMDB file
// name never names other bytes.
func (c config) download(ctx context.Context, f file) error {
	dest := filepath.Join(c.store, strings.TrimPrefix(f.path, "/"))
	if _, err := os.Stat(dest); err == nil {
		return nil
	} else if !errors.Is(err, fs.ErrNotExist) {
		return fmt.Errorf("check the store: %w", err)
	}
	body, err := c.get(ctx, c.tmdbImages+"/"+f.size+f.path, nil)
	if err != nil {
		return err
	}
	if err := os.WriteFile(dest, body, 0o644); err != nil {
		return fmt.Errorf("save %s: %w", f.path, err)
	}
	return nil
}

type omdbJSON struct {
	Response   string `json:"Response"`
	Error      string `json:"Error"`
	Awards     string `json:"Awards"`
	Metascore  string `json:"Metascore"`
	IMDbRating string `json:"imdbRating"`
	IMDbVotes  string `json:"imdbVotes"`
	Ratings    []struct {
		Source string `json:"Source"`
		Value  string `json:"Value"`
	} `json:"Ratings"`
}

const notAvailable = "N/A"

// omdb is OMDb's ratings and awards for the title IMDb numbers id.
func (c config) omdb(ctx context.Context, id string) (scenario.OMDbTitle, error) {
	var o omdbJSON
	if err := c.getJSON(ctx, c.omdbAPI, url.Values{"i": {id}, "apikey": {c.omdbKey}}, &o); err != nil {
		return scenario.OMDbTitle{}, err
	}
	if o.Response != "True" {
		return scenario.OMDbTitle{}, fmt.Errorf("OMDb has no title %s: %s", id, o.Error)
	}
	out := scenario.OMDbTitle{IMDbID: id}
	if o.Awards != notAvailable {
		out.Awards = o.Awards
	}
	if v, err := strconv.ParseFloat(o.IMDbRating, 64); err == nil {
		out.IMDbRating = &v
	}
	if v, err := strconv.ParseInt(strings.ReplaceAll(o.IMDbVotes, ",", ""), 10, 64); err == nil {
		out.IMDbVotes = &v
	}
	if v, err := strconv.Atoi(o.Metascore); err == nil {
		out.Metacritic = &v
	}
	for _, r := range o.Ratings {
		if v, err := strconv.Atoi(strings.TrimSuffix(r.Value, "%")); r.Source == "Rotten Tomatoes" && err == nil {
			out.RottenTomatoes = &v
		}
	}
	return out, nil
}

func (c config) getJSON(ctx context.Context, at string, q url.Values, v any) error {
	body, err := c.get(ctx, at, q)
	if err != nil {
		return err
	}
	if err := json.Unmarshal(body, v); err != nil {
		return fmt.Errorf("read %s: %w", at, err)
	}
	return nil
}

// get is the body at at with query q. Errors name at without q, which
// holds the API key.
func (c config) get(ctx context.Context, at string, q url.Values) ([]byte, error) {
	u := at
	if len(q) > 0 {
		u += "?" + q.Encode()
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, http.NoBody)
	if err != nil {
		return nil, fmt.Errorf("GET %s: %w", at, err)
	}
	res, err := c.client.Do(req)
	if err != nil {
		if ue, ok := errors.AsType[*url.Error](err); ok {
			err = ue.Err
		}
		return nil, fmt.Errorf("GET %s: %w", at, err)
	}
	defer func() { _ = res.Body.Close() }()
	body, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, fmt.Errorf("GET %s: %w", at, err)
	}
	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GET %s: %s %s", at, res.Status, body)
	}
	return body, nil
}
