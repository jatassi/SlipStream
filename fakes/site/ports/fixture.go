// Package ports is fixture-site's replay of every port's scrubbed
// exchanges, and the runner tier that proves each port against them.
//
// On disk, one directory per port, so a new port adds a directory and
// edits no Go:
//
//	fakes/site/ports/<port id>/
//	  fixture.yaml        source, settings, session and the cases with their exchanges
//	  bodies/<name>       response bodies, byte for byte as the site sent them
//	  want/<case>.json    the golden of each case, written by go test -update
//
// fixture.yaml:
//
//	source: prowlarr            # prowlarr (rewritten from its Files/Indexers) or synthetic
//	settings:                   # what the Admin entered, fictional and scrub-checked
//	  username: fixture-user
//	  apiKey: "{{apiKey}}"      # a placeholder, sent and matched as written
//	session: {cookie: "sid={{sid}}", pasted: true}   # held before each case, when the family needs one
//	cases:
//	  - name: search            # search, empty, auth and limit are required; loggedout for
//	                            # a family that holds a session; code for a CodeLogin
//	    call: {search: {mode: movie, ids: {imdbid: tt0000001}}}
//	    exchanges:
//	      - request:
//	          method: POST
//	          path: /api/torrents
//	          json: {passkey: "{{passkey}}", imdb: {id: 1}}   # a subset match
//	          noQuery: [freeleech]                              # parameters it must not carry
//	          rawQuery: "11;5;q=tt0000001"                      # the whole query, for one not joined by &
//	        response:
//	          status: 200
//	          headers: {Content-Type: application/json}
//	          body: search.json  # under bodies/
//
// A call is one of search, recent, fetch (a link, relative to the
// address) and test (with a typed code). A case may set its own session,
// and session: {} starts it with none. A case's settings are set over the
// fixture's, and its categories are the Indexer's, none when unset. A
// call that sends no request, such as a TV search on a port with none,
// has exchanges: [].
//
// A request matches an exchange on method, host (the port's fixture host
// when the exchange names none), path, and the headers and the query,
// form or JSON fields the exchange names. Exchanges answer in order
// within a case, so a loggedout case answers the logged-out page, the
// sign-in and then the page again.
//
// go run ./fakes/site/ports/record writes a case from a live capture or
// a body from a Prowlarr recording, and Scrub keeps real values out.
package ports

import (
	"bytes"
	"embed"
	"errors"
	"fmt"
	"io/fs"
	"maps"
	"path"
	"slices"
	"strings"

	"go.yaml.in/yaml/v3"

	"github.com/jatassi/slipstream/internal/ports/port"
	"github.com/jatassi/slipstream/internal/searching"
)

// FS holds every port's fixture. testport's is always there, so the
// patterns match with no port registered. They name files, so an empty
// bodies/ or want/ embeds nothing and breaks nothing.
//
//go:embed */fixture.yaml */bodies/* */want/*
var FS embed.FS

// Source is where a fixture's exchanges came from.
type Source string

const (
	// Prowlarr is a recording from Prowlarr's Files/Indexers, rewritten
	// to the fixture catalog.
	Prowlarr Source = "prowlarr"
	// Synthetic is written from the C# parser's shape.
	Synthetic Source = "synthetic"
	// Recorded is a live exchange the record tool captured.
	Recorded Source = "recorded"
)

type Fixture struct {
	Port     port.ID           `yaml:"-"`
	Source   Source            `yaml:"source"`
	Settings map[string]string `yaml:"settings,omitempty"`
	Session  *Session          `yaml:"session,omitempty"`
	Cases    []Case            `yaml:"cases"`
	// Bodies are bodies/ by file name, and Wants want/ by case name.
	Bodies map[string][]byte `yaml:"-"`
	Wants  map[string][]byte `yaml:"-"`
}

// Session is what the Node holds for the Indexer before a case runs.
type Session struct {
	// Cookie is name=value pairs joined by "; ".
	Cookie    string `yaml:"cookie,omitempty"`
	UserAgent string `yaml:"userAgent,omitempty"`
	Token     string `yaml:"token,omitempty"`
	// Pasted is a session from a browser cookie the Admin pasted.
	Pasted bool `yaml:"pasted,omitempty"`
}

type Case struct {
	Name string `yaml:"name"`
	Call Call   `yaml:"call"`
	// Session replaces the fixture's for this case.
	Session *Session `yaml:"session,omitempty"`
	// Settings are set over the fixture's for this case.
	Settings map[string]string `yaml:"settings,omitempty"`
	// Categories are the Newznab categories the Indexer has.
	Categories []int `yaml:"categories,omitempty"`
	// Exchanges is every request the case sends in order, exchanges: []
	// for a call that sends none.
	Exchanges []Exchange `yaml:"exchanges"`
}

