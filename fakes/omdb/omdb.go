// Package omdb is fake-omdb: OMDb's lookup of a title by IMDb id, with
// the ratings and awards the Scenario's omdb section lists. It takes any
// API key or none until a cue revokes it, and answers OMDb's daily cap once
// a key has spent the Scenario's daily limit or a cue spends it.
package omdb

import (
	"bytes"
	"cmp"
	"context"
	"encoding/json"
	"maps"
	"net/http"
	"regexp"
	"slices"
	"strconv"
	"sync"
	"time"

	"github.com/jatassi/slipstream/fakes/fake"
	"github.com/jatassi/slipstream/fakes/scenario"
	"github.com/jatassi/slipstream/internal/node"
	"github.com/jatassi/slipstream/internal/node/endpoint"
)

var (
	// RevokeKey refuses {"key": "…"} from now on with OMDb's bad-key
	// answer, or, with no key named, every key the fake has seen, none
	// included.
	RevokeKey = fake.CueSpec{Name: "omdb-revoke-key"}
	// Cap spends the rest of today's allowance of {"key": "…"}, or, with no
	// key named, of every key: OMDb answers "Request limit reached!" until
	// the next UTC midnight.
	Cap = fake.CueSpec{Name: "omdb-cap"}
)

var Fake = fake.Spec{
	Name: "omdb",
	TLS:  fake.FrontTLS,
	New:  newServer,
	Endpoints: func(e *node.EndpointsBlock, at fake.Address) {
		e.OMDb.API = endpoint.MustParseURL(at.URL(""))
		e.OMDb.KeyPage = endpoint.MustParseURL("https://www.omdb.invalid/apikey.aspx")
	},
	Online: fake.OnlineSuite("omdb"),
	Cues:   []fake.CueSpec{RevokeKey, Cap},
}

type server struct {
	titles map[string]scenario.OMDbTitle
	names  map[string]movieName
	limit  int

	mu      sync.Mutex
	seen    map[string]bool
	revoked map[string]bool
	// day is the UTC day calls and caps count in; a new day clears both.
	day       string
	calls     map[string]int
	capped    map[string]bool
	cappedAll bool
}

// movieName is what OMDb calls a title it also knows from TMDB's catalog.
type movieName struct {
	title, year string
}

func newServer(s scenario.Scenario, env fake.Env) (fake.Server, error) {
	srv := &server{titles: map[string]scenario.OMDbTitle{}, names: map[string]movieName{}, seen: map[string]bool{}, revoked: map[string]bool{}}
	if s.OMDb != nil {
		srv.limit = s.OMDb.DailyLimit
		for _, t := range s.OMDb.Titles {
			srv.titles[t.IMDbID] = t
		}
	}
	if s.TMDB != nil {
		for _, m := range s.TMDB.Movies {
			srv.names[m.IMDbID] = movieName{title: m.Title, year: year(m)}
		}
	}
	if err := fake.OnCue(env, RevokeKey.Name, srv.revoke); err != nil {
		return nil, err
	}
	if err := fake.OnCue(env, Cap.Name, srv.spend); err != nil {
		return nil, err
	}
	return fake.HTTP(env, fake.Routes{
		"title":     {Pattern: "GET /{$}", Handler: srv.title},
		"not-found": {Pattern: "/", Handler: http.NotFound},
	}), nil
}

// year is the year of the movie's earliest date, as OMDb's Year reads.
func year(m scenario.TMDBMovie) string {
	var days []string
	if m.Released != "" {
		days = append(days, m.Released)
	}
	for _, d := range m.Dates {
		days = append(days, d.Date)
	}
	if len(days) == 0 {
		return notAvailable
	}
	return slices.Min(days)[:4]
}

// Key names the key a cue plays on; nil is every key.
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
	return capped{Until: now.Truncate(24 * time.Hour).Add(24 * time.Hour)}, nil
}

// today rolls the counts over at UTC midnight and answers now. s.mu is
// held.
func (s *server) today() time.Time {
	now := time.Now().UTC()
	if day := now.Format(time.DateOnly); day != s.day {
		s.day, s.calls, s.capped, s.cappedAll = day, map[string]int{}, map[string]bool{}, false
	}
	return now
}

// verdict is what OMDb answers key before it looks at the id: its
// error words, or "" to go on.
func (s *server) verdict(key string) string {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.today()
	s.seen[key] = true
	if s.revoked[key] {
		return "Invalid API key!"
	}
	s.calls[key]++
	if s.cappedAll || s.capped[key] || (s.limit > 0 && s.calls[key] > s.limit) {
		return "Request limit reached!"
	}
	return ""
}

var imdbID = regexp.MustCompile(`^tt\d{7,}$`)

const notAvailable = "N/A"

type ratingJSON struct {
	Source string `json:"Source"`
	Value  string `json:"Value"`
}

type titleJSON struct {
	Title      string       `json:"Title"`
	Year       string       `json:"Year"`
	Awards     string       `json:"Awards"`
	Ratings    []ratingJSON `json:"Ratings"`
	Metascore  string       `json:"Metascore"`
	IMDbRating string       `json:"imdbRating"`
	IMDbVotes  string       `json:"imdbVotes"`
	IMDbID     string       `json:"imdbID"`
	Type       string       `json:"Type"`
	Response   string       `json:"Response"`
}

func (s *server) title(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	if words := s.verdict(q.Get("apikey")); words != "" {
		failure(w, http.StatusUnauthorized, words)
		return
	}
	id := q.Get("i")
	if !imdbID.MatchString(id) {
		failure(w, http.StatusOK, "Incorrect IMDb ID.")
		return
	}
	t, ok := s.titles[id]
	if !ok {
		failure(w, http.StatusOK, "Movie not found!")
		return
	}
	name := s.names[id]
	out := titleJSON{
		Title: cmp.Or(name.title, notAvailable), Year: cmp.Or(name.year, notAvailable), Awards: cmp.Or(t.Awards, notAvailable), Ratings: []ratingJSON{},
		Metascore: notAvailable, IMDbRating: notAvailable, IMDbVotes: notAvailable, IMDbID: id, Type: "movie", Response: "True",
	}
	if t.IMDbRating != nil {
		out.IMDbRating = strconv.FormatFloat(*t.IMDbRating, 'f', 1, 64)
		out.Ratings = append(out.Ratings, ratingJSON{"Internet Movie Database", out.IMDbRating + "/10"})
	}
	if t.IMDbVotes != nil {
		out.IMDbVotes = thousands(*t.IMDbVotes)
	}
	if t.RottenTomatoes != nil {
		out.Ratings = append(out.Ratings, ratingJSON{"Rotten Tomatoes", strconv.Itoa(*t.RottenTomatoes) + "%"})
	}
	if t.Metacritic != nil {
		out.Metascore = strconv.Itoa(*t.Metacritic)
		out.Ratings = append(out.Ratings, ratingJSON{"Metacritic", out.Metascore + "/100"})
	}
	reply(w, http.StatusOK, out)
}

// thousands writes n as OMDb writes votes, such as 2,412,519.
func thousands(n int64) string {
	s := strconv.FormatInt(n, 10)
	for i := len(s) - 3; i > 0; i -= 3 {
		s = s[:i] + "," + s[i:]
	}
	return s
}

func failure(w http.ResponseWriter, status int, words string) {
	reply(w, status, struct {
		Response string `json:"Response"`
		Error    string `json:"Error"`
	}{"False", words})
}

// reply writes v as OMDb does: no trailing newline and no HTML escapes.
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
	_, _ = w.Write(bytes.TrimSuffix(body.Bytes(), []byte("\n")))
}
