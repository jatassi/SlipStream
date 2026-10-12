// Package channel is a run's stop channel: HTTP/1.1 over a Unix socket,
// with the one route POST /v1/stop.
package channel

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"time"
)

type Server struct {
	srv    *http.Server
	l      net.Listener
	closed chan struct{}
	done   chan struct{}
}

// Serve answers POST /v1/stop on the socket at path with what stop
// returns once its work is done.
func Serve(ctx context.Context, path string, stop func() []byte) (*Server, error) {
	l, err := new(net.ListenConfig).Listen(ctx, "unix", path)
	if err != nil {
		return nil, fmt.Errorf("stop channel: %w", err)
	}
	mux := http.NewServeMux()
	mux.HandleFunc("POST /v1/stop", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(stop())
	})
	s := &Server{
		srv: &http.Server{
			Handler:           mux,
			ReadHeaderTimeout: 10 * time.Second,
			BaseContext:       func(net.Listener) context.Context { return context.WithoutCancel(ctx) },
		},
		l:      l,
		closed: make(chan struct{}),
		done:   make(chan struct{}),
	}
	go func() { _ = s.srv.Serve(l) }()
	go func() {
		<-s.closed
		s.srv.SetKeepAlivesEnabled(false)
		_ = s.srv.Shutdown(context.WithoutCancel(ctx))
		close(s.done)
	}()
	return s, nil
}

// Close stops accepting and removes the socket at once. A request already
// in flight still gets its answer; Done is closed once it has.
func (s *Server) Close() {
	_ = s.l.Close()
	close(s.closed)
}

func (s *Server) Done() <-chan struct{} { return s.done }

// ErrNoAnswer is a socket nothing serves: the supervisor is gone.
var ErrNoAnswer = errors.New("the stop channel did not answer")

// Stop sends POST /v1/stop to the socket at path and returns the answer.
func Stop(ctx context.Context, path string) ([]byte, error) {
	client := &http.Client{Transport: &http.Transport{
		DisableKeepAlives: true,
		DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
			return new(net.Dialer).DialContext(ctx, "unix", path)
		},
	}}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "http://stop-channel/v1/stop", http.NoBody)
	if err != nil {
		return nil, fmt.Errorf("stop channel: %w", err)
	}
	res, err := client.Do(req)
	if op, ok := errors.AsType[*net.OpError](err); ok && op.Op == "dial" && ctx.Err() == nil {
		return nil, fmt.Errorf("%w: %w", ErrNoAnswer, err)
	}
	if err != nil {
		return nil, fmt.Errorf("stop channel: %w", err)
	}
	defer func() { _ = res.Body.Close() }()
	body, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, fmt.Errorf("stop channel: %w", err)
	}
	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("stop channel: %d %s", res.StatusCode, bytes.TrimSpace(body))
	}
	return body, nil
}
