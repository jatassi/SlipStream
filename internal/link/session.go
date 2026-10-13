package link

import (
	"context"
	"crypto/tls"
	"net"
	"net/http"
	"net/netip"

	"connectrpc.com/connect/v2"
	"connectrpc.com/connect/v2/connecthttp"

	"github.com/jatassi/slipstream/internal/link/internal/mux"
	"github.com/jatassi/slipstream/internal/link/internal/wire"
	linkv1 "github.com/jatassi/slipstream/internal/pb/slipstream/link/v1"
	"github.com/jatassi/slipstream/internal/pb/slipstream/link/v1/linkv1connect"
)

// session is one authenticated connection to a peer. It is live once it
// is the peer table's entry for its key.
type session struct {
	key           Key
	mux           *mux.Session
	local, remote netip.AddrPort
	http          *http.Transport
	rpc           connect.Transport
	// welcomed is closed once the session is live. peer and matched are
	// set before, under the link's lock, and never change after.
	welcomed chan struct{}
	peer     NodeInfo
	matched  bool
}

func (s *session) state() Live {
	return Live{Peer: s.peer, Matched: s.matched, Local: s.local, Remote: s.remote}
}

func (s *session) linkService() linkv1connect.LinkServiceClient {
	return linkv1connect.NewLinkServiceClient(connect.NewClient(s.rpc))
}

// start runs a mux session over an authenticated tc and serves its
// streams until it ends.
func (l *Link) start(tc *tls.Conn, side func(net.Conn, mux.Config) (*mux.Session, error)) (*session, error) {
	pub, err := wire.PeerKey(tc.ConnectionState())
	if err != nil {
		_ = tc.Close()
		return nil, err
	}
	ms, err := side(tc, l.mux)
	if err != nil {
		_ = tc.Close()
		return nil, err
	}
	s := &session{
		key:      Key(pub),
		mux:      ms,
		local:    addrPort(tc.LocalAddr()),
		remote:   addrPort(tc.RemoteAddr()),
		welcomed: make(chan struct{}),
	}
	s.http = wire.Transport(l.cfg.Settings.StreamWindow, func(ctx context.Context) (net.Conn, error) {
		return wire.Open(ctx, ms, wire.RPC)
	})
	s.rpc = connecthttp.NewTransport(&http.Client{Transport: s.http}, "http://"+s.key.String())
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.closed {
		_ = ms.Close()
		return nil, net.ErrClosed
	}
	l.sessions[s] = struct{}{}
	l.wg.Go(func() { l.run(s) })
	return s, nil
}

// run serves s's streams by kind, then retires s once it ends.
func (l *Link) run(s *session) {
	for {
		st, err := s.mux.Accept()
		if err != nil {
			break
		}
		l.wg.Go(func() {
			k, err := wire.ReadKind(st)
			if serve := l.kinds[k]; err == nil && serve != nil {
				serve(s, st)
				return
			}
			_ = st.Reset()
		})
	}
	_ = s.mux.Close()
	s.http.CloseIdleConnections()
	l.mu.Lock()
	defer l.mu.Unlock()
	delete(l.sessions, s)
	if l.live[s.key] == s {
		delete(l.live, s.key)
		l.publish(Change{Key: s.key, State: Down{}})
	}
}

// welcome makes s its key's live session, closing the one it replaces. It
// is false when s has already ended.
func (l *Link) welcome(s *session, peer NodeInfo) bool {
	l.mu.Lock()
	select {
	case <-s.mux.Done():
		l.mu.Unlock()
		return false
	default:
	}
	old := l.live[s.key]
	if old == s {
		l.mu.Unlock()
		return true
	}
	s.peer, s.matched = peer, peer.Version == l.cfg.Node.Version
	l.live[s.key] = s
	close(s.welcomed)
	l.publish(Change{Key: s.key, State: s.state()})
	l.mu.Unlock()
	if old != nil {
		_ = old.mux.Close()
	}
	return true
}

