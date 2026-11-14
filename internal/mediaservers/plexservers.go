package mediaservers

import (
	"cmp"
	"context"
	"errors"
	"net/netip"
	"net/url"
	"slices"
	"strings"
	"sync"

	"github.com/jatassi/slipstream/internal/mediaserverlink"
	"github.com/jatassi/slipstream/internal/mediaserverlink/link"
	"github.com/jatassi/slipstream/internal/mediaservers/internal/db"
	"github.com/jatassi/slipstream/internal/nodes"
	msv1 "github.com/jatassi/slipstream/internal/pb/slipstream/mediaserver/v1"
	"github.com/jatassi/slipstream/internal/providers/plextv"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/storage"
)

// ranked is a server's connections in the order SlipStream picks from:
// local before remote, then plex.direct HTTPS, other HTTPS, plain HTTP,
// each in plex.tv's order. plextv already left the relays out.
func ranked(cs []plextv.Connection) []plextv.Connection {
	rank := func(c plextv.Connection) int {
		r := 2
		switch {
		case c.URI.Scheme == "https" && strings.HasSuffix(c.URI.Hostname(), ".plex.direct"):
			r = 0
		case c.URI.Scheme == "https":
			r = 1
		}
		if !c.Local {
			r += 3
		}
		return r
	}
	out := slices.Clone(cs)
	slices.SortStableFunc(out, func(a, b plextv.Connection) int { return cmp.Compare(rank(a), rank(b)) })
	return out
}

// answersAs is whether identifier responds at uri from node, read with no
// token from /identity.
func (m *MediaServers) answersAs(ctx context.Context, node role.NodeID, uri, identifier string) bool {
	ctx, cancel := context.WithTimeout(ctx, m.cfg.Timings.Call)
	defer cancel()
	resp, err := m.cfg.Link(node).Reach(ctx, &msv1.ReachRequest{Kind: "plex", Address: uri})
	if err != nil || mediaserverlink.ServerErrorOf(resp.GetProblem()) != nil {
		return false
	}
	id, err := mediaserverlink.IdentityOf(resp.GetIdentity())
	return err == nil && id.ID == identifier
}

// reachAll tests every connection from every Node at once: for each Node,
// whether each connection responded as identifier.
func (m *MediaServers) reachAll(ctx context.Context, identifier string, cs []plextv.Connection, from []role.NodeID) map[role.NodeID][]bool {
	out := make(map[role.NodeID][]bool, len(from))
	for _, n := range from {
		out[n] = make([]bool, len(cs))
	}
	var wg sync.WaitGroup
	for _, n := range from {
		for i, c := range cs {
			wg.Go(func() { out[n][i] = m.answersAs(ctx, n, c.URI.String(), identifier) })
		}
	}
	wg.Wait()
	return out
}

// first is the first connection that responded, "" when none did.
func first(cs []plextv.Connection, ok []bool) string {
	for i, c := range cs {
		if ok[i] {
			return c.URI.String()
		}
	}
	return ""
}

// holders are the Controller and every Node that holds Media Server link,
// the Controller first.
func (m *MediaServers) holders() []role.NodeID {
	self := m.cfg.Roster.Self()
	out := []role.NodeID{self}
	for _, n := range m.cfg.Roster.Holding(role.MediaServerLink) {
		if n != self {
			out = append(out, n)
		}
	}
	return out
}

// plexServers are the account's own servers, as plex.tv lists them now. A
// refusal marks the account refused.
func (m *MediaServers) plexServers(ctx context.Context, a account) ([]plextv.Server, error) {
	all, err := m.plexTV(a).Servers(ctx, a.jwt)
	if err != nil {
		if plextv.Refused(err) {
			m.refuse(ctx)
		}
		return nil, fromPlexTV(err)
	}
	return slices.DeleteFunc(all, func(s plextv.Server) bool { return !s.Owned }), nil
}

// hostIP is the IP address a connection's host names. A plex.direct host
// carries its IPv4 address in its first label, with dashes for dots.
func hostIP(u *url.URL) (netip.Addr, bool) {
	host := u.Hostname()
	if strings.HasSuffix(host, ".plex.direct") {
		label, _, _ := strings.Cut(host, ".")
		host = strings.ReplaceAll(label, "-", ".")
	}
	ip, err := netip.ParseAddr(host)
	return ip.Unmap(), err == nil
}

