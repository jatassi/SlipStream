// Package gestdown is fake-gestdown: Gestdown, Addic7ed's keyless TV
// proxy, as the recorded run of 2026-11-17 found it
// (subtitles/online/run-2026-11-17.md) and its server's code describes
// it. Show and subtitle ids are 22 base62 characters, a season pack's
// "sp_" and 22. A not-found answer is a bare JSON string and stays not
// found for 12 h, as Gestdown's cache keeps it. Downloads are raw SRT
// with CRLF line ends, or a pack's zip. Each origin has its own token
// bucket, whose 429 carries Retry-After.
package gestdown

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"math/big"
	"net/http"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"

	"golang.org/x/time/rate"

	"github.com/jatassi/slipstream/fakes/fake"
	"github.com/jatassi/slipstream/fakes/internal/subcatalog"
	"github.com/jatassi/slipstream/fakes/scenario"
	"github.com/jatassi/slipstream/internal/node"
	"github.com/jatassi/slipstream/internal/node/endpoint"
)

var (
	// Refreshing answers the next {"calls": n} episode searches, 1 when
	// unnamed, with 423: the show is being refreshed.
	Refreshing = fake.CueSpec{Name: "gestdown-refreshing"}
	// UpstreamOut answers every download from now on with a 429 whose JSON
	// error says Gestdown's Addic7ed accounts are out.
	UpstreamOut = fake.CueSpec{Name: "gestdown-upstream-out"}
	// Air lists the unaired catalog subtitle {"id": "…"} from now on,
	// though a search that found nothing stays cached for 12 h.
	Air = fake.CueSpec{Name: "gestdown-air"}
	// Bucket spends every origin's token bucket, so the next request
	// answers 429 with Retry-After.
	Bucket = fake.CueSpec{Name: "gestdown-bucket"}
)

var Fake = fake.Spec{
	Name: "gestdown",
	TLS:  fake.FrontTLS,
	New:  newServer,
	Endpoints: func(e *node.EndpointsBlock, at fake.Address) {
		e.Gestdown.API = endpoint.MustParseURL(at.URL(""))
	},
	Online: fake.OnlineSuite("gestdown"),
	Cues:   []fake.CueSpec{Refreshing, UpstreamOut, Air, Bucket},
}

// The shipped bucket: 50 tokens a minute, 200 deep.
const (
	refill = 1200 * time.Millisecond
	depth  = 200
)

// notFoundFor is how long Gestdown's cache keeps a not-found answer.
const notFoundFor = 12 * time.Hour

type server struct {
	catalog []subcatalog.Subtitle

	mu         sync.Mutex
	aired      map[string]bool
	missed     map[string]time.Time
	buckets    map[string]*rate.Limiter
	refreshing int
	upstream   bool
}

func newServer(s scenario.Scenario, env fake.Env) (fake.Server, error) {
	srv := &server{catalog: subcatalog.Of(s, env.Name), aired: map[string]bool{}, missed: map[string]time.Time{}, buckets: map[string]*rate.Limiter{}}
	for _, err := range []error{
		fake.OnCue(env, Refreshing.Name, srv.refresh),
		fake.OnCue(env, UpstreamOut.Name, srv.out),
		fake.OnCue(env, Air.Name, srv.air),
		fake.OnCue(env, Bucket.Name, srv.drain),
	} {
		if err != nil {
			return nil, err
		}
	}
	return fake.HTTP(env, fake.Routes{
		"show":      {Pattern: "GET /shows/external/tvdb/{tvdb}", Handler: srv.limited(srv.show)},
		"episode":   {Pattern: "GET /subtitles/get/{show}/{season}/{episode}/{lang}", Handler: srv.limited(srv.episode)},
		"download":  {Pattern: "GET /subtitles/download/{id}", Handler: srv.limited(srv.download)},
		"not-found": {Pattern: "/", Handler: func(w http.ResponseWriter, _ *http.Request) { reply(w, http.StatusNotFound, "Not found") }},
	}), nil
}

type Calls struct {
	Calls int `json:"calls"`
}

type answered struct {
	Calls int `json:"calls"`
}

func (s *server) refresh(_ context.Context, c Calls) (any, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.refreshing += max(c.Calls, 1)
	return answered{s.refreshing}, nil
}

type none struct{}

func (s *server) out(context.Context, none) (any, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.upstream = true
	return none{}, nil
}

type ID struct {
	ID string `json:"id"`
}

