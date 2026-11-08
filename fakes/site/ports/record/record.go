package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"log/slog"
	"maps"
	"mime"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"sync"

	"go.yaml.in/yaml/v3"

	"github.com/jatassi/slipstream/fakes/site/ports"
	"github.com/jatassi/slipstream/internal/ports/port"
	"github.com/jatassi/slipstream/internal/searching"
)

type liveConfig struct {
	// dir holds every port's fixture directory.
	dir      string
	entry    port.Entry
	caseName string
	address  string
	settings map[string]string
	call     ports.Call

	native    searching.Driver
	transport http.RoundTripper
}

func parseLive(args []string, reg port.Catalog) (liveConfig, error) {
	var c liveConfig
	var id, settings, search, fetch, code string
	var recent, test bool
	flags := flag.NewFlagSet("record live", flag.ContinueOnError)
	flags.StringVar(&id, "port", "", "the port's id")
	flags.StringVar(&c.caseName, "case", "", "the case to add or replace")
	flags.StringVar(&settings, "settings", "", "a YAML file mapping each setting to its value, kept out of the repository")
	flags.StringVar(&c.address, "address", "", "the site's address, the port's first by default")
	flags.StringVar(&c.dir, "dir", "fakes/site/ports", "the directory holding every port's fixture")
	flags.StringVar(&search, "search", "", "a search as a query string: mode, q, season, ep and the mode's ids")
	flags.BoolVar(&recent, "recent", false, "fetch the newest uploads")
	flags.StringVar(&fetch, "fetch", "", "fetch this link")
	flags.BoolVar(&test, "test", false, "run a Test")
	flags.StringVar(&code, "code", "", "the two-factor code a Test signs in with")
	if err := flags.Parse(args); err != nil {
		return c, err //nolint:wrapcheck // the flag package's words name the flag
	}
	e, ok := reg.Lookup(port.ID(id))
	if !ok {
		return c, fmt.Errorf("this binary carries no port %q", id)
	}
	c.entry = e
	if c.caseName == "" || settings == "" {
		return c, errors.New("name a case and a settings file")
	}
	if c.address == "" {
		c.address = e.IndexerURLs[0]
	}
	raw, err := os.ReadFile(settings)
	if err != nil {
		return c, fmt.Errorf("read the settings: %w", err)
	}
	if err := yaml.Unmarshal(raw, &c.settings); err != nil {
		return c, fmt.Errorf("read the settings: %w", err)
	}
	switch {
	case search != "":
		q, err := url.ParseQuery(search)
		if err != nil {
			return c, fmt.Errorf("read the search: %w", err)
		}
		s := &ports.Search{Mode: q.Get("mode"), Text: q.Get("q"), Episode: q.Get("ep"), IDs: map[string]string{}}
		if q.Has("season") {
			n, err := strconv.Atoi(q.Get("season"))
			if err != nil {
				return c, fmt.Errorf("season %q isn't a number", q.Get("season"))
			}
			s.Season = &n
		}
		for k := range q {
			if !slices.Contains([]string{"mode", "q", "season", "ep"}, k) {
				s.IDs[k] = q.Get(k)
			}
		}
		c.call.Search = s
	case recent:
		c.call.Recent = &struct{}{}
	case fetch != "":
		c.call.Fetch = fetch
	case test:
		c.call.Test = &ports.Test{Code: code}
	default:
		return c, errors.New("name a call: -search, -recent, -fetch or -test")
	}
	return c, nil
}

