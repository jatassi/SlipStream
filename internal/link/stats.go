package link

import (
	"net"
	"sync/atomic"
	"time"
)

// Stats is a live session's traffic.
type Stats struct {
	// Sent and Received are bytes on the session's TCP connection, TLS
	// records included.
	Sent, Received int64
	// RTT is the smoothed round trip of the session's pings, sent at the
	// keepalive cadence, and zero until the first is answered.
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
	return Stats{Sent: s.conn.sent.Load(), Received: s.conn.received.Load(), RTT: s.mux.RTT()}, true
}

type counted struct {
	net.Conn
	sent, received atomic.Int64
}

func (c *counted) Read(b []byte) (int, error) {
	n, err := c.Conn.Read(b)
	c.received.Add(int64(n))
	return n, err //nolint:wrapcheck // the connection's own error, as net.Conn returns it
}

func (c *counted) Write(b []byte) (int, error) {
	n, err := c.Conn.Write(b)
	c.sent.Add(int64(n))
	return n, err //nolint:wrapcheck // the connection's own error, as net.Conn returns it
}
