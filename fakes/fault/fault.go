// Package fault is the one fault vocabulary for both offline tiers. Wrap
// applies a Switch's Fault to a net.Conn, and Network is an in-memory
// network whose every path between two addresses carries a Switch. Its
// waits are channels and timers, so it runs inside a testing/synctest
// bubble on fake time.
package fault

import (
	"errors"
	"fmt"
	"net"
	"sync"
	"time"
)

// Fault is one of None, Stall, Reset, Latency or CutAfter.
type Fault interface{ fault() }

type (
	None struct{}
	// Stall drops writes and holds reads until the Switch changes, like a
	// path that lost its state.
	Stall struct{}
	// Reset kills every connection on the path and refuses new dials.
	Reset struct{}
	// Latency delays each write one way.
	Latency time.Duration
	// CutAfter lets each end write n more bytes, then cuts its connection.
	CutAfter int
)

func (None) fault()     {}
func (Stall) fault()    {}
func (Reset) fault()    {}
func (Latency) fault()  {}
func (CutAfter) fault() {}

var (
	ErrReset   = errors.New("connection reset")
	ErrCut     = errors.New("connection cut")
	ErrRefused = errors.New("connection refused")
)

// Switch holds one path's Fault. Only a test flipping it changes a fault.
// The zero Switch is None.
type Switch struct {
	mu      sync.Mutex
	fault   Fault
	changed chan struct{}
	conns   map[*conn]struct{}
}

func (s *Switch) Set(f Fault) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.fault = f
	if s.changed != nil {
		close(s.changed)
		s.changed = nil
	}
	if _, ok := f.(Reset); ok {
		for c := range s.conns {
			c.kill(ErrReset)
		}
		clear(s.conns)
	}
}

// state returns the current Fault and a channel closed when it next
// changes. The channel is made by the caller, so inside the bubble.
func (s *Switch) state() (f Fault, changed <-chan struct{}) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.changed == nil {
		s.changed = make(chan struct{})
	}
	if s.fault == nil {
		return None{}, s.changed
	}
	return s.fault, s.changed
}

func (s *Switch) add(c *conn) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.fault.(Reset); ok {
		c.kill(ErrReset)
		return
	}
	if s.conns == nil {
		s.conns = map[*conn]struct{}{}
	}
	s.conns[c] = struct{}{}
}

func (s *Switch) remove(c *conn) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.conns, c)
}

type conn struct {
	net.Conn
	sw *Switch

	once sync.Once
	dead chan struct{}
	err  error

	wmu       sync.Mutex
	countFrom <-chan struct{}
	sent      int
}

// Wrap applies sw's current Fault to every Read and Write on c. A Reset
// or a cut is final for c, as it is for a TCP connection.
func Wrap(c net.Conn, sw *Switch) net.Conn {
	w := &conn{Conn: c, sw: sw, dead: make(chan struct{})}
	sw.add(w)
	return w
}

func (c *conn) Read(b []byte) (int, error) {
	n, err := c.Conn.Read(b)
	for {
		f, changed := c.sw.state()
		if _, ok := f.(Stall); !ok {
			break
		}
		select {
		case <-changed:
		case <-c.dead:
			return 0, c.err
		}
	}
	return n, c.cause(err)
}

func (c *conn) Write(b []byte) (int, error) {
	c.wmu.Lock()
	defer c.wmu.Unlock()
	f, changed := c.sw.state()
	switch f := f.(type) {
	case None:
		return c.write(b)
	case Stall:
		return len(b), c.cause(nil)
	case Reset:
		return 0, c.cause(ErrReset)
	case Latency:
		t := time.NewTimer(time.Duration(f))
		defer t.Stop()
		select {
		case <-t.C:
			return c.write(b)
		case <-c.dead:
			return 0, c.err
		}
	case CutAfter:
		if changed != c.countFrom {
			c.countFrom, c.sent = changed, 0
		}
		left := max(int(f)-c.sent, 0)
		if len(b) <= left {
			n, err := c.write(b)
			c.sent += n
			return n, err
		}
		n, err := c.write(b[:left])
		c.sent += n
		if err != nil {
			return n, err
		}
		c.end(ErrCut)
		return n, ErrCut
	default:
		panic(fmt.Sprintf("fault: unknown Fault %T", f))
	}
}

func (c *conn) Close() error {
	c.end(net.ErrClosed)
	return nil
}

func (c *conn) write(b []byte) (int, error) {
	n, err := c.Conn.Write(b)
	return n, c.cause(err)
}

// cause reports why a dead conn failed in place of the pipe's own error.
func (c *conn) cause(err error) error {
	select {
	case <-c.dead:
		return c.err
	default:
		return err
	}
}

func (c *conn) end(err error) {
	c.sw.remove(c)
	c.kill(err)
}

func (c *conn) kill(err error) {
	c.once.Do(func() {
		c.err = err
		close(c.dead)
		_ = c.Conn.Close()
	})
}