// live runs c's call through a Node against the site, then adds what it
// sent and received to the port's fixture as c.caseName, scrubbed.
func live(ctx context.Context, c liveConfig) error {
	rec := &recorder{next: c.transport}
	node := searching.RemoteOf(searching.New(searching.Config{
		HTTP: &http.Client{Transport: rec}, Timings: searching.DefaultTimings(),
		Log: slog.New(slog.DiscardHandler), Native: c.native,
	}))
	job := searching.Job{
		ID: "record-" + c.caseName, Indexer: "record", Revision: 1, Kind: searching.Native, Port: string(c.entry.ID),
		Address: c.address, Settings: c.settings, Call: c.call.Searching(c.address),
	}
	var a *searching.Answer
	for a == nil {
		poll, err := node.Run(ctx, job)
		if err != nil {
			return fmt.Errorf("run the call: %w", err)
		}
		a = poll.Answer
	}
	if a.Outcome != nil {
		fmt.Fprintf(os.Stderr, "record: the call ended %v\n", a.Outcome)
	}
	base, err := url.Parse(c.address)
	if err != nil {
		return fmt.Errorf("read the address: %w", err)
	}
	r := rewriterOf(c, base.Hostname(), rec.exchanges(), a.Releases)
	f, err := existing(c.dir, c.entry.ID)
	if err != nil {
		return err
	}
	f.Settings = map[string]string{}
	for name, v := range c.settings {
		f.Settings[name] = placeholderOf(name, v)
	}
	kase := ports.Case{Name: c.caseName, Call: c.call}
	if q := c.call.Search; q != nil {
		scrubbed := *q
		if scrubbed.Text, err = rewritten(r, q.Text); err != nil {
			return err
		}
		kase.Call.Search = &scrubbed
	}
	if kase.Call.Fetch, err = rewritten(r, c.call.Fetch); err != nil {
		return err
	}
	added := map[string][]byte{}
	for i, x := range rec.exchanges() {
		ex, body, err := exchangeOf(r, base.Hostname(), x)
		if err != nil {
			return err
		}
		if body != nil {
			ex.Response.Body = fmt.Sprintf("%s-%d%s", c.caseName, i+1, extension(x.res.Header.Get("Content-Type")))
			added[ex.Response.Body] = body
			f.Bodies[ex.Response.Body] = body
		}
		kase.Exchanges = append(kase.Exchanges, ex)
	}
	if i := slices.IndexFunc(f.Cases, func(k ports.Case) bool { return k.Name == c.caseName }); i >= 0 {
		f.Cases[i] = kase
	} else {
		f.Cases = append(f.Cases, kase)
	}
	return write(c.dir, f, added)
}

// rewriterOf scrubs a capture: each setting's value becomes its
// placeholder, each cookie a response set and each bearer token its own,
// each of the port's hosts the fixture host, and each Release's title
// and info hash the catalog's.
func rewriterOf(c liveConfig, host string, xs []exchange, releases []searching.Release) *ports.Rewriter {
	r := &ports.Rewriter{Host: ports.Host(c.entry.ID), Hosts: append(hostsOf(c.entry), host), Replace: map[string]string{}}
	keep := func(value, with string) {
		if len(value) >= minSecret {
			r.Replace[value] = with
			r.Replace[url.QueryEscape(value)] = with
		}
	}
	for name, v := range c.settings {
		keep(v, placeholderOf(name, v))
	}
	for _, x := range xs {
		for _, ck := range x.res.Cookies() {
			keep(ck.Value, "{{"+identifier(ck.Name)+"}}")
		}
		if token, ok := strings.CutPrefix(x.req.Header.Get("Authorization"), "Bearer "); ok {
			keep(token, "{{token}}")
		}
	}
	for _, rel := range releases {
		r.Titles = append(r.Titles, rel.Title)
	}
	return r
}

// minSecret is the shortest value the rewrite swaps everywhere: a
// shorter one would swap unrelated text.
const minSecret = 4

func placeholderOf(name, v string) string {
	switch {
	case v == "":
		return ""
	case name == "username":
		return ports.FixtureUser
	case name == "email":
		return ports.FixtureEmail
	}
	return "{{" + identifier(name) + "}}"
}

var notIdentifier = regexp.MustCompile(`[^A-Za-z0-9]+`)

func identifier(name string) string {
	id := notIdentifier.ReplaceAllString(name, "")
	if id == "" || id[0] >= '0' && id[0] <= '9' {
		id = "value" + id
	}
	return id
}

func hostsOf(e port.Entry) []string {
	var out []string
	for _, raw := range slices.Concat(e.IndexerURLs, e.LegacyURLs) {
		if u, err := url.Parse(raw); err == nil {
			out = append(out, u.Hostname())
		}
	}
	return out
}

