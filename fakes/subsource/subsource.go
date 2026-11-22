// Package subsource is fake-subsource: SubSource's v1 API as the recorded
// run of 2026-11-17 found it (subtitles/online/run-2026-11-17.md). It
// takes the key as X-API-Key or api_key, finds a film or one season of a
// series per movieId, lists a title's subtitles in one language whatever
// seasonNumber and episodeNumber say, and serves zips whose pack members
// are out of episode order. Every 200 counts against three windows per
// key and carries them, with one X-Ratelimit-Reset for the minute's end.
package subsource

import (
	"bytes"
	"cmp"
	"context"
	"encoding/json"
	"fmt"
	"hash/fnv"
	"maps"
	"net/http"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/jatassi/slipstream/fakes/fake"
	"github.com/jatassi/slipstream/fakes/internal/subcatalog"
	"github.com/jatassi/slipstream/fakes/scenario"
	"github.com/jatassi/slipstream/internal/node"
	"github.com/jatassi/slipstream/internal/node/endpoint"
)

var (
	// RevokeKey refuses {"key": "…"} from now on with SubSource's invalid
	// key 401, or, with no key named, every key the fake has seen.
	RevokeKey = fake.CueSpec{Name: "subsource-revoke-key"}
	// Window spends {"window": "minute" | "hour" | "day"} of every key
	// until that window next resets: requests answer 429 with retryAfter.
	Window = fake.CueSpec{Name: "subsource-window"}
)

var Fake = fake.Spec{
	Name: "subsource",
	TLS:  fake.FrontTLS,
	New:  newServer,
	Endpoints: func(e *node.EndpointsBlock, at fake.Address) {
		e.SubSource.API = endpoint.MustParseURL(at.URL(""))
		e.SubSource.KeyPage = endpoint.MustParseURL("https://subsource.invalid/profile")
	},
	Online: fake.OnlineSuite("subsource"),
	Cues:   []fake.CueSpec{RevokeKey, Window},
}

// window is one of a free key's three, as the recorded headers name them.
type window struct {
	name   string
	suffix string
	limit  int
	length time.Duration
}

var windows = []window{
	{"minute", "", 60, time.Minute},
	{"hour", "-Hour", 1800, time.Hour},
	{"day", "-Day", 7200, 24 * time.Hour},
}

type server struct {
	catalog []subcatalog.Subtitle

	mu      sync.Mutex
	seen    map[string]bool
	revoked map[string]bool
	counts  map[string]*counts
	// spent holds each window a cue spent until it resets.
	spent map[string]time.Time
}

// counts are one key's requests in each window and when each window
// opened. The day window opens at UTC midnight; the others at the first
// request after the last closed.
type counts struct {
	used   [3]int
	opened [3]time.Time
}

func newServer(s scenario.Scenario, env fake.Env) (fake.Server, error) {
	srv := &server{catalog: subcatalog.Of(s, env.Name), seen: map[string]bool{}, revoked: map[string]bool{}, counts: map[string]*counts{}, spent: map[string]time.Time{}}
	if err := fake.OnCue(env, RevokeKey.Name, srv.revoke); err != nil {
		return nil, err
	}
	if err := fake.OnCue(env, Window.Name, srv.spend); err != nil {
		return nil, err
	}
	return fake.HTTP(env, fake.Routes{
		"titles":    {Pattern: "GET /api/v1/movies/search", Handler: srv.titles},
		"subtitles": {Pattern: "GET /api/v1/subtitles", Handler: srv.subtitles},
		"download":  {Pattern: "GET /api/v1/subtitles/{id}/download", Handler: srv.download},
		"not-found": {Pattern: "/", Handler: func(w http.ResponseWriter, _ *http.Request) {
			reply(w, http.StatusNotFound, map[string]any{"success": false, "error": "Not found", "message": "Route not found"})
		}},
	}), nil
}

type Key struct {
	Key *string `json:"key"`
}

type revoked struct {
	Revoked int `json:"revoked"`
}

