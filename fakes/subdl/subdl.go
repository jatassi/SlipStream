// Package subdl is fake-subdl: SubDL's v2 API as the recorded run of
// 2026-11-17 found it (subtitles/online/run-2026-11-17.md). It serves
// /api/v2/me, /api/v2/subtitles/search and /api/v2/subtitles/{id}/download
// on its api host with a strict Bearer key, and names each result's
// keyed url at fake-subdl-dl, which records any request as a divergence:
// SlipStream never follows those urls.
package subdl

import (
	"bytes"
	"context"
	"encoding/json"
	"maps"
	"net/http"
	"net/url"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/jatassi/slipstream/fakes/fake"
	"github.com/jatassi/slipstream/fakes/fault"
	"github.com/jatassi/slipstream/fakes/internal/subcatalog"
	"github.com/jatassi/slipstream/fakes/scenario"
	"github.com/jatassi/slipstream/internal/node"
	"github.com/jatassi/slipstream/internal/node/endpoint"
)

var (
	// RevokeKey refuses {"key": "…"} from now on with 401 unauthorized,
	// or, with no key named, every key the fake has seen.
	RevokeKey = fake.CueSpec{Name: "subdl-revoke-key"}
	// DailyLimit spends the rest of today's downloads of {"key": "…"}, or
	// of every key: downloads answer 429 quota_exceeded until the next
	// UTC midnight.
	DailyLimit = fake.CueSpec{Name: "subdl-daily-limit"}
	// WAF answers the next {"calls": n} requests, 1 when unnamed, with a
	// bare 403 challenge page.
	WAF = fake.CueSpec{Name: "subdl-waf"}
	// Busy answers the next {"calls": n} requests, 1 when unnamed, with 429
	// service_busy.
	Busy = fake.CueSpec{Name: "subdl-busy"}
)

var Fake = fake.Spec{
	Name: "subdl",
	TLS:  fake.FrontTLS,
	New:  newServer,
	Endpoints: func(e *node.EndpointsBlock, at fake.Address) {
		e.SubDL.API = endpoint.MustParseURL(at.URL(""))
		e.SubDL.KeyPage = endpoint.MustParseURL("https://subdl.invalid/panel/api")
	},
	Online: fake.OnlineSuite("subdl"),
	Cues:   []fake.CueSpec{RevokeKey, DailyLimit, WAF, Busy},
}

// The free plan's name and daily allowance, as /me reported them.
const (
	planName        = "Free"
	searchesPerDay  = 2000
	downloadsPerDay = 50
	// searchLag is how many searches /me's counter trails by.
	searchLag = 2
)

const docsURL = "https://subdl.com/developers#errors"

type server struct {
	catalog []subcatalog.Subtitle
	fronts  fake.Fronts

	mu      sync.Mutex
	seen    map[string]bool
	revoked map[string]bool
	// day is the UTC day the counts and caps count in; a new day clears
	// them.
	day       string
	searches  map[string]int
	downloads map[string]int
	capped    map[string]bool
	cappedAll bool
	waf, busy int
}

func newServer(s scenario.Scenario, env fake.Env) (fake.Server, error) {
	srv := &server{catalog: subcatalog.Of(s, env.Name), fronts: env.Fronts, seen: map[string]bool{}, revoked: map[string]bool{}}
	for name, play := range map[string]func(context.Context, Key) (any, error){RevokeKey.Name: srv.revoke, DailyLimit.Name: srv.spend} {
		if err := fake.OnCue(env, name, play); err != nil {
			return nil, err
		}
	}
	if err := fake.OnCue(env, WAF.Name, func(_ context.Context, c Calls) (any, error) { return srv.answerNext(&srv.waf, c), nil }); err != nil {
		return nil, err
	}
	if err := fake.OnCue(env, Busy.Name, func(_ context.Context, c Calls) (any, error) { return srv.answerNext(&srv.busy, c), nil }); err != nil {
		return nil, err
	}
	return fake.HTTP(env, fake.Routes{
		"me":       {Pattern: "GET /api/v2/me", Handler: srv.me},
		"search":   {Pattern: "GET /api/v2/subtitles/search", Handler: srv.search},
		"download": {Pattern: "GET /api/v2/subtitles/{id}/download", Handler: srv.download},
		"not-found": {Pattern: "/", Handler: func(w http.ResponseWriter, _ *http.Request) {
			failure(w, http.StatusNotFound, "not_found", "Not Found")
		}},
	}), nil
}

