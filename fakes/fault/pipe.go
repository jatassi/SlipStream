package fault

import (
	"io"
	"net"
	"os"
	"sync"
	"time"
)

// Pipe is net.Pipe with an unbounded buffer each way. A net.Pipe write
// waits for the peer's read, so two peers that both write before reading
// deadlock, where TCP's buffers let them through.
func Pipe(a, b net.Addr) (atA, atB net.Conn) {
	ab, ba := &stream{}, &stream{}
	return &end{in: ba, out: ab, local: a, remote: b}, &end{in: ab, out: ba, local: b, remote: a}
}

// stream is one direction of a pipe.
type stream struct {
	mu       sync.Mutex
	buf      []byte
	eof      bool // the writing end closed
	shut     bool // the reading end closed
	deadline time.Time
	timer    *time.Timer
	wake     chan struct{}
}

// changed wakes a waiting Read. Call it with mu held.
func (s *stream) changed() {
	if s.wake != nil {
		close(s.wake)
		s.wake = nil
	}
}

func (s *stream) read(b []byte) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for {
		switch {
		case s.shut:
			return 0, net.ErrClosed
		case len(s.buf) > 0:
			n := copy(b, s.buf)
			s.buf = s.buf[n:]
			return n, nil
		case s.eof:
			return 0, io.EOF
		case !s.deadline.IsZero() && !time.Now().Before(s.deadline):
			return 0, os.ErrDeadlineExceeded
		}
		if s.wake == nil {
			s.wake = make(chan struct{})
		}
		wake := s.wake
		s.mu.Unlock()
		<-wake
		s.mu.Lock()
	}
}

func (s *stream) write(b []byte) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	switch {
	case s.eof:
		return 0, net.ErrClosed
	case s.shut:
		return 0, io.ErrClosedPipe
	}
	s.buf = append(s.buf, b...)
	s.changed()
	return len(b), nil
}

func (s *stream) setDeadline(t time.Time) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.deadline = t
	if s.timer != nil {
		s.timer.Stop()
		s.timer = nil
	}
	if !t.IsZero() {
		s.timer = time.AfterFunc(time.Until(t), func() {
			s.mu.Lock()
			defer s.mu.Unlock()
			s.changed()
		})
	}
	s.changed()
}

func (s *stream) close(reading bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if reading {
		s.shut = true
		if s.timer != nil {
			s.timer.Stop()
		}
	} else {
		s.eof = true
	}
	s.changed()
}

type end struct {
	in, out       *stream
	local, remote net.Addr

	mu            sync.Mutex
	writeDeadline time.Time
}

func (e *end) Read(b []byte) (int, error) { return e.in.read(b) }

// Write never waits, so a write deadline only fails writes made after it.
func (e *end) Write(b []byte) (int, error) {
	e.mu.Lock()
	expired := !e.writeDeadline.IsZero() && !time.Now().Before(e.writeDeadline)
	e.mu.Unlock()
	if expired {
		return 0, os.ErrDeadlineExceeded
	}
	return e.out.write(b)
}

func (e *end) Close() error {
	e.in.close(true)
	e.out.close(false)
	return nil
}

// CloseWrite ends what this end writes; the peer reads io.EOF after the
// buffered bytes, as a half-closed socket does.
func (e *end) CloseWrite() error {
	e.out.close(false)
	return nil
}

func (e *end) LocalAddr() net.Addr  { return e.local }
func (e *end) RemoteAddr() net.Addr { return e.remote }

func (e *end) SetDeadline(t time.Time) error {
	_ = e.SetWriteDeadline(t)
	return e.SetReadDeadline(t)
}

func (e *end) SetReadDeadline(t time.Time) error {
	e.in.setDeadline(t)
	return nil
}

func (e *end) SetWriteDeadline(t time.Time) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.writeDeadline = t
	return nil
}