// Call is one of Search, Recent, Fetch and Test.
type Call struct {
	Search *Search   `yaml:"search,omitempty"`
	Recent *struct{} `yaml:"recent,omitempty"`
	Fetch  string    `yaml:"fetch,omitempty"`
	Test   *Test     `yaml:"test,omitempty"`
}

// Search is searching.Query as a fixture writes it.
type Search struct {
	Mode    string            `yaml:"mode"`
	IDs     map[string]string `yaml:"ids,omitempty"`
	Text    string            `yaml:"text,omitempty"`
	Season  *int              `yaml:"season,omitempty"`
	Episode string            `yaml:"episode,omitempty"`
}

// Test is a Test call, with the two-factor code the Admin typed.
type Test struct {
	Code string `yaml:"code,omitempty"`
}

type Exchange struct {
	Request  Match    `yaml:"request"`
	Response Response `yaml:"response"`
}

// Match is what a request must carry. Every field it names must be
// there with that value. Others may be too, except the query parameters
// NoQuery names.
type Match struct {
	Method  string            `yaml:"method,omitempty"`
	Host    string            `yaml:"host,omitempty"`
	Path    string            `yaml:"path"`
	Headers map[string]string `yaml:"headers,omitempty"`
	Query   map[string]string `yaml:"query,omitempty"`
	NoQuery []string          `yaml:"noQuery,omitempty"`
	// RawQuery is the whole query as sent, for a site whose query isn't
	// name=value pairs joined by "&".
	RawQuery string            `yaml:"rawQuery,omitempty"`
	Form     map[string]string `yaml:"form,omitempty"`
	JSON     map[string]any    `yaml:"json,omitempty"`
}

type Response struct {
	Status  int               `yaml:"status,omitempty"`
	Headers map[string]string `yaml:"headers,omitempty"`
	// Body names a file under bodies/, "" for none.
	Body string `yaml:"body,omitempty"`
}

// Host is the host fixture-site serves port id's fixture at, the host of
// fixture-site's site labeled id.
func Host(id port.ID) string { return string(id) + ".fixture-site.fake.localhost" }

// Address is the base address a fixture's Indexer has.
func Address(id port.ID) string { return "https://" + Host(id) + "/" }

// Required are the cases every fixture of a port in family a has. A port
// with TV search also has a TV search with a season and an episode.
func Required(a port.Auth) []string {
	out := []string{"search", "empty", "recent", "auth", "limit"}
	if a.HoldsSession() {
		out = append(out, "loggedout")
	}
	if a == port.FormWithCode {
		out = append(out, "code")
	}
	return out
}

// TVCase is a fixture's TV search with a season and an episode.
func (f Fixture) TVCase() (Case, bool) {
	i := slices.IndexFunc(f.Cases, func(c Case) bool {
		s := c.Call.Search
		return s != nil && s.Mode == "tvsearch" && s.Season != nil && *s.Season > 0 && s.Episode != ""
	})
	if i < 0 {
		return Case{}, false
	}
	return f.Cases[i], true
}

// Skeleton is the fixture prowlarr-sync new writes for a port of family
// a: every case Required names and a TV search, each with a call and no
// exchanges yet.
func Skeleton(id port.ID, a port.Auth) Fixture {
	calls := map[string]Call{
		"search": {Search: &Search{Mode: "search", Text: "The Quiet Orchard"}},
		"empty":  {Search: &Search{Mode: "search", Text: "Juniper Falls"}},
		"tv":     {Search: &Search{Mode: "tvsearch", Text: "Harbor Lights", Season: new(2), Episode: "3"}},
	}
	f := Fixture{Port: id, Source: Synthetic, Settings: map[string]string{}}
	for _, name := range append(Required(a), "tv") {
		c := Case{Name: name, Call: Call{Recent: &struct{}{}}, Exchanges: []Exchange{}}
		if call, ok := calls[name]; ok {
			c.Call = call
		}
		if a.HoldsSession() && (name == "auth" || name == "code") {
			c.Session = &Session{}
		}
		f.Cases = append(f.Cases, c)
	}
	return f
}

// Load reads port id's fixture from fsys, strictly: every case has one
// call and a unique name, and every body it names is there.
func Load(fsys fs.FS, id port.ID) (Fixture, error) {
	raw, err := fs.ReadFile(fsys, path.Join(string(id), "fixture.yaml"))
	if errors.Is(err, fs.ErrNotExist) {
		return Fixture{}, fmt.Errorf("port %s has no fixture: add %s/fixture.yaml", id, id)
	}
	if err != nil {
		return Fixture{}, fmt.Errorf("read the fixture of %s: %w", id, err)
	}
	f, err := Parse(id, raw)
	if err != nil {
		return Fixture{}, err
	}
	if f.Bodies, err = files(fsys, path.Join(string(id), "bodies"), ""); err != nil {
		return Fixture{}, err
	}
	if f.Wants, err = files(fsys, path.Join(string(id), "want"), ".json"); err != nil {
		return Fixture{}, err
	}
	var errs []error
	for _, c := range f.Cases {
		for i, x := range c.Exchanges {
			if _, ok := f.Bodies[x.Response.Body]; x.Response.Body != "" && !ok {
				errs = append(errs, fmt.Errorf("fixture %s, case %s, exchange %d: bodies/%s isn't there", id, c.Name, i+1, x.Response.Body))
			}
		}
	}
	return f, errors.Join(errs...)
}

