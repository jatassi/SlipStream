package wire

import (
	"net"
	"sync"
)

// Conns is a net.Listener yielding the conns handed to Push, so an
// http.Server serves streams the link accepted and sorted by kind.
type Conns struct {
	ch   chan net.Conn
	done chan struct{}
	once sync.Once
}

func NewConns() *Conns {
	return &Conns{ch: make(chan net.Conn), done: make(chan struct{})}
}

// Push hands c to the next Accept, or closes it once the listener closed.
func (l *Conns) Push(c net.Conn) {
	select {
	case l.ch <- c:
	case <-l.done:
		_ = c.Close()
	}
}

func (l *Conns) Accept() (net.Conn, error) {
	select {
	case c := <-l.ch:
		return c, nil
	case <-l.done:
		return nil, net.ErrClosed
	}
}

func (l *Conns) Close() error {
	l.once.Do(func() { close(l.done) })
	return nil
}

func (l *Conns) Addr() net.Addr { return linkAddr{} }

type linkAddr struct{}

func (linkAddr) Network() string { return "link" }
func (linkAddr) String() string  { return "link" }