func (s *server) air(_ context.Context, a ID) (any, error) {
	if !slices.ContainsFunc(s.catalog, func(c subcatalog.Subtitle) bool { return c.ID == a.ID && c.Unaired }) {
		return nil, fmt.Errorf("the catalog has no unaired subtitle %q", a.ID)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.aired[a.ID] = true
	return none{}, nil
}

type drained struct {
	Origins int `json:"origins"`
}

func (s *server) drain(context.Context, none) (any, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now()
	for _, b := range s.buckets {
		b.AllowN(now, int(b.TokensAt(now)))
	}
	return drained{len(s.buckets)}, nil
}

// limited takes one token from the origin's bucket before h, or answers
// 429 with the seconds until the next token.
func (s *server) limited(h http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		origin := fake.OriginOf(r.Context())
		s.mu.Lock()
		b := s.buckets[origin]
		if b == nil {
			b = rate.NewLimiter(rate.Every(refill), depth)
			s.buckets[origin] = b
		}
		now := time.Now()
		ok := b.AllowN(now, 1)
		wait := refill - time.Duration(b.TokensAt(now)*float64(refill))
		s.mu.Unlock()
		if !ok {
			w.Header().Set("Retry-After", strconv.Itoa(int((wait+time.Second-1)/time.Second)))
			w.Header().Set("X-Retry-Reason", "Too many requests from this address")
			w.WriteHeader(http.StatusTooManyRequests)
			return
		}
		h(w, r)
	}
}

// notFound answers path's cached not-found when one is held, or with miss
// a fresh one, cached for 12 h. It answers whether it answered.
func (s *server) notFound(w http.ResponseWriter, path string, miss bool, words string) bool {
	s.mu.Lock()
	now := time.Now()
	at, cached := s.missed[path]
	cached = cached && now.Before(at.Add(notFoundFor))
	if miss && !cached {
		s.missed[path], cached = now, true
	}
	s.mu.Unlock()
	if cached {
		w.Header().Set("Cache-Control", "public, max-age=43200")
		reply(w, http.StatusNotFound, words)
	}
	return cached
}

// base62 is 22 characters of base 62 drawn from seed: Gestdown's id
// shape.
func base62(seed string) string {
	const digits = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"
	sum := sha256.Sum256([]byte(seed))
	v := new(big.Int).SetBytes(sum[:])
	out := make([]byte, 22)
	base, digit := big.NewInt(62), new(big.Int)
	for i := range out {
		v.DivMod(v, base, digit)
		out[i] = digits[digit.Int64()]
	}
	return string(out)
}

func showID(tvdb string) string { return base62("show/" + tvdb) }

func subtitleID(c subcatalog.Subtitle) string {
	if c.Pack() {
		return "sp_" + base62("pack/"+c.ID)
	}
	return base62("subtitle/" + c.ID)
}

func (s *server) listed(c subcatalog.Subtitle) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return !c.Unaired || s.aired[c.ID]
}

type showJSON struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	NbSeasons int    `json:"nbSeasons"`
	Seasons   []int  `json:"seasons"`
	TVDBID    int    `json:"tvDbId"`
	TMDBID    *int   `json:"tmdbId"`
	Slug      string `json:"slug"`
}

func (s *server) show(w http.ResponseWriter, r *http.Request) {
	tvdb := r.PathValue("tvdb")
	words := "Couldn't find show: " + tvdb
	if s.notFound(w, r.URL.Path, false, words) {
		return
	}
	var show *showJSON
	for _, c := range s.catalog {
		if c.TVDB != tvdb {
			continue
		}
		if show == nil {
			id, _ := strconv.Atoi(tvdb)
			show = &showJSON{ID: showID(tvdb), Name: c.Title, TVDBID: id, Slug: strings.ToLower(strings.ReplaceAll(c.Title, " ", "-"))}
			if tmdb, err := strconv.Atoi(c.TMDB); err == nil {
				show.TMDBID = &tmdb
			}
		}
		if !slices.Contains(show.Seasons, c.Season) {
			show.Seasons = append(show.Seasons, c.Season)
		}
	}
	if show == nil {
		s.notFound(w, r.URL.Path, true, words)
		return
	}
	slices.Sort(show.Seasons)
	show.NbSeasons = len(show.Seasons)
	w.Header().Set("Cache-Control", "public, max-age=14400")
	reply(w, http.StatusOK, map[string]any{"shows": []*showJSON{show}})
}

