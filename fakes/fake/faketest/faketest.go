// Package faketest serves one fake.Spec in a test, over loopback the way
// the world serves it or over an in-memory network inside synctest.Test.
package faketest

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"net/netip"
	"os"
	"path/filepath"
	"slices"
	"sync"
	"testing"

	"github.com/jatassi/slipstream/fakes/events"
	"github.com/jatassi/slipstream/fakes/fake"
	"github.com/jatassi/slipstream/fakes/fault"
	"github.com/jatassi/slipstream/fakes/internal/ca"
	"github.com/jatassi/slipstream/fakes/internal/releases"
	"github.com/jatassi/slipstream/fakes/scenario"
	"github.com/jatassi/slipstream/internal/node"
)

// Served is a running fake and a client that reaches it, trusting its
// certificate when it serves TLS and dialing each site host the Spec's
// Endpoints puts in endpoints.searching at the fake, as a Node does. Roots
// is the CA its certificates chain to, for a client of another protocol.
// Faults plays on every connection and request, and a test may set and
// clear its rules as it goes. RunDir holds the event log its captures
// write. Media is the fake's media dir, where a test places the files a
// Media Server fake reads.
type Served struct {
	Address fake.Address
	Client  *http.Client
	Roots   *x509.CertPool
	Faults  *fault.Plan
	RunDir  string
	Media   string
	env     fake.Env
	from    func(origin string) *http.Client
}

// From is a client whose requests arrive through origin's front, which
// WithOrigins declared, so a fault rule can name origin, and whose dials a
// refuse rule on that front refuses. InMemory only.
func (s Served) From(t testing.TB, origin string) *http.Client {
	t.Helper()
	if s.from == nil {
		t.Fatal("faketest: From needs a fake served InMemory")
	}
	return s.from(origin)
}

// Cue plays the fake's cue name with args, as fakeworld cue does, and
// answers its result.
func (s Served) Cue(t testing.TB, name string, args any) any {
	t.Helper()
	raw, err := json.Marshal(args)
	if err != nil {
		t.Fatal(err)
	}
	out, err := s.env.Cue(t.Context(), name, raw)
	if err != nil {
		t.Fatal(err)
	}
	return out
}

// Records are the records the fake's captures wrote so far.
func (s Served) Records(t testing.TB) []events.Record {
	t.Helper()
	recs, err := events.Read(s.RunDir)
	if err != nil {
		t.Fatal(err)
	}
	return recs
}

type options struct {
	releases *releases.Catalog
	rules    []fault.Rule
	fronts   fake.Fronts
	origin   string
	origins  []string
	media    string
	shared   *fake.Shared
}

type Option func(*options)

// WithReleases hands the fake a release catalog, as the world does.
func WithReleases(c *releases.Catalog) Option {
	return func(o *options) { o.releases = c }
}

// Catalog is a release catalog of one release per name, each a payload of
// size bytes drawn from its name and placed in a test folder as
// <name>.mkv, as the world's catalog places it.
func Catalog(t testing.TB, size int, names ...string) *releases.Catalog {
	t.Helper()
	dir := t.TempDir()
	c := &releases.Catalog{}
	for _, name := range names {
		payload := releases.Filler(name, int64(size))
		file := filepath.Join(dir, name+".mkv")
		if err := os.WriteFile(file, payload, 0o600); err != nil {
			t.Fatal(err)
		}
		meta, hash, err := releases.Metainfo(name+".mkv", nil, payload, "https://127.0.0.1/announce/faketest", false)
		if err != nil {
			t.Fatal(err)
		}
		c.Releases = append(c.Releases, releases.Release{
			Release: scenario.Release{Name: name}, Dir: dir, Files: []releases.File{{Path: name + ".mkv", Size: int64(size)}}, Size: int64(size), Metainfo: meta, InfoHash: hash,
		})
	}
	return c
}

// WithFronts hands the fake the world's fronts, as the world does.
func WithFronts(fronts fake.Fronts) Option {
	return func(o *options) { o.fronts = fronts }
}

