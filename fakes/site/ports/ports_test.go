package ports

import (
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"log/slog"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"testing/synctest"
	"time"

	"github.com/jatassi/slipstream/fakes/fake"
	"github.com/jatassi/slipstream/fakes/fake/faketest"
	"github.com/jatassi/slipstream/fakes/scenario"
	"github.com/jatassi/slipstream/fakes/site/ports/internal/testport"
	"github.com/jatassi/slipstream/internal/ports/all"
	"github.com/jatassi/slipstream/internal/ports/port"
	"github.com/jatassi/slipstream/internal/searching"
)

var update = flag.Bool("update", false, "rewrite each case's want/<case>.json from what the port returns")

// TestPorts runs every port the binary carries against its fixture.
func TestPorts(t *testing.T) {
	reg := all.Registry()
	runPorts(t, reg, port.Driver(reg))
}

// TestTheHarnessRunsTestport proves the harness on testport, which
// testport.Driver runs by hand.
func TestTheHarnessRunsTestport(t *testing.T) {
	runPorts(t, testport.Registry(), testport.Driver{})
}

// runPorts runs each case of each port of reg through native through
// the Node's searching, as the Controller would send it.
func runPorts(t *testing.T, reg port.Registry, native searching.Driver) {
	for _, e := range reg.Entries() {
		t.Run(string(e.ID), func(t *testing.T) {
			f, err := Embedded(e.ID)
			if err != nil {
				t.Fatal(err)
			}
			fits(t, e, f)
			for _, c := range f.Cases {
				t.Run(c.Name, func(t *testing.T) {
					synctest.Test(t, func(t *testing.T) { runCase(t, reg, native, e, f, c) })
				})
			}
		})
	}
}

// fits fails a fixture that lacks a case its family needs, sets a
// setting the port doesn't have, or keeps a golden no case writes.
func fits(t *testing.T, e port.Entry, f Fixture) {
	t.Helper()
	for _, name := range Required(e.Auth) {
		if _, ok := f.Case(name); !ok {
			t.Errorf("the fixture has no %s case, which every port of its family needs", name)
		}
	}
	for name := range f.Settings {
		if !slices.ContainsFunc(e.Fields, func(fl port.Field) bool { return fl.Name == name }) {
			t.Errorf("the fixture sets %s, which isn't one of the port's settings", name)
		}
	}
	for name := range f.Wants {
		if _, ok := f.Case(name); !ok {
			t.Errorf("want/%s.json is the golden of no case", name)
		}
	}
}

func runCase(t *testing.T, reg port.Registry, native searching.Driver, e port.Entry, f Fixture, c Case) {
	var served Served
	spec := fake.Spec{
		Name: "fixture-site", TLS: fake.FrontTLS, Cues: []fake.CueSpec{PortCase},
		New: func(_ scenario.Scenario, env fake.Env) (fake.Server, error) {
			routes := fake.Routes{}
			var err error
			served, err = Serve(env, reg, Embedded, routes)
			return fake.HTTP(env, routes), err
		},
	}
	site := faketest.InMemory(t, spec, scenario.Scenario{})
	site.Cue(t, PortCase.Name, CaseArgs{Port: string(e.ID), Case: c.Name})
	node := searching.RemoteOf(searching.New(searching.Config{
		HTTP: site.Client, Timings: searching.DefaultTimings(), Log: slog.New(slog.DiscardHandler), Native: native,
	}))
	job := searching.Job{
		ID: "job-" + c.Name, Indexer: "fixture", Revision: 1, Kind: searching.Native, Port: string(e.ID),
		Address: Address(e.ID), Settings: f.Settings, Session: sessionOf(f.SessionOf(c), e.ID), Call: c.Call.Searching(Address(e.ID)),
	}
	var a searching.Answer
	for {
		poll, err := node.Run(t.Context(), job)
		if err != nil {
			t.Fatalf("Run = %v", err)
		}
		if poll.Answer != nil {
			a = *poll.Answer
			break
		}
	}
	golden(t, f, c.Name, wantOf(a))
	replay := served[e.ID]
	for _, why := range replay.Strays() {
		t.Errorf("stray request: %s", why)
	}
	hits := replay.Hits()
	if len(hits) != len(c.Exchanges) {
		t.Errorf("the case got %d matching requests for its %d exchanges. Each exchange matches exactly one request.", len(hits), len(c.Exchanges))
	}
	spaced(t, hits, max(e.RateLimit, port.Floor))
}

// spaced fails two requests to one host closer than floor on the fake
// clock, a followed redirect's included.
func spaced(t *testing.T, hits []Hit, floor time.Duration) {
	t.Helper()
	last := map[string]time.Time{}
	for _, h := range hits {
		if at, ok := last[h.Host]; ok && h.At.Sub(at) < floor {
			t.Errorf("exchange %d reached %s %s after the one before, under the %s floor", h.Exchange+1, h.Host, h.At.Sub(at), floor)
		}
		last[h.Host] = h.At
	}
}

func sessionOf(s *Session, id port.ID) *searching.Session {
	if s == nil {
		return nil
	}
	out := &searching.Session{UserAgent: s.UserAgent, Token: s.Token, Pasted: s.Pasted, Earned: time.Now()}
	for pair := range strings.SplitSeq(s.Cookie, ";") {
		if name, value, ok := strings.Cut(strings.TrimSpace(pair), "="); ok {
			out.Cookies = append(out.Cookies, searching.Cookie{Name: name, Value: value, Domain: Host(id), Path: "/", HostOnly: true})
		}
	}
	return out
}

