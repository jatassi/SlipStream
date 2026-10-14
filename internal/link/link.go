// Package link is the Node link: one pinned TLS 1.3 session per peer key,
// multiplexed into streams that carry ConnectRPC both ways. It owns the
// Node key, dialing and re-dialing, the frozen Pair and Hello, and the
// lockstep gate that closes every other service between versions.
package link

import (
	"context"
	"crypto/ed25519"
	"crypto/tls"
	"fmt"
	"log/slog"
	"maps"
	"math/rand/v2"
	"net"
	"net/http"
	"net/netip"
	"slices"
	"strings"
	"sync"

	"connectrpc.com/connect/v2"
	"connectrpc.com/connect/v2/connecthttp"

	"github.com/jatassi/slipstream/internal/link/internal/mux"
	"github.com/jatassi/slipstream/internal/link/internal/wire"
	linkv1 "github.com/jatassi/slipstream/internal/pb/slipstream/link/v1"
	"github.com/jatassi/slipstream/internal/pb/slipstream/link/v1/linkv1connect"
)

// Network is where the link dials. Production passes the OS network and
// the simulation tier a fault.Host.
type Network interface {
	Listen(network, address string) (net.Listener, error)
	DialContext(ctx context.Context, network, address string) (net.Conn, error)
}

// PairFunc answers a peer's Pair. The link calls it for a peer admitted as
// Pairing or Pinned and answers every other peer NOT_A_CONTROLLER itself.
type PairFunc func(ctx context.Context, peer Key, req *linkv1.PairRequest) (*linkv1.PairResponse, error)

type Config struct {
	Key PrivateKey
	// Node is this Node's build. Hello sends Advertised as its addresses.
	Node     NodeInfo
	Timings  Timings
	Settings Settings
	Net      Network
	// Rand is the Node's seeded link stream, safe for concurrent draws.
	Rand  *rand.Rand
	Trust Trust
	// Lockstep serves every path outside slipstream.link.v1, on sessions
	// whose peer runs this version.
	Lockstep http.Handler
	Pair     PairFunc
	Log      *slog.Logger
}

// Link holds this Node's sessions, at most one live per peer key.
type Link struct {
	cfg       Config
	cert      tls.Certificate
	mux       mux.Config
	advertise Addresses
	// kinds serves an accepted stream by the kind it opens with. A kind
	// with no entry is reset.
	kinds map[wire.Kind]func(*session, Stream)
	rpc   *wire.Conns
	http  *http.Server

	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup

	mu       sync.Mutex
	closed   bool
	bound    netip.AddrPort
	sessions map[*session]struct{}
	live     map[Key]*session
	kept     map[Key]*keeper
	subs     map[*subscriber]struct{}
}

func New(cfg Config) (*Link, error) {
	mc := mux.Config{Keepalive: cfg.Timings.Keepalive, WriteTimeout: cfg.Timings.WriteTimeout, StreamWindow: cfg.Settings.StreamWindow}
	if err := mc.Check(); err != nil {
		return nil, fmt.Errorf("link: %w", err)
	}
	cert, err := wire.Certificate(cfg.Key.key)
	if err != nil {
		return nil, err
	}
	var advertise Addresses
	if len(cfg.Settings.Advertise) > 0 {
		if advertise, err = ParseAddresses(cfg.Settings.Advertise); err != nil {
			return nil, fmt.Errorf("link.advertise: %w", err)
		}
	}
	ctx, cancel := context.WithCancel(context.Background())
	l := &Link{
		cfg:       cfg,
		cert:      cert,
		mux:       mc,
		advertise: advertise,
		rpc:       wire.NewConns(),
		ctx:       ctx,
		cancel:    cancel,
		sessions:  map[*session]struct{}{},
		live:      map[Key]*session{},
		kept:      map[Key]*keeper{},
		subs:      map[*subscriber]struct{}{},
	}
	l.kinds = map[wire.Kind]func(*session, Stream){
		wire.RPC: func(s *session, c Stream) { l.rpc.Push(rpcConn{c, s}) },
	}
	l.http = wire.Server(cfg.Settings.StreamWindow, l.routes(), withSession, cfg.Log)
	l.wg.Go(func() { _ = l.http.Serve(l.rpc) })
	return l, nil
}

// Close ends every session and dial and waits for the link's goroutines.
func (l *Link) Close() error {
	l.mu.Lock()
	if l.closed {
		l.mu.Unlock()
		return nil
	}
	l.closed = true
	sessions := slices.Collect(maps.Keys(l.sessions))
	l.mu.Unlock()
	l.cancel()
	for _, s := range sessions {
		_ = s.mux.Close()
	}
	_ = l.rpc.Close()
	_ = l.http.Close()
	l.wg.Wait()
	return nil
}

// spawn runs fn on a goroutine Close waits for, unless the link is closed.
func (l *Link) spawn(fn func()) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.closed {
		return false
	}
	l.wg.Go(fn)
	return true
}

// Serve accepts sessions on ln until ctx ends or Close.
func (l *Link) Serve(ctx context.Context, ln net.Listener) error {
	l.mu.Lock()
	l.bound = addrPort(ln.Addr())
	l.mu.Unlock()
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	stopLink := context.AfterFunc(l.ctx, cancel)
	defer stopLink()
	stopListener := context.AfterFunc(ctx, func() { _ = ln.Close() })
	defer stopListener()
	for {
		c, err := ln.Accept()
		if err != nil {
			if ctx.Err() != nil {
				return nil
			}
			return fmt.Errorf("link: accept: %w", err)
		}
		if !l.spawn(func() { l.accept(c) }) {
			_ = c.Close()
			return nil
		}
	}
}

