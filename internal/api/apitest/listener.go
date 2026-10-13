package apitest

import (
	"bufio"
	"context"
	"fmt"
	"io"
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
		return nil, fmt.Errorf("dial the in-memory listener: %w", ctx.Err())
	}
}

// Client sends every request to l, each on a new connection, so no idle
// connection outlives a test.
func (l *Listener) Client() *http.Client {
	return &http.Client{Transport: &http.Transport{DialContext: l.Dial, DisableKeepAlives: true}}
}

// Stream reads a Server-Sent Events body as a browser does, each message
// as it arrives. A net.Pipe has no buffer, so a message the client leaves
// unread would miss the server's write deadline.
type Stream struct {
	body     io.Closer
	messages chan string
	end      chan error
}

func NewStream(body io.ReadCloser) *Stream {
	s := &Stream{body: body, messages: make(chan string, 4096), end: make(chan error, 1)}
	go func() {
		r := bufio.NewReader(body)
		for {
			msg, err := readMessage(r)
			if err != nil {
				s.end <- err
				close(s.messages)
				return
			}
			s.messages <- msg
		}
	}()
	return s
}

// Next returns the next message, its lines up to the blank line that ends
// it, or the error that ended the stream: io.EOF when the server closed it,
// or ctx's error when ctx ends first.
func (s *Stream) Next(ctx context.Context) (string, error) {
	select {
	case msg, ok := <-s.messages:
		if ok {
			return msg, nil
		}
		return "", <-s.end
	case <-ctx.Done():
		return "", fmt.Errorf("read the stream: %w", ctx.Err())
	}
}

// Close drops the connection, as a closed browser tab does.
func (s *Stream) Close() error {
	if err := s.body.Close(); err != nil {
		return fmt.Errorf("close the stream: %w", err)
	}
	return nil
}

func readMessage(r *bufio.Reader) (string, error) {
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