type subtitleJSON struct {
	SubtitleID      string   `json:"subtitleId"`
	Version         string   `json:"version"`
	Completed       bool     `json:"completed"`
	HearingImpaired bool     `json:"hearingImpaired"`
	Corrected       bool     `json:"corrected"`
	HD              bool     `json:"hd"`
	DownloadURI     string   `json:"downloadUri"`
	Language        string   `json:"language"`
	Discovered      string   `json:"discovered"`
	DownloadCount   int      `json:"downloadCount"`
	Source          string   `json:"source"`
	Qualities       []string `json:"qualities"`
	Release         *string  `json:"release"`
}

// names are Gestdown's language words for the languages a catalog lists.
var names = map[string]string{"en": "English", "fr": "French", "de": "German", "es": "Spanish", "pt-BR": "Portuguese (Brazilian)"}

func (s *server) episode(w http.ResponseWriter, r *http.Request) {
	s.mu.Lock()
	refreshing := s.refreshing > 0
	if refreshing {
		s.refreshing--
	}
	s.mu.Unlock()
	if refreshing {
		reply(w, http.StatusLocked, "Show is being refreshed, try again later")
		return
	}
	const words = "Couldn't find any subtitles for this episode"
	if s.notFound(w, r.URL.Path, false, words) {
		return
	}
	season, _ := strconv.Atoi(r.PathValue("season"))
	episode, _ := strconv.Atoi(r.PathValue("episode"))
	var singles, packs []subcatalog.Subtitle
	title := ""
	for _, c := range s.catalog {
		if showID(c.TVDB) != r.PathValue("show") || c.Season != season || !slices.Contains(c.Episodes, episode) || c.Language != r.PathValue("lang") || !s.listed(c) {
			continue
		}
		title = c.Title
		if c.Pack() {
			packs = append(packs, c)
		} else {
			singles = append(singles, c)
		}
	}
	found := singles
	if len(found) == 0 {
		found = packs
	}
	if len(found) == 0 {
		s.notFound(w, r.URL.Path, true, words)
		return
	}
	var matching []subtitleJSON
	for _, c := range found {
		m := subtitleJSON{
			SubtitleID: subtitleID(c), Version: c.Release, Completed: true, HearingImpaired: c.HI, HD: true,
			DownloadURI: "/subtitles/download/" + subtitleID(c), Language: names[c.Language], Discovered: "2026-11-01T00:00:00Z",
			DownloadCount: 120, Source: "Addic7ed", Qualities: []string{"1080p"}, Release: &c.Release,
		}
		matching = append(matching, m)
		m.SubtitleID, m.Completed, m.Version = base62("unfinished/"+c.ID), false, c.Release+" WIP"
		m.DownloadURI = "/subtitles/download/" + m.SubtitleID
		matching = append(matching, m)
	}
	w.Header().Set("Cache-Control", "public, max-age=7200")
	reply(w, http.StatusOK, map[string]any{
		"matchingSubtitles": matching,
		"episode":           map[string]any{"season": season, "number": episode, "title": fmt.Sprintf("Episode %d", episode), "show": title, "discovered": "2026-11-01T00:00:00Z"},
	})
}

func (s *server) download(w http.ResponseWriter, r *http.Request) {
	s.mu.Lock()
	out := s.upstream
	s.mu.Unlock()
	if out {
		reply(w, http.StatusTooManyRequests, map[string]string{"error": "No more Addic7ed credentials available, try again later"})
		return
	}
	i := slices.IndexFunc(s.catalog, func(c subcatalog.Subtitle) bool { return subtitleID(c) == r.PathValue("id") && s.listed(c) })
	if i < 0 {
		reply(w, http.StatusNotFound, "Subtitle not found")
		return
	}
	c := s.catalog[i]
	w.Header().Set("X-Robots-Tag", "noindex, nofollow")
	w.Header().Set("Cache-Control", "public, max-age=691200")
	if c.Pack() {
		b, err := subcatalog.Zip(subcatalog.Members(c))
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/zip")
		w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s.%s.zip"`, c.Release, c.Language))
		_, _ = w.Write(b)
		return
	}
	w.Header().Set("Content-Type", "text/srt")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s.%s.srt"`, c.Release, c.Language))
	_, _ = w.Write(bytes.ReplaceAll(subcatalog.Members(c)[0].Bytes, []byte("\n"), []byte("\r\n")))
}

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