// kept are the response headers a fixture keeps.
var kept = []string{"Content-Type", "Location", "Retry-After", "Set-Cookie"}

func rewritten(r *ports.Rewriter, s string) (string, error) {
	out, err := r.Rewrite([]byte(s))
	return string(out), err
}

func exchangeOf(r *ports.Rewriter, host string, x exchange) (ports.Exchange, []byte, error) {
	var errs []error
	scrub := func(s string) string {
		out, err := rewritten(r, s)
		errs = append(errs, err)
		return out
	}
	m := ports.Match{Method: x.req.Method, Path: scrub(x.req.URL.Path), Query: values(x.req.URL.Query(), scrub)}
	if h := x.req.URL.Hostname(); h != host {
		m.Host = scrub(h)
	}
	for _, k := range []string{"Authorization", "Cookie"} {
		if v := x.req.Header.Get(k); v != "" {
			if m.Headers == nil {
				m.Headers = map[string]string{}
			}
			m.Headers[k] = scrub(v)
		}
	}
	ct, _, _ := mime.ParseMediaType(x.req.Header.Get("Content-Type"))
	switch {
	case ct == "application/x-www-form-urlencoded":
		form, err := url.ParseQuery(string(x.reqBody))
		errs = append(errs, err)
		m.Form = values(form, scrub)
	case ct == "application/json" && len(x.reqBody) > 0:
		errs = append(errs, json.Unmarshal([]byte(scrub(string(x.reqBody))), &m.JSON))
	}
	res := ports.Response{Status: x.res.StatusCode}
	for _, k := range kept {
		if vs := x.res.Header.Values(k); len(vs) > 0 {
			if res.Headers == nil {
				res.Headers = map[string]string{}
			}
			res.Headers[k] = scrub(strings.Join(vs, "\n"))
		}
	}
	var body []byte
	if len(x.resBody) > 0 {
		var err error
		body, err = r.Rewrite(x.resBody)
		errs = append(errs, err)
	}
	return ports.Exchange{Request: m, Response: res}, body, errors.Join(errs...)
}

func values(v url.Values, scrub func(string) string) map[string]string {
	if len(v) == 0 {
		return nil
	}
	out := map[string]string{}
	for k, vs := range v {
		out[k] = scrub(strings.Join(vs, ","))
	}
	return out
}

func extension(contentType string) string {
	ct, _, _ := mime.ParseMediaType(contentType)
	switch {
	case strings.HasSuffix(ct, "json"):
		return ".json"
	case strings.HasSuffix(ct, "xml"):
		return ".xml"
	case ct == "text/html":
		return ".html"
	case ct == "application/x-bittorrent":
		return ".torrent"
	}
	return ".txt"
}

// existing is the port's fixture in dir, or a new recorded one.
func existing(dir string, id port.ID) (ports.Fixture, error) {
	if _, err := os.Stat(filepath.Join(dir, string(id), "fixture.yaml")); errors.Is(err, fs.ErrNotExist) {
		return ports.Fixture{Port: id, Source: ports.Recorded, Bodies: map[string][]byte{}}, nil
	}
	f, err := ports.Load(os.DirFS(dir), id)
	if f.Bodies == nil {
		f.Bodies = map[string][]byte{}
	}
	return f, err
}

// write writes the added bodies, and f's fixture.yaml when f has cases,
// only once Scrub passes all of f.
func write(dir string, f ports.Fixture, added map[string][]byte) error {
	if found := ports.Scrub(f); len(found) > 0 {
		return fmt.Errorf("the scrub check refuses the fixture, and nothing was written:\n%s", strings.Join(found, "\n"))
	}
	at := filepath.Join(dir, string(f.Port))
	if err := os.MkdirAll(filepath.Join(at, "bodies"), 0o755); err != nil {
		return fmt.Errorf("create the fixture: %w", err)
	}
	for _, name := range slices.Sorted(maps.Keys(added)) {
		if err := os.WriteFile(filepath.Join(at, "bodies", name), added[name], 0o644); err != nil {
			return fmt.Errorf("write the body: %w", err)
		}
	}
	if len(f.Cases) == 0 {
		return nil
	}
	raw, err := f.Marshal()
	if err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(at, "fixture.yaml"), raw, 0o644); err != nil {
		return fmt.Errorf("write the fixture: %w", err)
	}
	return nil
}

