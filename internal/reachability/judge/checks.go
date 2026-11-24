package judge

import (
	"net/netip"
	"slices"
	"strings"

	"github.com/jatassi/slipstream/internal/netmodel"
)

// row is one check of the table. Each verdict rule is a row or a case in a
// row's Eval. Rows emit FixIDs and FactIDs with Params, never sentences.
type row struct {
	ID  CheckID
	Hop HopKind
	// Only is the one family the row runs for; nil runs it for every
	// offered family. An Exposure row has no family at all.
	Only       *netmodel.Family
	Applies    func(in *familyIn) bool
	LocalGuess bool // a passing outside test overrules it into a fact
	Eval       func(in *familyIn) outcome
}

type outcome struct {
	Result   Result
	Fact     bool
	Points   Verdict
	Observed Value
	Fix      *Fix
	Facts    []Fact // added to the check's hop
	byPlex   bool
}

func passes() outcome                      { return outcome{Result: Passes} }
func notReached() outcome                  { return outcome{Result: NotReached} }
func notUsed() outcome                     { return outcome{Result: NotUsed} }
func fails(v Verdict, fx *Fix) outcome     { return outcome{Result: Fails, Points: v, Fix: fx} }
func notTested(v Verdict, fx *Fix) outcome { return outcome{Result: NotTested, Points: v, Fix: fx} }
func fix(id FixID, p Params) *Fix          { return &Fix{ID: id, Params: p} }
func asFact(v Value, facts ...Fact) outcome {
	return outcome{Result: Passes, Fact: true, Observed: v, Facts: facts}
}
func (o outcome) with(facts ...Fact) outcome { o.Facts = append(o.Facts, facts...); return o }
func (o outcome) seen(v Value) outcome       { o.Observed = v; return o }
func (f *Fix) action(a Action) *Fix          { f.Action = &a; return f }

func ptr[T any](v T) *T { return &v }

func always(*familyIn) bool { return true }

func addrs(as []netip.Addr) Value {
	s := make([]string, len(as))
	for i, a := range as {
		s[i] = a.String()
	}
	return Value{Text: strings.Join(s, ", ")}
}

var rowDNSA = row{ID: CheckDNSA, Hop: DNSHop, Only: ptr(netmodel.IPv4), Applies: always,
	Eval: func(in *familyIn) outcome {
		d, r := in.dns, in.f.Rules
		seen := addrs(d.a)
		nonGlobal := slices.IndexFunc(d.a, func(a netip.Addr) bool { return !r.Global(a) })
		switch {
		case in.route.Kind == Unplaced:
			// Only home resolvers answered, all non-Global. A neutral fact;
			// outside.connect carries the Not tested step.
			return asFact(seen, Fact{ID: FactSplitHorizon, Params: Params{Host: in.host, Addr: d.a[0]}})
		case len(d.a) == 0 && len(d.aaaa) > 0:
			// IPv6 alone never replaces the relay (ADR 0022).
			return fails(NeedsRelay, fix(FixNoA, Params{Host: in.host, Program: in.subject()}))
		case in.exposure.Kind == UIPortal && !slices.ContainsFunc(d.a, r.Global):
			return fails(Fixable, fix(FixPrivateAddress, Params{URL: in.exposure.Public.String()})).seen(seen)
		case len(d.a) == 0:
			return fails(Fixable, fix(FixDNSFailed, Params{Host: in.host, Raw: d.errA.String()}))
		case in.direct() && in.o.public4.IsValid() && slices.ContainsFunc(d.a, in.o.previous):
			return fails(Fixable, fix(FixStaleA, Params{Host: in.host, Addr: in.o.facts.Previous4, Want: in.o.public4})).seen(seen)
		case in.direct() && nonGlobal >= 0:
			return fails(Fixable, fix(FixPrivateA, Params{Host: in.host, Addr: d.a[nonGlobal]})).seen(seen)
		case in.direct():
			if i := slices.IndexFunc(d.a, func(a netip.Addr) bool { return !in.o.holds(a) }); i >= 0 {
				return fails(Fixable, fix(FixExtraA, Params{Host: in.host, Addr: d.a[i]})).seen(seen)
			}
		}
		return passes().seen(seen)
	}}