// Key names the key a cue plays on; nil is every key.
type Key struct {
	Key *string `json:"key"`
}

// Calls is how many requests a cue answers.
type Calls struct {
	Calls int `json:"calls"`
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

type capped struct {
	Until time.Time `json:"until"`
}

func (s *server) spend(_ context.Context, a Key) (any, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := s.today()
	if a.Key == nil {
		s.cappedAll = true
	} else {
		s.capped[*a.Key] = true
	}
	return capped{Until: midnight(now)}, nil
}

type answered struct {
	Calls int `json:"calls"`
}

func (s *server) answerNext(n *int, c Calls) answered {
	s.mu.Lock()
	defer s.mu.Unlock()
	*n += max(c.Calls, 1)
	return answered{*n}
}

// today rolls the counts over at UTC midnight and answers now. s.mu is
// held.
func (s *server) today() time.Time {
	now := time.Now().UTC()
	if day := now.Format(time.DateOnly); day != s.day {
		s.day, s.searches, s.downloads, s.capped, s.cappedAll = day, map[string]int{}, map[string]int{}, map[string]bool{}, false
	}
	return now
}

func midnight(now time.Time) time.Time { return now.UTC().Truncate(24 * time.Hour).Add(24 * time.Hour) }

// stamp is SubDL's ISO 8601 UTC time, as 2026-11-18T00:00:00.000Z.
func stamp(t time.Time) string { return t.UTC().Format("2006-01-02T15:04:05.000Z") }

// key is the request's key once the challenge page, a busy service and
// the key check let it through, or "" with the answer written.
func (s *server) key(w http.ResponseWriter, r *http.Request) string {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.today()
	switch {
	case s.waf > 0:
		s.waf--
		w.Header().Set("Content-Type", "text/html; charset=UTF-8")
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte("<!DOCTYPE html><title>Just a moment...</title>"))
		return ""
	case s.busy > 0:
		s.busy--
		failure(w, http.StatusTooManyRequests, "service_busy", "Service is busy, try again shortly")
		return ""
	}
	key, ok := strings.CutPrefix(r.Header.Get("Authorization"), "Bearer ")
	if !ok {
		key = r.Header.Get("X-API-Key")
	}
	switch {
	case key == "":
		failure(w, http.StatusUnauthorized, "unauthorized", "Missing API key. Send it as an `Authorization: Bearer <key>` or `X-API-Key` header.")
		return ""
	case s.revoked[key]:
		failure(w, http.StatusUnauthorized, "unauthorized", "Not Authorized")
		return ""
	}
	s.seen[key] = true
	return key
}

type windowJSON struct {
	Used      int    `json:"used"`
	Limit     int    `json:"limit"`
	Remaining int    `json:"remaining"`
	Period    string `json:"period"`
	ResetAt   string `json:"reset_at"`
}

func window(used, limit int, reset time.Time) windowJSON {
	return windowJSON{Used: used, Limit: limit, Remaining: max(limit-used, 0), Period: "day", ResetAt: stamp(reset)}
}

