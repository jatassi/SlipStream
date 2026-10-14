package fault

import (
	"fmt"
	"net"
	"sync"
	"time"
)

// conn plays the connection fault its Plan holds it under on both its
// reads and its writes, so one wrapper on one end faults both ways. Under
// Stall it drops writes and holds reads, like a path that lost its state:
// a write that waited would hang a graceful close on a stalled link. It
// re-reads the Plan on every read and write, so a cleared throttle speeds
// a live connection up at once.
type conn struct {
	net.Conn
	plan  *Plan
	front Front

	// held and gen are guarded by plan.mu. gen changes whenever held does,
	// so a Cut counts from the moment its rule took the connection.
	held *entry
	gen  uint64

	once sync.Once
	dead chan struct{}
	err  error

	rmu, wmu sync.Mutex
	in, out  budget
}

// budget is one direction's bytes under a Cut, counted for one hold.
type budget struct {
	gen  uint64
	used int64
}

func (b *budget) left(c Cut, gen uint64) int64 {
	if b.gen != gen {
		b.gen, b.used = gen, 0
	}
	return max(int64(c)-b.used, 0)
}

// Conn wraps c, one end of a connection to f, so it plays the connection
// faults p's rules hold it under. The connection counts as one occurrence.
func (p *Plan) Conn(c net.Conn, f Front) net.Conn {
	w := &conn{Conn: c, plan: p, front: f, dead: make(chan struct{})}
	p.mu.Lock()
	firing := p.openLocked(w)
	p.mu.Unlock()
	if firing != nil {
		p.report([]Firing{*firing})
	}
	return w
}

// FrontOf is the front a connection Plan.Conn wrapped was reached
// through, looking beneath every wrapper that names the connection it
// wraps with NetConn, as a TLS connection does.
func FrontOf(c net.Conn) (Front, bool) {
	for {
		if w, ok := c.(*conn); ok {
			return w.front, true
		}
		t, ok := c.(interface{ NetConn() net.Conn })
		if !ok {
			return Front{}, false
		}
		c = t.NetConn()
	}
}

// Listener wraps each connection l accepts with p.Conn.
func (p *Plan) Listener(l net.Listener, f Front) net.Listener {
	return wrapping{Listener: l, plan: p, front: f}
}

type wrapping struct {
	net.Listener
	plan  *Plan
	front Front
}

func (l wrapping) Accept() (net.Conn, error) {
	c, err := l.Listener.Accept()
	if err != nil {
		return nil, err //nolint:wrapcheck // the listener's own error, as net.Listener callers expect
	}
	return l.plan.Conn(c, l.front), nil
}

func (c *conn) state() (f Fault, gen uint64, changed <-chan struct{}) {
	c.plan.mu.Lock()
	defer c.plan.mu.Unlock()
	if c.held != nil {
		f = c.held.Fault
	}
	return f, c.gen, c.plan.changedLocked()
}

// step is the most bytes a Rate passes at once, so a cleared throttle
// takes effect within a sixteenth of a second of traffic.
func step(r Rate) int { return int(max(int64(r)/16, 1)) }

func (c *conn) Write(b []byte) (int, error) {
	c.wmu.Lock()
	defer c.wmu.Unlock()
	written := 0
	for written < len(b) {
		n, done, err := c.writeStep(b[written:])
		written += n
		if done || err != nil {
			return written, err
		}
	}
	return written, nil
}

// writeStep writes some of b under the fault c holds now. done is set
// once no more of b goes out.
func (c *conn) writeStep(b []byte) (n int, done bool, err error) {
	f, gen, changed := c.state()
	switch f := f.(type) {
	case nil:
		n, err = c.write(b)
		return n, true, err
	case Stall:
		return len(b), true, c.cause(nil)
	case Latency:
		if again, err := c.pause(time.Duration(f), changed); again || err != nil {
			return 0, false, err
		}
		n, err = c.write(b)
		return n, true, err
	case Throttle:
		if n, err = c.write(b[:min(len(b), step(Rate(f)))]); err != nil {
			return n, true, err
		}
		_, err = c.pause(pace(n, Rate(f)), changed)
		return n, false, err
	case Cut:
		left := c.out.left(f, gen)
		n, err = c.write(b[:min(int64(len(b)), left)])
		c.out.used += int64(n)
		if err == nil && int64(n) == left {
			c.cut()
			err = ErrCut
		}
		return n, false, err
	case Reset:
		return 0, true, c.cause(ErrReset)
	default:
		panic(fmt.Sprintf("fault: %s held a connection", f))
	}
}

func (c *conn) Read(b []byte) (int, error) {
	c.rmu.Lock()
	defer c.rmu.Unlock()
	f, gen, _ := c.state()
	switch f := f.(type) {
	case Throttle:
		b = b[:min(len(b), step(Rate(f)))]
	case Cut:
		left := c.in.left(f, gen)
		if left == 0 {
			c.cut()
			return 0, ErrCut
		}
		b = b[:min(int64(len(b)), left)]
	}
	n, err := c.Conn.Read(b)
	for {
		m, again, held := c.deliver(n)
		if held != nil {
			return 0, held
		}
		if !again {
			return m, c.cause(err)
		}
	}
}

// deliver holds n bytes just read under the fault c holds now, and says
// how many to return. again is set when the Plan changed while it held
// them, so they wait on the new state.
func (c *conn) deliver(n int) (m int, again bool, err error) {
	f, gen, changed := c.state()
	switch f := f.(type) {
	case Stall:
		return n, true, c.wait(changed)
	case Latency:
		again, err = c.pause(time.Duration(f), changed)
		return n, again, err
	case Throttle:
		_, err = c.pause(pace(n, Rate(f)), changed)
		return n, false, err
	case Cut:
		n = int(min(int64(n), c.in.left(f, gen)))
		c.in.used += int64(n)
		if c.in.left(f, gen) == 0 {
			c.cut()
		}
	}
	return n, false, nil
}

// pace is how long n bytes take at r.
func pace(n int, r Rate) time.Duration {
	return time.Duration(int64(n) * int64(time.Second) / int64(r))
}

// wait blocks until the Plan changes, or fails once c is dead.
func (c *conn) wait(changed <-chan struct{}) error {
	select {
	case <-changed:
		return nil
	case <-c.dead:
		return c.err
	}
}

// pause waits d. It ends early with again set if the Plan changes, and
// fails once c is dead.
func (c *conn) pause(d time.Duration, changed <-chan struct{}) (again bool, err error) {
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-t.C:
		return false, nil
	case <-changed:
		return true, nil
	case <-c.dead:
		return false, c.err
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

// cause reports why a dead conn failed in place of the transport's own
// error.
func (c *conn) cause(err error) error {
	select {
	case <-c.dead:
		return c.err
	default:
		return err
	}
}

func (c *conn) cut() { c.end(ErrCut) }

func (c *conn) end(err error) {
	c.plan.mu.Lock()
	c.plan.dropLocked(c)
	c.plan.mu.Unlock()
	c.kill(err, false)
}

// kill ends c with err. An abort resets the transport, so its peer fails
// at once: a TCP connection sends RST, and a Pipe hands its peer err.
func (c *conn) kill(err error, abort bool) {
	c.once.Do(func() {
		c.err = err
		close(c.dead)
		if abort {
			switch t := c.Conn.(type) {
			case *end:
				t.abort(err)
				return
			case interface{ SetLinger(sec int) error }:
				_ = t.SetLinger(0)
			}
		}
		_ = c.Conn.Close()
	})
}
