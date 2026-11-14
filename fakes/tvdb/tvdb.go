// Package tvdb is fake-tvdb: TVDB API v4's login, series search, extended
// series records, episodes in each order, updates list and languages,
// from the Scenario's tvdb catalog. It takes any key, and any PIN beside
// it, until a cue rejects them, and answers every call but login with 401
// unless it carries a live token.
package tvdb

import (
	"cmp"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"hash/fnv"
	"maps"
	"net/http"
	"path"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/jatassi/slipstream/fakes/fake"
	"github.com/jatassi/slipstream/fakes/scenario"
	"github.com/jatassi/slipstream/internal/node"
	"github.com/jatassi/slipstream/internal/node/endpoint"
)

var (
	// ExpireTokens ends every token the fake has issued, so the next call
	// on each answers 401 and a new login works.
	ExpireTokens = fake.CueSpec{Name: "tvdb-expire-tokens"}
	// RejectKey refuses logins with {"key": "…"} from now on, with TVDB's
	// body for a bad key, and ends its tokens; with no key named, every
	// key the fake has seen.
	RejectKey = fake.CueSpec{Name: "tvdb-reject-key"}
	// RejectPin refuses logins that send a PIN beside {"key": "…"}, with a
	// body that names the PIN, and ends their tokens; with no key named,
	// beside every key.
	RejectPin = fake.CueSpec{Name: "tvdb-reject-pin"}
	// Change makes the record of series {"id": n} differ on its next
	// fetch and lists it as updated now.
	Change = fake.CueSpec{Name: "tvdb-change"}
	// Delete has TVDB drop series {"id": n}: it answers 404 and is listed
	// as deleted now.
	Delete = fake.CueSpec{Name: "tvdb-delete"}
	// AddSeason gives series {"id": n} its next Aired season, of as many
	// episodes as its last, one airing each week from tomorrow. An ended
	// series returns: it reads continuing.
	AddSeason = fake.CueSpec{Name: "tvdb-add-season"}
	// MoveEpisode moves episode {"id": n} to {"season": s, "number": e} in
	// Aired order, as a special TVDB places in a season.
	MoveEpisode = fake.CueSpec{Name: "tvdb-move-episode"}
	// RemoveEpisode drops episode {"id": n} from every order.
	RemoveEpisode = fake.CueSpec{Name: "tvdb-remove-episode"}
)

var Fake = fake.Spec{
	Name: "tvdb",
	TLS:  fake.FrontTLS,
	New:  newServer,
	Endpoints: func(e *node.EndpointsBlock, at fake.Address) {
		e.TVDB.API = endpoint.MustParseURL(at.URL(root))
		e.TVDB.KeyPage = endpoint.MustParseURL("https://thetvdb.invalid/api-information")
	},
	Online: fake.OnlineSuite("tvdb"),
	Cues:   []fake.CueSpec{ExpireTokens, RejectKey, RejectPin, Change, Delete, AddSeason, MoveEpisode, RemoveEpisode},
}

const (
	root = "/v4"
	// artworkHost is where TVDB's records say their artwork lives;
	// fake-image-cdn serves the same paths.
	artworkHost = "https://artworks.thetvdb.com"
	// episodesPage, updatesPage and searchLimit are TVDB's page lengths.
	episodesPage = 500
	updatesPage  = 1000
	searchLimit  = 50
	// tokenLifetime is TVDB's own month.
	tokenLifetime = 30 * 24 * time.Hour
	// addedEpisodesFirstID is the first id an episode AddSeason makes
	// takes, past every block the Scenario reserves.
	addedEpisodesFirstID = scenario.TVDBGeneratedFirstID + 9_000_000
)

