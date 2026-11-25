package plextv

import (
	"context"
	"net"
	"net/netip"
	"time"

	"golang.org/x/net/proxy"

	"github.com/jatassi/slipstream/fakes/fake"
	"github.com/jatassi/slipstream/fakes/fault"
	"github.com/jatassi/slipstream/fakes/internal/netsite"
	"github.com/jatassi/slipstream/fakes/internal/plexaccount"
	"github.com/jatassi/slipstream/fakes/outside"
)

// connectingBack is plex.tv in a Scenario with a network: beside the API
// it plays Plex's own check of each server with Remote Access on.
type connectingBack struct {
	fake.Server
	s *state
}

func (c connectingBack) Serve(ctx context.Context, l net.Listener) error {
	go c.s.connectBack(ctx)
	return c.Server.Serve(ctx, l)
}

// connectBack connects from the internet, through the outside dialer's
// world front, to the public address and port of each server with Remote
// Access on every connectBackEvery, and records mapped or failed in the
// account and in ConnectBacks. A
// mapping the Scenario or a cue set stays. The first check waits one
// interval, since the world starts its Nodes after its fakes.
func (s *state) connectBack(ctx context.Context) {
	at, _ := s.fronts.Find(fault.Target(outside.Fake.Name), netsite.World)
	socks, err := proxy.SOCKS5("tcp", at.AddrPort.String(), nil, &net.Dialer{Timeout: connectBackEvery})
	if err != nil {
		return
	}
	dialer, _ := socks.(proxy.ContextDialer)
	s.mu.Lock()
	for _, srv := range s.owned.Servers() {
		s.cued[srv.Identifier] = srv.Remote.Mapping.State != "unknown"
	}
	s.mu.Unlock()
	tick := time.NewTicker(connectBackEvery)
	defer tick.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-tick.C:
		}
		for _, srv := range s.owned.Servers() {
			s.check(ctx, dialer, srv)
		}
	}
}

func (s *state) check(ctx context.Context, dialer proxy.ContextDialer, srv plexaccount.Server) {
	s.mu.Lock()
	public, skip := s.public(srv), !srv.Remote.On || s.cued[srv.Identifier]
	s.mu.Unlock()
	if skip || !public.IsValid() {
		return
	}
	state := "failed"
	if c, err := dialer.DialContext(ctx, "tcp", public.String()); err == nil {
		_ = c.Close()
		state = "mapped"
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.cued[srv.Identifier] {
		return
	}
	s.owned.SetMapping(srv.Identifier, plexaccount.MappingOf(state, public))
	s.connectBacks.Record(ctx, map[string]string{"server": srv.Name, "state": state})
}

// public is srv's public address and port: its Node's site address and
// the port it listens on there. s.mu holds.
func (s *state) public(srv plexaccount.Server) netip.AddrPort {
	at, ok := s.fronts.Find(srv.Target, srv.Node)
	if s.site == nil || !ok {
		return netip.AddrPort{}
	}
	return netip.AddrPortFrom(s.site.Source(srv.Node), at.AddrPort.Port())
}
