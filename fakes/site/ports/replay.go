package ports

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"maps"
	"net"
	"net/http"
	"net/url"
	"reflect"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/jatassi/slipstream/fakes/fake"
	"github.com/jatassi/slipstream/internal/ports/port"
)

// PortCase has a port's fixture answer from one case, in order, from now
// on: {"port": "<port id>", "case": "<case>"}. Case "" answers any
// request from whichever exchange of any case matches it first.
var PortCase = fake.CueSpec{Name: "fixture-site-port-case"}

// Stray is the status a Replay answers a request no exchange matches
// with, its body naming the request and the exchange it expected.
const Stray = 599

// Replay serves a fixture at its port's host. With a case in use, each
// request must match the case's next exchange, and the case starts over
// after its last. With none, a request gets the first exchange of any
// case that matches it.
type Replay struct {
	f Fixture

	mu     sync.Mutex
	inUse  *Case
	next   int
	hits   []Hit
	strays []string
}

// Hit is one request a Replay answered from an exchange.
type Hit struct {
	At       time.Time
	Host     string
	Exchange int
}

func NewReplay(f Fixture) *Replay { return &Replay{f: f} }

// Use answers from case name from now on, from its first exchange, and
// forgets the hits and strays so far. "" answers from any case.
func (r *Replay) Use(name string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.inUse, r.next, r.hits, r.strays = nil, 0, nil, nil
	if name == "" {
		return nil
	}
	c, ok := r.f.Case(name)
	if !ok {
		return fmt.Errorf("the fixture of %s has no case %q", r.f.Port, name)
	}
	r.inUse = &c
	return nil
}

// Hits are the requests answered from the case in use since Use.
func (r *Replay) Hits() []Hit {
	r.mu.Lock()
	defer r.mu.Unlock()
	return slices.Clone(r.hits)
}

// Strays are the requests no exchange matched since Use, each with why.
func (r *Replay) Strays() []string {
	r.mu.Lock()
	defer r.mu.Unlock()
	return slices.Clone(r.strays)
}

func (r *Replay) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	body, err := io.ReadAll(req.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	x, why := r.answer(req, body)
	if x == nil {
		http.Error(w, why, Stray)
		return
	}
	for k, v := range x.Response.Headers {
		if strings.EqualFold(k, "Set-Cookie") {
			for c := range strings.SplitSeq(v, "\n") {
				w.Header().Add(k, c)
			}
			continue
		}
		w.Header().Set(k, v)
	}
	w.WriteHeader(statusOf(x.Response.Status))
	_, _ = w.Write(r.f.Bodies[x.Response.Body])
}

func statusOf(s int) int {
	if s == 0 {
		return http.StatusOK
	}
	return s
}

// answer is the exchange req gets, or nil and why none matches.
func (r *Replay) answer(req *http.Request, body []byte) (x *Exchange, why string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.inUse == nil {
		for _, c := range r.f.Cases {
			for i := range c.Exchanges {
				if r.matches(c.Exchanges[i].Request, req, body) == "" {
					return &c.Exchanges[i], ""
				}
			}
		}
		why = fmt.Sprintf("no exchange of the fixture of %s matches %s", r.f.Port, described(req))
		r.strays = append(r.strays, why)
		return nil, why
	}
	if r.next == len(r.inUse.Exchanges) {
		r.next = 0
	}
	x = &r.inUse.Exchanges[r.next]
	if why = r.matches(x.Request, req, body); why != "" {
		why = fmt.Sprintf("case %s of %s expected exchange %d, and %s: %s", r.inUse.Name, r.f.Port, r.next+1, described(req), why)
		r.strays = append(r.strays, why)
		return nil, why
	}
	r.hits = append(r.hits, Hit{At: time.Now(), Host: hostOf(req), Exchange: r.next})
	r.next++
	return x, ""
}

func described(req *http.Request) string {
	return fmt.Sprintf("%s %s%s?%s", req.Method, hostOf(req), req.URL.Path, req.URL.RawQuery)
}

func hostOf(req *http.Request) string {
	if h, _, err := net.SplitHostPort(req.Host); err == nil {
		return h
	}
	return req.Host
}