// nodeAt is the Node one of whose addresses has ip.
func nodeAt(peers map[role.NodeID]nodes.Peer, ip netip.Addr) (role.NodeID, bool) {
	for id, p := range peers {
		if p.Removed {
			continue
		}
		for _, a := range p.Addresses {
			if ap, err := netip.ParseAddrPort(a); err == nil && ap.Addr().Unmap() == ip {
				return id, true
			}
		}
	}
	return role.NodeID{}, false
}

func (s service) PlexServers(ctx context.Context) ([]PlexServerView, error) {
	a, err := s.m.signedIn(ctx)
	if err != nil {
		return nil, err
	}
	servers, err := s.m.plexServers(ctx, a)
	if err != nil {
		return nil, err
	}
	var (
		names map[role.NodeID]nodes.Named
		peers map[role.NodeID]nodes.Peer
	)
	err = s.m.cfg.Library.Read(ctx, func(ctx context.Context, r storage.Reader[storage.Library]) error {
		if names, err = s.m.names(ctx, r); err != nil {
			return err
		}
		peers, err = nodes.PeersIn(ctx, r)
		return err
	})
	if err != nil {
		return nil, err
	}
	from := s.m.holders()
	out := make([]PlexServerView, 0, len(servers))
	for _, srv := range servers {
		cs := ranked(srv.Connections)
		out = append(out, s.m.plexServerView(srv, cs, s.m.reachAll(ctx, srv.Identifier, cs, from), from, names, peers))
	}
	return out, nil
}

// plexServerView is srv with what each Node reached, and the Node a local
// connection's address belongs to as where it runs and, when that Node
// holds Media Server link, the Node that reaches it.
func (m *MediaServers) plexServerView(srv plextv.Server, cs []plextv.Connection, reached map[role.NodeID][]bool, from []role.NodeID,
	names map[role.NodeID]nodes.Named, peers map[role.NodeID]nodes.Peer,
) PlexServerView {
	ref := func(n role.NodeID) NodeRef { return NodeRef{ID: n.String(), Name: names[n].Name} }
	v := PlexServerView{MachineIdentifier: srv.Identifier, Name: srv.Name, Version: srv.Version, Connections: []PlexConnection{}, SuggestedReaching: ref(m.cfg.Roster.Self())}
	for i, c := range cs {
		pc := PlexConnection{URI: c.URI.String(), Local: c.Local, HTTPS: c.URI.Scheme == "https", ReachedFrom: []Reached{}}
		for _, n := range from {
			pc.ReachedFrom = append(pc.ReachedFrom, Reached{NodeID: n.String(), OK: reached[n][i]})
		}
		v.Connections = append(v.Connections, pc)
	}
	for _, c := range cs {
		ip, ok := hostIP(c.URI)
		if !ok || !c.Local {
			continue
		}
		if n, ok := nodeAt(peers, ip); ok {
			runsOn := ref(n)
			v.SuggestedRunsOn = &runsOn
			if m.cfg.Roster.Holds(role.MediaServerLink, n) {
				v.SuggestedReaching = runsOn
			}
			break
		}
	}
	return v
}

// plexDraft fills a Plex draft from the account: the server's token, and
// for an address the Admin didn't type, the first connection that
// responds as the server from the reaching Node, and from the Controller
// for the fallback.
func (m *MediaServers) plexDraft(ctx context.Context, d Draft, names map[role.NodeID]nodes.Named) (Draft, account, link.Credential, error) {
	a, err := m.signedIn(ctx)
	if err != nil {
		return d, a, link.Credential{}, err
	}
	servers, err := m.plexServers(ctx, a)
	if err != nil {
		return d, a, link.Credential{}, err
	}
	i := slices.IndexFunc(servers, func(s plextv.Server) bool { return s.Identifier == d.PlexServer })
	if i < 0 {
		return d, a, link.Credential{}, ErrPlexServerNotFound
	}
	srv := servers[i]
	cs := ranked(srv.Connections)
	self := m.cfg.Roster.Self()
	if d.Address == "" || d.FallbackAddress == "" {
		reached := m.reachAll(ctx, srv.Identifier, cs, slices.Compact([]role.NodeID{d.Reaching, self}))
		if d.Address == "" {
			if d.Address = first(cs, reached[d.Reaching]); d.Address == "" {
				return d, a, link.Credential{}, &UnreachedError{Label: "Plex", Node: names[d.Reaching].Name, Address: listed(cs, srv.Name)}
			}
		}
		if fb := first(cs, reached[self]); d.FallbackAddress == "" && fb != d.Address {
			d.FallbackAddress = fb
		}
	}
	return d, a, link.Credential{Token: srv.Token}, nil
}

