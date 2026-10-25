package link

import (
	"context"
	"crypto/ed25519"
	"crypto/tls"
	"errors"
	"fmt"
	"net/netip"
	"sync"
	"time"

	"github.com/jatassi/slipstream/internal/backoff"
	"github.com/jatassi/slipstream/internal/joincode"
	"github.com/jatassi/slipstream/internal/link/internal/wire"
	linkv1 "github.com/jatassi/slipstream/internal/pb/slipstream/link/v1"
)

// atCeiling is a retry count whose backoff is redial_max.
const atCeiling = 64

type keeper struct {
	cancel context.CancelFunc
	// target and refused are guarded by the link's lock.
	target  Target
	refused linkv1.RefusedReason
}

// Keep holds a session to t, re-dialing after every end until Drop or
// Close. Keeping a kept key again replaces its addresses for the next dial.
func (l *Link) Keep(t Target) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.closed {
		return
	}
	if k := l.kept[t.Key]; k != nil {
		k.target = t
		return
	}
	ctx, cancel := context.WithCancel(l.ctx)
	k := &keeper{cancel: cancel, target: t}
	l.kept[t.Key] = k
	l.wg.Go(func() { l.keep(ctx, k) })
}

// Drop stops keeping key and closes its session.
func (l *Link) Drop(key Key) {
	l.mu.Lock()
	if k := l.kept[key]; k != nil {
		k.cancel()
		delete(l.kept, key)
		if k.refused != 0 && l.live[key] == nil {
			l.publish(Change{Key: key, State: Down{}})
		}
	}
	l.mu.Unlock()
	l.Reset(key)
}

// Reset closes key's live session and leaves its keeper, which dials
// again.
func (l *Link) Reset(key Key) {
	l.mu.Lock()
	s := l.live[key]
	l.mu.Unlock()
	if s != nil {
		_ = s.mux.Close()
	}
}

// answered records the peer's refusal of k's last Hello, which Down
// carries while no session is live. welcome clears it on a Welcome.
func (l *Link) answered(k *keeper, refused linkv1.RefusedReason) {
	l.mu.Lock()
	defer l.mu.Unlock()
	key := k.target.Key
	if l.kept[key] != k || k.refused == refused {
		return
	}
	k.refused = refused
	if l.live[key] == nil {
		l.publish(Change{Key: key, State: Down{Refused: refused}})
	}
}

// down is key's state with no live session. Callers hold l.mu.
func (l *Link) down(key Key) Down {
	if k := l.kept[key]; k != nil {
		return Down{Refused: k.refused}
	}
	return Down{}
}

// keep dials, says Hello, awaits the session's end and backs off. A
// Welcome resets the attempt count, and a refused Hello waits at
// redial_max, logged once per change.
func (l *Link) keep(ctx context.Context, k *keeper) {
	var n int
	var refusal string
	for {
		l.mu.Lock()
		t := k.target
		s := l.live[t.Key]
		l.mu.Unlock()
		if s == nil {
			var err error
			s, err = l.attempt(ctx, t)
			if l.turnedAway(k, err, &refusal) {
				n = atCeiling
			}
		}
		if s != nil {
			n = 0
			select {
			case <-s.mux.Done():
			case <-ctx.Done():
				return
			}
		}
		wait := time.NewTimer(backoff.Delay(l.cfg.Timings.RedialMin, l.cfg.Timings.RedialMax, n, l.cfg.Rand))
		n++
		select {
		case <-wait.C:
		case <-ctx.Done():
			wait.Stop()
			return
		}
	}
}

// turnedAway records how k's peer answered an attempt and logs a refused
// or superseded Hello once per change of refusal. It is true for those
// two answers.
func (l *Link) turnedAway(k *keeper, err error, refusal *string) bool {
	peer := k.target.Key.Fingerprint()
	var refused *RefusedError
	var superseded *SupersededError
	switch {
	case err == nil:
		*refusal = ""
	case errors.As(err, &refused) || errors.As(err, &superseded):
		if err.Error() != *refusal {
			*refusal = err.Error()
			l.cfg.Log.Warn("link refused", "peer", peer, "err", err)
		}
		var reason linkv1.RefusedReason
		if refused != nil {
			reason = refused.Reason
		}
		l.answered(k, reason)
		return true
	default:
		l.cfg.Log.Debug("link dial failed", "peer", peer, "err", err)
	}
	return false
}

