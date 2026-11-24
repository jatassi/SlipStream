package judgetest

import (
	"fmt"
	"net/netip"
	"reflect"
	"slices"
	"testing"
	"time"

	"github.com/jatassi/slipstream/internal/netmodel"
	"github.com/jatassi/slipstream/internal/reachability/judge"
)

// The presets are the remote-* Scenarios' shapes: atlas, the Controller,
// runs Plex on its home network.

// Open is remote-open: atlas holds the public address, seedbox is outside.
func Open() *B {
	return base("http://plex.open.localhost:32400").Class(netmodel.Open).Outside().Resolves("203.0.113.9")
}

// Mappable is remote-mappable: the router answers UPnP IGD and its WAN
// address is the STUN answer, with nothing forwarded.
func Mappable() *B {
	return base("http://plex.mappable.localhost:32400").Class(netmodel.Mappable).Outside().Resolves("203.0.113.9")
}

// SingleMachine is remote-single-machine: one Controller behind a router
// that answers no mapping protocol, and no other Node.
func SingleMachine() *B {
	return base("http://plex.single.localhost:32400").Class(netmodel.BehindRouter).Resolves("203.0.113.9")
}

// CGNAT is remote-cgnat: the router's WAN address is in 100.64.0.0/10.
func CGNAT() *B {
	return base("http://plex.cgnat.localhost:32400").Class(netmodel.CGNAT).Outside().Resolves("203.0.113.50")
}

// DoubleNAT is remote-double-nat: the router's WAN address is private.
func DoubleNAT() *B {
	return base("http://plex.double.localhost:32400").Class(netmodel.DoubleNAT).Outside().Resolves("203.0.113.9")
}

// VPN is remote-vpn: atlas's egress is a WireGuard tunnel.
func VPN() *B {
	return base("http://plex.vpn.localhost:32400").Class(netmodel.ThroughVPN).Outside().Resolves("203.0.113.9")
}

// OwnProxy is remote-own-proxy, Jackson's shape: the A names the Admin's
// proxy at 198.51.100.8, which isn't paired.
func OwnProxy() *B {
	return base("https://plex.proxy.fake.localhost").Class(netmodel.Mappable).Resolves("198.51.100.8").PlexMapping("mapped", false)
}

// PairedProxy is OwnProxy with the proxy paired as the Worker the-vps.
func PairedProxy() *B {
	return base("https://plex.proxy.fake.localhost").Class(netmodel.Mappable).Proxy().Resolves("198.51.100.8")
}

// IPv6 is remote-ipv6: Open, with atlas at 2001:db8:1::20 and an AAAA.
func IPv6() *B {
	return base("http://plex.v6.localhost:32400").Class(netmodel.Open).V6("2001:db8:1::20/64").Outside().
		Resolves("203.0.113.9", "2001:db8:1::20")
}

// OpenUI is remote-open for the UI and portal at https://slipstream.open.localhost.
func OpenUI() *B {
	return Open().UI("https://slipstream.open.localhost")
}

// Want is what a row asserts: ids and params, never words.
type Want struct {
	Verdict  judge.Verdict
	Tested   bool
	Card     judge.Card
	Deciding judge.CheckID
	Family   netmodel.Family // the deciding check's; zero for one with no family
	Fix      judge.FixID
	Params   judge.Params
	Facts    []judge.FactID // each shown somewhere on the route
	// Action is the fix's button, asserted on the rows that name one.
	Action *judge.Action
}

type Row struct {
	Name  string
	Facts judge.Facts
	Want  Want
}

var (
	v4 = netmodel.IPv4
	v6 = netmodel.IPv6
)

func forward() judge.Params {
	return judge.Params{Node: Atlas, Port: 32400, LocalAddr: AtlasLAN, LocalPort: 32400}
}

func row(name string, b *B, w Want) Row { return Row{Name: name, Facts: b.Facts(), Want: w} }

