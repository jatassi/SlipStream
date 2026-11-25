package reachability

import (
	"bytes"
	"context"
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"net/http"
	"net/netip"
	"time"

	"github.com/jatassi/slipstream/internal/netmodel"
	netv1 "github.com/jatassi/slipstream/internal/pb/slipstream/network/v1"
	"github.com/jatassi/slipstream/internal/reachability/judge"
	"github.com/jatassi/slipstream/internal/role"
)

// EchoPath is where a tester fetches the echo of the UI and portal.
const EchoPath = "/api/v1/remote-access/echo/"

func (r *Reachability) resolve(ctx context.Context, node role.NodeID, host string) (judge.DNSAnswer, error) {
	resp, err := r.cfg.Network(node).Resolve(ctx, &netv1.ResolveRequest{Host: host})
	if err != nil {
		return judge.DNSAnswer{}, err
	}
	errA, err1 := netmodel.Enum(int32(resp.GetErrorA()), judge.NoResolver)
	errAAAA, err2 := netmodel.Enum(int32(resp.GetErrorAaaa()), judge.NoResolver)
	if err := errors.Join(err1, err2); err != nil {
		return judge.DNSAnswer{}, fmt.Errorf("read the lookup error: %w", err)
	}
	a := judge.DNSAnswer{By: node, ErrA: errA, ErrAAAA: errAAAA}
	for _, b := range resp.GetA() {
		if x, err := netmodel.ParseAddr(b); err == nil {
			a.A = append(a.A, x)
		}
	}
	for _, b := range resp.GetAaaa() {
		if x, err := netmodel.ParseAddr(b); err == nil {
			a.AAAA = append(a.AAAA, x)
		}
	}
	return a, nil
}

// test has the first eligible tester of each offered family connect to the
// public address as a player would, then connect back to the origin on a
// Direct route and to the origin from a paired proxy on an own-proxy one.
// A tester that can't be asked becomes a tester-away fact and the next is
// tried.
func (g *gathered) test(ctx context.Context, r *Reachability) {
	pl := judge.Place(g.f)
	for _, fam := range pl.Offered {
		testers, _ := judge.Testers(g.f, pl, fam)
		for _, t := range testers {
			d, err := g.player(ctx, r, t, fam)
			if err != nil {
				g.f.Skipped = append(g.f.Skipped, judge.Fact{ID: judge.FactTesterAway, Family: &fam, Params: judge.Params{Tester: t}})
				continue
			}
			g.f.Dials = append(g.f.Dials, d)
			g.legs(ctx, r, pl, t, fam)
			break
		}
	}
}

func (g *gathered) player(ctx context.Context, r *Reachability, t role.NodeID, fam netmodel.Family) (judge.Dial, error) {
	req := &netv1.DialRequest{
		Target: &netv1.DialRequest_Host{Host: g.host}, Port: uint32(g.f.Exposure.PublicPort()),
		Family: netmodel.FamilyOnWire(fam), Scheme: schemeOnWire(g.scheme), Path: g.identityPath(),
	}
	var echoID, answer string
	if g.f.Exposure.Kind == judge.UIPortal {
		echoID, answer = r.echoes.issue(r.now(), r.cfg.Timings.Echo)
		req.Path = EchoPath + echoID
	}
	resp, err := r.cfg.Network(t).Dial(ctx, req)
	if echoID != "" {
		arrived := r.echoes.take(echoID)
		if err != nil {
			return judge.Dial{}, err
		}
		d, err := dialOf(judge.PlayerDial, t, fam, g.f.Exposure.PublicPort(), g.host, resp, func(body []byte) judge.IdentitySeen {
			if string(bytes.TrimSpace(body)) == answer {
				return judge.Matches
			}
			return judge.OtherIdentity
		})
		d.Echo = arrived
		return d, err
	}
	if err != nil {
		return judge.Dial{}, err
	}
	return dialOf(judge.PlayerDial, t, fam, g.f.Exposure.PublicPort(), g.host, resp, g.identity)
}

// legs are the connection back to the origin's public address from the
// tester on a Direct route, and from a paired proxy to the origin on an
// own-proxy route. Each target is derived here and vetted before it is
// sent.
func (g *gathered) legs(ctx context.Context, r *Reachability, pl judge.Placement, t role.NodeID, fam netmodel.Family) {
	fr := pl.Routes[fam]
	origin := g.f.Nodes[pl.Origin]
	public4 := r.cfg.Rules.Public4(origin.Observed())
	if fam != netmodel.IPv4 || !public4.IsValid() {
		return
	}
	switch {
	case fr.Kind == judge.Direct:
		target := netip.AddrPortFrom(public4, g.f.Exposure.PublicPort())
		if judge.Vet(g.f, pl, t, target, judge.FromPublic4) {
			g.connect(ctx, r, judge.ConnectBackDial, t, fam, target)
		}
	case fr.Kind == judge.OwnProxy && !fr.ProxyNode.IsZero() && r.cfg.Roster.InStep(fr.ProxyNode):
		target := netip.AddrPortFrom(public4, g.f.Exposure.Local)
		if judge.Vet(g.f, pl, fr.ProxyNode, target, judge.FromPublic4) {
			g.connect(ctx, r, judge.ProxyLegDial, fr.ProxyNode, fam, target)
		}
	}
}

