package main

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"go.yaml.in/yaml/v3"

	"github.com/jatassi/slipstream/fakes/scenario"
)

const lanternMovie = `{
  "id": 4101, "imdb_id": "tt0004101", "title": "The Lantern Keeper", "original_title": "Le Gardien du phare",
  "original_language": "fr", "overview": "A keeper tends the last lantern.", "tagline": "Keep it lit.",
  "release_date": "2011-05-04", "runtime": 118, "status": "Released", "vote_average": 7.4, "vote_count": 2210,
  "genres": [{"id": 18, "name": "Drama"}, {"id": 9648, "name": "Mystery"}],
  "production_companies": [{"id": 610, "name": "Tidewater Pictures", "logo_path": "/tidewater.png", "origin_country": "FR"}, {"id": 611, "name": "Second Co", "logo_path": null, "origin_country": "US"}],
  "belongs_to_collection": {"id": 720, "name": "Lantern Collection", "poster_path": null, "backdrop_path": null},
  "external_ids": {"imdb_id": "tt0004101"},
  "release_dates": {"results": [
    {"iso_3166_1": "JP", "release_dates": [{"certification": "G", "iso_639_1": "", "note": "", "release_date": "2011-09-01T00:00:00.000Z", "type": 3}]},
    {"iso_3166_1": "FR", "release_dates": [{"certification": "TP", "iso_639_1": "fr", "note": "Cannes", "release_date": "2011-05-04T00:00:00.000Z", "type": 1}]},
    {"iso_3166_1": "US", "release_dates": [
      {"certification": "PG-13", "iso_639_1": "", "note": "", "release_date": "2011-10-07T00:00:00.000Z", "type": 3},
      {"certification": "", "iso_639_1": "", "note": "", "release_date": "2012-01-17T00:00:00.000Z", "type": 4}
    ]}
  ]},
  "alternative_titles": {"titles": [
    {"iso_3166_1": "JP", "title": "Ranpu", "type": ""},
    {"iso_3166_1": "US", "title": "Lantern Keeper", "type": "short title"},
    {"iso_3166_1": "DE", "title": "Der Leuchtturmwärter", "type": ""}
  ]},
  "credits": {
    "cast": [
      {"id": 9002, "name": "Second Billed", "character": "Mate", "order": 1, "profile_path": null},
      {"id": 9001, "name": "Top Billed", "character": "Keeper", "order": 0, "profile_path": "/topBilled.jpg"},
      {"id": 9003, "name": "Third", "character": "c3", "order": 2, "profile_path": null},
      {"id": 9004, "name": "Fourth", "character": "c4", "order": 3, "profile_path": null},
      {"id": 9005, "name": "Fifth", "character": "c5", "order": 4, "profile_path": null},
      {"id": 9006, "name": "Sixth", "character": "c6", "order": 5, "profile_path": null},
      {"id": 9007, "name": "Seventh", "character": "c7", "order": 6, "profile_path": null},
      {"id": 9008, "name": "Eighth", "character": "c8", "order": 7, "profile_path": null},
      {"id": 9009, "name": "Ninth", "character": "c9", "order": 8, "profile_path": null},
      {"id": 9010, "name": "Tenth", "character": "c10", "order": 9, "profile_path": null},
      {"id": 9011, "name": "Eleventh", "character": "c11", "order": 10, "profile_path": "/eleventh.jpg"}
    ],
    "crew": [
      {"id": 9100, "name": "Ada Lumen", "department": "Directing", "job": "Director", "profile_path": "/adaLumen.jpg"},
      {"id": 9100, "name": "Ada Lumen", "department": "Writing", "job": "Screenplay", "profile_path": "/adaLumen.jpg"},
      {"id": 9101, "name": "Bo Quill", "department": "Writing", "job": "Novel", "profile_path": null},
      {"id": 9102, "name": "Cy Money", "department": "Production", "job": "Producer", "profile_path": "/cyMoney.jpg"},
      {"id": 9103, "name": "Di Second", "department": "Directing", "job": "Assistant Director", "profile_path": null}
    ]
  },
  "videos": {"results": [
    {"iso_639_1": "en", "iso_3166_1": "US", "name": "Teaser", "key": "teaserKey", "site": "YouTube", "type": "Teaser", "official": true},
    {"iso_639_1": "en", "iso_3166_1": "US", "name": "Fan cut", "key": "fanKey", "site": "YouTube", "type": "Trailer", "official": false},
    {"iso_639_1": "en", "iso_3166_1": "US", "name": "Trailer", "key": "vimeoKey", "site": "Vimeo", "type": "Trailer", "official": true},
    {"iso_639_1": "en", "iso_3166_1": "US", "name": "Official Trailer", "key": "lanternTr1", "site": "YouTube", "type": "Trailer", "official": true}
  ]},
  "images": {
    "posters": [
      {"file_path": "/frPoster.jpg", "iso_639_1": "fr", "width": 2000, "height": 3000, "vote_average": 9.1, "vote_count": 30},
      {"file_path": "/enPosterLow.jpg", "iso_639_1": "en", "width": 2000, "height": 3000, "vote_average": 5.1, "vote_count": 3},
      {"file_path": "/enPoster.jpg", "iso_639_1": "en", "width": 1400, "height": 2100, "vote_average": 5.6, "vote_count": 12}
    ],
    "backdrops": [
      {"file_path": "/enBackdrop.jpg", "iso_639_1": "en", "width": 3840, "height": 2160, "vote_average": 9.0, "vote_count": 9},
      {"file_path": "/plainBackdrop.jpg", "iso_639_1": null, "width": 1920, "height": 1080, "vote_average": 4.2, "vote_count": 5}
    ],
    "logos": [
      {"file_path": "/enLogo.svg", "iso_639_1": "en", "width": 1200, "height": 380, "vote_average": 5.3, "vote_count": 2}
    ]
  },
  "translations": {"translations": [
    {"iso_639_1": "en", "iso_3166_1": "US", "data": {"title": "", "overview": "A keeper tends the last lantern.", "tagline": ""}},
    {"iso_639_1": "fr", "iso_3166_1": "FR", "data": {"title": "Le Gardien du phare", "overview": "Un gardien veille.", "tagline": ""}},
    {"iso_639_1": "es", "iso_3166_1": "MX", "data": {"title": "El faro", "overview": "", "tagline": ""}},
    {"iso_639_1": "it", "iso_3166_1": "IT", "data": {"title": "Il faro", "overview": "", "tagline": ""}},
    {"iso_639_1": "de", "iso_3166_1": "DE", "data": {"title": "", "overview": "", "tagline": ""}}
  ]}
}`