// attempt dials t and says Hello, all within handshake.
func (l *Link) attempt(ctx context.Context, t Target) (*session, error) {
	ctx, cancel := context.WithTimeout(ctx, l.cfg.Timings.Handshake)
	defer cancel()
	s, err := l.race(ctx, t.Addresses, pinned(t.Key))
	if err != nil {
		return nil, err
	}
	if err := l.hello(ctx, s); err != nil {
		_ = s.mux.Close()
		return nil, err
	}
	return s, nil
}

// Paired is a Pair the Controller accepted, with the session live.
type Paired struct {
	NodeID []byte
	// Name is this Node's name as the Controller recorded it.
	Name           string
	ControllerName string
	Controller     Key
	// Node is the Controller's, from its Welcome.
	Node NodeInfo
	// Remote is the Controller's address on the session Pair made live.
	Remote netip.AddrPort
}

// Pair dials code's Controller, checking its key against the code, sends
// Pair{secret, name} and then Hello, all within join_dial. It returns once
// the session is live.
// A Controller's refusal is a *RefusedError, and a peer whose key the
// code does not name fails with ErrKeyMismatch.
func (l *Link) Pair(ctx context.Context, code joincode.Code, name string) (Paired, error) {
	addrs, err := ParseAddresses(code.Addresses())
	if err != nil {
		return Paired{}, err
	}
	ctx, cancel := context.WithTimeout(ctx, l.cfg.Timings.JoinDial)
	defer cancel()
	stop := context.AfterFunc(l.ctx, cancel)
	defer stop()
	s, err := l.race(ctx, addrs, func(k Key) error {
		if !code.Names(k[:]) {
			return fmt.Errorf("%w: it holds key %s", ErrKeyMismatch, k.Fingerprint())
		}
		return nil
	})
	if err != nil {
		return Paired{}, err
	}
	return l.pairOn(ctx, s, code, name)
}