func (s *server) me(w http.ResponseWriter, r *http.Request) {
	key := s.key(w, r)
	if key == "" {
		return
	}
	s.mu.Lock()
	now := s.today()
	searches, downloads := max(s.searches[key]-searchLag, 0), s.downloads[key]
	s.mu.Unlock()
	month := time.Date(now.Year(), now.Month()+1, 1, 0, 0, 0, 0, time.UTC)
	reply(w, http.StatusOK, map[string]any{
		"user": map[string]any{"id": 1, "name": "slipstream", "username": nil},
		"plan": map[string]any{"name": planName},
		"usage": map[string]any{
			"search":    window(searches, searchesPerDay, midnight(now)),
			"downloads": window(downloads, downloadsPerDay, midnight(now)),
			"ai": map[string]any{"translations": map[string]any{
				"period": "month", "reset_at": stamp(month), "quota": map[string]any{"used": 0, "limit": 0, "remaining": 0},
			}},
		},
	})
}

type unpackJSON struct {
	Name    string `json:"name"`
	URL     string `json:"url"`
	Season  int    `json:"season"`
	Episode *int   `json:"episode"`
	Format  string `json:"format"`
}

type subtitleJSON struct {
	Author       string       `json:"author"`
	Comment      string       `json:"comment"`
	Episode      *int         `json:"episode"`
	EpisodeEnd   *int         `json:"episode_end"`
	EpisodeFrom  *int         `json:"episode_from"`
	FPS          *float64     `json:"fps"`
	Framerate    *string      `json:"framerate"`
	FullSeason   bool         `json:"full_season"`
	HI           bool         `json:"hi"`
	Lang         string       `json:"lang"`
	Language     string       `json:"language"`
	Name         string       `json:"name"`
	ReleaseName  string       `json:"release_name"`
	Releases     []string     `json:"releases"`
	Season       *int         `json:"season"`
	SubtitlePage string       `json:"subtitlePage"`
	UnpackFiles  []unpackJSON `json:"unpack_files"`
	URL          string       `json:"url"`
}

type resultJSON struct {
	SDID   int    `json:"sd_id"`
	Type   string `json:"type"`
	Name   string `json:"name"`
	IMDbID string `json:"imdb_id"`
	TMDBID int    `json:"tmdb_id"`
}

// names are SubDL's lang words for the languages a catalog lists.
var names = map[string]string{"en": "english", "fr": "french", "de": "german", "es": "spanish"}

func (s *server) search(w http.ResponseWriter, r *http.Request) {
	key := s.key(w, r)
	if key == "" {
		return
	}
	q := r.URL.Query()
	s.mu.Lock()
	now := s.today()
	s.searches[key]++
	remaining := max(searchesPerDay-s.searches[key], 0)
	s.mu.Unlock()
	langs := strings.Split(q.Get("languages"), ",")
	tv := q.Get("type") == "tv"
	season, _ := strconv.Atoi(q.Get("season"))
	episode, _ := strconv.Atoi(q.Get("episode"))
	dl := s.dlHost(r.Context())
	out := map[string]any{"status": true, "results": []resultJSON{}, "subtitles": []subtitleJSON{}}
	var found []subtitleJSON
	for _, c := range s.catalog {
		switch {
		case q.Get("imdb_id") != "" && c.IMDb != q.Get("imdb_id"),
			q.Get("tmdb_id") != "" && c.TMDB != q.Get("tmdb_id"),
			q.Get("imdb_id") == "" && q.Get("tmdb_id") == "",
			tv != (c.Season > 0),
			tv && q.Has("season") && c.Season != season,
			tv && q.Has("episode") && !slices.Contains(c.Episodes, episode),
			!slices.Contains(langs, code(c.Language)):
			continue
		}
		found = append(found, item(c, dl, key))
	}
	if len(found) > 0 {
		out["subtitles"] = found
		kind := "movie"
		if tv {
			kind = "tv"
		}
		out["results"] = []resultJSON{{SDID: 1, Type: kind, IMDbID: q.Get("imdb_id")}}
	}
	w.Header().Set("X-Ratelimit-Limit", strconv.Itoa(searchesPerDay))
	w.Header().Set("X-Ratelimit-Remaining", strconv.Itoa(remaining))
	w.Header().Set("X-Ratelimit-Reset", stamp(midnight(now)))
	reply(w, http.StatusOK, out)
}