// listed names a server's best connection in copy, or the server when
// plex.tv listed none.
func listed(cs []plextv.Connection, server string) string {
	if len(cs) == 0 {
		return server
	}
	return cs[0].URI.String()
}

// reread reads the account's servers again: a server whose token changed,
// or whose saved connection is gone from plex.tv's list and no longer
// responds as the server, gets the new token or the first connection that
// responds, and a new settings version.
func (m *MediaServers) reread(ctx context.Context) error {
	a, err := m.signedIn(ctx)
	if errors.Is(err, ErrPlexNotSignedIn) {
		return nil
	}
	if err != nil {
		return err
	}
	servers, err := m.plexServers(ctx, a)
	if err != nil {
		return err
	}
	var rows []db.MediaserversServer
	err = m.cfg.Library.Read(ctx, func(ctx context.Context, r storage.Reader[storage.Library]) error {
		rows, err = db.New(r).ListPlexServers(ctx)
		return err
	})
	if err != nil {
		return err
	}
	for _, row := range rows {
		i := slices.IndexFunc(servers, func(s plextv.Server) bool { return s.Identifier == row.Identity })
		if i < 0 {
			m.cfg.Log.Warn("the Plex account no longer lists a Plex server", "server", row.Name)
			continue
		}
		if err := m.repick(ctx, row, servers[i]); err != nil {
			return err
		}
	}
	return nil
}

func (m *MediaServers) repick(ctx context.Context, row db.MediaserversServer, srv plextv.Server) error {
	cred, err := unsealCredential(m.cfg.Sealer, row.ID, row.SecretsSealed)
	if err != nil {
		return err
	}
	cs := ranked(srv.Connections)
	self := m.cfg.Roster.Self()
	address, controller := row.Address, cmpOr(deref(row.FallbackAddress), row.Address)
	pick := func(at string, node role.NodeID) string {
		if slices.ContainsFunc(cs, func(c plextv.Connection) bool { return c.URI.String() == at }) || m.answersAs(ctx, node, at, srv.Identifier) {
			return at
		}
		return cmpOr(first(cs, m.reachAll(ctx, srv.Identifier, cs, []role.NodeID{node})[node]), at)
	}
	address, controller = pick(address, row.ReachingNodeID), pick(controller, self)
	if address == row.Address && controller == cmpOr(deref(row.FallbackAddress), row.Address) && srv.Token == cred.Token {
		return nil
	}
	cred.Token = srv.Token
	sealed, err := sealCredential(m.cfg.Sealer, row.ID, cred)
	if err != nil {
		return err
	}
	next := row
	next.Address, next.FallbackAddress, next.SecretsSealed, next.SettingsVersion = address, nil, sealed, row.SettingsVersion+1
	if controller != address {
		next.FallbackAddress = &controller
	}
	err = m.cfg.Library.Write(ctx, func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
		return db.New(tx).UpdateServer(ctx, db.UpdateServerParams{
			ID: next.ID, Name: next.Name, Address: next.Address, FallbackAddress: next.FallbackAddress, BrowserAddress: next.BrowserAddress,
			Identity: next.Identity, Version: next.Version, MonitorDelayMs: next.MonitorDelayMs, ReachingNodeID: next.ReachingNodeID,
			RunsOnNodeID: next.RunsOnNodeID, SecretsSealed: next.SecretsSealed, SettingsVersion: next.SettingsVersion, Fallback: next.Fallback,
			CheckedAt: next.CheckedAt, At: storage.Now(),
		})
	})
	if err != nil {
		return err
	}
	m.cfg.Log.Info("a Plex server's token or connection changed", "server", row.Name)
	m.cfg.Touch(row.ReachingNodeID)
	m.changed(ctx, row.ID)
	return nil
}

// refuse marks the account's sign-in refused, which every Plex server's
// state reads at once.
func (m *MediaServers) refuse(ctx context.Context) {
	var ids []link.ServerID
	err := m.cfg.Library.Write(ctx, func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
		q := db.New(tx)
		n, err := q.RefusePlex(ctx, storage.Now())
		if err != nil || n == 0 {
			return err
		}
		rows, err := q.ListPlexServers(ctx)
		for _, r := range rows {
			ids = append(ids, r.ID)
		}
		return err
	})
	if err != nil {
		m.cfg.Log.Error("record that plex.tv refused the Plex sign-in", "err", err.Error())
		return
	}
	if len(ids) > 0 {
		m.cfg.Log.Warn("plex.tv refused the Plex sign-in", "servers", len(ids))
		m.changed(ctx, ids...)
	}
}