func (g *gathered) connect(ctx context.Context, r *Reachability, kind judge.DialKind, t role.NodeID, fam netmodel.Family, target netip.AddrPort) {
	addr := target.Addr().As4()
	req := &netv1.DialRequest{
		Target: &netv1.DialRequest_Addr{Addr: addr[:]}, Port: uint32(target.Port()), Family: netmodel.FamilyOnWire(fam),
		Scheme: schemeOnWire(g.scheme), Sni: g.host, Path: g.identityPath(),
	}
	resp, err := r.cfg.Network(t).Dial(ctx, req)
	if err != nil {
		return
	}
	if d, err := dialOf(kind, t, fam, target.Port(), g.host, resp, g.identity); err == nil {
		g.f.Dials = append(g.f.Dials, d)
	}
}

func schemeOnWire(s string) netv1.Scheme {
	if s == "https" {
		return netv1.Scheme_SCHEME_HTTPS
	}
	return netv1.Scheme_SCHEME_HTTP
}

// identityPath is where a Media Server says who it is without a sign-in.
func (g *gathered) identityPath() string {
	if g.kind == string(judge.Plex) {
		return "/identity"
	}
	return "/System/Info/Public"
}

// dialOf is the attempt that got furthest. The body is weighed here on the
// Controller: a tester never interprets identity. An answer no tester
// could give is refused.
func dialOf(kind judge.DialKind, t role.NodeID, fam netmodel.Family, port uint16, host string, resp *netv1.DialResponse, identity func([]byte) judge.IdentitySeen) (judge.Dial, error) {
	d := judge.Dial{Kind: kind, Tester: t, Family: fam}
	var best *netv1.Attempt
	for _, a := range resp.GetAttempts() {
		if best == nil || a.GetStage() > best.GetStage() {
			best = a
		}
	}
	if best == nil {
		return d, nil
	}
	if a, err := netmodel.ParseAddr(best.GetAddr()); err == nil {
		d.Addr = netip.AddrPortFrom(a, port)
	}
	var err error
	if d.Stage, err = netmodel.Enum(int32(best.GetStage()), judge.HTTPDone); err != nil {
		return d, fmt.Errorf("read the stage: %w", err)
	}
	if c := best.GetTls(); c != nil {
		problem, err := netmodel.Enum(int32(c.GetProblem()), judge.Handshake)
		if err != nil {
			return d, fmt.Errorf("read the TLS problem: %w", err)
		}
		d.TLS = &judge.TLSFacts{Problem: problem, Names: c.GetNames(), NotAfter: time.UnixMilli(c.GetNotAfterMs()).UTC()}
	}
	if d.Stage < judge.HTTPDone {
		return d, nil
	}
	status := int(best.GetStatus())
	switch {
	case status == http.StatusUnauthorized, status == http.StatusForbidden,
		status >= 300 && status < 400 && best.GetLocationHost() != "" && best.GetLocationHost() != host:
		d.Identity = judge.SignIn
	case status >= 200 && status < 300:
		d.Identity = identity(best.GetBody())
	default:
		d.Identity = judge.OtherIdentity
	}
	return d, nil
}

// identity is whether body names the Media Server SlipStream added: Plex's
// machineIdentifier, or Jellyfin's and Emby's Id.
func (g *gathered) identity(body []byte) judge.IdentitySeen {
	if g.ident != "" && g.ident == identityIn(body) {
		return judge.Matches
	}
	return judge.OtherIdentity
}

func identityIn(body []byte) string {
	var j struct {
		MediaContainer struct {
			MachineIdentifier string `json:"machineIdentifier"`
		} `json:"MediaContainer"`
		ID string `json:"Id"`
	}
	if json.Unmarshal(body, &j) == nil {
		return j.MediaContainer.MachineIdentifier + j.ID
	}
	var x struct {
		MachineIdentifier string `xml:"machineIdentifier,attr"`
	}
	if xml.Unmarshal(body, &x) == nil {
		return x.MachineIdentifier
	}
	return ""
}
