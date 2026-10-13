// Package mux is the link's stream multiplexer and the only yamux importer,
// so switching libraries is this one file. The yamux wire protocol is on
// the frozen path.
package mux

import (
	"context"
	"fmt"
	"io"
	"net"
	"time"

	"github.com/libp2p/go-yamux/v5"
)

type Config struct {
	// Keepalive is how often a session pings; a ping unanswered within
	// WriteTimeout closes it.
	Keepalive    time.Duration
	WriteTimeout time.Duration
	// StreamWindow is the largest receive window one stream grows to.
	StreamWindow uint32
}

// MinStreamWindow is yamux's initial window, which a stream never shrinks
// below.
const MinStreamWindow = 256 << 10

// Check reports a Config no session can run with.
func (c Config) Check() error {
	if c.StreamWindow < MinStreamWindow {
		return fmt.Errorf("stream_window %d is under %d bytes", c.StreamWindow, MinStreamWindow)
	}
	return nil
}

func (c Config) yamux() *yamux.Config {
	y := yamux.DefaultConfig()
	y.KeepAliveInterval = c.Keepalive
	y.ConnectionWriteTimeout = c.WriteTimeout
	y.MaxStreamWindowSize = c.StreamWindow
	y.LogOutput = io.Discard
	return y
}

// Stream is one multiplexed stream. Reset abandons it in both directions.
type Stream interface {
	net.Conn
	Reset() error
}

type Session struct {
	s *yamux.Session
}

func Client(conn net.Conn, cfg Config) (*Session, error) {
	s, err := yamux.Client(conn, cfg.yamux(), nil)
	if err != nil {
		return nil, fmt.Errorf("mux: %w", err)
	}
	return &Session{s}, nil
}

func Server(conn net.Conn, cfg Config) (*Session, error) {
	s, err := yamux.Server(conn, cfg.yamux(), nil)
	if err != nil {
		return nil, fmt.Errorf("mux: %w", err)
	}
	return &Session{s}, nil
}

func (s *Session) Open(ctx context.Context) (Stream, error) {
	st, err := s.s.OpenStream(ctx)
	if err != nil {
		return nil, fmt.Errorf("mux: open stream: %w", err)
	}
	return st, nil
}

func (s *Session) Accept() (Stream, error) {
	st, err := s.s.AcceptStream()
	if err != nil {
		return nil, fmt.Errorf("mux: accept stream: %w", err)
	}
	return st, nil
}

// Close ends the session, its streams and its connection.
func (s *Session) Close() error {
	if err := s.s.Close(); err != nil {
		return fmt.Errorf("mux: %w", err)
	}
	return nil
}

// Done is closed once the session has ended, by either side, a failed
// keepalive or its connection.
func (s *Session) Done() <-chan struct{} { return s.s.CloseChan() }
