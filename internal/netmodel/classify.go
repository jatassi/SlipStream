package netmodel

import (
	"net/netip"
	"slices"
)

// Note is a fact a class rule adds beside the class.
type Note uint8

const (
	// STUNVariesPort is STUN servers seeing one address on different ports.
	STUNVariesPort Note = iota + 1
	// STUNVariesAddress is STUN servers seeing different addresses.
	STUNVariesAddress
)

// Classify is the class rule; the first rule that matches wins. The Node
// calls it for its summary and the judge calls it again on the full facts,
// so the Controller decides the class.
func (r Rules) Classify(o Observed) (Class4, []Note) {
	mapped := stunMapped(o.STUN)
	if len(mapped) == 0 {
		return Class4Unknown, nil
	}
	stun := mapped[0].Addr()
	var notes []Note
	varies := false
	for _, m := range mapped[1:] {
		if m.Addr() != stun {
			varies = true
		} else if m.Port() != mapped[0].Port() && !slices.Contains(notes, STUNVariesPort) {
			notes = append(notes, STUNVariesPort)
		}
	}
	wan := o.Gateway.WAN
	p := o.Gateway.Protocols
	switch {
	case holds(o.Interfaces, stun):
		return Open, notes
	case egressTunnel(o):
		return ThroughVPN, notes
	case varies:
		return BehindRouter, append(notes, STUNVariesAddress)
	case shared.Contains(wan):
		// The gateway's answer only: STUN can't carry 100.64.0.0/10.
		return CGNAT, notes
	case wan.IsValid() && (!r.Global(wan) || wan != stun):
		return DoubleNAT, notes
	case wan == stun && (p.PCP == Yes || p.NATPMP == Yes || p.UPnP == Yes):
		return Mappable, notes
	}
	return BehindRouter, notes
}

// Public4 is the STUN answer, zero under ThroughVPN, where it is the
// tunnel's exit, and with no answer.
func (r Rules) Public4(o Observed) netip.Addr {
	c, _ := r.Classify(o)
	if c == ThroughVPN || c == Class4Unknown {
		return netip.Addr{}
	}
	return stunMapped(o.STUN)[0].Addr()
}

func (r Rules) Class6(ifaces []Iface) Class6 {
	c := Class6None
	for _, i := range ifaces {
		for _, a := range i.Addrs {
			if !a.Prefix.Addr().Is6() {
				continue
			}
			k := r.Of(a.Prefix.Addr())
			if k == Global {
				return Global6
			}
			if k == ULA || k == LinkLocal {
				c = LocalOnly
			}
		}
	}
	return c
}

// Prefixes6 is the sorted set of Global /64s the interfaces hold.
func (r Rules) Prefixes6(ifaces []Iface) []netip.Prefix {
	var out []netip.Prefix
	for _, i := range ifaces {
		for _, a := range i.Addrs {
			if ad := a.Prefix.Addr(); ad.Is6() && r.Global(ad) {
				out = append(out, Slash64(ad))
			}
		}
	}
	return sortPrefixes(out)
}

func sortPrefixes(ps []netip.Prefix) []netip.Prefix {
	slices.SortFunc(ps, func(a, b netip.Prefix) int { return a.Addr().Compare(b.Addr()) })
	return slices.Compact(ps)
}

func stunMapped(as []STUNAnswer) []netip.AddrPort {
	var out []netip.AddrPort
	for _, a := range as {
		if a.Err == STUNNone && a.Mapped.IsValid() {
			out = append(out, netip.AddrPortFrom(a.Mapped.Addr().Unmap(), a.Mapped.Port()))
		}
	}
	return out
}

func holds(ifaces []Iface, a netip.Addr) bool {
	return NetworkFacts{Interfaces: ifaces}.Holds(a)
}

func egressTunnel(o Observed) bool {
	for _, i := range o.Interfaces {
		if i.Name == o.Egress.Iface && i.Tunnel {
			return true
		}
	}
	return false
}