// rpcConn is an RPC-kind stream, carrying its session to the handlers.
type rpcConn struct {
	net.Conn
	s *session
}

type sessionKey struct{}

func withSession(ctx context.Context, c net.Conn) context.Context {
	if rc, ok := c.(rpcConn); ok {
		return context.WithValue(ctx, sessionKey{}, rc.s)
	}
	return ctx
}

func sessionOf(ctx context.Context) *session {
	s, _ := ctx.Value(sessionKey{}).(*session)
	return s
}

// PeerOf is the key of the peer whose call ctx serves.
func PeerOf(ctx context.Context) (Key, bool) {
	if s := sessionOf(ctx); s != nil {
		return s.key, true
	}
	return Key{}, false
}

// routes serves Pair and Hello itself, refuses the rest of
// slipstream.link.v1 until its owners mount it, and gates the lockstep
// handler.
func (l *Link) routes() http.Handler {
	srv := connect.NewServer()
	linkv1connect.RegisterLinkServiceHandler(srv, linkService{l})
	m := http.NewServeMux()
	connecthttp.Mount(m, srv)
	m.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if frozen(r.URL.Path) {
			http.NotFound(w, r)
			return
		}
		if err := l.inStep(r.Context(), sessionOf(r.Context())); err != nil {
			_ = errorWriter.Write(w, r, err)
			return
		}
		l.cfg.Lockstep.ServeHTTP(w, r)
	})
	return m
}

// inStep waits for s's Hello, then reports whether s is live with a peer
// on this version. A peer may call before its own side has the Welcome.
func (l *Link) inStep(ctx context.Context, s *session) error {
	select {
	case <-s.welcomed:
	case <-s.mux.Done():
	case <-ctx.Done():
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	switch {
	case l.live[s.key] != s:
		return rpcError(connect.CodeUnavailable, ErrDisconnected)
	case !s.matched:
		return rpcError(connect.CodeFailedPrecondition, ErrMismatched)
	}
	return nil
}

type linkService struct{ l *Link }

func (h linkService) Pair(ctx context.Context, req *linkv1.PairRequest) (*linkv1.PairResponse, error) {
	s := sessionOf(ctx)
	switch h.l.cfg.Trust.Admit(s.key).(type) {
	case Pairing, Pinned:
		return h.l.cfg.Pair(ctx, s.key, req)
	}
	return &linkv1.PairResponse{Outcome: &linkv1.PairResponse_Refused{
		Refused: &linkv1.Refused{Reason: linkv1.RefusedReason_REFUSED_REASON_NOT_A_CONTROLLER},
	}}, nil
}

func (h linkService) Hello(ctx context.Context, req *linkv1.HelloRequest) (*linkv1.HelloResponse, error) {
	s := sessionOf(ctx)
	switch a := h.l.cfg.Trust.Admit(s.key).(type) {
	case Pinned:
		peer, err := parseNodeInfo(req.GetNode())
		if err != nil {
			return nil, rpcError(connect.CodeInvalidArgument, err)
		}
		if !h.l.welcome(s, peer) {
			return nil, rpcError(connect.CodeUnavailable, ErrDisconnected)
		}
		return &linkv1.HelloResponse{Outcome: &linkv1.HelloResponse_Welcome{
			Welcome: &linkv1.Welcome{Node: h.l.info().proto()},
		}}, nil
	case Superseded:
		return &linkv1.HelloResponse{Outcome: &linkv1.HelloResponse_Superseded{
			Superseded: &linkv1.Superseded{ByKey: a.By[:], Addresses: a.Addresses.Proto()},
		}}, nil
	}
	return &linkv1.HelloResponse{Outcome: &linkv1.HelloResponse_Refused{
		Refused: &linkv1.Refused{Reason: linkv1.RefusedReason_REFUSED_REASON_UNKNOWN_KEY},
	}}, nil
}
