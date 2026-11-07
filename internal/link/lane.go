package link

import (
	"context"
	"crypto/ed25519"
	"crypto/tls"
	"fmt"
	"net"

	"github.com/jatassi/slipstream/internal/link/internal/wire"
)

// LaneHandler serves lanes: TLS connections on the link's listener,
// pinned to the peer's key as a session is, that carry one Transfer's
// chunks with no mux and no Hello.
type LaneHandler interface {
	// Admit reports, inside the handshake, whether key may open a lane
	// now. Trust is never asked. A peer on another build fails the
	// handshake before Admit runs.
	Admit(key Key) bool
	// Serve owns c until it returns. ctx ends when the link closes. The
	// link closes c once Serve returns or the link closes, and closing c
	// closes its TCP connection with no close_notify.
	Serve(ctx context.Context, key Key, c net.Conn)
}

// HandleLanes serves lanes with h. Call it before Serve and before any
// dial. With no handler the link refuses every lane in the handshake.
func (l *Link) HandleLanes(h LaneHandler) {
	if l.lanes != nil {
		panic("link: lanes already have a handler")
	}
	l.lanes = h
}

// DialLane opens a lane to key. It races addrs as a session dial does,
// IPv6 at once and IPv4 after fallback_delay, within handshake. With zero
// addrs it dials the addresses the link keeps key at, which is how a
// Worker reaches its Controller, since no intent carries the Controller's
// addresses. A peer on another build refuses the lane with
// no_application_protocol, and a peer from before lanes fails this
// side's handshake. TLS 1.3 ends this side's handshake before the peer
// checks this Node's key, so a peer whose lane handler refuses it shows
// as a TLS alert on the lane's first read. Closing the lane closes its
// TCP connection with no close_notify. Its bytes count in LaneBytes(key).
func (l *Link) DialLane(ctx context.Context, key Key, addrs Addresses) (net.Conn, error) {
	if addrs.IsZero() {
		l.mu.Lock()
		if k := l.kept[key]; k != nil {
			addrs = k.target.Addresses
		}
		l.mu.Unlock()
	}
	if addrs.IsZero() {
		return nil, fmt.Errorf("no address to dial a lane to %s", key.Fingerprint())
	}
	ctx, cancel := context.WithTimeout(ctx, l.cfg.Timings.Handshake)
	defer cancel()
	stop := context.AfterFunc(l.ctx, cancel)
	defer stop()
	pin := pinned(key)
	c, tc, err := l.race(ctx, addrs, wire.LaneConfig(l.cert, l.lane, func(pub ed25519.PublicKey, _ string) error { return pin(Key(pub)) }))
	if err != nil {
		return nil, fmt.Errorf("dial a lane to %s: %w", key.Fingerprint(), err)
	}
	return l.countLane(c, tc, key), nil
}

// LaneBytes is every byte sent and received on lanes with key since the
// link started, closed lanes included, so it never goes backwards.
func (l *Link) LaneBytes(key Key) Stats {
	l.mu.Lock()
	t := l.laneTraffic[key]
	l.mu.Unlock()
	if t == nil {
		return Stats{}
	}
	return t.stats()
}

// serveLane hands an accepted lane to the lane handler until it returns
// or the link closes.
func (l *Link) serveLane(c *counted, tc *tls.Conn) {
	pub, err := wire.PeerKey(tc.ConnectionState())
	if err != nil {
		_ = c.Close()
		return
	}
	key := Key(pub)
	lane := l.countLane(c, tc, key)
	stop := context.AfterFunc(l.ctx, func() { _ = lane.Close() })
	defer stop()
	l.lanes.Serve(l.ctx, key, lane)
	_ = lane.Close()
}

// countLane makes c, under TLS as tc, a lane with key: its bytes so far
// and from now on count in key's lane traffic. Nothing reads or writes c
// meanwhile, since its handshake is over and nobody holds the lane yet.
func (l *Link) countLane(c *counted, tc *tls.Conn, key Key) net.Conn {
	l.mu.Lock()
	t := l.laneTraffic[key]
	if t == nil {
		t = &traffic{}
		l.laneTraffic[key] = t
	}
	l.mu.Unlock()
	t.sent.Add(c.t.sent.Load())
	t.received.Add(c.t.received.Load())
	c.t = t
	return unnotified{tc, c}
}
