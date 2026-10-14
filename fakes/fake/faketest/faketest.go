// Package faketest serves one fake.Spec in a test, over loopback the way
// the world serves it or over an in-memory network inside synctest.Test.
package faketest

import (
	"context"
	"crypto/tls"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"net/netip"
	"sync"
	"testing"

	"github.com/jatassi/slipstream/fakes/fake"
	"github.com/jatassi/slipstream/fakes/fault"
	"github.com/jatassi/slipstream/fakes/internal/ca"
	"github.com/jatassi/slipstream/fakes/internal/releases"
	"github.com/jatassi/slipstream/fakes/scenario"
)

// Served is a running fake and a client that reaches it, trusting its
// certificate when it serves TLS. Faults plays on every connection and
// request, and a test may set and clear its rules as it goes.
type Served struct {
	Address fake.Address
	Client  *http.Client
	Faults  *fault.Plan
}

type options struct {
	releases *releases.Catalog
	rules    []fault.Rule
}

type Option func(*options)

// WithReleases hands the fake a release catalog, as the world does.
func WithReleases(c *releases.Catalog) Option {
	return func(o *options) { o.releases = c }
}

// WithFaults sets rules before the fake serves. Their Target is the
// fake's name, and its one front carries connections and HTTP.
func WithFaults(rules ...fault.Rule) Option {
	return func(o *options) { o.rules = append(o.rules, rules...) }
}

// Loopback serves spec on 127.0.0.1, behind a leaf from a run CA when
// spec.TLS is set. It stops when the test ends.
func Loopback(t testing.TB, spec fake.Spec, s scenario.Scenario, opts ...Option) Served {
	t.Helper()
	l, err := new(net.ListenConfig).Listen(t.Context(), "tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	return serve(t, spec, s, l, &http.Transport{}, newCA(t), opts...)
}

// InMemory serves spec over an in-memory network, so it runs inside
// synctest.Test. The client reaches the fake whatever host it dials.
func InMemory(t testing.TB, spec fake.Spec, s scenario.Scenario, opts ...Option) Served {
	t.Helper()
	l := &pipeListener{conns: make(chan net.Conn), closed: make(chan struct{})}
	return serve(t, spec, s, l, &http.Transport{
		DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) { return l.dial(ctx) },
	}, newCA(t), opts...)
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
	front := fault.Front{Target: fault.Target(spec.Name), Layers: fault.Conn | fault.HTTP}
	plan := fault.NewPlan([]fault.Front{front}, nil)
	for _, r := range o.rules {
		if _, err := plan.Set(r); err != nil {
			t.Fatal(err)
		}
	}
	at := fake.Address{AddrPort: netip.MustParseAddrPort(l.Addr().String()), TLS: spec.TLS}
	l = plan.Listener(l, front)
	if spec.TLS {
		leaf, err := authority.Leaf("127.0.0.1", spec.Name+".fake.localhost")
		if err != nil {
			t.Fatal(err)
		}
		l = tls.NewListener(l, &tls.Config{Certificates: []tls.Certificate{leaf}, MinVersion: tls.VersionTLS12})
		tr.TLSClientConfig = &tls.Config{RootCAs: authority.Pool(), MinVersion: tls.VersionTLS12}
	}
	env := fake.NewEnv(s, spec.Name, slog.New(slog.NewTextHandler(t.Output(), nil)), plan)
	env.Releases = o.releases
	server, err := spec.New(s, env)
	if err != nil {
		t.Fatal(err)
	}
	done := make(chan error, 1)
	go func() { done <- server.Serve(t.Context(), l) }()
	t.Cleanup(func() {
		tr.CloseIdleConnections()
		if err := <-done; err != nil {
			t.Errorf("fake %s: %v", spec.Name, err)
		}
	})
	return Served{Address: at, Client: &http.Client{Transport: tr}, Faults: plan}
}

// pipeListener hands each dial's far end of a fault.Pipe to Accept. Every
// block is on a channel, so a bubble's goroutines block durably.
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

func (l *pipeListener) dial(ctx context.Context) (net.Conn, error) {
	server, client := fault.Pipe(l.Addr(), l.Addr())
	select {
	case l.conns <- server:
		return client, nil
	case <-l.closed:
		return nil, net.ErrClosed
	case <-ctx.Done():
		return nil, fmt.Errorf("in-memory dial: %w", ctx.Err())
	}
}