// rowDNSAAAA runs for IPv6 whenever the deciding resolvers saw an AAAA
// answer or error, offered or not. A failing or non-Global AAAA offers
// nothing, yet clients that ask for IPv6 stall on it.
var rowDNSAAAA = row{ID: CheckDNSAAAA, Hop: DNSHop, Only: ptr(netmodel.IPv6), Applies: aaaaSeen,
	Eval: func(in *familyIn) outcome {
		d, r := in.dns, in.f.Rules
		seen := addrs(d.aaaa)
		switch {
		case in.pl.Routes[netmodel.IPv4].Kind == Unplaced:
			return asFact(seen)
		case d.errAAAA != DNSNone:
			return fails(Fixable, fix(FixAAAAFailed, Params{Host: in.host, Raw: d.errAAAA.String()}))
		}
		if i := slices.IndexFunc(d.aaaa, func(a netip.Addr) bool { return !r.Global(a) }); i >= 0 {
			return fails(Fixable, fix(FixPrivateAAAA, Params{Host: in.host, Addr: d.aaaa[i]})).seen(seen)
		}
		if !in.direct() {
			return passes().seen(seen)
		}
		if i := slices.IndexFunc(d.aaaa, in.o.previous); i >= 0 {
			p := Params{Host: in.host, Addr: d.aaaa[i], Node: in.o.node, Want: in.f.Pins[in.o.node]}
			if len(in.o.slash64) > 0 {
				p.Prefix = in.o.slash64[0]
			}
			return fails(Fixable, fix(FixPrefixChange, p)).seen(seen)
		}
		if i := slices.IndexFunc(d.aaaa, func(a netip.Addr) bool { return !in.o.holds(a) }); i >= 0 {
			return fails(Fixable, fix(FixAAAAOffOrigin, Params{Host: in.host, Addr: d.aaaa[i], Node: in.o.node})).seen(seen)
		}
		return passes().seen(seen)
	}}

func aaaaSeen(in *familyIn) bool {
	d := in.dns
	return len(d.aaaa) > 0 || d.errAAAA != DNSNone && (d.errAAAA != NXDomain || d.errA != NXDomain)
}

// notReaching is every origin row but class on a Media Server that runs on
// no Node: the reaching Node's bind and firewall say nothing about it.
func notReaching(in *familyIn) bool { return in.pl.OriginLabel != Reaching }

// rowBind reads the program on the Origin hop's port. Another program
// there, Caddy or IIS, is judged by its own socket, and the service's own
// bind is a fact.
var rowBind = row{ID: CheckOriginBind, Hop: OriginHop, Applies: notReaching, LocalGuess: true,
	Eval: func(in *familyIn) outcome {
		l := in.listener
		if !in.direct() {
			return asFact(Value{})
		}
		o := passes()
		if !in.ours() {
			id := FactOtherProgram
			if in.own.Loopback {
				id = FactOwnBind
			}
			o.Facts = append(o.Facts, Fact{ID: id, Params: Params{Node: in.o.node, Program: l.Program.Name, Port: l.Port}})
		}
		p := Params{Node: in.o.node, Program: l.Program.Name, Port: l.Port, Family: ptr(in.fam)}
		switch {
		case l.Loopback:
			o.Result, o.Points, o.Fix = Fails, Fixable, in.bindFix()
		case l.Family(in.fam) == netmodel.Unknown:
			o.Result, o.Points, o.Fix = NotTested, Fixable, fix(FixBindUnknown, p)
		case l.Family(in.fam) == netmodel.No:
			o.Result, o.Points, o.Fix = Fails, Fixable, fix(FixBindFamily, p)
		}
		return o
	}}

var rowFirewall = row{ID: CheckOriginFirewall, Hop: OriginHop, Applies: notReaching, LocalGuess: true,
	Eval: func(in *familyIn) outcome {
		if !in.direct() {
			return asFact(Value{Text: in.firewall.Because})
		}
		p := Params{Node: in.o.node, Port: in.port, Tool: in.firewall.Tool, Program: in.listener.Program.Name, Family: ptr(in.fam)}
		switch {
		case in.firewall.Policy == netmodel.Blocks:
			return fails(Fixable, &Fix{ID: FixFirewall, Params: p, Command: true})
		case in.firewall.Policy == netmodel.PolicyUnknown && in.listener.Docker:
			return notTested(Fixable, fix(FixFirewallDocker, p))
		case in.firewall.Policy == netmodel.PolicyUnknown:
			return notTested(Fixable, fix(FixFirewallUnknown, p).action(Action{Kind: PairNode}))
		case in.firewall.Partial:
			return notTested(Fixable, fix(FixFirewallUnknown, p).action(Action{Kind: PairNode})).
				with(Fact{ID: FactFirewalldPartial, Params: Params{Node: in.o.node, Port: in.port}})
		}
		return passes()
	}}