func (s *server) revoke(_ context.Context, a Key) (any, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	keys := slices.Collect(maps.Keys(s.seen))
	if a.Key != nil {
		keys = []string{*a.Key}
	}
	for _, k := range keys {
		s.revoked[k] = true
	}
	return revoked{len(keys)}, nil
}

type Spend struct {
	Window string `json:"window"`
}

type spent struct {
	Until time.Time `json:"until"`
}

func (s *server) spend(_ context.Context, a Spend) (any, error) {
	i := slices.IndexFunc(windows, func(w window) bool { return w.name == a.Window })
	if i < 0 {
		return nil, fmt.Errorf("window is %q, not minute, hour or day", a.Window)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now().UTC()
	until := now.Add(windows[i].length)
	if a.Window == "day" {
		until = now.Truncate(24 * time.Hour).Add(24 * time.Hour)
	}
	s.spent[a.Window] = until
	return spent{Until: until}, nil
}

// admit is the request's key once the key check and the windows let it
// through, counted, or "" with the answer written.
func (s *server) admit(w http.ResponseWriter, r *http.Request) string {
	key := cmp.Or(r.Header.Get("X-API-Key"), r.URL.Query().Get("api_key"))
	s.mu.Lock()
	defer s.mu.Unlock()
	switch {
	case key == "":
		reply(w, http.StatusUnauthorized, map[string]string{"error": "API key required", "message": "Please provide an API key in the X-API-Key header or api_key query parameter"})
		return ""
	case s.revoked[key]:
		reply(w, http.StatusUnauthorized, map[string]string{"error": "Invalid API key", "message": "The provided API key is invalid or inactive"})
		return ""
	}
	s.seen[key] = true
	now := time.Now().UTC()
	c := s.counts[key]
	if c == nil {
		c = &counts{}
		s.counts[key] = c
	}
	wait := time.Duration(0)
	for i, win := range windows {
		opened := now.Truncate(24 * time.Hour)
		if win.name != "day" {
			opened = c.opened[i]
			if !now.Before(opened.Add(win.length)) {
				opened = now
			}
		}
		if !opened.Equal(c.opened[i]) {
			c.opened[i], c.used[i] = opened, 0
		}
		if until, ok := s.spent[win.name]; ok && now.Before(until) {
			c.used[i] = win.limit
			wait = max(wait, until.Sub(now))
		} else if c.used[i] >= win.limit {
			wait = max(wait, opened.Add(win.length).Sub(now))
		}
	}
	s.rated(w, c)
	if wait > 0 {
		seconds := int((wait + time.Second - 1) / time.Second)
		reply(w, http.StatusTooManyRequests, map[string]any{"success": false, "error": "Too many requests", "message": fmt.Sprintf("Rate limit exceeded. Try again in %d seconds.", seconds), "retryAfter": seconds})
		return ""
	}
	for i := range windows {
		c.used[i]++
	}
	s.rated(w, c)
	return key
}

// rated writes the windows' headers as the recorded run saw them. s.mu is
// held.
func (s *server) rated(w http.ResponseWriter, c *counts) {
	h := w.Header()
	for i, win := range windows {
		h.Set("X-Ratelimit-Limit"+win.suffix, strconv.Itoa(win.limit))
		h.Set("X-Ratelimit-Remaining"+win.suffix, strconv.Itoa(max(win.limit-c.used[i], 0)))
	}
	h.Set("X-Ratelimit-Window", "60")
	h.Set("X-Ratelimit-Reset", c.opened[0].Add(time.Minute).Format("2006-01-02T15:04:05.000Z"))
}

// title is one movieId: a film, or one season of a series.
type title struct {
	ID     int
	Name   string
	Year   int
	IMDb   string
	Season int
}

func (s *server) titleOf(c subcatalog.Subtitle) title {
	h := fnv.New32a()
	_, _ = fmt.Fprintf(h, "%s/%d", c.IMDb, c.Season)
	return title{ID: int(h.Sum32()%90000) + 10000, Name: c.Title, Year: c.Year, IMDb: c.IMDb, Season: c.Season}
}

type titleJSON struct {
	MovieID        int               `json:"movieId"`
	Title          string            `json:"title"`
	AlternateTitle *string           `json:"alternateTitle"`
	ReleaseYear    int               `json:"releaseYear"`
	Type           string            `json:"type"`
	IMDbID         string            `json:"imdbId"`
	TMDBID         *int              `json:"tmdbId"`
	Season         *int              `json:"season"`
	SubtitleCount  int               `json:"subtitleCount"`
	Posters        map[string]string `json:"posters"`
	SubsourceLink  string            `json:"subsourceLink"`
}

func (s *server) titles(w http.ResponseWriter, r *http.Request) {
	if s.admit(w, r) == "" {
		return
	}
	q := r.URL.Query()
	season, _ := strconv.Atoi(q.Get("season"))
	found := map[int]*titleJSON{}
	var order []int
	for _, c := range s.catalog {
		t := s.titleOf(c)
		switch q.Get("searchType") {
		case "imdb":
			if t.IMDb != q.Get("imdb") {
				continue
			}
		case "text":
			if !strings.Contains(strings.ToLower(t.Name), strings.ToLower(q.Get("q"))) {
				continue
			}
		default:
			reply(w, http.StatusBadRequest, map[string]any{"success": false, "error": "Invalid parameters", "message": "searchType must be imdb or text"})
			return
		}
		if q.Has("season") && t.Season != season {
			continue
		}
		if found[t.ID] == nil {
			slug := strings.ToLower(strings.ReplaceAll(t.Name, " ", "-"))
			j := &titleJSON{MovieID: t.ID, Title: t.Name, ReleaseYear: t.Year, Type: "movie", IMDbID: t.IMDb, Posters: map[string]string{}, SubsourceLink: "/subtitles/" + slug}
			if t.Season > 0 {
				j.Type, j.Season, j.SubsourceLink = "tvseries", &t.Season, fmt.Sprintf("/subtitles/%s/season-%d", slug, t.Season)
			}
			found[t.ID] = j
			order = append(order, t.ID)
		}
		found[t.ID].SubtitleCount++
	}
	data := []*titleJSON{}
	for _, id := range order {
		data = append(data, found[id])
	}
	reply(w, http.StatusOK, map[string]any{"success": true, "data": data})
}

// subtitleID is the catalog subtitle's id on SubSource.
func subtitleID(c subcatalog.Subtitle) int {
	h := fnv.New32a()
	_, _ = h.Write([]byte(c.ID))
	return int(h.Sum32()%9000000) + 1000000
}

// names are SubSource's language names for the languages a catalog
// lists.
var names = map[string]string{"en": "english", "fr": "french", "de": "german", "es": "spanish", "pt-BR": "brazilian_portuguese", "fa": "farsi_persian"}

type subtitleJSON struct {
	SubtitleID      int      `json:"subtitleId"`
	MovieID         int      `json:"movieId"`
	Language        string   `json:"language"`
	ReleaseInfo     []string `json:"releaseInfo"`
	HearingImpaired bool     `json:"hearingImpaired"`
	ForeignParts    bool     `json:"foreignParts"`
	Framerate       *string  `json:"framerate"`
	ReleaseType     string   `json:"releaseType"`
	ProductionType  string   `json:"productionType"`
	Files           int      `json:"files"`
	Size            int      `json:"size"`
	Downloads       int      `json:"downloads"`
	Rating          struct {
		Good  int `json:"good"`
		Bad   int `json:"bad"`
		Total int `json:"total"`
	} `json:"rating"`
	Contributors []contributorJSON `json:"contributors"`
	UploaderID   int               `json:"uploaderId"`
	Commentary   string            `json:"commentary"`
	Preview      string            `json:"preview"`
	CreatedAt    string            `json:"createdAt"`
	Link         string            `json:"link"`
}

type contributorJSON struct {
	DisplayName string `json:"displayname"`
	ID          int    `json:"id"`
}

func (s *server) subtitles(w http.ResponseWriter, r *http.Request) {
	if s.admit(w, r) == "" {
		return
	}
	q := r.URL.Query()
	movie, _ := strconv.Atoi(q.Get("movieId"))
	limit := min(max(atoi(q.Get("limit"), 50), 1), 100)
	data := []subtitleJSON{}
	for _, c := range s.catalog {
		if s.titleOf(c).ID != movie || names[c.Language] != q.Get("language") {
			continue
		}
		data = append(data, item(c, movie))
	}
	slices.SortFunc(data, func(a, b subtitleJSON) int { return strings.Compare(b.CreatedAt, a.CreatedAt) })
	total := len(data)
	data = data[:min(limit, total)]
	reply(w, http.StatusOK, map[string]any{
		"success": true, "data": data,
		"pagination": map[string]int{"limit": limit, "page": 1, "pages": max((total+limit-1)/limit, 1), "total": total},
		"filters":    map[string]any{"movieId": movie, "language": q.Get("language")},
	})
}

func atoi(s string, otherwise int) int {
	if n, err := strconv.Atoi(s); err == nil {
		return n
	}
	return otherwise
}

func item(c subcatalog.Subtitle, movie int) subtitleJSON {
	id := subtitleID(c)
	out := subtitleJSON{
		SubtitleID: id, MovieID: movie, Language: names[c.Language], ReleaseInfo: []string{c.Release}, HearingImpaired: c.HI,
		ReleaseType: "web", ProductionType: "translated", Files: len(subcatalog.Members(c)), Size: 4096, Downloads: id % 997,
		Contributors: []contributorJSON{{DisplayName: "slipstream", ID: 1}}, UploaderID: 1, Commentary: "", Preview: "1\n00:00:00,500 --> 00:00:03,000\nLine 1",
		CreatedAt: time.Unix(int64(1700000000+id%1000000), 0).UTC().Format("2006-01-02T15:04:05.000Z"),
		Link:      fmt.Sprintf("/subtitle/%s/%s/%d", strings.ToLower(strings.ReplaceAll(c.Title, " ", "-")), names[c.Language], id),
	}
	if c.Pack() {
		out.ReleaseInfo = []string{fmt.Sprintf("%s - Season %d (WEBRip)", c.Title, c.Season)}
	}
	out.Rating.Good, out.Rating.Bad, out.Rating.Total = 9, 1, 10
	return out
}

// members are a download's files. A pack's are named as SubSource's are,
// "<series>  S01E06  <name>.srt", and its later episodes come first.
func members(c subcatalog.Subtitle) []subcatalog.Member {
	ms := subcatalog.Members(c)
	if !c.Pack() {
		return ms
	}
	for i := range ms {
		ms[i].Name = fmt.Sprintf("%s  S%02dE%02d  Episode %d.%s", c.Title, c.Season, ms[i].Episode, ms[i].Episode, c.Format.Ext())
	}
	half := len(ms) / 2
	return append(ms[half:], ms[:half]...)
}

func (s *server) download(w http.ResponseWriter, r *http.Request) {
	i := slices.IndexFunc(s.catalog, func(c subcatalog.Subtitle) bool { return strconv.Itoa(subtitleID(c)) == r.PathValue("id") })
	if s.admit(w, r) == "" {
		return
	}
	if i < 0 {
		reply(w, http.StatusNotFound, map[string]any{"success": false, "error": "Subtitle not found", "message": "No subtitle has this id"})
		return
	}
	c := s.catalog[i]
	b, err := subcatalog.Zip(members(c))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	slug := strings.ToLower(strings.ReplaceAll(c.Title, " ", "-"))
	w.Header().Set("Content-Type", "application/zip")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s_%s_%d-[SubSource].zip"`, slug, names[c.Language], subtitleID(c)))
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(b)
}

// reply writes v as SubSource does: no HTML escapes.
func reply(w http.ResponseWriter, status int, v any) {
	var body bytes.Buffer
	enc := json.NewEncoder(&body)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(v); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_, _ = w.Write(body.Bytes())
}