type importConfig struct {
	dir       string
	entry     port.Entry
	recording string
	body      string
	r         *ports.Rewriter
}

func parseImport(args []string, reg port.Catalog) (importConfig, error) {
	var c importConfig
	var id string
	var titles, replaces, hosts list
	flags := flag.NewFlagSet("record import", flag.ContinueOnError)
	flags.StringVar(&id, "port", "", "the port's id")
	flags.StringVar(&c.recording, "recording", "", "the Prowlarr recording to rewrite")
	flags.StringVar(&c.body, "body", "", "its name under the port's bodies/")
	flags.StringVar(&c.dir, "dir", "fakes/site/ports", "the directory holding every port's fixture")
	flags.Var(&titles, "title", "a title the recording holds that isn't a dotted release name (repeatable)")
	flags.Var(&replaces, "replace", "old=new, a literal to swap (repeatable)")
	flags.Var(&hosts, "host", "a host besides the port's own addresses' to swap for the fixture host (repeatable)")
	if err := flags.Parse(args); err != nil {
		return c, err //nolint:wrapcheck // the flag package's words name the flag
	}
	e, ok := reg.Lookup(port.ID(id))
	if !ok {
		return c, fmt.Errorf("this binary carries no port %q", id)
	}
	if c.recording == "" || c.body == "" || filepath.Base(c.body) != c.body {
		return c, errors.New("name a recording and the body's file name")
	}
	c.entry = e
	c.r = &ports.Rewriter{Host: ports.Host(e.ID), Hosts: append(hostsOf(e), hosts...), Titles: titles, Replace: map[string]string{}}
	for _, kv := range replaces {
		old, nu, ok := strings.Cut(kv, "=")
		if !ok || old == "" {
			return c, fmt.Errorf("-replace %q isn't old=new", kv)
		}
		c.r.Replace[old] = nu
	}
	return c, nil
}

// importRecording writes the recording, rewritten, as a body of the
// port's fixture.
func importRecording(c importConfig) error {
	raw, err := os.ReadFile(c.recording)
	if err != nil {
		return fmt.Errorf("read the recording: %w", err)
	}
	body, err := c.r.Rewrite(bytes.TrimPrefix(raw, []byte("\xef\xbb\xbf")))
	if err != nil {
		return err
	}
	added := map[string][]byte{c.body: body}
	return write(c.dir, ports.Fixture{Port: c.entry.ID, Bodies: added}, added)
}

type list []string

func (l *list) String() string { return strings.Join(*l, ",") }

func (l *list) Set(v string) error {
	*l = append(*l, v)
	return nil
}

// recorder is a transport that keeps every exchange it carries.
type recorder struct {
	next http.RoundTripper
	mu   sync.Mutex
	xs   []exchange
}

type exchange struct {
	req     *http.Request
	reqBody []byte
	res     *http.Response
	resBody []byte
}

func (r *recorder) RoundTrip(req *http.Request) (*http.Response, error) {
	var reqBody []byte
	if req.Body != nil && req.GetBody != nil {
		body, err := req.GetBody()
		if err != nil {
			return nil, fmt.Errorf("copy the request: %w", err)
		}
		reqBody, err = io.ReadAll(body)
		if err != nil {
			return nil, fmt.Errorf("copy the request: %w", err)
		}
	}
	res, err := r.next.RoundTrip(req)
	if err != nil {
		return nil, err //nolint:wrapcheck // the transport's own error
	}
	resBody, err := io.ReadAll(res.Body)
	_ = res.Body.Close()
	if err != nil {
		return nil, fmt.Errorf("read the response: %w", err)
	}
	res.Body = io.NopCloser(bytes.NewReader(resBody))
	r.mu.Lock()
	defer r.mu.Unlock()
	r.xs = append(r.xs, exchange{req: req, reqBody: reqBody, res: res, resBody: resBody})
	return res, nil
}

func (r *recorder) exchanges() []exchange {
	r.mu.Lock()
	defer r.mu.Unlock()
	return slices.Clone(r.xs)
}