type server struct {
	catalog  scenario.TVDB
	lifetime time.Duration

	mu     sync.Mutex
	issued int
	tokens map[string]session
	seen   map[string]bool
	// rejectedKeys refuse a login; rejectedPins refuse one with a PIN.
	rejectedKeys map[string]bool
	rejectedPins map[string]bool
	// edited holds each series a cue reshaped, whole.
	edited map[int64]scenario.TVDBSeries
	// orders holds each series' episodes in an order, as episodesIn
	// sorts them, since a client pages a long series one request at a
	// time. An edit or a change drops the series' entries.
	orders   map[orderKey][]placed
	revision map[int64]int
	deleted  map[int64]bool
	updates  []update
	added    int64
}

type session struct {
	key, pin string
	expires  time.Time
}

type update struct {
	id      int64
	deleted bool
	at      time.Time
}

func newServer(s scenario.Scenario, env fake.Env) (fake.Server, error) {
	srv := &server{
		lifetime: tokenLifetime, tokens: map[string]session{}, seen: map[string]bool{}, rejectedKeys: map[string]bool{}, rejectedPins: map[string]bool{},
		edited: map[int64]scenario.TVDBSeries{}, orders: map[orderKey][]placed{}, revision: map[int64]int{}, deleted: map[int64]bool{}, added: addedEpisodesFirstID,
	}
	if s.TVDB != nil {
		srv.catalog = *s.TVDB
		srv.lifetime = cmp.Or(s.TVDB.TokenLifetime, tokenLifetime)
	}
	plays := []error{
		fake.OnCue(env, ExpireTokens.Name, srv.expire),
		fake.OnCue(env, RejectKey.Name, func(_ context.Context, a Key) (any, error) { return srv.reject(a, srv.rejectedKeys, false), nil }),
		fake.OnCue(env, RejectPin.Name, func(_ context.Context, a Key) (any, error) { return srv.reject(a, srv.rejectedPins, true), nil }),
		fake.OnCue(env, Change.Name, func(_ context.Context, a ID) (any, error) { return srv.change(a.ID) }),
		fake.OnCue(env, Delete.Name, func(_ context.Context, a ID) (any, error) { return srv.remove(a.ID) }),
		fake.OnCue(env, AddSeason.Name, func(_ context.Context, a ID) (any, error) { return srv.addSeason(a.ID) }),
		fake.OnCue(env, MoveEpisode.Name, func(_ context.Context, a Move) (any, error) { return srv.moveEpisode(a) }),
		fake.OnCue(env, RemoveEpisode.Name, func(_ context.Context, a ID) (any, error) { return srv.removeEpisode(a.ID) }),
	}
	for _, err := range plays {
		if err != nil {
			return nil, err
		}
	}
	return fake.HTTP(env, fake.Routes{
		"login":     {Pattern: "POST " + root + "/login", Handler: srv.login},
		"search":    {Pattern: "GET " + root + "/search", Handler: srv.authed(srv.search)},
		"series":    {Pattern: "GET " + root + "/series/{id}/extended", Handler: srv.authed(srv.series)},
		"episodes":  {Pattern: "GET " + root + "/series/{id}/episodes/{order}/{lang}", Handler: srv.authed(srv.episodes)},
		"updates":   {Pattern: "GET " + root + "/updates", Handler: srv.authed(srv.updated)},
		"languages": {Pattern: "GET " + root + "/languages", Handler: srv.authed(srv.languages)},
		"not-found": {Pattern: "/", Handler: func(w http.ResponseWriter, _ *http.Request) { failure(w, http.StatusNotFound, notFoundWords) }},
	}), nil
}

// Key names the key a RejectKey or RejectPin cue plays on; nil is every
// key seen.
type Key struct {
	Key *string `json:"key"`
}

// ID names the series, or for RemoveEpisode the episode, a cue plays on.
type ID struct {
	ID int64 `json:"id"`
}

// Move names an episode and its new place in Aired order.
type Move struct {
	ID     int64 `json:"id"`
	Season int   `json:"season"`
	Number int   `json:"number"`
}

type ended struct {
	Ended int `json:"ended"`
}