var rowClass = row{ID: CheckOriginClass, Hop: OriginHop, Only: ptr(netmodel.IPv4), LocalGuess: true,
	Applies: func(in *familyIn) bool { return in.direct() },
	Eval: func(in *familyIn) outcome {
		node := in.o.node
		switch in.class {
		case netmodel.Open:
			return passes()
		case netmodel.Mappable:
			switch {
			case in.liveMapping():
				return passes()
			case in.refusedOtherPort():
				return notTested(Fixable, fix(FixManualForward, in.forwardParams()))
			case in.plex():
				return notTested(Fixable, fix(FixMappablePlex, Params{Node: node}))
			}
			return notTested(Fixable, fix(FixMappable, Params{Node: node, Port: in.publicPort}))
		case netmodel.CGNAT:
			o := fails(NeedsRelay, fix(FixCGNAT, Params{Node: node, Program: in.subject()}))
			if in.plex() {
				o.Facts = append(o.Facts, Fact{ID: FactPlexRelayQuality})
			}
			return o
		case netmodel.DoubleNAT:
			return fails(Fixable, fix(FixDoubleNAT, Params{Node: node, Port: in.publicPort, Addr: in.o.facts.Gateway.WAN}))
		case netmodel.ThroughVPN:
			return notTested(Fixable, fix(FixVPN, in.forwardParams()))
		case netmodel.BehindRouter, netmodel.Class4Unknown:
			if in.class == netmodel.BehindRouter && !in.o.facts.Gateway.Addr.IsValid() {
				return notTested(Fixable, fix(FixNoGateway, Params{Node: node, Port: in.publicPort}))
			}
		}
		return notTested(Fixable, fix(FixForwardNotSeen, in.forwardParams()).action(Action{Kind: PairNode}))
	}}

var rowConnect = row{ID: CheckOutsideConnect, Hop: PlayerHop, Applies: always,
	Eval: func(in *familyIn) outcome {
		d := in.player
		switch {
		case d != nil && d.Stage >= Connected:
			o := passes().seen(Value{Text: d.Addr.String()})
			if in.route.Kind == OwnProxy {
				o.Facts = append(o.Facts, Fact{ID: FactTestedThroughProxy, Params: Params{Tester: d.Tester, Addr: in.route.Proxy}})
			}
			return o
		case in.plexCounts():
			// Plex's own check passes outside.connect and counts as tested,
			// even beside a failed test, which becomes a fact: a tester can
			// sit behind its own egress filter.
			o := passes()
			o.byPlex = true
			if d != nil {
				o.Facts = append(o.Facts, Fact{ID: FactPlexTested, Params: Params{Tester: d.Tester, Addr: d.Addr.Addr(), Port: d.Addr.Port()}})
			}
			return o
		case d == nil && in.fam == netmodel.IPv6:
			// Untested IPv6 is Fixable until S60 (IPv6 direct reach) reads
			// the router's IPv6 firewall.
			return notTested(Fixable, fix(FixUntestedIPv6, Params{Node: in.o.node, Port: in.publicPort}))
		case d == nil:
			return notTested(in.untestedPoints(), fix(FixNotTested, Params{}).action(Action{Kind: PairNode}))
		}
		return fails(Fixable, in.cause()).seen(Value{Text: d.Addr.String()})
	}}

// rowConnectBack places a failure for cause. It is a fact with a result,
// so it never decides a verdict on its own.
var rowConnectBack = row{ID: CheckConnectBack, Hop: OriginHop,
	Applies: func(in *familyIn) bool { return in.direct() },
	Eval: func(in *familyIn) outcome {
		d := in.connectBack
		if d == nil {
			return notUsed()
		}
		o := outcome{Result: Passes, Fact: true, Observed: Value{Text: d.Addr.String()}}
		if d.Stage < Connected {
			o.Result = Fails
			o.Facts = []Fact{{ID: FactConnectBack, Params: Params{Tester: d.Tester, Addr: d.Addr.Addr(), Port: d.Addr.Port()}}}
		}
		return o
	}}