// Rows is the verdict spec: one row per verdict rule and deciding check.
func Rows() []Row {
	reachable := Want{Verdict: judge.Reachable, Tested: true, Card: judge.NoCard}
	with := func(w Want, facts ...judge.FactID) Want { w.Facts = facts; return w }
	button := func(w Want, k judge.ActionKind) Want { w.Action = &judge.Action{Kind: k, Node: Atlas}; return w }
	untested := func(v judge.Verdict, check judge.CheckID, fam netmodel.Family, fx judge.FixID, p judge.Params, facts ...judge.FactID) Want {
		return Want{Verdict: v, Card: judge.Untested, Deciding: check, Family: fam, Fix: fx, Params: p, Facts: facts}
	}
	failing := func(v judge.Verdict, tested bool, check judge.CheckID, fam netmodel.Family, fx judge.FixID, p judge.Params, facts ...judge.FactID) Want {
		card := judge.Warn
		if v == judge.NeedsRelay {
			card = judge.Fail
		}
		return Want{Verdict: v, Tested: tested, Card: card, Deciding: check, Family: fam, Fix: fx, Params: p, Facts: facts}
	}
	confirm := untested(judge.Reachable, judge.CheckOutsideConnect, v4, judge.FixNotTested, judge.Params{})
	firewall := func(fam netmodel.Family) judge.Params {
		return judge.Params{Node: Atlas, Port: 32400, Tool: netmodel.WindowsDefender, Program: "Plex Media Server", Family: &fam}
	}
	plexOn := judge.Params{Node: Atlas, Program: "Plex Media Server", Port: 32400}
	ui := "slipstream.open.localhost"
	proxyCustom := "http://" + netip.AddrPortFrom(ProxyIP, 32400).String()
	return []Row{
		// Jackson's own proxy (U2).
		row("own proxy tested from atlas", OwnProxy().Tested(v4, Passed), with(reachable, judge.FactTestedThroughProxy, judge.FactPlexMapping)),
		row("own proxy with a custom URL listed with its default port",
			OwnProxy().Custom(proxyCustom+",https://plex.proxy.fake.localhost:443").Tested(v4, Passed), reachable),
		row("own proxy untested", OwnProxy(),
			untested(judge.Fixable, judge.CheckOutsideConnect, v4, judge.FixNotTested, judge.Params{})),
		row("own proxy certificate expired", OwnProxy().Tested(v4, Expired),
			failing(judge.Fixable, false, judge.CheckTLSCertificate, v4, judge.FixCertificate,
				judge.Params{Host: "plex.proxy.fake.localhost", Date: Expires, Raw: "expired"})),
		row("own proxy certificate for another name", OwnProxy().Tested(v4, WrongName),
			failing(judge.Fixable, false, judge.CheckTLSCertificate, v4, judge.FixCertificate,
				judge.Params{Host: "plex.proxy.fake.localhost", Date: Expires.AddDate(1, 0, 0), Raw: "wrong-name"})),
		row("own proxy sign-in page before Plex", OwnProxy().Tested(v4, SignInPage),
			failing(judge.Fixable, false, judge.CheckIdentity, v4, judge.FixSignInServer, judge.Params{Host: "plex.proxy.fake.localhost", Program: "Plex"})),
		row("Plex doesn't list the public URL", OwnProxy().Custom(proxyCustom).Tested(v4, Passed),
			failing(judge.Fixable, true, judge.CheckPlexCustomURL, 0, judge.FixCustomURL, judge.Params{URL: "https://plex.proxy.fake.localhost"})),
		row("paired proxy closed with its leg home open", PairedProxy().Tested(v4, Closed).ProxyLeg(true),
			failing(judge.Fixable, false, judge.CheckOutsideConnect, v4, judge.FixProxyPort, judge.Params{Port: 443, Addr: ProxyIP},
				judge.FactProxyLeg)),
		row("paired proxy and its leg home both closed", PairedProxy().Tested(v4, Closed).ProxyLeg(false),
			failing(judge.Fixable, false, judge.CheckOutsideConnect, v4, judge.FixProxyForward, judge.Params{Addr: ProxyIP, Node: Atlas, Port: 32400})),
		row("paired proxy closed and Plex on this machine only", PairedProxy().Loopback().Tested(v4, Closed).ProxyLeg(false),
			failing(judge.Fixable, false, judge.CheckOutsideConnect, v4, judge.FixBindProgram, plexOn)),

		// A single machine (U3, stories 30 and 31).
		row("single machine with no tester", SingleMachine(),
			untested(judge.Fixable, judge.CheckOriginClass, v4, judge.FixForwardNotSeen, forward())),
		row("single machine where Plex's own check passes", SingleMachine().PlexMapping("mapped", false), reachable),
		row("Plex's mapping still pending counts for nothing", SingleMachine().PlexMapping("mapped", true),
			untested(judge.Fixable, judge.CheckOriginClass, v4, judge.FixForwardNotSeen, forward(), judge.FactPlexMapping)),
		row("Plex's own check passes beside a failed test", Mappable().PlexMapping("mapped", false).Tested(v4, Closed),
			with(reachable, judge.FactPlexTested)),

		// Classes (U4 to U6, U11).
		row("open and tested", Open().Tested(v4, Passed), reachable),
		row("open and untested is Reachable on local proof", Open(), confirm),
		row("mappable with no mapping", Mappable(),
			untested(judge.Fixable, judge.CheckOriginClass, v4, judge.FixMappablePlex, judge.Params{Node: Atlas})),
		row("mappable UI with no mapping", Mappable().UI("https://slipstream.mappable.localhost:8484"),
			button(untested(judge.Fixable, judge.CheckOriginClass, v4, judge.FixMappable, judge.Params{Node: Atlas, Port: 8484}), judge.MapPorts)),
		row("mappable with SlipStream's mapping, untested", Mappable().Mapped(time.Hour), confirm),
		row("mappable granted on another port", Mappable().Refused(netmodel.OtherPort),
			untested(judge.Fixable, judge.CheckOriginClass, v4, judge.FixManualForward, forward())),
		row("mappable and the router refuses SlipStream", Mappable().Refused(netmodel.NotAuthorized),
			untested(judge.Fixable, judge.CheckOriginClass, v4, judge.FixManualForward, forward())),
		row("mappable and closed from outside", Mappable().Tested(v4, Closed),
			failing(judge.Fixable, false, judge.CheckOutsideConnect, v4, judge.FixMappablePlex, judge.Params{Node: Atlas})),
		row("a permanent mapping", Mappable().Mapped(0).Tested(v4, Passed), with(reachable, judge.FactMappingPermanent)),
		row("CGNAT", CGNAT().Tested(v4, Closed),
			failing(judge.NeedsRelay, false, judge.CheckOriginClass, v4, judge.FixCGNAT, judge.Params{Node: Atlas, Program: "Plex"}, judge.FactPlexRelayQuality)),
		row("CGNAT with IPv6 that works", CGNAT().V6("2001:db8:1::20/64").Resolves("203.0.113.50", "2001:db8:1::20").Tested(v6, Passed),
			failing(judge.NeedsRelay, false, judge.CheckOriginClass, v4, judge.FixCGNAT, judge.Params{Node: Atlas, Program: "Plex"}, judge.FactIPv6ViewersDirect)),
		row("double NAT", DoubleNAT(),
			failing(judge.Fixable, false, judge.CheckOriginClass, v4, judge.FixDoubleNAT, judge.Params{Node: Atlas, Port: 32400, Addr: ip("192.168.0.5")})),
		row("double NAT whose outer router forwards", DoubleNAT().Tested(v4, Passed), reachable),
		row("STUN answers that differ by address", Mappable().STUNVaries(true),
			untested(judge.Fixable, judge.CheckOriginClass, v4, judge.FixForwardNotSeen, forward(), judge.FactSTUNVariesAddress)),
		row("STUN answers that differ by port", Mappable().STUNVaries(false).Tested(v4, Passed), with(reachable, judge.FactSTUNVariesPort)),
		row("VPN egress", VPN(), untested(judge.Fixable, judge.CheckOriginClass, v4, judge.FixVPN, forward())),

		// Bind and firewall (stories 22 and 37).
		row("firewall blocks, closed from outside", Open().Firewall(netmodel.Blocks).Tested(v4, Closed),
			failing(judge.Fixable, false, judge.CheckOutsideConnect, v4, judge.FixFirewall, firewall(v4))),
		row("firewall unknown", Open().Firewall(netmodel.PolicyUnknown),
			untested(judge.Fixable, judge.CheckOriginFirewall, v4, judge.FixFirewallUnknown, firewall(v4))),
		row("firewall unknown is overruled by a passing test", Open().Firewall(netmodel.PolicyUnknown).Tested(v4, Passed), reachable),
		row("firewalld allows with its other rules unread", Open().FirewalldPartial(),
			untested(judge.Fixable, judge.CheckOriginFirewall, v4, judge.FixFirewallUnknown,
				judge.Params{Node: Atlas, Port: 32400, Tool: netmodel.Firewalld, Program: "Plex Media Server", Family: &v4}, judge.FactFirewalldPartial)),
		row("Docker publishes the port", Open().Docker(),
			untested(judge.Fixable, judge.CheckOriginFirewall, v4, judge.FixFirewallDocker, firewall(v4))),
		row("Plex on this machine only", Open().Loopback(),
			failing(judge.Fixable, false, judge.CheckOriginBind, v4, judge.FixBindProgram, plexOn)),
		row("SlipStream on this machine only", OpenUI().Loopback(),
			failing(judge.Fixable, false, judge.CheckOriginBind, v4, judge.FixBind, judge.Params{Node: Atlas})),
		row("bind families unknown", Open().ServerListens(netmodel.Unknown, netmodel.Unknown),
			untested(judge.Fixable, judge.CheckOriginBind, v4, judge.FixBindUnknown,
				judge.Params{Node: Atlas, Program: "Plex Media Server", Port: 32400, Family: &v4})),
		row("Plex listens on IPv4 only", IPv6().ServerListens(netmodel.Yes, netmodel.No).Tested(v4, Passed).Tested(v6, Closed),
			failing(judge.Fixable, false, judge.CheckOutsideConnect, v6, judge.FixBindFamily,
				judge.Params{Node: Atlas, Program: "Plex Media Server", Port: 32400, Family: &v6})),
		row("the firewall blocks IPv6 only", IPv6().FirewallFor(v6, netmodel.Blocks).Tested(v4, Passed).Tested(v6, Closed),
			failing(judge.Fixable, false, judge.CheckOutsideConnect, v6, judge.FixFirewall, firewall(v6))),
		row("closed with nothing local to blame", Open().Tested(v4, Closed).ConnectBack(false),
			failing(judge.Fixable, false, judge.CheckOutsideConnect, v4, judge.FixNoGateway, judge.Params{Node: Atlas, Port: 32400}, judge.FactConnectBack)),
		row("something else responds", Open().Tested(v4, Something),
			failing(judge.Fixable, false, judge.CheckIdentity, v4, judge.FixIdentity, judge.Params{Host: "plex.open.localhost", Port: 32400, Program: "Plex"})),
		row("connected with no response", Open().Tested(v4, NoResponse),
			failing(judge.Fixable, false, judge.CheckIdentity, v4, judge.FixIdentity, judge.Params{Host: "plex.open.localhost", Port: 32400, Program: "Plex"})),

		// DNS (stories 27 and 34).
		row("an A on the home's old address", Open().Previous4("203.0.113.7").Resolves("203.0.113.7"),
			failing(judge.Fixable, false, judge.CheckDNSA, v4, judge.FixStaleA,
				judge.Params{Host: "plex.open.localhost", Addr: ip("203.0.113.7"), Want: HomeWAN})),
		row("an extra A", Open().Resolves("203.0.113.9", "198.51.100.99"),
			failing(judge.Fixable, false, judge.CheckDNSA, v4, judge.FixExtraA, judge.Params{Host: "plex.open.localhost", Addr: ip("198.51.100.99")})),
		row("a private A seen from outside", Open().Resolves("192.168.1.20"),
			failing(judge.Fixable, false, judge.CheckDNSA, v4, judge.FixPrivateA, judge.Params{Host: "plex.open.localhost", Addr: AtlasLAN})),
		row("the UI's Address is private", OpenUI().Resolves("192.168.1.20"),
			failing(judge.Fixable, false, judge.CheckDNSA, v4, judge.FixPrivateAddress, judge.Params{URL: "https://slipstream.open.localhost"})),
		row("NXDOMAIN", Open().DNSError(v4, judge.NXDomain),
			failing(judge.Fixable, false, judge.CheckDNSA, v4, judge.FixDNSFailed, judge.Params{Host: "plex.open.localhost", Raw: "NXDOMAIN"})),
		row("split horizon with no outside resolver", SingleMachine().Resolves("192.168.1.20"),
			untested(judge.Fixable, judge.CheckOutsideConnect, v4, judge.FixNotTested, judge.Params{}, judge.FactSplitHorizon)),
		row("split horizon beside an outside answer", Open().ResolvesAt(Atlas, "192.168.1.20").Tested(v4, Passed),
			with(reachable, judge.FactSplitHorizon)),

		// IPv6 (U12, ADR 0022).
		row("IPv6 tested both ways", IPv6().Tested(v4, Passed).Tested(v6, Passed), reachable),
		row("an AAAA with no A", IPv6().Resolves("2001:db8:1::20").Tested(v6, Passed),
			failing(judge.NeedsRelay, false, judge.CheckDNSA, v4, judge.FixNoA, judge.Params{Host: "plex.v6.localhost", Program: "Plex"}, judge.FactIPv6ViewersDirect)),
		row("untested IPv6", IPv6().Tested(v4, Passed),
			untested(judge.Fixable, judge.CheckOutsideConnect, v6, judge.FixUntestedIPv6, judge.Params{Node: Atlas, Port: 32400})),
		row("a broken AAAA", IPv6().Tested(v4, Passed).Tested(v6, Closed),
			failing(judge.Fixable, false, judge.CheckOutsideConnect, v6, judge.FixBrokenAAAA,
				judge.Params{Host: "plex.v6.localhost", Addr: AtlasV6, Port: 32400, Node: Atlas})),
		row("a private AAAA offers nothing and fails", Open().Resolves("203.0.113.9", "fd72::20").Tested(v4, Passed),
			failing(judge.Fixable, true, judge.CheckDNSAAAA, v6, judge.FixPrivateAAAA, judge.Params{Host: "plex.open.localhost", Addr: ip("fd72::20")})),
		row("an AAAA lookup that fails", Open().DNSError(v6, judge.ServFail).Tested(v4, Passed),
			failing(judge.Fixable, true, judge.CheckDNSAAAA, v6, judge.FixAAAAFailed, judge.Params{Host: "plex.open.localhost", Raw: "SERVFAIL"})),
		row("an AAAA in the old prefix", IPv6().Class(netmodel.Open).V6("2001:db8:2::20/64").Previous6("2001:db8:1::/64").Tested(v4, Passed),
			failing(judge.Fixable, false, judge.CheckDNSAAAA, v6, judge.FixPrefixChange,
				judge.Params{Host: "plex.v6.localhost", Addr: AtlasV6, Node: Atlas, Prefix: pfx("2001:db8:2::/64")})),
		row("an AAAA in the old prefix with a pinned address",
			IPv6().Class(netmodel.Open).V6("2001:db8:2::20/64").Previous6("2001:db8:1::/64").Pin("2001:db8:2::20").Tested(v4, Passed),
			failing(judge.Fixable, false, judge.CheckDNSAAAA, v6, judge.FixPrefixChange,
				judge.Params{Host: "plex.v6.localhost", Addr: AtlasV6, Node: Atlas, Prefix: pfx("2001:db8:2::/64"), Want: ip("2001:db8:2::20")})),
		row("an AAAA off the origin", IPv6().Resolves("203.0.113.9", "2001:db8:1::20", "2001:db8:5::1").Tested(v4, Passed),
			failing(judge.Fixable, false, judge.CheckDNSAAAA, v6, judge.FixAAAAOffOrigin,
				judge.Params{Host: "plex.v6.localhost", Addr: ip("2001:db8:5::1"), Node: Atlas})),
		row("an AAAA on the origin beside a proxy A", OwnProxy().V6("2001:db8:1::20/64").Resolves("198.51.100.8", "2001:db8:1::20").Tested(v4, Passed),
			untested(judge.Fixable, judge.CheckOutsideConnect, v6, judge.FixUntestedIPv6, judge.Params{Node: Atlas, Port: 443}, judge.FactIPv6SkipsProxy)),

		// Testers and trust (U8, U9, U17).
		row("an inside tester never tests", Open().Inside().Away(Seedbox),
			with(confirm, judge.FactSameNetwork, judge.FactTesterAway)),
		row("a contradicted claim", Open().LinkFrom(Seedbox, "198.51.100.20"), with(confirm, judge.FactClaimContradicted)),
		row("a link over a private network", Open().LinkFrom(Seedbox, "100.101.7.3").Tested(v4, Passed),
			with(reachable, judge.FactLinkUnconfirmed)),
		row("a Worker that serves the Exposure", Open().Serves(Seedbox), with(confirm, judge.FactServesIt)),
		row("a public address that keeps changing", Open().Churn().Tested(v4, Passed), with(reachable, judge.FactAddressChurn)),
		row("a Media Server on no Node", Open().Reaching().Loopback(), with(confirm, judge.FactClassFromReaching)),

		// The UI and portal and its echo (U7, U16, story 21).
		row("plain HTTP", Open().UI("http://slipstream.open.localhost:8484").Tested(v4, Passed),
			failing(judge.Fixable, true, judge.CheckHTTPPlain, 0, judge.FixPlainHTTP, judge.Params{})),
		row("the echo from outside", OpenUI().Tested(v4, Passed), reachable),
		row("the echo through Caddy", OpenUI().Caddy().Loopback().Tested(v4, Passed).Echo(ThroughCaddy),
			with(reachable, judge.FactEchoThroughProgram, judge.FactOwnBind)),
		row("a loopback echo with no program in front", OpenUI().Tested(v4, Passed).Echo(ThroughCaddy),
			untested(judge.Reachable, judge.CheckIdentity, v4, judge.FixEchoUnconfirmed, judge.Params{Tester: Seedbox}, judge.FactEchoSource)),
		row("the echo through the tester's own stream", OpenUI().Tested(v4, Passed).Echo(OwnStream),
			untested(judge.Reachable, judge.CheckIdentity, v4, judge.FixEchoUnconfirmed, judge.Params{Tester: Seedbox}, judge.FactEchoOwnStream)),
		row("the echo from the tester itself", OpenUI().Tested(v4, Passed).Echo(TesterLoopback),
			untested(judge.Reachable, judge.CheckIdentity, v4, judge.FixEchoUnconfirmed, judge.Params{Tester: Seedbox}, judge.FactEchoTesterAddress)),
		row("the echo from another address", OpenUI().Tested(v4, Passed).Echo(OtherSource),
			untested(judge.Reachable, judge.CheckIdentity, v4, judge.FixEchoUnconfirmed, judge.Params{Tester: Seedbox}, judge.FactEchoSource)),
		row("an echo the Controller forgot", OpenUI().Tested(v4, Passed).Echo(Dropped),
			untested(judge.Reachable, judge.CheckIdentity, v4, judge.FixEchoUnconfirmed, judge.Params{Tester: Seedbox})),
		row("a sign-in page before the UI", OpenUI().Tested(v4, SignInPage),
			untested(judge.Reachable, judge.CheckIdentity, v4, judge.FixSignInUI, judge.Params{Host: ui}, judge.FactSignInPage)),
	}
}

