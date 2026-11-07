package main

import (
	"bytes"
	"cmp"
	"encoding/json"
	"image"
	"image/color"
	"image/jpeg"
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

const keepersSeries = `{"status": "success", "data": {
  "id": 81189, "name": "등대지기", "image": "https://artworks.thetvdb.com/banners/v4/series/81189/posters/k33pers.jpg",
  "aliases": [{"language": "eng", "name": "Lantern Keepers"}, {"language": "kor", "name": "등대"}],
  "status": {"id": 2, "name": "Ended", "recordType": "series", "keepUpdated": false},
  "originalCountry": "kor", "originalLanguage": "kor", "overview": "두 사람이 등대를 지킨다.",
  "originalNetwork": {"id": 1, "name": "Harbor One"}, "latestNetwork": {"id": 2, "name": "Stream Nine"},
  "genres": [{"id": 3, "name": "Drama", "slug": "drama"}, {"id": 4, "name": "Mystery", "slug": "mystery"}],
  "remoteIds": [{"id": "tt7001001", "type": 2, "sourceName": "IMDB"}, {"id": "5001", "type": 12, "sourceName": "TheMovieDB.com"}, {"id": "https://keepers.example", "type": 4, "sourceName": "Official Website"}],
  "seasons": [
    {"id": 9000, "number": 0, "name": "Specials", "image": "", "type": {"id": 1, "type": "official"}},
    {"id": 9001, "number": 1, "image": "https://artworks.thetvdb.com/banners/v4/season/9001/posters/s1.jpg", "type": {"id": 1, "type": "official"}},
    {"id": 9002, "number": 2, "name": "The Return", "image": "https://artworks.thetvdb.com/banners/v4/season/9002/posters/s2.webp", "type": {"id": 1, "type": "official"}},
    {"id": 9101, "number": 1, "name": "Disc One", "image": "https://artworks.thetvdb.com/banners/v4/season/9101/posters/d1.jpg", "type": {"id": 2, "type": "dvd"}},
    {"id": 9201, "number": 1, "type": {"id": 4, "type": "alternate"}}
  ],
  "seasonTypes": [{"id": 1, "type": "official"}, {"id": 2, "type": "dvd"}, {"id": 3, "type": "absolute"}, {"id": 4, "type": "alternate"}],
  "translations": {
    "nameTranslations": [{"language": "eng", "name": "Keepers", "isAlias": true}, {"language": "eng", "name": "The Lantern Keepers"}, {"language": "kor", "name": "등대지기", "isPrimary": true}],
    "overviewTranslations": [{"language": "kor", "overview": "두 사람이 등대를 지킨다."}, {"language": "eng", "overview": "Two keepers tend a lighthouse."}]
  }
}}`

const (
	keepersAiredFirst = `{"status": "success", "data": {"series": {"id": 81189}, "episodes": [
  {"id": 701, "seasonNumber": 0, "number": 1, "aired": "2019-12-24", "runtime": 20, "name": "Behind the Light", "overview": "", "image": "https://artworks.thetvdb.com/banners/v4/episode/701/screencap/e701.jpg"},
  {"id": 702, "seasonNumber": 1, "number": 1, "aired": "2019-05-02", "runtime": 52, "name": "The First Night", "overview": "The keepers arrive.", "image": "https://artworks.thetvdb.com/banners/v4/episode/702/screencap/e702.jpg"},
  {"id": 703, "seasonNumber": 1, "number": 2, "aired": "2019-05-09", "runtime": null, "name": "Fog", "overview": "Fog rolls in.", "image": null}
]}, "links": {"prev": null, "self": "%s?page=0", "next": "%s?page=1", "total_items": 4, "page_size": 3}}`
	keepersAiredNext = `{"status": "success", "data": {"series": {"id": 81189}, "episodes": [
  {"id": 704, "seasonNumber": 2, "number": 1, "aired": "2021-01-01", "runtime": 55, "name": "Home", "overview": "They return.", "image": "https://artworks.thetvdb.com/banners/v4/episode/704/screencap/e704.webp"}
]}, "links": {"prev": "%s?page=0", "self": "%s?page=1", "next": null, "total_items": 4, "page_size": 3}}`
	keepersDVD = `{"status": "success", "data": {"series": {"id": 81189}, "episodes": [
  {"id": 702, "seasonNumber": 1, "number": 1}, {"id": 703, "seasonNumber": 1, "number": 2}, {"id": 704, "seasonNumber": 1, "number": 3}, {"id": 799, "seasonNumber": 1, "number": 4}
]}, "links": {"next": null}}`
	keepersAbsolute = `{"status": "success", "data": {"series": {"id": 81189}, "episodes": [
  {"id": 701, "seasonNumber": 0, "number": 1}, {"id": 702, "seasonNumber": 1, "number": 1}, {"id": 703, "seasonNumber": 1, "number": 2}, {"id": 704, "seasonNumber": 1, "number": 3}
]}, "links": {"next": null}}`
)

const keepersTMDB = `{
  "id": 5001, "name": "The Lantern Keepers", "original_name": "등대지기", "original_language": "ko",
  "overview": "Two keepers tend a lighthouse.", "first_air_date": "2019-05-02", "vote_average": 8.2, "vote_count": 410,
  "created_by": [{"id": 9200, "credit_id": "c1", "name": "Ines Calder", "original_name": "Ines Calder", "profile_path": "/inesCalder.jpg"}],
  "networks": [{"id": 49, "name": "Harbor One", "logo_path": "/harborOne.svg", "origin_country": "KR"}, {"id": 50, "name": "Stream Nine", "logo_path": "/streamNine.png", "origin_country": "US"}],
  "credits": {
    "cast": [
      {"id": 9201, "name": "Second Lead", "character": "Bo", "order": 1, "profile_path": null},
      {"id": 9202, "name": "Mara Venn", "character": "Ada", "order": 0, "profile_path": "/maraVenn.jpg"}
    ],
    "crew": [{"id": 9300, "name": "Episode Director", "department": "Directing", "job": "Director", "profile_path": "/director.jpg"}]
  },
  "content_ratings": {"results": [
    {"descriptors": [], "iso_3166_1": "KR", "rating": "15"},
    {"descriptors": [], "iso_3166_1": "DE", "rating": ""},
    {"descriptors": [], "iso_3166_1": "GB", "rating": "15"},
    {"descriptors": [], "iso_3166_1": "US", "rating": "TV-MA"}
  ]},
  "images": {
    "posters": [{"file_path": "/koPoster.jpg", "iso_639_1": "ko", "width": 2000, "height": 3000, "vote_average": 9, "vote_count": 9}, {"file_path": "/keepersPoster.jpg", "iso_639_1": "en", "width": 1000, "height": 1500, "vote_average": 5.5, "vote_count": 4}],
    "backdrops": [{"file_path": "/keepersBackdrop.jpg", "iso_639_1": null, "width": 3840, "height": 2160, "vote_average": 5.2, "vote_count": 3}],
    "logos": []
  },
  "videos": {"results": [{"iso_639_1": "en", "iso_3166_1": "US", "name": "Official Trailer", "key": "keepersTr1", "site": "YouTube", "type": "Trailer", "official": true}]},
  "external_ids": {"imdb_id": "tt7001001", "tvdb_id": 81189}
}`

const (
	keepersOMDb       = `{"Title":"The Lantern Keepers","Year":"2019–2021","Awards":"N/A","Ratings":[{"Source":"Internet Movie Database","Value":"8.3/10"}],"Metascore":"N/A","imdbRating":"8.3","imdbVotes":"52,120","imdbID":"tt7001001","Type":"series","totalSeasons":"3","Response":"True"}`
	keepersOMDbFirst  = `{"Title":"The Lantern Keepers","Season":"1","totalSeasons":"3","Episodes":[{"Title":"The First Night","Released":"2019-05-02","Episode":"1","imdbRating":"8.4","imdbID":"tt7001002"},{"Title":"Fog","Released":"2019-05-09","Episode":"2","imdbRating":"N/A","imdbID":"tt7001003"},{"Title":"Unknown","Released":"N/A","Episode":"3","imdbRating":"N/A","imdbID":"N/A"}],"Response":"True"}`
	keepersOMDbSecond = `{"Title":"The Lantern Keepers","Season":"2","totalSeasons":"3","Episodes":[{"Title":"Home","Released":"2021-01-01","Episode":"1","imdbRating":"9.0","imdbID":"tt7001004"}],"Response":"True"}`
	seasonNotFound    = `{"Response":"False","Error":"Series or season not found!"}`
)

// artworkSizes are the originals TVDB's artwork host serves, by the kind
// of path.
var artworkSizes = map[string][2]int{"/series/": {1000, 1500}, "/season/": {680, 1000}, "/episode/": {1920, 1080}}

type world struct {
	mu        sync.Mutex
	downloads []string
	tvdbIDs   map[string]string
}

func (w *world) note(path string) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.downloads = append(w.downloads, path)
}

