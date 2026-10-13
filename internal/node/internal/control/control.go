// Package control is a Node's local request channel: an AF_UNIX stream
// socket in the data dir on every OS, one request per connection and one JSON
// object per line each way. Access is by the run dir's OS permissions only.
package control

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net"
	"os"
	"sync"
	"time"
)

type Code string

const (
	CodeUnknownOp  Code = "unknown_op"
	CodeNotReady   Code = "not_ready"
	CodeBadRequest Code = "bad_request"
	CodeFailed     Code = "failed"
)

type Error struct {
	Code    Code   `json:"code"`
	Message string `json:"message"`
}

func (e *Error) Error() string { return fmt.Sprintf("%s: %s", e.Code, e.Message) }

const (
	ioTimeout = 5 * time.Second
	maxLine   = 1 << 20
)

type request struct {
	Op   string          `json:"op"`
	Body json.RawMessage `json:"body"`
}

type reply struct {
	Body  json.RawMessage `json:"body,omitempty"`
	Error *Error          `json:"error,omitempty"`
}

// Handler answers one request through send, once or more. An error it
// returns is the connection's last reply: an *Error as is, any other error
// with CodeFailed.
type Handler func(ctx context.Context, body json.RawMessage, send func(any) error) error

type Dispatch func(op string) (Handler, error)

type Server struct {
	ln       net.Listener
	dispatch Dispatch
	ctx      context.Context
	cancel   context.CancelFunc
	accepted chan struct{}
	conns    sync.WaitGroup
}

// ListenFunc binds a listener, as net.ListenConfig.Listen does.
type ListenFunc func(ctx context.Context, network, address string) (net.Listener, error)