// pairOn sends Pair{secret, name} and then Hello on s, closing s unless
// both succeed.
func (l *Link) pairOn(ctx context.Context, s *session, code joincode.Code, name string) (Paired, error) {
	paired, err := l.pair(ctx, s, code.Secret(), name)
	if err == nil {
		err = l.hello(ctx, s)
	}
	if err != nil {
		_ = s.mux.Close()
		return Paired{}, err
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	return Paired{NodeID: paired.GetNodeId(), Name: paired.GetName(), ControllerName: paired.GetControllerName(), Controller: s.key, Node: s.peer, Remote: s.remote}, nil
}

func (l *Link) pair(ctx context.Context, s *session, secret [joincode.SecretSize]byte, name string) (*linkv1.Paired, error) {
	res, err := s.linkService().Pair(ctx, &linkv1.PairRequest{Secret: secret[:], Name: name})
	if err != nil {
		return nil, fmt.Errorf("pair: %w", err)
	}
	switch o := res.GetOutcome().(type) {
	case *linkv1.PairResponse_Paired:
		return o.Paired, nil
	case *linkv1.PairResponse_Refused:
		return nil, &RefusedError{Reason: o.Refused.GetReason()}
	}
	return nil, errors.New("pair: the peer responded with neither Paired nor Refused")
}

// hello says Hello on s and makes it live on a Welcome.
func (l *Link) hello(ctx context.Context, s *session) error {
	res, err := s.linkService().Hello(ctx, &linkv1.HelloRequest{Node: l.info().proto()})
	if err != nil {
		return fmt.Errorf("hello: %w", err)
	}
	switch o := res.GetOutcome().(type) {
	case *linkv1.HelloResponse_Welcome:
		peer, err := parseNodeInfo(o.Welcome.GetNode())
		if err != nil {
			return fmt.Errorf("hello: %w", err)
		}
		if !l.welcome(s, peer) {
			return ErrDisconnected
		}
		return nil
	case *linkv1.HelloResponse_Superseded:
		by, err := KeyFromBytes(o.Superseded.GetByKey())
		if err != nil {
			return fmt.Errorf("hello: superseded: %w", err)
		}
		addrs, err := AddressesFromProto(o.Superseded.GetAddresses())
		if err != nil {
			return fmt.Errorf("hello: superseded: %w", err)
		}
		return &SupersededError{By: by, Addresses: addrs}
	case *linkv1.HelloResponse_Refused:
		return &RefusedError{Reason: o.Refused.GetReason()}
	}
	return fmt.Errorf("hello: the peer responded with %T, which this build doesn't support", res.GetOutcome())
}

// race dials every address, IPv6 at once and IPv4 after fallback_delay
// unless an IPv6 dial connected first, or at once when every IPv6 dial
// failed. The first session through TLS wins and the rest are closed.
func (l *Link) race(ctx context.Context, addrs Addresses, pin func(Key) error) (*session, error) {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	v6, v4 := addrs.families()
	r := &racer{l: l, ctx: ctx, pin: pin, results: make(chan dialed, len(v6)+len(v4)), connected6: make(chan struct{}, len(v6))}
	r.dial(v6, true)
	pending, failed6, connected := len(v6), 0, false
	startV4 := func() {
		pending += len(v4)
		r.dial(v4, false)
		v4 = nil
	}
	if len(v6) == 0 {
		startV4()
	}
	fallback := time.NewTimer(l.cfg.Timings.FallbackDelay)
	defer fallback.Stop()
	var errs []error
	for pending > 0 {
		select {
		case <-r.connected6:
			connected = true
		case <-fallback.C:
			if !connected {
				startV4()
			}
		case d := <-r.results:
			pending--
			if d.err == nil {
				return d.s, nil
			}
			errs = append(errs, d.err)
			if d.ipv6 {
				failed6++
			}
			if d.ipv6 && failed6 == len(v6) {
				startV4()
			}
		}
	}
	return nil, errors.Join(errs...)
}

// racer runs one race's dials. The first to finish TLS claims the race;
// any that finish after close their session.
type racer struct {
	l          *Link
	ctx        context.Context
	pin        func(Key) error
	results    chan dialed
	connected6 chan struct{}

	mu  sync.Mutex
	won bool
}

type dialed struct {
	s    *session
	err  error
	ipv6 bool
}

func (r *racer) dial(addrs []netip.AddrPort, ipv6 bool) {
	for _, a := range addrs {
		if !r.l.spawn(func() { r.results <- r.connect(a, ipv6) }) {
			r.results <- dialed{nil, errClosed, ipv6}
		}
	}
}

func (r *racer) connect(a netip.AddrPort, ipv6 bool) dialed {
	s, err := r.l.connect(r.ctx, a, r.pin, func() {
		if ipv6 {
			r.connected6 <- struct{}{}
		}
	})
	if err != nil {
		return dialed{nil, err, ipv6}
	}
	r.mu.Lock()
	lost := r.won
	r.won = true
	r.mu.Unlock()
	if lost {
		_ = s.mux.Close()
		return dialed{nil, errLostRace, ipv6}
	}
	return dialed{s, nil, ipv6}
}

var (
	errLostRace = errors.New("another address connected first")
	errClosed   = errors.New("the link is closed")
)

// connect dials a, runs TLS with pin as the check of the peer's key, and
// starts the session.
func (l *Link) connect(ctx context.Context, a netip.AddrPort, pin func(Key) error, connected func()) (*session, error) {
	raw, err := l.cfg.Net.DialContext(ctx, "tcp", a.String())
	if err != nil {
		return nil, fmt.Errorf("dial %s: %w", a, err)
	}
	connected()
	c := &counted{Conn: raw}
	tc := tls.Client(c, wire.ClientConfig(l.cert, func(pub ed25519.PublicKey) error { return pin(Key(pub)) }))
	if err := tc.HandshakeContext(ctx); err != nil {
		_ = c.Close()
		return nil, fmt.Errorf("handshake with %s: %w", a, err)
	}
	return l.start(c, tc, true)
}
