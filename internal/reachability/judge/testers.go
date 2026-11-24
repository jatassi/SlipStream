package judge

import (
	"cmp"
	"net/netip"
	"slices"

	"github.com/jatassi/slipstream/internal/netmodel"
	"github.com/jatassi/slipstream/internal/role"
)

// Testers orders the eligible testers for one family: the Relaying Node,
// then the Controller, then by name. It says why each other candidate was
// skipped, and adds link-unconfirmed for a kept Worker whose link couldn't
// confirm its claim. A Direct route takes only testers outside the
// origin's network, because a connection from inside can pass through
// hairpin NAT and prove nothing.
func Testers(f Facts, pl Placement, fam netmodel.Family) ([]role.NodeID, []Fact) {
	fr, ok := pl.Routes[fam]
	if !ok || fr.Kind == Unplaced {
		return nil, nil
	}
	cs := slices.Clone(f.Candidates)
	slices.SortStableFunc(cs, func(a, b Candidate) int {
		return cmp.Or(-cmp.Compare(btoi(a.Relaying), btoi(b.Relaying)), -cmp.Compare(btoi(a.Self), btoi(b.Self)), cmp.Compare(a.Name, b.Name))
	})
	var keep []role.NodeID
	var facts []Fact
	for _, c := range cs {
		fact, ok := eligible(f, pl, fam, c)
		if fact.ID != "" {
			fact.Family = &fam
			facts = append(facts, fact)
		}
		if ok {
			keep = append(keep, c.Node)
		}
	}
	return keep, facts
}

// eligible is whether c may test the family's route, with the fact that
// says why not, or that its claim stands unconfirmed.
func eligible(f Facts, pl Placement, fam netmodel.Family, c Candidate) (Fact, bool) {
	fr := pl.Routes[fam]
	nf := f.Nodes[c.Node]
	public4 := f.Rules.Public4(nf.Observed())
	switch {
	case c.Serves && !c.Self:
		return Fact{ID: FactServesIt, Params: Params{Tester: c.Node}}, false
	case !c.InStep:
		return Fact{ID: FactTesterAway, Params: Params{Tester: c.Node}}, false
	case fam == netmodel.IPv4 && !public4.IsValid(), fam == netmodel.IPv6 && f.Rules.Class6(nf.Interfaces) != netmodel.Global6:
		return Fact{}, false
	case fr.Kind == Direct && c.Node == pl.Origin:
		return Fact{}, false
	case slices.ContainsFunc(fr.Dialed, nf.Holds):
		return Fact{ID: FactAtAddress, Params: Params{Tester: c.Node, Addr: fr.Dialed[0]}}, false
	case fr.Kind == Direct && sameNetwork(f, pl.Origin, c.Node):
		return Fact{ID: FactSameNetwork, Params: Params{Tester: c.Node, Node: pl.Origin}}, false
	}
	if a, bad := testerDNS(f, c.Node, fam); bad {
		return Fact{ID: FactTesterDNS, Params: Params{Tester: c.Node, Host: f.Exposure.Public.Hostname(), Addr: a}}, false
	}
	if fr.Kind == Direct && !c.Self {
		fact, refused := linkClaim(f, c, fam)
		return fact, !refused
	}
	return Fact{}, true
}

// linkClaim compares a Worker's Global link address with the claim of its
// family. A non-Global link (an overlay) or one of the other family leaves
// the claim unconfirmed and the Worker kept.
func linkClaim(f Facts, c Candidate, fam netmodel.Family) (Fact, bool) {
	nf := f.Nodes[c.Node]
	a := c.Link.Addr.Unmap()
	if !f.Rules.Global(a) {
		return Fact{ID: FactLinkUnconfirmed, Family: &fam, Params: Params{Tester: c.Node, Reason: ReasonPrivate}}, false
	}
	if (fam == netmodel.IPv6) != a.Is6() {
		return Fact{ID: FactLinkUnconfirmed, Family: &fam, Params: Params{Tester: c.Node, Reason: ReasonOtherFamily}}, false
	}
	if a.Is4() {
		claim := f.Rules.Public4(nf.Observed())
		if a != claim {
			return Fact{ID: FactClaimContradicted, Family: &fam, Params: Params{Tester: c.Node, Addr: claim, Want: a}}, true
		}
		return Fact{}, false
	}
	if !slices.Contains(f.Rules.Prefixes6(nf.Interfaces), netmodel.Slash64(a)) {
		return Fact{ID: FactClaimContradicted, Family: &fam, Params: Params{Tester: c.Node, Prefix: firstPrefix(f.Rules.Prefixes6(nf.Interfaces)), Want: a}}, true
	}
	return Fact{}, false
}