// Listen removes a socket left at path by a process that held the same
// lock, then binds path through listen and serves.
func Listen(ctx context.Context, listen ListenFunc, path string, dispatch Dispatch) (*Server, error) {
	if err := os.Remove(path); err != nil && !errors.Is(err, fs.ErrNotExist) {
		return nil, fmt.Errorf("remove stale control socket: %w", err)
	}
	ln, err := listen(ctx, "unix", path)
	if err != nil {
		return nil, fmt.Errorf("listen on control socket: %w", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	s := &Server{ln: ln, dispatch: dispatch, ctx: ctx, cancel: cancel, accepted: make(chan struct{})}
	go s.accept()
	return s, nil
}

func (s *Server) accept() {
	defer close(s.accepted)
	for {
		conn, err := s.ln.Accept()
		if err != nil {
			return
		}
		s.conns.Go(func() { s.serve(conn) })
	}
}

// Close stops accepting and cancels in-flight handlers' contexts. Handlers
// still running keep their connections until Wait.
func (s *Server) Close() error {
	s.cancel()
	err := s.ln.Close()
	<-s.accepted
	if err != nil {
		return fmt.Errorf("close control socket: %w", err)
	}
	return nil
}

// Wait waits for every connection to finish. Call it after Close.
func (s *Server) Wait() { s.conns.Wait() }

func (s *Server) serve(conn net.Conn) {
	defer func() { _ = conn.Close() }()
	send := func(r reply) error {
		line, err := json.Marshal(r)
		if err != nil {
			return fmt.Errorf("encode reply: %w", err)
		}
		if err := conn.SetWriteDeadline(time.Now().Add(ioTimeout)); err != nil {
			return fmt.Errorf("set write deadline: %w", err)
		}
		if _, err := conn.Write(append(line, '\n')); err != nil {
			return fmt.Errorf("write reply: %w", err)
		}
		return nil
	}
	h, body, err := s.read(conn)
	if err == nil {
		err = h(s.ctx, body, func(v any) error {
			b, err := json.Marshal(v)
			if err != nil {
				return fmt.Errorf("encode reply body: %w", err)
			}
			return send(reply{Body: b})
		})
	}
	if err != nil {
		e, ok := errors.AsType[*Error](err)
		if !ok {
			e = &Error{Code: CodeFailed, Message: err.Error()}
		}
		_ = send(reply{Error: e})
	}
}

func (s *Server) read(conn net.Conn) (Handler, json.RawMessage, error) {
	if err := conn.SetReadDeadline(time.Now().Add(ioTimeout)); err != nil {
		return nil, nil, fmt.Errorf("set read deadline: %w", err)
	}
	line, err := readLine(bufio.NewReader(io.LimitReader(conn, maxLine+1)))
	if err != nil {
		return nil, nil, &Error{Code: CodeBadRequest, Message: err.Error()}
	}
	var req request
	if err := Decode(line, &req); err != nil {
		return nil, nil, err
	}
	h, err := s.dispatch(req.Op)
	if err != nil {
		return nil, nil, err
	}
	return h, req.Body, nil
}

func readLine(r *bufio.Reader) ([]byte, error) {
	line, err := r.ReadBytes('\n')
	if errors.Is(err, io.EOF) && len(line) > maxLine {
		return nil, fmt.Errorf("line exceeds %d bytes", maxLine)
	}
	if err != nil {
		return nil, fmt.Errorf("read line: %w", err)
	}
	return line, nil
}

// Decode reads a request or body strictly, failing with CodeBadRequest.
func Decode(data []byte, v any) error {
	if len(bytes.TrimSpace(data)) == 0 {
		data = []byte("{}")
	}
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.DisallowUnknownFields()
	if err := dec.Decode(v); err != nil {
		return &Error{Code: CodeBadRequest, Message: err.Error()}
	}
	return nil
}

type Conn struct {
	conn *net.UnixConn
	r    *bufio.Reader
}

// Dial connects to the socket at path. The connection's deadline is ctx's.
func Dial(ctx context.Context, path string) (*Conn, error) {
	var d net.Dialer
	conn, err := d.DialUnix(ctx, "unix", nil, &net.UnixAddr{Name: path, Net: "unix"})
	if err != nil {
		return nil, fmt.Errorf("dial control socket: %w", err)
	}
	if deadline, ok := ctx.Deadline(); ok {
		if err := conn.SetDeadline(deadline); err != nil {
			return nil, errors.Join(fmt.Errorf("set deadline: %w", err), conn.Close())
		}
	}
	return &Conn{conn: conn, r: bufio.NewReader(io.LimitReader(conn, maxLine))}, nil
}

// Send writes the request and then closes the write side, which works
// around Windows AF_UNIX reads that never see the Node close the connection.
func (c *Conn) Send(op string, body any) error {
	b, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("encode %s request: %w", op, err)
	}
	line, err := json.Marshal(request{Op: op, Body: b})
	if err != nil {
		return fmt.Errorf("encode %s request: %w", op, err)
	}
	if _, err := c.conn.Write(append(line, '\n')); err != nil {
		return fmt.Errorf("send %s request: %w", op, err)
	}
	if err := c.conn.CloseWrite(); err != nil {
		return fmt.Errorf("send %s request: %w", op, err)
	}
	return nil
}

// Recv reads the next reply into v. An error reply is an *Error, and a
// connection the Node closed without another reply is io.EOF.
func (c *Conn) Recv(v any) error {
	line, err := c.r.ReadBytes('\n')
	if errors.Is(err, io.EOF) && len(line) == 0 {
		return io.EOF
	}
	if err != nil {
		return fmt.Errorf("read reply: %w", err)
	}
	var r reply
	if err := json.Unmarshal(line, &r); err != nil {
		return fmt.Errorf("decode reply: %w", err)
	}
	if r.Error != nil {
		return r.Error
	}
	if err := json.Unmarshal(r.Body, v); err != nil {
		return fmt.Errorf("decode reply body: %w", err)
	}
	return nil
}

func (c *Conn) SetDeadline(t time.Time) error {
	if err := c.conn.SetDeadline(t); err != nil {
		return fmt.Errorf("set deadline: %w", err)
	}
	return nil
}

func (c *Conn) Close() error {
	if err := c.conn.Close(); err != nil {
		return fmt.Errorf("close control connection: %w", err)
	}
	return nil
}
