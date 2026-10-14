// Package control is a run's control API: HTTP/1.1 over a Unix socket,
// which the supervisor, or the test process for an in-process run, serves
// with the world's handler. Every route is a write; reads go to the
// run's files.
package control

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"syscall"
	"time"
)

type Server struct {
	srv    *http.Server
	l      net.Listener
	closed chan struct{}
	done   chan struct{}
}

// Serve answers h's routes on the socket at path.
func Serve(ctx context.Context, path string, h http.Handler) (*Server, error) {
	l, err := new(net.ListenConfig).Listen(ctx, "unix", path)
	if err != nil {
		return nil, fmt.Errorf("control socket: %w", err)
	}
	s := &Server{
		srv: &http.Server{
			Handler:           h,
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
var ErrNoAnswer = errors.New("the control socket did not answer")

// Do sends req on its own connection to the socket at path and returns
// the answer's status and body. It closes its write side once req is
// sent, since a Windows AF_UNIX read never sees the server close.
func Do(ctx context.Context, path string, req *http.Request) (status int, body []byte, err error) {
	conn, err := new(net.Dialer).DialUnix(ctx, "unix", nil, &net.UnixAddr{Name: path, Net: "unix"})
	if err != nil {
		if ctx.Err() == nil {
			return 0, nil, fmt.Errorf("%w: %w", ErrNoAnswer, err)
		}
		return 0, nil, fmt.Errorf("control socket: %w", err)
	}
	defer func() { _ = conn.Close() }()
	if deadline, ok := ctx.Deadline(); ok {
		if err := conn.SetDeadline(deadline); err != nil {
			return 0, nil, fmt.Errorf("control socket: %w", err)
		}
	}
	if err := req.Write(conn); err != nil {
		return 0, nil, fmt.Errorf("control socket: %w", err)
	}
	// macOS answers ENOTCONN once the server has answered and closed; the
	// answer stays readable.
	if err := conn.CloseWrite(); err != nil && !errors.Is(err, syscall.ENOTCONN) {
		return 0, nil, fmt.Errorf("control socket: %w", err)
	}
	res, err := http.ReadResponse(bufio.NewReader(conn), req)
	if err != nil {
		return 0, nil, fmt.Errorf("control socket: %w", err)
	}
	defer func() { _ = res.Body.Close() }()
	if body, err = io.ReadAll(res.Body); err != nil {
		return 0, nil, fmt.Errorf("control socket: %w", err)
	}
	return res.StatusCode, bytes.TrimSpace(body), nil
}