const sparseMovie = `{"id": 4007, "imdb_id": "tt0004007", "title": "Short Film", "original_title": "Short Film", "original_language": "en", "runtime": 12, "status": "Released",
  "genres": [], "production_companies": [], "belongs_to_collection": null, "external_ids": {"imdb_id": "tt0004007"},
  "release_dates": {"results": []}, "alternative_titles": {"titles": []}, "credits": {"cast": [], "crew": []},
  "videos": {"results": []}, "images": {"posters": [], "backdrops": [], "logos": []}, "translations": {"translations": []}}`

const lanternOMDb = `{"Title":"The Lantern Keeper","Year":"2011","Awards":"Won 1 BAFTA. 4 wins & 9 nominations total","Ratings":[{"Source":"Internet Movie Database","Value":"7.6/10"},{"Source":"Rotten Tomatoes","Value":"91%"},{"Source":"Metacritic","Value":"78/100"}],"Metascore":"78","imdbRating":"7.6","imdbVotes":"1,204,311","imdbID":"tt0004101","Type":"movie","Response":"True"}`

const sparseOMDb = `{"Title":"Short Film","Year":"2020","Awards":"N/A","Ratings":[],"Metascore":"N/A","imdbRating":"N/A","imdbVotes":"N/A","imdbID":"tt0004007","Type":"movie","Response":"True"}`

const scenarioFile = `# yaml-language-server: $schema=../scenario.schema.json
description: >-
  A folded description that runs long enough to wrap across two lines in
  the file, which a full rewrite would unwrap.
seed: real
nodes:
  - name: home
    setup: controller
    roles: [storage, metadata]
tmdb:
  # Hand-written movies the recorder replaces.
  movies:
    - id: 9000001
      imdb_id: tt9000001
      title: The Quiet Orchard
      released: 2021-04-16
      runtime: 104m
  generated: {count: 3, seed: kept}

library:
  roots:
    - {name: movies, node: home}
`

type upstream struct {
	mu     sync.Mutex
	images []string
}

