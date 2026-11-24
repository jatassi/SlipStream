package judgetest

import (
	"net/netip"

	"github.com/jatassi/slipstream/internal/netmodel"
	"github.com/jatassi/slipstream/internal/reachability/judge"
	"github.com/jatassi/slipstream/internal/role"
)

// Outcome is how a tester's connection to the public address ended.
type Outcome uint8

const (
	Passed     Outcome = iota
	Closed             // refused at the address
	Expired            // the certificate expired
	WrongName          // the certificate names another host
	SignInPage         // a sign-in page answered
	Something          // something else answered
	NoResponse         // connected, then no HTTP response
)

// Tested has the first tester judge.Testers picks connect to the public
// address over fam, ending as o. With no tester it changes nothing.
func (b *B) Tested(fam netmodel.Family, o Outcome) *B {
	ts, _ := judge.Testers(b.f, judge.Place(b.f), fam)
	if len(ts) == 0 {
		return b
	}
	return b.TestedBy(ts[0], fam, o)
}

// TestedBy is Tested from tester t, eligible or not.
func (b *B) TestedBy(t role.NodeID, fam netmodel.Family, o Outcome) *B {
	pl := judge.Place(b.f)
	fr := pl.Routes[fam]
	d := judge.Dial{Kind: judge.PlayerDial, Tester: t, Family: fam, Stage: judge.HTTPDone, Identity: judge.Matches}
	if len(fr.Dialed) > 0 {
		d.Addr = netip.AddrPortFrom(fr.Dialed[0], b.f.Exposure.PublicPort())
	}
	if b.f.Exposure.Public.Scheme == "https" {
		d.TLS = &judge.TLSFacts{Names: []string{b.f.Exposure.Public.Hostname()}, NotAfter: Expires.AddDate(1, 0, 0)}
	}
	if d.TLS == nil && (o == Expired || o == WrongName) {
		o = Passed
	}
	switch o {
	case Passed:
	case Closed:
		d.Stage, d.TLS, d.Identity = judge.Resolved, nil, judge.Unread
	case Expired:
		d.TLS.Problem, d.TLS.NotAfter = judge.Expired, Expires
		d.Stage, d.Identity = judge.TLSDone, judge.Unread
	case WrongName:
		d.TLS.Problem, d.TLS.Names = judge.WrongName, []string{"other.example.org"}
		d.Stage, d.Identity = judge.TLSDone, judge.Unread
	case SignInPage:
		d.Identity = judge.SignIn
	case Something:
		d.Identity = judge.OtherIdentity
	case NoResponse:
		d.Stage, d.Identity = judge.TLSDone, judge.Unread
		if d.TLS == nil {
			d.Stage = judge.Connected
		}
	}
	if b.f.Exposure.Kind == judge.UIPortal && d.Identity == judge.Matches {
		d.Echo = b.arrival(pl, t, fam, Outside)
	}
	b.f.Dials = append(b.f.Dials, d)
	return b
}

// Arrival is how the echo's fetch came in.
type Arrival uint8

const (
	// Outside is from the tester's public address to a bound listener.
	Outside Arrival = iota
	// ThroughCaddy is over loopback from another program on the origin.
	ThroughCaddy
	// OwnStream is through the tester's own Serve stream.
	OwnStream
	// TesterLoopback is from a loopback address on the tester itself.
	TesterLoopback
	// OtherSource is from an address that isn't the tester's.
	OtherSource
	// Dropped is an echo id the Controller forgot.
	Dropped
)

// Echo sets how the IPv4 test's echo arrived.
func (b *B) Echo(a Arrival) *B {
	pl := judge.Place(b.f)
	for i, d := range b.f.Dials {
		if d.Kind == judge.PlayerDial && d.Family == netmodel.IPv4 {
			b.f.Dials[i].Echo = b.arrival(pl, d.Tester, d.Family, a)
		}
	}
	return b
}

func (b *B) arrival(pl judge.Placement, t role.NodeID, fam netmodel.Family, a Arrival) *judge.Arrival {
	nf := b.f.Nodes[t]
	src := b.f.Rules.Public4(nf.Observed())
	if fam == netmodel.IPv6 {
		src = SeedboxV6
	}
	e := &judge.Arrival{At: pl.Origin, Source: src}
	switch a {
	case Outside:
	case ThroughCaddy:
		e.Source = ip("127.0.0.1")
	case OwnStream:
		e.Via = t
	case TesterLoopback:
		e.At, e.Source = t, ip("127.0.0.1")
	case OtherSource:
		e.Source = ip("198.51.100.20")
	case Dropped:
		return nil
	}
	return e
}

// ConnectBack has the IPv4 tester connect to atlas's Public4 on the public
// port.
func (b *B) ConnectBack(ok bool) *B {
	for _, d := range b.f.Dials {
		if d.Kind != judge.PlayerDial || d.Family != netmodel.IPv4 {
			continue
		}
		cb := judge.Dial{Kind: judge.ConnectBackDial, Tester: d.Tester, Family: netmodel.IPv4,
			Addr: netip.AddrPortFrom(HomeWAN, b.f.Exposure.PublicPort()), Stage: judge.Connected}
		if !ok {
			cb.Stage = judge.Resolved
		}
		b.f.Dials = append(b.f.Dials, cb)
		return b
	}
	return b
}

// ProxyLeg has the-vps connect from the proxy to the origin's port.
func (b *B) ProxyLeg(ok bool) *B {
	d := judge.Dial{Kind: judge.ProxyLegDial, Tester: VPS, Family: netmodel.IPv4,
		Addr: netip.AddrPortFrom(HomeWAN, b.f.Exposure.Local), Stage: judge.Connected}
	if !ok {
		d.Stage = judge.Resolved
	}
	b.f.Dials = append(b.f.Dials, d)
	return b
}