func (w *world) took() []string {
	w.mu.Lock()
	defer w.mu.Unlock()
	return slices.Sorted(slices.Values(w.downloads))
}

// servers are TMDB, OMDb and TVDB, each holding one movie or one series.
func (w *world) servers(t *testing.T) config {
	t.Helper()
	serve := func(h http.HandlerFunc) string {
		s := httptest.NewServer(h)
		t.Cleanup(s.Close)
		return s.URL
	}
	tmdbAPI := serve(func(rw http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		body, ok := map[string]string{
			"/3/movie/4101": lanternMovie,
			"/3/tv/5001":    strings.Replace(keepersTMDB, `"tvdb_id": 81189`, `"tvdb_id": `+cmp.Or(w.tvdbIDs["5001"], "81189"), 1),
		}[r.URL.Path]
		if !ok || q.Get("api_key") != "tmdb-key" || q.Get("include_image_language") != "en,null" ||
			r.URL.Path == "/3/tv/5001" && q.Get("append_to_response") != "credits,content_ratings,images,videos,external_ids" {
			http.Error(rw, `{"status_code":34}`, http.StatusNotFound)
			return
		}
		_, _ = rw.Write([]byte(body))
	})
	tmdbImages := serve(func(rw http.ResponseWriter, r *http.Request) {
		w.note(r.URL.Path)
		_, _ = rw.Write([]byte("bytes of " + r.URL.Path)) //nolint:gosec // a test server's plain-text body
	})
	omdb := serve(func(rw http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		body, ok := map[[2]string]string{
			{"tt0004101", ""}: lanternOMDb, {"tt7001001", ""}: keepersOMDb,
			{"tt7001001", "1"}: keepersOMDbFirst, {"tt7001001", "2"}: keepersOMDbSecond,
		}[[2]string{q.Get("i"), q.Get("Season")}]
		if !ok || q.Get("apikey") != "omdb-key" {
			body = seasonNotFound
		}
		_, _ = rw.Write([]byte(body))
	})
	var tvdb string
	tvdb = serve(func(rw http.ResponseWriter, r *http.Request) { tvdbAPI(rw, r, tvdb) })
	artwork := serve(func(rw http.ResponseWriter, r *http.Request) {
		w.note(r.URL.Path)
		for kind, size := range artworkSizes {
			if strings.Contains(r.URL.Path, kind) {
				_, _ = rw.Write(picture(t, size[0], size[1]))
				return
			}
		}
		http.NotFound(rw, r)
	})
	return config{
		tmdbAPI: tmdbAPI + "/3", tmdbImages: tmdbImages + "/t/p", omdbAPI: omdb + "/", tvdbAPI: tvdb + "/v4", tvdbArtwork: artwork,
		tmdbKey: "tmdb-key", omdbKey: "omdb-key", tvdbKey: "tvdb-key", tvdbPin: "4321",
		movies: []int64{4101}, series: []int64{81189}, client: &http.Client{Timeout: 10 * time.Second},
	}
}