var rowTLS = row{ID: CheckTLSCertificate, Hop: TLSHop, Applies: (*familyIn).https,
	Eval: func(in *familyIn) outcome {
		d := in.player
		switch {
		case d == nil || d.Stage < Connected:
			return notReached()
		case d.TLS == nil || d.Stage < TLSDone:
			return fails(Fixable, fix(FixCertificate, Params{Host: in.host, Raw: Handshake.String()}))
		case d.TLS.Problem != TLSNone:
			return fails(Fixable, fix(FixCertificate, Params{Host: in.host, Date: d.TLS.NotAfter, Raw: d.TLS.Problem.String()})).
				seen(Value{Text: strings.Join(d.TLS.Names, ", ")})
		}
		return passes()
	}}

var rowIdentity = row{ID: CheckIdentity, Hop: OriginHop, Applies: always,
	Eval: func(in *familyIn) outcome {
		d := in.player
		switch {
		case d == nil || d.Stage < Connected:
			return notReached()
		case in.https() && (d.TLS == nil || d.TLS.Problem != TLSNone):
			return notReached()
		case d.Stage < HTTPDone, d.Identity == OtherIdentity, d.Identity == Unread:
			return fails(Fixable, fix(FixIdentity, Params{Host: in.host, Port: in.publicPort, Program: in.subject()}))
		case d.Identity == SignIn && in.exposure.Kind == UIPortal:
			// A sign-in page in front of the UI is a fact, and the family
			// isn't proven, because nothing can test past it.
			return notTested(in.untestedPoints(), fix(FixSignInUI, Params{Host: in.host})).
				with(Fact{ID: FactSignInPage, Params: Params{Host: in.host}})
		case d.Identity == SignIn:
			return fails(Fixable, fix(FixSignInServer, Params{Host: in.host, Program: in.subject()}))
		case in.exposure.Kind == UIPortal:
			ok, why := AcceptEcho(in.f, in.pl, *d)
			o := passes()
			if !ok {
				o = notTested(in.untestedPoints(), fix(FixEchoUnconfirmed, Params{Tester: d.Tester}))
			}
			if why.ID != "" {
				o.Facts = append(o.Facts, why)
			}
			return o
		}
		return passes()
	}}

var familyRows = []row{rowDNSA, rowDNSAAAA, rowBind, rowFirewall, rowClass, rowConnect, rowConnectBack, rowTLS, rowIdentity}

// exposureRows have no family and point for the Exposure directly.
var exposureRows = []row{
	{ID: CheckHTTPPlain, Hop: TLSHop, Applies: func(in *familyIn) bool { return in.exposure.Kind == UIPortal },
		Eval: func(in *familyIn) outcome {
			if !in.https() {
				return fails(Fixable, fix(FixPlainHTTP, Params{}))
			}
			return passes()
		}},
	{ID: CheckPlexCustomURL, Hop: MediaServerHop,
		Applies: func(in *familyIn) bool { return in.plex() && !strings.HasSuffix(in.host, ".plex.direct") },
		Eval: func(in *familyIn) outcome {
			if in.listsPublic() {
				return passes()
			}
			return fails(Fixable, fix(FixCustomURL, Params{URL: in.exposure.Public.String()})).
				seen(Value{Text: strings.Join(in.server.Custom, ", ")})
		}},
	{ID: CheckPlexMapping, Hop: MediaServerHop,
		Applies: func(in *familyIn) bool { return in.plex() && in.server.Mapping != nil },
		Eval: func(in *familyIn) outcome {
			// Never fails: it feeds plexCounts and liveMapping.
			m := in.server.Mapping
			return asFact(Value{Text: m.Public.String()}, Fact{ID: FactPlexMapping, Params: Params{Raw: m.State}})
		}},
}

func (in *familyIn) check(r row, o outcome, perFamily bool) Check {
	c := Check{ID: r.ID, Hop: r.Hop, Result: o.Result, Fact: o.Fact, Points: o.Points, Observed: o.Observed, Fix: o.Fix}
	if !perFamily {
		return c
	}
	c.Family = ptr(in.fam)
	if r.ID == CheckIdentity && in.exposure.Kind == MediaServer {
		c.Hop = MediaServerHop
	}
	switch {
	case c.Hop == OriginHop && r.ID != CheckIdentity:
		c.By = in.o.node
	case in.player != nil:
		c.By = in.player.Tester
	}
	return c
}