// WithMedia hands the fake dir as its media dir, in place of a new test
// folder.
func WithMedia(dir string) Option {
	return func(o *options) { o.media = dir }
}

// WithShared hands the fake shared, as the world hands every fake of a
// run one, so a test serves fakes that share state.
func WithShared(shared *fake.Shared) Option {
	return func(o *options) { o.shared = shared }
}

// As has every request arrive from origin, as through that Node's front.
func As(origin string) Option {
	return func(o *options) { o.origin = origin }
}

// WithOrigins declares one more front per origin, which Served.From
// dials, as the world's fronts give each Node its own.
func WithOrigins(origins ...string) Option {
	return func(o *options) { o.origins = append(o.origins, origins...) }
}

// WithFaults sets rules before the fake serves, after the Scenario's own
// rules for the fake. Their Target is the fake's name, and its one front
// carries connections and HTTP.
func WithFaults(rules ...fault.Rule) Option {
	return func(o *options) { o.rules = append(o.rules, rules...) }
}

// Loopback serves spec on 127.0.0.1, behind a leaf from a run CA for a
// FrontTLS spec and with that leaf in its Env for an OwnTLS one. It stops when the test ends.
func Loopback(t testing.TB, spec fake.Spec, s scenario.Scenario, opts ...Option) Served {
	t.Helper()
	l, err := new(net.ListenConfig).Listen(t.Context(), "tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	return serve(t, spec, s, l, &http.Transport{DialContext: new(net.Dialer).DialContext}, newCA(t), opts...)
}

// InMemory serves spec over an in-memory network, so it runs inside
// synctest.Test. The client reaches the fake whatever host it dials.
func InMemory(t testing.TB, spec fake.Spec, s scenario.Scenario, opts ...Option) Served {
	t.Helper()
	l := &pipeListener{conns: make(chan net.Conn), closed: make(chan struct{})}
	served := serve(t, spec, s, l, &http.Transport{
		DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) { return l.dial(ctx, "") },
	}, newCA(t), opts...)
	tr, _ := served.Client.Transport.(*http.Transport)
	served.from = func(origin string) *http.Client {
		via := tr.Clone()
		front := fault.Front{Target: fault.Target(spec.Name), Origin: origin, Layers: fault.Conn | fault.HTTP}
		via.DialContext = func(ctx context.Context, _, _ string) (net.Conn, error) {
			if served.Faults.Refused(front) {
				return nil, &net.OpError{Op: "dial", Net: "tcp", Addr: l.Addr(), Err: fault.ErrRefused}
			}
			return l.dial(ctx, origin)
		}
		return &http.Client{Transport: via}
	}
	return served
}

func newCA(t testing.TB) *ca.CA {
	t.Helper()
	authority, err := ca.New("faketest")
	if err != nil {
		t.Fatal(err)
	}
	return authority
}