func (u *upstream) servers(t *testing.T) config {
	t.Helper()
	serve := func(h http.HandlerFunc) string {
		s := httptest.NewServer(h)
		t.Cleanup(s.Close)
		return s.URL
	}
	movies := map[string]string{"4101": lanternMovie, "4007": sparseMovie}
	titles := map[string]string{"tt0004101": lanternOMDb, "tt0004007": sparseOMDb}
	tmdbAPI := serve(func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		body, ok := movies[strings.TrimPrefix(r.URL.Path, "/3/movie/")]
		if !ok || q.Get("api_key") != "tmdb-key" || q.Get("include_image_language") != "en,null" ||
			q.Get("append_to_response") != "external_ids,release_dates,alternative_titles,credits,videos,images,translations" {
			http.Error(w, `{"status_code":34}`, http.StatusNotFound)
			return
		}
		_, _ = w.Write([]byte(body))
	})
	images := serve(func(w http.ResponseWriter, r *http.Request) {
		u.mu.Lock()
		u.images = append(u.images, r.URL.Path)
		u.mu.Unlock()
		_, _ = w.Write([]byte("bytes of " + r.URL.Path)) //nolint:gosec // a test server's plain-text body
	})
	omdb := serve(func(w http.ResponseWriter, r *http.Request) {
		body, ok := titles[r.URL.Query().Get("i")]
		if !ok || r.URL.Query().Get("apikey") != "omdb-key" {
			body = `{"Response":"False","Error":"Movie not found!"}`
		}
		_, _ = w.Write([]byte(body))
	})
	return config{
		tmdbAPI: tmdbAPI + "/3", tmdbImages: images + "/t/p", omdbAPI: omdb + "/",
		tmdbKey: "tmdb-key", omdbKey: "omdb-key", movies: []int64{4101, 4007, 4101}, client: &http.Client{Timeout: 10 * time.Second},
	}
}

func ptr[T any](v T) *T { return &v }