// golden compares w with want/<case>.json, or rewrites it under -update.
func golden(t *testing.T, f Fixture, name string, w Want) {
	t.Helper()
	got, err := json.MarshalIndent(w, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	got = append(got, '\n')
	if *update {
		path := filepath.Join(string(f.Port), "want", name+".json")
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, got, 0o644); err != nil {
			t.Fatal(err)
		}
		return
	}
	want, ok := f.Wants[name]
	if !ok {
		t.Fatalf("want/%s.json isn't there. Run go test ./fakes/site/ports -run TestPorts -update to write it, and read it before committing:\n%s", name, got)
	}
	if !bytes.Equal(got, want) {
		t.Errorf("the case returned\n%s\nand want/%s.json holds\n%s", got, name, want)
	}
}

// Want is a case's golden: the Releases with every field the contract
// carries, or the outcome, and whether the session changed.
type Want struct {
	Releases       []Release `json:"releases,omitzero"`
	Outcome        *Outcome  `json:"outcome,omitempty"`
	Got            *Got      `json:"got,omitempty"`
	Report         *Report   `json:"report,omitempty"`
	SessionChanged bool      `json:"sessionChanged"`
}

type Release struct {
	Title      string            `json:"title"`
	Size       int64             `json:"size"`
	Published  string            `json:"published"`
	Seeders    int               `json:"seeders"`
	Leechers   int               `json:"leechers"`
	Grabs      int               `json:"grabs"`
	Link       string            `json:"link"`
	Magnet     string            `json:"magnet"`
	InfoHash   string            `json:"infoHash"`
	Categories []int             `json:"categories"`
	IDs        map[string]string `json:"ids"`
	Factors    Factors           `json:"factors"`
	MinRatio   float64           `json:"minRatio"`
	MinSeed    string            `json:"minSeed"`
	Details    string            `json:"details"`
	Source     *searching.Source `json:"source"`
	Flags      []searching.Flag  `json:"flags"`
}

type Outcome struct {
	Kind       string `json:"kind"`
	Cause      string `json:"cause,omitempty"`
	Text       string `json:"text,omitempty"`
	RetryAfter string `json:"retryAfter,omitempty"`
}

// Factors and Flags are a Release's, named as the wire names them.
type Factors struct {
	Download float64 `json:"download"`
	Upload   float64 `json:"upload"`
}

type Got struct {
	TorrentBytes int    `json:"torrentBytes,omitempty"`
	Magnet       string `json:"magnet,omitempty"`
}

type Report struct {
	Modes      map[string][]string `json:"modes"`
	Categories []int               `json:"categories"`
	Items      int                 `json:"items"`
	Sources    []string            `json:"sources"`
}

func wantOf(a searching.Answer) Want {
	w := Want{SessionChanged: a.Session != nil}
	if a.Outcome == nil && a.Got == nil && a.Report == nil {
		w.Releases = []Release{}
	}
	for _, r := range a.Releases {
		rel := Release{
			Title: r.Title, Size: r.Size, Seeders: r.Seeders, Leechers: r.Leechers, Grabs: r.Grabs,
			Link: r.Link.Reveal(), Magnet: r.Magnet.Reveal(), Categories: orEmpty(r.Categories), IDs: r.IDs,
			Factors:  Factors(r.Factors),
			MinRatio: r.Floor.MinRatio, MinSeed: r.Floor.MinSeed.String(), Details: r.Details, Source: r.Source, Flags: orEmpty(r.Flags),
		}
		if !r.Published.IsZero() {
			rel.Published = r.Published.UTC().Format(time.RFC3339)
		}
		if r.InfoHash != nil {
			rel.InfoHash = r.InfoHash.String()
		}
		w.Releases = append(w.Releases, rel)
	}
	w.Outcome = outcomeOf(a.Outcome)
	switch g := a.Got.(type) {
	case searching.Torrent:
		w.Got = &Got{TorrentBytes: len(g.Raw)}
	case searching.Magnet:
		w.Got = &Got{Magnet: g.Link.Reveal()}
	}
	if r := a.Report; r != nil {
		w.Report = &Report{Modes: r.Capabilities.Modes, Categories: r.Capabilities.Categories, Items: r.Items, Sources: r.Sources}
	}
	return w
}

func outcomeOf(err error) *Outcome {
	if err == nil {
		return nil
	}
	if s, ok := errors.AsType[*searching.Stopped](err); ok {
		return &Outcome{Kind: "stopped", Cause: string(s.Cause), Text: s.Text}
	}
	if l, ok := errors.AsType[*searching.Limited](err); ok {
		o := &Outcome{Kind: "limited"}
		if l.RetryAfter > 0 {
			o.RetryAfter = l.RetryAfter.String()
		}
		return o
	}
	return &Outcome{Kind: "failing", Text: err.Error()}
}

func orEmpty[T any](s []T) []T {
	if s == nil {
		return []T{}
	}
	return s
}

// TestScrubbed runs the scrub check over every fixture.
func TestScrubbed(t *testing.T) {
	dirs, err := FS.ReadDir(".")
	if err != nil {
		t.Fatal(err)
	}
	for _, d := range dirs {
		t.Run(d.Name(), func(t *testing.T) {
			f, err := Embedded(port.ID(d.Name()))
			if err != nil {
				t.Fatal(err)
			}
			for _, found := range Scrub(f) {
				t.Error(found)
			}
		})
	}
}
