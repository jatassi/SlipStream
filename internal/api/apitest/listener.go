package apitest

import (
	"bufio"
	"context"
	"net"
	"net/http"
	"strings"
	"sync"
)

// Listener is an in-memory net.Listener: each Dial hands Accept one end of
// a net.Pipe. It touches no network, so a server on it runs inside a
// testing/synctest bubble.
type Listener struct {
	addr   net.Addr
	conns  chan net.Conn
	closed chan struct{}
	once   sync.Once
}

// NewListener reports addr as its address.
func NewListener(addr net.Addr) *Listener {
	return &Listener{addr: addr, conns: make(chan net.Conn), closed: make(chan struct{})}
}

func (l *Listener) Accept() (net.Conn, error) {
	select {
	case c := <-l.conns:
		return c, nil
	case <-l.closed:
		return nil, net.ErrClosed
	}
}

func (l *Listener) Close() error {
	l.once.Do(func() { close(l.closed) })
	return nil
}

func (l *Listener) Addr() net.Addr { return l.addr }

// Dial ignores its network and address: there is one server to reach.
func (l *Listener) Dial(ctx context.Context, _, _ string) (net.Conn, error) {
	client, server := net.Pipe()
	select {
	case l.conns <- server:
		return client, nil
	case <-l.closed:
		return nil, net.ErrClosed
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

// Client sends every request to l, each on a new connection, so no idle
// connection outlives a test.
func (l *Listener) Client() *http.Client {
	return &http.Client{Transport: &http.Transport{DialContext: l.Dial, DisableKeepAlives: true}}
}

// ReadMessage reads one Server-Sent Events message, its lines up to the
// blank line that ends it.
func ReadMessage(r *bufio.Reader) (string, error) {
	var msg strings.Builder
	for {
		line, err := r.ReadString('\n')
		if err != nil {
			return msg.String(), err
		}
		if line == "\n" {
			return msg.String(), nil
		}
		msg.WriteString(line)
	}
}