// tvdbAPI answers TVDB's login, the series and its episodes in each
// order, at root.
func tvdbAPI(rw http.ResponseWriter, r *http.Request, root string) {
	if r.Method == http.MethodPost && r.URL.Path == "/v4/login" {
		var login struct{ APIKey, Pin string }
		if err := json.NewDecoder(r.Body).Decode(&login); err != nil || login.APIKey != "tvdb-key" || login.Pin != "4321" {
			http.Error(rw, `{"status":"failure","message":"Unauthorized","data":null}`, http.StatusUnauthorized)
			return
		}
		_, _ = rw.Write([]byte(`{"status":"success","data":{"token":"tvdb-token"}}`))
		return
	}
	if r.Header.Get("Authorization") != "Bearer tvdb-token" {
		http.Error(rw, `{"status":"failure","message":"Unauthorized","data":null}`, http.StatusUnauthorized)
		return
	}
	aired := root + "/v4/series/81189/episodes/official/eng"
	body, ok := map[string]string{
		"/v4/series/81189/extended?meta=translations&short=true": keepersSeries,
		"/v4/series/81189/episodes/official/eng?page=0":          strings.ReplaceAll(keepersAiredFirst, "%s", aired),
		"/v4/series/81189/episodes/official/eng?page=1":          strings.ReplaceAll(keepersAiredNext, "%s", aired),
		"/v4/series/81189/episodes/dvd/eng?page=0":               keepersDVD,
		"/v4/series/81189/episodes/absolute/eng?page=0":          keepersAbsolute,
	}[r.URL.Path+"?"+r.URL.RawQuery]
	if !ok {
		http.Error(rw, `{"status":"failure","message":"NotFoundException","data":null}`, http.StatusNotFound)
		return
	}
	_, _ = rw.Write([]byte(body))
}

