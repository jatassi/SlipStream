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
	"github.com/jatassi/slipstream/fakes/scenario"
)

// Served is a running fake and a client that reaches it, trusting its
// certificate when it serves TLS.
type Served struct {
	Address fake.Address
	Client  *http.Client
}

// Loopback serves spec on 127.0.0.1, behind a leaf from a run CA when
// spec.TLS is set. It stops when the test ends.
func Loopback(t testing.TB, spec fake.Spec, s scenario.Scenario) Served {
	t.Helper()
	l, err := new(net.ListenConfig).Listen(t.Context(), "tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	return serve(t, spec, s, l, &http.Transport{}, newCA(t))
}

// InMemory serves spec over an in-memory network, so it runs inside
// synctest.Test. The client reaches the fake whatever host it dials.
func InMemory(t testing.TB, spec fake.Spec, s scenario.Scenario) Served {
	t.Helper()
	l := &pipeListener{conns: make(chan net.Conn), closed: make(chan struct{})}
	return serve(t, spec, s, l, &http.Transport{
		DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) { return l.dial(ctx) },
	}, newCA(t))
}

func newCA(t testing.TB) *ca.CA {
	t.Helper()
	authority, err := ca.New("faketest")
	if err != nil {
		t.Fatal(err)
	}
	return authority
}

func serve(t testing.TB, spec fake.Spec, s scenario.Scenario, l net.Listener, tr *http.Transport, authority *ca.CA) Served {
	t.Helper()
	at := fake.Address{AddrPort: netip.MustParseAddrPort(l.Addr().String()), TLS: spec.TLS}
	if spec.TLS {
		leaf, err := authority.Leaf("127.0.0.1", spec.Name+".fake.localhost")
		if err != nil {
			t.Fatal(err)
		}
		l = tls.NewListener(l, &tls.Config{Certificates: []tls.Certificate{leaf}, MinVersion: tls.VersionTLS12})
		tr.TLSClientConfig = &tls.Config{RootCAs: authority.Pool(), MinVersion: tls.VersionTLS12}
	}
	server, err := spec.New(s, fake.NewEnv(s, spec.Name, slog.New(slog.NewTextHandler(t.Output(), nil))))
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
	return Served{Address: at, Client: &http.Client{Transport: tr}}
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