// Parse reads fixture.yaml strictly and checks its cases.
func Parse(id port.ID, raw []byte) (Fixture, error) {
	f := Fixture{Port: id}
	dec := yaml.NewDecoder(bytes.NewReader(raw))
	dec.KnownFields(true)
	if err := dec.Decode(&f); err != nil {
		return Fixture{}, fmt.Errorf("fixture %s: %w", id, err)
	}
	var errs []error
	if !slices.Contains([]Source{Prowlarr, Synthetic, Recorded}, f.Source) {
		errs = append(errs, fmt.Errorf("fixture %s: source %q is none of prowlarr, synthetic and recorded", id, f.Source))
	}
	seen := map[string]bool{}
	for _, c := range f.Cases {
		if c.Name == "" || seen[c.Name] {
			errs = append(errs, fmt.Errorf("fixture %s: case %q is unnamed or named twice", id, c.Name))
		}
		seen[c.Name] = true
		if n := c.Call.count(); n != 1 {
			errs = append(errs, fmt.Errorf("fixture %s, case %s: a call is one of search, recent, fetch and test, and this one sets %d", id, c.Name, n))
		}
		if c.Exchanges == nil {
			errs = append(errs, fmt.Errorf("fixture %s, case %s: the case lists no exchanges, and one that sends no request says exchanges: []", id, c.Name))
		}
	}
	return f, errors.Join(errs...)
}

// Case is the case named name.
func (f Fixture) Case(name string) (Case, bool) {
	i := slices.IndexFunc(f.Cases, func(c Case) bool { return c.Name == name })
	if i < 0 {
		return Case{}, false
	}
	return f.Cases[i], true
}

// SessionOf is the session c starts with, nil for none.
func (f Fixture) SessionOf(c Case) *Session {
	s := f.Session
	if c.Session != nil {
		s = c.Session
	}
	if s == nil || *s == (Session{}) {
		return nil
	}
	return s
}

// SettingsOf are the settings c runs with.
func (f Fixture) SettingsOf(c Case) map[string]string {
	out := map[string]string{}
	maps.Copy(out, f.Settings)
	maps.Copy(out, c.Settings)
	return out
}

// Marshal is f as fixture.yaml.
func (f Fixture) Marshal() ([]byte, error) {
	var b bytes.Buffer
	enc := yaml.NewEncoder(&b)
	enc.SetIndent(2)
	if err := enc.Encode(f); err != nil {
		return nil, fmt.Errorf("write the fixture of %s: %w", f.Port, err)
	}
	return b.Bytes(), nil
}

// Searching is c as a Job's call, a relative fetch link resolved
// against address.
func (c Call) Searching(address string) searching.Call {
	switch {
	case c.Search != nil:
		return searching.SearchCall{Query: searching.Query{
			Mode: c.Search.Mode, IDs: c.Search.IDs, Text: c.Search.Text, Season: c.Search.Season, Episode: c.Search.Episode,
		}}
	case c.Fetch != "":
		link := c.Fetch
		if strings.HasPrefix(link, "/") {
			link = strings.TrimSuffix(address, "/") + link
		}
		return searching.FetchCall{Link: searching.LinkOf(link)}
	case c.Test != nil:
		return searching.TestCall{Code: c.Test.Code}
	}
	return searching.RecentCall{}
}

func (c Call) count() int {
	n := 0
	for _, set := range []bool{c.Search != nil, c.Recent != nil, c.Fetch != "", c.Test != nil} {
		if set {
			n++
		}
	}
	return n
}

// files are dir's files by name, without suffix, nil when dir isn't
// there.
func files(fsys fs.FS, dir, suffix string) (map[string][]byte, error) {
	entries, err := fs.ReadDir(fsys, dir)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", dir, err)
	}
	out := map[string][]byte{}
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), suffix) {
			continue
		}
		raw, err := fs.ReadFile(fsys, path.Join(dir, e.Name()))
		if err != nil {
			return nil, fmt.Errorf("read %s: %w", e.Name(), err)
		}
		out[strings.TrimSuffix(e.Name(), suffix)] = raw
	}
	return out, nil
}