func picture(t *testing.T, width, height int) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	for y := range height {
		for x := range width {
			img.Set(x, y, color.RGBA{R: uint8(x), G: uint8(y), B: 0x80, A: 0xff})
		}
	}
	var b bytes.Buffer
	if err := jpeg.Encode(&b, img, nil); err != nil {
		t.Fatal(err)
	}
	return b.Bytes()
}

const seriesScenarioFile = `# yaml-language-server: $schema=../scenario.schema.json
description: A Scenario the recorder fills.
seed: real
nodes:
  - name: home
    setup: controller
    roles: [storage, metadata]
tvdb:
  series:
    - {id: 1001, name: A Fictional Series, status: ended}
  token_lifetime: 1h
library:
  roots:
    - {name: series, node: home}
`

func TestRecordWritesASeriesAndAMovieInOneRun(t *testing.T) {
	var w world
	c := w.servers(t)
	dir := t.TempDir()
	c.scenario, c.store = filepath.Join(dir, "real.yaml"), filepath.Join(dir, "files")
	if err := os.WriteFile(c.scenario, []byte(seriesScenarioFile), 0o644); err != nil {
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
		TVDB scenario.TVDB `yaml:"tvdb"`
		OMDb scenario.OMDb `yaml:"omdb"`
	}
	if err := yaml.Unmarshal(first, &got); err != nil {
		t.Fatal(err)
	}
	wantTVDB := scenario.TVDB{TokenLifetime: time.Hour, Series: []scenario.TVDBSeries{{
		ID: 81189, Name: "등대지기", English: "The Lantern Keepers", Aliases: []string{"Lantern Keepers"}, Status: "ended", Network: "Harbor One",
		IMDb: "tt7001001", TMDB: 5001, Overview: "Two keepers tend a lighthouse.", Genres: []string{"Drama", "Mystery"},
		OriginalLanguage: "kor", OriginalCountry: "kor", Image: "/banners/v4/series/81189/posters/k33pers.jpg",
		Seasons: []scenario.TVDBSeason{
			{Number: 0, Name: "Specials", Episodes: []scenario.TVDBEpisode{
				{ID: 701, Number: 1, Name: "Behind the Light", Aired: "2019-12-24", Runtime: 20, Image: "/banners/v4/episode/701/screencap/e701.jpg"},
			}},
			{Number: 1, Image: "/banners/v4/season/9001/posters/s1.jpg", Episodes: []scenario.TVDBEpisode{
				{ID: 702, Number: 1, Name: "The First Night", Aired: "2019-05-02", Runtime: 52, Overview: "The keepers arrive.", Image: "/banners/v4/episode/702/screencap/e702.jpg"},
				{ID: 703, Number: 2, Name: "Fog", Aired: "2019-05-09", Overview: "Fog rolls in."},
			}},
			{Number: 2, Name: "The Return", Episodes: []scenario.TVDBEpisode{
				{ID: 704, Number: 1, Name: "Home", Aired: "2021-01-01", Runtime: 55, Overview: "They return."},
			}},
		},
		DVD:      map[int64]scenario.TVDBNumber{702: {Season: 1, Episode: 1}, 703: {Season: 1, Episode: 2}, 704: {Season: 1, Episode: 3}},
		Absolute: map[int64]int{702: 1, 703: 2, 704: 3},
	}}}
	if !reflect.DeepEqual(got.TVDB, wantTVDB) {
		t.Errorf("tvdb =\n%+v\nwant\n%+v", got.TVDB, wantTVDB)
	}
	wantSeries := []scenario.TMDBSeries{{
		ID: 5001, TVDB: 81189, Name: "The Lantern Keepers", OriginalName: "등대지기", OriginalLanguage: "ko",
		Overview: "Two keepers tend a lighthouse.", FirstAired: "2019-05-02",
		Network:        &scenario.TMDBNetwork{ID: 49, Name: "Harbor One", Logo: "/harborOne.svg"},
		Creators:       []scenario.TMDBCast{{ID: 9200, Name: "Ines Calder", Profile: "/inesCalder.jpg"}},
		Credits:        scenario.TMDBCredits{Cast: []scenario.TMDBCast{{ID: 9202, Name: "Mara Venn", Character: "Ada", Profile: "/maraVenn.jpg"}, {ID: 9201, Name: "Second Lead", Character: "Bo"}}},
		Certifications: []scenario.TMDBCertification{{Country: "US", Rating: "TV-MA"}, {Country: "GB", Rating: "15"}},
		Trailer:        "keepersTr1",
		Images: []scenario.TMDBImage{
			{Shape: "poster", Language: "en", Votes: 5.5, Path: "/keepersPoster.jpg", Width: 1000, Height: 1500},
			{Shape: "backdrop", Votes: 5.2, Path: "/keepersBackdrop.jpg", Width: 3840, Height: 2160},
		},
		VoteAverage: 8.2, VoteCount: 410,
	}}
	if !reflect.DeepEqual(got.TMDB.Series, wantSeries) {
		t.Errorf("tmdb series =\n%+v\nwant\n%+v", got.TMDB.Series, wantSeries)
	}
	if len(got.TMDB.Movies) != 1 || got.TMDB.Movies[0].ID != 4101 {
		t.Errorf("tmdb movies = %+v, want The Lantern Keeper", got.TMDB.Movies)
	}
	wantOMDb := scenario.OMDb{Titles: []scenario.OMDbTitle{
		{IMDbID: "tt0004101", IMDbRating: ptr(7.6), IMDbVotes: ptr[int64](1204311), RottenTomatoes: ptr(91), Metacritic: ptr(78), Awards: "Won 1 BAFTA. 4 wins & 9 nominations total"},
		{IMDbID: "tt7001001", IMDbRating: ptr(8.3), IMDbVotes: ptr[int64](52120), Type: "series", Seasons: []scenario.OMDbSeason{
			{Number: 1, Episodes: []scenario.OMDbEpisode{{Episode: 1, IMDbID: "tt7001002", Rating: ptr(8.4)}, {Episode: 2, IMDbID: "tt7001003"}}},
			{Number: 2, Episodes: []scenario.OMDbEpisode{{Episode: 1, IMDbID: "tt7001004", Rating: ptr(9.0)}}},
		}},
	}}
	if !reflect.DeepEqual(got.OMDb, wantOMDb) {
		t.Errorf("omdb = %+v, want %+v", got.OMDb, wantOMDb)
	}
	if before, _, _ := strings.Cut(seriesScenarioFile, "tvdb:\n"); !strings.HasPrefix(string(first), before) || !strings.Contains(string(first), "\nlibrary:\n  roots:\n    - {name: series, node: home}\n") {
		t.Errorf("the Scenario file =\n%s\nwant every line outside tmdb, tvdb and omdb as it was", first)
	}

	wantDownloads := []string{
		"/banners/v4/episode/701/screencap/e701.jpg", "/banners/v4/episode/702/screencap/e702.jpg",
		"/banners/v4/season/9001/posters/s1.jpg", "/banners/v4/series/81189/posters/k33pers.jpg",
		"/t/p/original/enLogo.svg", "/t/p/original/harborOne.svg", "/t/p/w1280/keepersBackdrop.jpg", "/t/p/w1280/plainBackdrop.jpg",
		"/t/p/w185/adaLumen.jpg", "/t/p/w185/inesCalder.jpg", "/t/p/w185/maraVenn.jpg", "/t/p/w185/topBilled.jpg",
		"/t/p/w500/enPoster.jpg", "/t/p/w500/keepersPoster.jpg",
	}
	if got := w.took(); !reflect.DeepEqual(got, wantDownloads) {
		t.Errorf("downloaded\n%v\nwant\n%v", got, wantDownloads)
	}
	checkScaled(t, c.store, map[string][2]int{
		"tvdb/v4/series/81189/posters/k33pers.jpg": {500, 750},
		"tvdb/v4/season/9001/posters/s1.jpg":       {342, 502},
		"tvdb/v4/episode/702/screencap/e702.jpg":   {400, 225},
		"tvdb/v4/episode/701/screencap/e701.jpg":   {400, 225},
	})
	if data, err := os.ReadFile(filepath.Join(c.store, "harborOne.svg")); err != nil || string(data) != "bytes of /t/p/original/harborOne.svg" {
		t.Errorf("the store's harborOne.svg holds %q, %v; want the network logo", data, err)
	}

	w.downloads = nil
	if err := run(t.Context(), c); err != nil {
		t.Fatal(err)
	}
	again, err := os.ReadFile(c.scenario)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(again, first) || len(w.took()) != 0 {
		t.Errorf("a second run changed the file to\n%s\nand downloaded %v; want no change", again, w.took())
	}
}