// Check judges r and reports every way the result misses r.Want, printing
// the deciding check, fix id and params.
func Check(t testing.TB, r Row) {
	t.Helper()
	j := judge.Judge(r.Facts)
	w := r.Want
	got := Want{Verdict: j.Verdict, Tested: j.Tested, Card: j.Card()}
	if j.Deciding != nil {
		got.Deciding = j.Deciding.ID
		if j.Deciding.Family != nil {
			got.Family = *j.Deciding.Family
		}
	}
	if fx := j.Fix(); fx != nil {
		got.Fix, got.Params = fx.ID, fx.Params
		if w.Action != nil {
			got.Action = fx.Action
		}
	}
	shown := Shown(j)
	for _, id := range w.Facts {
		if slices.Contains(shown, id) {
			got.Facts = append(got.Facts, id)
		}
	}
	if !reflect.DeepEqual(got, w) {
		t.Errorf("%s:\n got  %s\n want %s\n facts shown %v", r.Name, describe(got), describe(w), shown)
	}
}

func describe(w Want) string {
	return fmt.Sprintf("%v tested=%v card=%v deciding=%s/%v fix=%s params=%+v facts=%v action=%+v",
		w.Verdict, w.Tested, w.Card, w.Deciding, w.Family, w.Fix, w.Params, w.Facts, w.Action)
}

// Shown is every fact on j's route: hop facts, the facts of fixes'
// checks, and the Exposure's own.
func Shown(j judge.Judged) []judge.FactID {
	var out []judge.FactID
	for _, f := range j.Notes {
		out = append(out, f.ID)
	}
	for _, f := range j.Facts {
		out = append(out, f.ID)
	}
	for _, h := range j.Hops {
		for _, f := range h.Facts {
			out = append(out, f.ID)
		}
	}
	return out
}