// accept runs TLS and the mux on c, then gives the peer handshake to say
// Hello before closing the session.
func (l *Link) accept(c net.Conn) {
	ctx, cancel := context.WithTimeout(l.ctx, l.cfg.Timings.Handshake)
	defer cancel()
	tc := tls.Server(c, wire.ServerConfig(l.cert, l.admit))
	if err := tc.HandshakeContext(ctx); err != nil {
		l.cfg.Log.Debug("link handshake failed", "remote", c.RemoteAddr().String(), "err", err)
		_ = c.Close()
		return
	}
	s, err := l.start(tc, mux.Server)
	if err != nil {
		return
	}
	select {
	case <-s.welcomed:
	case <-s.mux.Done():
	case <-ctx.Done():
		_ = s.mux.Close()
	}
}

// admit is the accepting side's pin check: any key Trust does not refuse.
func (l *Link) admit(pub ed25519.PublicKey) error {
	k := Key(pub)
	if _, refused := l.cfg.Trust.Admit(k).(Refused); refused {
		return fmt.Errorf("peer key %s is refused", k.Fingerprint())
	}
	return nil
}

// State is key's session as this Node sees it.
func (l *Link) State(key Key) State {
	l.mu.Lock()
	defer l.mu.Unlock()
	if s := l.live[key]; s != nil {
		return s.state()
	}
	return Down{}
}

// Subscribe yields every Change in order until ctx ends or Close. The link
// never waits on a slow reader: changes queue for it.
func (l *Link) Subscribe(ctx context.Context) <-chan Change {
	out := make(chan Change)
	sub := &subscriber{wake: make(chan struct{}, 1)}
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.closed {
		close(out)
		return out
	}
	l.subs[sub] = struct{}{}
	l.wg.Go(func() {
		defer close(out)
		defer func() {
			l.mu.Lock()
			delete(l.subs, sub)
			l.mu.Unlock()
		}()
		for {
			l.mu.Lock()
			batch := sub.queue
			sub.queue = nil
			l.mu.Unlock()
			for _, c := range batch {
				select {
				case out <- c:
				case <-ctx.Done():
					return
				case <-l.ctx.Done():
					return
				}
			}
			select {
			case <-sub.wake:
			case <-ctx.Done():
				return
			case <-l.ctx.Done():
				return
			}
		}
	})
	return out
}

type subscriber struct {
	queue []Change
	wake  chan struct{}
}

// publish queues c for every subscriber. Callers hold l.mu, so subscribers
// see changes in the order the peer table took them.
func (l *Link) publish(c Change) {
	for sub := range l.subs {
		sub.queue = append(sub.queue, c)
		select {
		case sub.wake <- struct{}{}:
		default:
		}
	}
}

// Advertised is the address record this Node gives out: link.advertise
// when it is set, else the bound link address, or for an unspecified one
// every up, non-loopback, non-link-local unicast address it covers. It is
// false before Serve or with no such address.
func (l *Link) Advertised() (Addresses, bool) {
	l.mu.Lock()
	bound := l.bound
	l.mu.Unlock()
	if !bound.IsValid() {
		return Addresses{}, false
	}
	if !l.advertise.IsZero() {
		return l.advertise, true
	}
	var host []netip.Addr
	if bound.Addr().IsUnspecified() {
		host = hostAddrs()
	}
	return advertise(bound, host)
}

func (l *Link) info() NodeInfo {
	n := l.cfg.Node
	n.Addresses, _ = l.Advertised()
	return n
}

// Client calls key's services. A call with no live session fails with
// ErrDisconnected, and a lockstep call to a peer on another version with
// ErrMismatched, both before a byte leaves.
func (l *Link) Client(key Key) *connect.Client {
	return connect.NewClient(peerTransport{l, key})
}

// UpdateClient calls key's UpdateService, which stays open between
// versions so a Worker can update.
func (l *Link) UpdateClient(key Key) linkv1connect.UpdateServiceClient {
	return linkv1connect.NewUpdateServiceClient(l.Client(key))
}

// peerTransport sends each call on the peer's live session at call time.
type peerTransport struct {
	l   *Link
	key Key
}

func (t peerTransport) NewClientStream(ctx context.Context, spec connect.Spec) (connect.ClientStream, error) {
	t.l.mu.Lock()
	s := t.l.live[t.key]
	t.l.mu.Unlock()
	switch {
	case s == nil:
		return nil, rpcError(connect.CodeUnavailable, ErrDisconnected)
	case !s.matched && !frozen(spec.Procedure):
		return nil, rpcError(connect.CodeFailedPrecondition, ErrMismatched)
	}
	return s.rpc.NewClientStream(ctx, spec) //nolint:wrapcheck // the session's transport reports the call's own error
}

var frozenPrefix = "/" + string(linkv1.File_slipstream_link_v1_link_proto.Package()) + "."

func frozen(path string) bool { return strings.HasPrefix(path, frozenPrefix) }

var errorWriter = connecthttp.NewErrorWriter()

// rpcError is err as a Connect error with code, matching err under
// errors.Is on this side.
func rpcError(code connect.Code, err error) *connect.Error {
	return connect.NewError(code, err.Error()).WithCause(err)
}

func addrPort(a net.Addr) netip.AddrPort {
	ta, ok := a.(*net.TCPAddr)
	if !ok {
		return netip.AddrPort{}
	}
	ap := ta.AddrPort()
	return netip.AddrPortFrom(ap.Addr().Unmap(), ap.Port())
}