func TestRecordWritesTheMoviesTheirRatingsAndTheirImages(t *testing.T) {
	var u upstream
	c := u.servers(t)
	dir := t.TempDir()
	c.scenario, c.store = filepath.Join(dir, "real.yaml"), filepath.Join(dir, "files")
	if err := os.Mkdir(c.store, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(c.scenario, []byte(scenarioFile), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := run(t.Context(), c); err != nil {
		t.Fatal(err)
	}
	first, err := os.ReadFile(c.scenario)
	if err != nil {
		t.Fatal(err)
	}

	var got struct {
		TMDB scenario.TMDB `yaml:"tmdb"`
		OMDb scenario.OMDb `yaml:"omdb"`
	}
	if err := yaml.Unmarshal(first, &got); err != nil {
		t.Fatal(err)
	}
	want := scenario.TMDB{Generated: &scenario.TMDBGenerated{Count: 3, Seed: "kept"}, Movies: []scenario.TMDBMovie{
		{ID: 4007, IMDbID: "tt0004007", Title: "Short Film", OriginalTitle: "Short Film", OriginalLanguage: "en", Runtime: 12 * time.Minute, Status: "Released"},
		{
			ID: 4101, IMDbID: "tt0004101", Title: "The Lantern Keeper", OriginalTitle: "Le Gardien du phare", OriginalLanguage: "fr",
			Overview: "A keeper tends the last lantern.", Tagline: "Keep it lit.", Genres: []string{"Drama", "Mystery"}, Status: "Released",
			Runtime: 118 * time.Minute, VoteAverage: 7.4, VoteCount: 2210, Trailer: "lanternTr1",
			Dates: []scenario.TMDBDate{
				{Country: "US", Kind: "cinema", Date: "2011-10-07", Certification: "PG-13"},
				{Country: "US", Kind: "digital", Date: "2012-01-17"},
				{Country: "FR", Kind: "premiere", Date: "2011-05-04", Certification: "TP", Note: "Cannes"},
			},
			Titles:       []scenario.TMDBTitle{{Country: "US", Title: "Lantern Keeper", Kind: "short title"}, {Country: "DE", Title: "Der Leuchtturmwärter"}},
			Translations: []scenario.TMDBTranslation{{Language: "fr-FR", Title: "Le Gardien du phare", Overview: "Un gardien veille."}},
			Credits: scenario.TMDBCredits{
				Cast: []scenario.TMDBCast{
					{ID: 9001, Name: "Top Billed", Character: "Keeper", Profile: "/topBilled.jpg"}, {ID: 9002, Name: "Second Billed", Character: "Mate"},
					{ID: 9003, Name: "Third", Character: "c3"}, {ID: 9004, Name: "Fourth", Character: "c4"}, {ID: 9005, Name: "Fifth", Character: "c5"},
					{ID: 9006, Name: "Sixth", Character: "c6"}, {ID: 9007, Name: "Seventh", Character: "c7"}, {ID: 9008, Name: "Eighth", Character: "c8"},
					{ID: 9009, Name: "Ninth", Character: "c9"}, {ID: 9010, Name: "Tenth", Character: "c10"},
				},
				Crew: []scenario.TMDBCrew{
					{ID: 9100, Name: "Ada Lumen", Department: "Directing", Job: "Director", Profile: "/adaLumen.jpg"},
					{ID: 9100, Name: "Ada Lumen", Department: "Writing", Job: "Screenplay", Profile: "/adaLumen.jpg"},
					{ID: 9101, Name: "Bo Quill", Department: "Writing", Job: "Novel"},
				},
			},
			Collection: &scenario.TMDBNamed{ID: 720, Name: "Lantern Collection"},
			Studio:     &scenario.TMDBNamed{ID: 610, Name: "Tidewater Pictures"},
			Images: []scenario.TMDBImage{
				{Shape: "poster", Language: "en", Votes: 5.6, Path: "/enPoster.jpg", Width: 1400, Height: 2100},
				{Shape: "backdrop", Votes: 4.2, Path: "/plainBackdrop.jpg", Width: 1920, Height: 1080},
				{Shape: "logo", Language: "en", Votes: 5.3, Path: "/enLogo.svg", Width: 1200, Height: 380},
			},
		},
	}}
	if !reflect.DeepEqual(got.TMDB, want) {
		t.Errorf("tmdb =\n%+v\nwant\n%+v", got.TMDB, want)
	}
	wantOMDb := scenario.OMDb{Titles: []scenario.OMDbTitle{
		{IMDbID: "tt0004007"},
		{IMDbID: "tt0004101", IMDbRating: ptr(7.6), IMDbVotes: ptr[int64](1204311), RottenTomatoes: ptr(91), Metacritic: ptr(78), Awards: "Won 1 BAFTA. 4 wins & 9 nominations total"},
	}}
	if !reflect.DeepEqual(got.OMDb, wantOMDb) {
		t.Errorf("omdb = %+v, want %+v", got.OMDb, wantOMDb)
	}

	before, _, _ := strings.Cut(scenarioFile, "tmdb:\n")
	if !strings.HasPrefix(string(first), before) || !strings.Contains(string(first), "\n\nlibrary:\n  roots:\n    - {name: movies, node: home}\n") {
		t.Errorf("the Scenario file =\n%s\nwant every line outside tmdb and omdb as it was", first)
	}
	if !strings.Contains(string(first), "  # Hand-written movies the recorder replaces.\n  movies:\n") {
		t.Errorf("the Scenario file =\n%s\nwant the comment in tmdb kept", first)
	}

	wantImages := []string{"/t/p/original/enLogo.svg", "/t/p/w1280/plainBackdrop.jpg", "/t/p/w185/adaLumen.jpg", "/t/p/w185/topBilled.jpg", "/t/p/w500/enPoster.jpg"}
	if got := slices.Sorted(slices.Values(u.images)); !reflect.DeepEqual(got, wantImages) {
		t.Errorf("downloaded %v, want %v", got, wantImages)
	}
	for _, p := range wantImages {
		name := p[strings.LastIndex(p, "/")+1:]
		if data, err := os.ReadFile(filepath.Join(c.store, name)); err != nil || string(data) != "bytes of "+p {
			t.Errorf("the store's %s holds %q, %v; want the bytes of %s", name, data, err, p)
		}
	}

	u.images = nil
	if err := run(t.Context(), c); err != nil {
		t.Fatal(err)
	}
	again, err := os.ReadFile(c.scenario)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(again, first) || len(u.images) != 0 {
		t.Errorf("a second run changed the file to\n%s\nand downloaded %v; want no change", again, u.images)
	}
}

func TestRecordStopsOnAMovieTMDBOrOMDbLacks(t *testing.T) {
	var u upstream
	for _, c := range []struct {
		id      int64
		omdbKey string
		want    string
	}{
		{4999, "omdb-key", "/3/movie/4999: 404 Not Found"},
		{4007, "spent-key", "OMDb has no title tt0004007: Movie not found!"},
	} {
		cfg := u.servers(t)
		cfg.movies, cfg.omdbKey = []int64{c.id}, c.omdbKey
		cfg.scenario, cfg.store = filepath.Join(t.TempDir(), "real.yaml"), t.TempDir()
		if err := os.WriteFile(cfg.scenario, []byte(scenarioFile), 0o644); err != nil {
			t.Fatal(err)
		}
		err := run(t.Context(), cfg)
		if err == nil || !strings.Contains(err.Error(), c.want) || strings.Contains(err.Error(), "-key") {
			t.Errorf("run = %v; want an error holding %q and no API key", err, c.want)
		}
		if data, _ := os.ReadFile(cfg.scenario); string(data) != scenarioFile {
			t.Errorf("a failed run wrote the Scenario:\n%s", data)
		}
	}
}
