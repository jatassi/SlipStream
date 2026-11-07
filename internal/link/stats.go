package link

import (
	"net"
	"sync/atomic"
	"time"
)

// Stats is the traffic of a live session, or of every lane with a peer.
type Stats struct {
	// Sent and Received are bytes on TCP connections, TLS records
	// included.
	Sent, Received int64
	// RTT is the smoothed round trip of the session's pings, sent at the
	// keepalive cadence, and zero until the first is answered. Lanes have
	// none.
	RTT time.Duration
}

// Stats is key's live session's traffic, false with no live session.
func (l *Link) Stats(key Key) (Stats, bool) {
	l.mu.Lock()
	s := l.live[key]
	l.mu.Unlock()
	if s == nil {
		return Stats{}, false
	}
	st := s.conn.t.stats()
	st.RTT = s.mux.RTT()
	return st, true
}

// traffic is bytes on one or more TCP connections.
type traffic struct {
	sent, received atomic.Int64
}

func (t *traffic) stats() Stats {
	return Stats{Sent: t.sent.Load(), Received: t.received.Load()}
}

// counted counts a TCP connection's bytes into t.
type counted struct {
	net.Conn
	t *traffic
}

func newCounted(c net.Conn) *counted { return &counted{Conn: c, t: &traffic{}} }

func (c *counted) Read(b []byte) (int, error) {
	n, err := c.Conn.Read(b)
	c.t.received.Add(int64(n))
	return n, err //nolint:wrapcheck // the connection's own error, as net.Conn returns it
}

func (c *counted) Write(b []byte) (int, error) {
	n, err := c.Conn.Write(b)
	c.t.sent.Add(int64(n))
	return n, err //nolint:wrapcheck // the connection's own error, as net.Conn returns it
}