// code is a catalog language as SubDL's language field writes it.
func code(lang string) string {
	if lang == "pt-BR" {
		return "BR_PT"
	}
	return strings.ToUpper(lang)
}

// aiComment is what SubDL's comment says of a machine translation.
const aiComment = "ترجمه ماشینی"

func item(c subcatalog.Subtitle, dl, key string) subtitleJSON {
	keyed := func(path string) string {
		return (&url.URL{Scheme: "https", Host: dl, Path: path, RawQuery: url.Values{"api_key": {key}}.Encode()}).String()
	}
	out := subtitleJSON{
		Author: "slipstream", HI: c.HI, Lang: names[c.Language], Language: code(c.Language), Name: "SUBDL::" + c.Release + ".zip",
		ReleaseName: c.Release, Releases: []string{c.Release}, SubtitlePage: "/s/info/" + c.ID, URL: keyed("/subtitle/" + c.ID + ".zip"),
	}
	if c.AI {
		out.Author, out.Comment = "SUBDL AI", aiComment
	}
	if c.Season > 0 {
		out.Season = &c.Season
	}
	switch {
	case c.Pack():
		out.EpisodeFrom, out.EpisodeEnd = &c.Episodes[0], &c.Episodes[len(c.Episodes)-1]
		out.FullSeason = c.Season%2 == 1
	case len(c.Episodes) == 1:
		out.Episode = &c.Episodes[0]
	}
	for i, m := range members(c) {
		u := unpackJSON{Name: m.Name, URL: keyed("/subtitle/" + c.ID + "/" + strconv.Itoa(i) + "/" + url.PathEscape(m.Name)), Format: c.Format.Ext()}
		if c.Pack() && m.Episode > 0 {
			u.Episode = &m.Episode
		}
		out.UnpackFiles = append(out.UnpackFiles, u)
	}
	return out
}

// dlHost is fake-subdl-dl's front for the request's origin, or SubDL's
// own dl host outside a world.
func (s *server) dlHost(ctx context.Context) string {
	if at, ok := s.fronts.Find(fault.Target(DL.Name), fake.OriginOf(ctx)); ok {
		return at.AddrPort.String()
	}
	return "dl.subdl.invalid"
}

func (s *server) download(w http.ResponseWriter, r *http.Request) {
	key := s.key(w, r)
	if key == "" {
		return
	}
	i := slices.IndexFunc(s.catalog, func(c subcatalog.Subtitle) bool { return c.ID == r.PathValue("id") })
	if i < 0 {
		failure(w, http.StatusNotFound, "not_found", "Subtitle not found")
		return
	}
	s.mu.Lock()
	s.today()
	spent := s.cappedAll || s.capped[key] || s.downloads[key] >= downloadsPerDay
	if !spent {
		s.downloads[key]++
	}
	s.mu.Unlock()
	if spent {
		failure(w, http.StatusTooManyRequests, "quota_exceeded", "Daily download quota exceeded")
		return
	}
	b, err := subcatalog.Zip(members(s.catalog[i]))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/zip")
	w.Header().Set("Content-Disposition", "attachment; filename*=UTF-8''"+url.PathEscape(s.catalog[i].Release+".zip"))
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(b)
}

// members are a download's files: a pack opens with SubDL's notice.
func members(c subcatalog.Subtitle) []subcatalog.Member {
	if c.Pack() {
		return append([]subcatalog.Member{subcatalog.Junk(c)}, subcatalog.Members(c)...)
	}
	return subcatalog.Members(c)
}

func failure(w http.ResponseWriter, status int, code, message string) {
	reply(w, status, map[string]any{"error": map[string]string{"code": code, "message": message, "docs_url": docsURL}})
}

// reply writes v as SubDL does: no HTML escapes.
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