func (s *server) expire(context.Context, struct{}) (any, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	n := len(s.tokens)
	clear(s.tokens)
	return ended{n}, nil
}

type rejected struct {
	Rejected int `json:"rejected"`
	Ended    int `json:"ended"`
}

// reject adds a's keys to into and ends each token they hold, only those
// with a PIN when pins is set.
func (s *server) reject(a Key, into map[string]bool, pins bool) rejected {
	s.mu.Lock()
	defer s.mu.Unlock()
	keys := slices.Collect(maps.Keys(s.seen))
	if a.Key != nil {
		keys = []string{*a.Key}
	}
	out := rejected{Rejected: len(keys)}
	for _, k := range keys {
		into[k] = true
	}
	for t, sess := range s.tokens {
		if slices.Contains(keys, sess.key) && (!pins || sess.pin != "") {
			delete(s.tokens, t)
			out.Ended++
		}
	}
	return out
}

const (
	unauthorized  = "Unauthorized"
	pinRefused    = "Unauthorized: the subscriber PIN does not match this key"
	notFoundWords = "NotFoundException: the record was not found"
)

func (s *server) login(w http.ResponseWriter, r *http.Request) {
	var body struct {
		APIKey string `json:"apikey"`
		Pin    string `json:"pin"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		failure(w, http.StatusBadRequest, "InvalidBody: "+err.Error())
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if body.APIKey != "" {
		s.seen[body.APIKey] = true
	}
	switch {
	case body.APIKey == "" || s.rejectedKeys[body.APIKey]:
		failure(w, http.StatusUnauthorized, unauthorized)
		return
	case body.Pin != "" && s.rejectedPins[body.APIKey]:
		failure(w, http.StatusUnauthorized, pinRefused)
		return
	}
	s.issued++
	expires := time.Now().Add(s.lifetime)
	t := jwt(s.issued, expires)
	s.tokens[t] = session{key: body.APIKey, pin: body.Pin, expires: expires}
	reply(w, http.StatusOK, envelope{Data: struct {
		Token string `json:"token"`
	}{t}, Status: "success"})
}

// jwt is the n-th token, a JWT whose payload claims expires as exp.
func jwt(n int, expires time.Time) string {
	part := func(v any) string {
		b, err := json.Marshal(v)
		if err != nil {
			panic(err)
		}
		return base64.RawURLEncoding.EncodeToString(b)
	}
	return part(map[string]string{"alg": "HS256", "typ": "JWT"}) + "." +
		part(map[string]any{"exp": expires.Unix(), "id": "fake-tvdb", "n": n}) + "." +
		base64.RawURLEncoding.EncodeToString(fmt.Appendf(nil, "fake-tvdb signature %d", n))
}

// authed answers 401 before h runs unless the request carries a live
// token.
func (s *server) authed(h http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		t, ok := strings.CutPrefix(r.Header.Get("Authorization"), "Bearer ")
		s.mu.Lock()
		sess, live := s.tokens[t]
		live = ok && live && time.Now().Before(sess.expires)
		s.mu.Unlock()
		if !live {
			failure(w, http.StatusUnauthorized, unauthorized)
			return
		}
		h(w, r)
	}
}

// live is series id as TVDB holds it now, with its revision.
func (s *server) live(id int64) (scenario.TVDBSeries, int, bool) {
	s.mu.Lock()
	edited, ok := s.edited[id]
	rev, gone := s.revision[id], s.deleted[id]
	s.mu.Unlock()
	if gone {
		return scenario.TVDBSeries{}, 0, false
	}
	if ok {
		return edited, rev, true
	}
	series, ok := s.catalog.Find(id)
	return series, rev, ok
}

type changed struct {
	Revision int  `json:"revision"`
	Deleted  bool `json:"deleted"`
}

func (s *server) change(id int64) (any, error) {
	if _, _, ok := s.live(id); !ok {
		return nil, fmt.Errorf("tvdb lists no series %d", id)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.revision[id]++
	s.forget(id)
	s.updates = append(s.updates, update{id: id, at: time.Now()})
	return changed{Revision: s.revision[id]}, nil
}

func (s *server) remove(id int64) (any, error) {
	if _, _, ok := s.live(id); !ok {
		return nil, fmt.Errorf("tvdb lists no series %d", id)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.deleted[id] = true
	s.updates = append(s.updates, update{id: id, deleted: true, at: time.Now()})
	return changed{Revision: s.revision[id], Deleted: true}, nil
}

type orderKey struct {
	series int64
	order  int
}

// ordered is series' episodes in order t.
func (s *server) ordered(series scenario.TVDBSeries, t seasonType) []placed {
	k := orderKey{series.ID, t.ID}
	s.mu.Lock()
	defer s.mu.Unlock()
	out, ok := s.orders[k]
	if !ok {
		out = episodesIn(series, t)
		s.orders[k] = out
	}
	return out
}

// forget drops series id's orders. Callers hold s.mu.
func (s *server) forget(id int64) {
	for k := range s.orders {
		if k.series == id {
			delete(s.orders, k)
		}
	}
}

// edit reshapes series id with f and lists it as updated now.
func (s *server) edit(id int64, f func(*scenario.TVDBSeries) error) error {
	series, _, ok := s.live(id)
	if !ok {
		return fmt.Errorf("tvdb lists no series %d", id)
	}
	series.Seasons = slices.Clone(series.Seasons)
	for i := range series.Seasons {
		series.Seasons[i].Episodes = slices.Clone(series.Seasons[i].Episodes)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := f(&series); err != nil {
		return err
	}
	s.edited[id] = series
	s.forget(id)
	s.updates = append(s.updates, update{id: id, at: time.Now()})
	return nil
}

type added struct {
	Season   int     `json:"season"`
	Episodes []int64 `json:"episodes"`
}

func (s *server) addSeason(id int64) (any, error) {
	var out added
	err := s.edit(id, func(series *scenario.TVDBSeries) error {
		next, count := 1, 1
		for _, season := range series.Seasons {
			if season.Number >= next {
				next, count = season.Number+1, max(len(season.Episodes), 1)
			}
		}
		day := time.Now().UTC().AddDate(0, 0, 1)
		season := scenario.TVDBSeason{Number: next}
		for n := range count {
			season.Episodes = append(season.Episodes, scenario.TVDBEpisode{
				ID: s.added, Number: n + 1, Name: fmt.Sprintf("Season %d, Part %d", next, n+1), Aired: day.AddDate(0, 0, 7*n).Format(time.DateOnly), Runtime: 45,
			})
			out.Episodes = append(out.Episodes, s.added)
			s.added++
		}
		series.Seasons = append(series.Seasons, season)
		if series.Status == "ended" {
			series.Status = "continuing"
		}
		out.Season = next
		return nil
	})
	return out, err
}

type moved struct {
	Series int64 `json:"series"`
}

func (s *server) moveEpisode(m Move) (any, error) {
	id, err := s.seriesOf(m.ID)
	if err != nil {
		return nil, err
	}
	return moved{id}, s.edit(id, func(series *scenario.TVDBSeries) error {
		e := take(series, m.ID)
		e.Number = m.Number
		at := slices.IndexFunc(series.Seasons, func(season scenario.TVDBSeason) bool { return season.Number == m.Season })
		if at < 0 {
			series.Seasons = append(series.Seasons, scenario.TVDBSeason{Number: m.Season})
			at = len(series.Seasons) - 1
		}
		series.Seasons[at].Episodes = append(series.Seasons[at].Episodes, e)
		return nil
	})
}

func (s *server) removeEpisode(episode int64) (any, error) {
	id, err := s.seriesOf(episode)
	if err != nil {
		return nil, err
	}
	return moved{id}, s.edit(id, func(series *scenario.TVDBSeries) error {
		take(series, episode)
		delete(series.DVD, episode)
		delete(series.Absolute, episode)
		return nil
	})
}

// take removes episode id from series' Aired seasons and answers it.
func take(series *scenario.TVDBSeries, id int64) scenario.TVDBEpisode {
	for i, season := range series.Seasons {
		if j := slices.IndexFunc(season.Episodes, func(e scenario.TVDBEpisode) bool { return e.ID == id }); j >= 0 {
			e := season.Episodes[j]
			series.Seasons[i].Episodes = slices.Delete(season.Episodes, j, j+1)
			return e
		}
	}
	return scenario.TVDBEpisode{}
}

// seriesOf is the live series that lists episode id.
func (s *server) seriesOf(id int64) (int64, error) {
	for series := range s.catalog.All() {
		if live, _, ok := s.live(series.ID); ok && slices.ContainsFunc(live.Seasons, func(season scenario.TVDBSeason) bool {
			return slices.ContainsFunc(season.Episodes, func(e scenario.TVDBEpisode) bool { return e.ID == id })
		}) {
			return series.ID, nil
		}
	}
	return 0, fmt.Errorf("tvdb lists no episode %d", id)
}

// envelope is every TVDB answer.
type envelope struct {
	Status string     `json:"status"`
	Data   any        `json:"data"`
	Links  *linksJSON `json:"links,omitempty"`
}

type linksJSON struct {
	Prev       *string `json:"prev"`
	Self       *string `json:"self"`
	Next       *string `json:"next"`
	TotalItems int     `json:"total_items"`
	PageSize   int     `json:"page_size"`
}

// pageLinks are the links of page (from 0) of total items size to a page,
// each a URL of r with only its page changed.
func pageLinks(r *http.Request, page, total, size int) *linksJSON {
	at := func(p int) *string {
		u := *r.URL
		u.Scheme, u.Host = "https", r.Host
		q := u.Query()
		q.Set("page", strconv.Itoa(p))
		u.RawQuery = q.Encode()
		s := u.String()
		return &s
	}
	l := &linksJSON{Self: at(page), TotalItems: total, PageSize: size}
	if page > 0 {
		l.Prev = at(page - 1)
	}
	if (page+1)*size < total {
		l.Next = at(page + 1)
	}
	return l
}

func pageParam(v string) int {
	if n, err := strconv.Atoi(v); err == nil && n > 0 {
		return n
	}
	return 0
}

func failure(w http.ResponseWriter, status int, words string) {
	reply(w, status, struct {
		Status  string `json:"status"`
		Message string `json:"message"`
		Data    any    `json:"data"`
	}{"failure", words, nil})
}

func reply(w http.ResponseWriter, status int, v any) {
	body, err := json.Marshal(v)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_, _ = w.Write(body)
}

func hash(parts ...string) uint64 {
	h := fnv.New64a()
	for _, p := range parts {
		_, _ = h.Write([]byte(p))
		_, _ = h.Write([]byte{0})
	}
	return h.Sum64()
}

// recorded is the URL of the artwork path p a Scenario recorded, else the
// one artwork makes up.
func recorded(p, record string, id int64, kind string) string {
	if p != "" {
		return artworkHost + p
	}
	return artwork(record, id, kind)
}

// thumbnail is the URL of the _t thumbnail TVDB keeps beside the image at
// u.
func thumbnail(u string) string {
	ext := path.Ext(u)
	return strings.TrimSuffix(u, ext) + "_t" + ext
}

// artwork is the URL TVDB names an image of kind for record by, fixed for
// both.
func artwork(record string, id int64, kind string) string {
	key := strconv.FormatInt(id, 10)
	return artworkHost + "/banners/v4/" + record + "/" + key + "/" + kind + "/" + strconv.FormatUint(hash(record, key, kind), 36) + ".jpg"
}