func serve(t testing.TB, spec fake.Spec, s scenario.Scenario, l net.Listener, tr *http.Transport, authority *ca.CA, opts ...Option) Served {
	t.Helper()
	var o options
	for _, opt := range opts {
		opt(&o)
	}
	fronts := map[string]fault.Front{"": {Target: fault.Target(spec.Name), Origin: o.origin, Layers: fault.Conn | fault.HTTP}}
	all := []fault.Front{fronts[""]}
	for _, origin := range o.origins {
		fronts[origin] = fault.Front{Target: fault.Target(spec.Name), Origin: origin, Layers: fault.Conn | fault.HTTP}
		all = append(all, fronts[origin])
	}
	plan := fault.NewPlan(all, nil)
	at := fake.Address{AddrPort: netip.MustParseAddrPort(l.Addr().String()), TLS: spec.TLS == fake.FrontTLS}
	l = fronted{Listener: l, plan: plan, fronts: fronts}
	if spec.Endpoints != nil {
		var e node.EndpointsBlock
		spec.Endpoints(&e, at)
		tr.DialContext = e.Searching.Dial(tr.DialContext)
	}
	leaf, err := authority.Leaf(spec.LeafNames()...)
	if err != nil {
		t.Fatal(err)
	}
	served := &tls.Config{Certificates: []tls.Certificate{leaf}, MinVersion: tls.VersionTLS12}
	if spec.TLS == fake.FrontTLS {
		l = tls.NewListener(l, served)
		tr.TLSClientConfig = &tls.Config{RootCAs: authority.Pool(), MinVersion: tls.VersionTLS12}
	}
	runDir := t.TempDir()
	log, err := events.Create(runDir, func(events.Record) []events.Record { return nil })
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = log.Close() })
	env := fake.NewEnv(s, spec, slog.New(slog.NewTextHandler(t.Output(), nil)), plan, log)
	if o.media == "" {
		o.media = t.TempDir()
	}
	env.Releases, env.Fronts, env.Files, env.Dir, env.Media = o.releases, o.fronts, runDir, t.TempDir(), o.media
	if o.shared != nil {
		env.Shared = o.shared
	}
	if spec.TLS == fake.OwnTLS {
		env.Leaf = served
	}
	server, err := spec.New(s, env)
	if err != nil {
		t.Fatal(err)
	}
	for _, r := range slices.Concat(ownRules(s, spec), o.rules) {
		if _, err := plan.Set(r); err != nil {
			t.Fatal(err)
		}
	}
	done := make(chan error, 1)
	go func() { done <- server.Serve(t.Context(), l) }()
	t.Cleanup(func() {
		tr.CloseIdleConnections()
		if err := <-done; err != nil {
			t.Errorf("fake %s: %v", spec.Name, err)
		}
	})
	return Served{Address: at, Client: &http.Client{Transport: tr}, Roots: authority.Pool(), Faults: plan, RunDir: runDir, Media: o.media, env: env}
}

// ownRules are the Scenario's fault rules whose Target is spec's fake, as
// the world sets them at launch.
func ownRules(s scenario.Scenario, spec fake.Spec) []fault.Rule {
	var out []fault.Rule
	for _, r := range s.Faults {
		if r.Target == fault.Target(spec.Name) {
			out = append(out, r)
		}
	}
	return out
}

// pipeListener hands each dial's far end of a fault.Pipe to Accept. Every
// block is on a channel, so a bubble's goroutines block durably.
// fronted wraps each accepted connection in its origin's front, the
// default front for a connection from no origin.
type fronted struct {
	net.Listener
	plan   *fault.Plan
	fronts map[string]fault.Front
}

func (l fronted) Accept() (net.Conn, error) {
	c, err := l.Listener.Accept()
	if err != nil {
		return nil, err //nolint:wrapcheck // the listener's own error, as net.Listener callers expect
	}
	origin := ""
	if o, ok := c.(originConn); ok {
		c, origin = o.Conn, o.origin
	}
	return l.plan.Conn(c, l.fronts[origin]), nil
}

// originConn is the fake's end of an in-memory connection From dialed.
type originConn struct {
	net.Conn
	origin string
}

type pipeListener struct {
	conns  chan net.Conn
	closed chan struct{}
	once   sync.Once
}

func (l *pipeListener) Accept() (net.Conn, error) {
	select {
	case c := <-l.conns:
		return c, nil
	case <-l.closed:
		return nil, net.ErrClosed
	}
}

func (l *pipeListener) Close() error {
	l.once.Do(func() { close(l.closed) })
	return nil
}

func (l *pipeListener) Addr() net.Addr {
	return net.TCPAddrFromAddrPort(netip.MustParseAddrPort("127.0.0.1:443"))
}

func (l *pipeListener) dial(ctx context.Context, origin string) (net.Conn, error) {
	server, client := fault.Pipe(l.Addr(), l.Addr())
	select {
	case l.conns <- originConn{Conn: server, origin: origin}:
		return client, nil
	case <-l.closed:
		return nil, net.ErrClosed
	case <-ctx.Done():
		return nil, fmt.Errorf("in-memory dial: %w", ctx.Err())
	}
}