func firstPrefix(ps []netip.Prefix) netip.Prefix {
	if len(ps) == 0 {
		return netip.Prefix{}
	}
	return ps[0]
}

// testerDNS is a tester whose own resolution of the public host answers a
// non-Global address or one of its own.
func testerDNS(f Facts, n role.NodeID, fam netmodel.Family) (netip.Addr, bool) {
	nf := f.Nodes[n]
	for _, d := range f.DNS {
		if d.By != n {
			continue
		}
		as := d.A
		if fam == netmodel.IPv6 {
			as = d.AAAA
		}
		for _, a := range as {
			if !f.Rules.Global(a) || nf.Holds(a) {
				return a, true
			}
		}
	}
	return netip.Addr{}, false
}

func btoi(b bool) int {
	if b {
		return 1
	}
	return 0
}

// TargetSource is where a connect-back target came from.
type TargetSource uint8

const (
	// FromPublic4 is the origin's Public4.
	FromPublic4 TargetSource = iota
	// FromPublished is a Media Server's published IPv6 connection.
	FromPublished
)

// Vet decides whether tester may be sent to target: Global unicast under
// the rules, outside the tester's own networks, on one of the Exposure's
// ports, and for a published IPv6 connection inside a /64 the origin holds.
// A target taken from a Worker's claim passes here or is never sent.
func Vet(f Facts, pl Placement, tester role.NodeID, target netip.AddrPort, from TargetSource) bool {
	a := target.Addr().Unmap()
	nf := f.Nodes[tester]
	switch {
	case !f.Rules.Global(a):
		return false
	case a == f.Rules.Public4(nf.Observed()) || nf.OnLink(a):
		return false
	case target.Port() != f.Exposure.PublicPort() && target.Port() != f.Exposure.Local:
		return false
	case from == FromPublished:
		return a.Is6() && slices.Contains(f.Rules.Prefixes6(f.Nodes[pl.Origin].Interfaces), netmodel.Slash64(a))
	}
	return true
}

// AcceptEcho decides whether the echo's arrival proves an outside fetch.
// It refuses a fetch that came through the tester's own Serve stream
// (echo-own-stream), or whose source is a non-Global interface address or
// a loopback of the tester (echo-tester-address). On a Direct route the
// source must be the tester's Public4 or lie in one of its Global /64s
// (echo-source), except that a loopback source at an origin whose public
// port belongs to another program is accepted with echo-through-program:
// Caddy or IIS forwarded it, and SlipStream can't see the tester.
func AcceptEcho(f Facts, pl Placement, d Dial) (bool, Fact) {
	e := d.Echo
	if e == nil {
		return false, Fact{}
	}
	fam := d.Family
	tf := f.Nodes[d.Tester]
	src := e.Source.Unmap()
	refuse := func(id FactID, p Params) (bool, Fact) {
		p.Tester = d.Tester
		return false, Fact{ID: id, Family: &fam, Params: p}
	}
	switch {
	case !e.Via.IsZero() && e.Via == d.Tester:
		return refuse(FactEchoOwnStream, Params{})
	case src.IsLoopback() && e.At == d.Tester, !f.Rules.Global(src) && !src.IsLoopback() && tf.Holds(src):
		return refuse(FactEchoTesterAddress, Params{Addr: src})
	case pl.Routes[fam].Kind != Direct:
		return true, Fact{}
	case src.Is4() && src == f.Rules.Public4(tf.Observed()),
		src.Is6() && slices.Contains(f.Rules.Prefixes6(tf.Interfaces), netmodel.Slash64(src)):
		return true, Fact{}
	case src.IsLoopback() && e.At == pl.Origin:
		in := newFamilyIn(f, pl, fam)
		if !in.ours() {
			return true, Fact{ID: FactEchoThroughProgram, Family: &fam, Params: Params{Node: pl.Origin, Program: in.listener.Program.Name, Port: in.listener.Port}}
		}
	}
	return refuse(FactEchoSource, Params{Addr: src})
}