// matches is why req doesn't match m, "" when it does.
func (r *Replay) matches(m Match, req *http.Request, body []byte) string {
	method := m.Method
	if method == "" {
		method = http.MethodGet
	}
	want := m.Host
	if want == "" {
		want = Host(r.f.Port)
	}
	switch {
	case req.Method != method:
		return "its method isn't " + method
	case hostOf(req) != want:
		return "its host isn't " + want
	case req.URL.Path != m.Path:
		return "its path isn't " + m.Path
	}
	for k, v := range sorted(m.Headers) {
		if got := req.Header.Get(k); got != v {
			return fmt.Sprintf("header %s is %q, not %q", k, got, v)
		}
	}
	if why := fields("query", req.URL.Query(), m.Query); why != "" {
		return why
	}
	if m.Form != nil {
		form, err := url.ParseQuery(string(body))
		if err != nil {
			return "its body isn't a form"
		}
		if why := fields("form", form, m.Form); why != "" {
			return why
		}
	}
	if m.JSON != nil {
		return jsonFields(body, m.JSON)
	}
	return ""
}

func sorted(m map[string]string) func(func(string, string) bool) {
	return func(yield func(string, string) bool) {
		for _, k := range slices.Sorted(maps.Keys(m)) {
			if !yield(k, m[k]) {
				return
			}
		}
	}
}

// fields is why got lacks one of want's fields, a repeated one compared
// as its values joined by commas.
func fields(what string, got url.Values, want map[string]string) string {
	for k, v := range sorted(want) {
		vs, ok := got[k]
		if !ok {
			return fmt.Sprintf("its %s has no %s", what, k)
		}
		if joined := strings.Join(vs, ","); joined != v {
			return fmt.Sprintf("its %s's %s is %q, not %q", what, k, joined, v)
		}
	}
	return ""
}

func jsonFields(body []byte, want map[string]any) string {
	var got any
	if err := json.Unmarshal(body, &got); err != nil {
		return "its body isn't JSON"
	}
	raw, err := json.Marshal(want)
	if err != nil {
		return err.Error()
	}
	var norm any
	_ = json.Unmarshal(raw, &norm)
	if path := subset(norm, got, "$"); path != "" {
		return "its JSON differs at " + path
	}
	return ""
}

// subset is the first path where got lacks want, "" when got holds it:
// an object holds want's keys with values that hold theirs, and anything
// else equals want.
func subset(want, got any, at string) string {
	w, isObject := want.(map[string]any)
	if !isObject {
		if !reflect.DeepEqual(want, got) {
			return at
		}
		return ""
	}
	g, ok := got.(map[string]any)
	if !ok {
		return at
	}
	for _, k := range slices.Sorted(maps.Keys(w)) {
		if p := subset(w[k], g[k], at+"."+k); p != "" {
			return p
		}
	}
	return ""
}

// Served is every registered port's Replay that fixture-site serves.
type Served map[port.ID]*Replay

// Serve adds a route to routes for each port of reg whose host no route
// serves yet, a family fake's taking precedence, answering from the
// port's fixture as load loads it. It plays PortCase for those ports.
func Serve(env fake.Env, reg port.Catalog, load Loader, routes fake.Routes) (Served, error) {
	taken := map[string]bool{}
	for _, rt := range routes {
		_, at, _ := strings.Cut(rt.Pattern, " ")
		h, _, _ := strings.Cut(at, "/")
		taken[h] = true
	}
	served := Served{}
	for _, e := range reg.Entries() {
		if taken[Host(e.ID)] {
			continue
		}
		f, err := load(e.ID)
		if err != nil {
			return nil, err
		}
		r := NewReplay(f)
		served[e.ID] = r
		routes[string(e.ID)+".replay"] = fake.Route{Pattern: Host(e.ID) + "/", Handler: r.ServeHTTP}
	}
	err := fake.OnCue(env, PortCase.Name, func(_ context.Context, a CaseArgs) (any, error) {
		r, ok := served[port.ID(a.Port)]
		if !ok {
			return nil, fmt.Errorf("fixture-site replays no port %q", a.Port)
		}
		return struct{}{}, r.Use(a.Case)
	})
	return served, err
}

// Loader loads a port's fixture.
type Loader func(port.ID) (Fixture, error)

// Embedded loads fixtures from FS.
func Embedded(id port.ID) (Fixture, error) { return Load(FS, id) }

// CaseArgs are PortCase's arguments.
type CaseArgs struct {
	Port string `json:"port"`
	Case string `json:"case"`
}