// checkScaled checks each JPEG the store holds is scaled down to its
// width and height.
func checkScaled(t *testing.T, store string, want map[string][2]int) {
	t.Helper()
	for name, size := range want {
		data, err := os.ReadFile(filepath.Join(store, filepath.FromSlash(name)))
		if err != nil {
			t.Fatal(err)
		}
		if cfg, format, err := image.DecodeConfig(bytes.NewReader(data)); err != nil || format != "jpeg" || cfg.Width != size[0] || cfg.Height != size[1] {
			t.Errorf("the store's %s is a %dx%d %s, %v; want a %dx%d JPEG", name, cfg.Width, cfg.Height, format, err, size[0], size[1])
		}
	}
}

func TestRecordStopsOnASeriesItCantMatch(t *testing.T) {
	for _, c := range []struct {
		name     string
		tvdbPin  string
		tmdbTVDB string
		want     string
	}{
		{"a refused PIN", "1111", "", "POST "},
		{"a TMDB record naming another TVDB series", "4321", "81190", "TVDB series 81189 names TMDB series 5001, which doesn't name it back"},
	} {
		t.Run(c.name, func(t *testing.T) {
			w := world{tvdbIDs: map[string]string{"5001": c.tmdbTVDB}}
			cfg := w.servers(t)
			cfg.tvdbPin = c.tvdbPin
			cfg.scenario, cfg.store = filepath.Join(t.TempDir(), "real.yaml"), t.TempDir()
			if err := os.WriteFile(cfg.scenario, []byte(seriesScenarioFile), 0o644); err != nil {
				t.Fatal(err)
			}
			err := run(t.Context(), cfg)
			if err == nil || !strings.Contains(err.Error(), c.want) || strings.Contains(err.Error(), "-key") || strings.Contains(err.Error(), "4321") {
				t.Errorf("run = %v; want an error holding %q and no key or PIN", err, c.want)
			}
			if data, _ := os.ReadFile(cfg.scenario); string(data) != seriesScenarioFile {
				t.Errorf("a failed run wrote the Scenario:\n%s", data)
			}
		})
	}
}
