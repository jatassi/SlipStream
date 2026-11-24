package netmodel

import (
	"bytes"
	"errors"
	"fmt"
	"net/netip"
	"time"

	"google.golang.org/protobuf/types/known/emptypb"

	netv1 "github.com/jatassi/slipstream/internal/pb/slipstream/network/v1"
	"github.com/jatassi/slipstream/internal/role"
)

// Settings is a Node's network section. Script is the scripted readers'
// file, set only by the Fake world's scripted-network divergence.
type Settings struct {
	Script string `yaml:"script"`
}

// RulesOf is the classification rules a Node's network section asks for:
// a scripted Node lives on the documentation prefixes.
func RulesOf(s Settings) Rules { return Rules{DocumentationGlobal: s.Script != ""} }

// Def is the network Node-wide channel. Its facts are notable when the
// revision changes. It has no results.
var Def = role.WideDef[*netv1.Intent, *netv1.Facts, *emptypb.Empty]{
	Name:    role.Network,
	Notable: func(last, next *netv1.Facts) bool { return !bytes.Equal(last.GetRevision(), next.GetRevision()) },
}

// SummaryOnWire is s as the channel's facts, observed at at.
func SummaryOnWire(s Summary, at time.Time) *netv1.Facts {
	m := &netv1.Facts{Watched: s.Watched, ObservedAtMs: at.UnixMilli()}
	if !s.Watched {
		return m
	}
	m.Class = netv1.Class4(s.Class)
	m.Public4 = addrBytes(s.Public4)
	m.Revision = s.Revision[:]
	for _, p := range s.Prefix6 {
		a := p.Addr().As16()
		m.Prefixes6 = append(m.Prefixes6, a[:8])
	}
	return m
}

// ParseSummary is the channel's facts as a Summary.
func ParseSummary(m *netv1.Facts) (Summary, error) {
	s := Summary{Watched: m.GetWatched()}
	if !s.Watched {
		return s, nil
	}
	class, err := Enum[Class4](int32(m.GetClass()), BehindRouter)
	if err != nil {
		return Summary{}, fmt.Errorf("class: %w", err)
	}
	s.Class = class
	if s.Public4, err = ParseAddr(m.GetPublic4()); err != nil {
		return Summary{}, fmt.Errorf("public4: %w", err)
	}
	for _, b := range m.GetPrefixes6() {
		if len(b) != 8 {
			return Summary{}, fmt.Errorf("prefix6 of %d bytes, want 8", len(b))
		}
		var a [16]byte
		copy(a[:], b)
		s.Prefix6 = append(s.Prefix6, netip.PrefixFrom(netip.AddrFrom16(a), 64))
	}
	if len(m.GetRevision()) != len(s.Revision) {
		return Summary{}, fmt.Errorf("revision of %d bytes, want %d", len(m.GetRevision()), len(s.Revision))
	}
	copy(s.Revision[:], m.GetRevision())
	return s, nil
}

// FactsOnWire is f as a Node sends it, without the addresses only the
// Controller keeps.
func FactsOnWire(f NetworkFacts) *netv1.NetworkFacts {
	m := &netv1.NetworkFacts{
		AtMs:    f.At.UnixMilli(),
		Watched: f.Watched,
		Egress:  &netv1.Egress{Iface: f.Egress.Iface, Local: addrBytes(f.Egress.Local)},
		Gateway: &netv1.Gateway{
			Addr:   addrBytes(f.Gateway.Addr),
			Pcp:    netv1.Tri(f.Gateway.Protocols.PCP),
			NatPmp: netv1.Tri(f.Gateway.Protocols.NATPMP),
			Upnp:   netv1.Tri(f.Gateway.Protocols.UPnP),
			Wan:    addrBytes(f.Gateway.WAN),
		},
	}
	for _, fw := range f.Gateway.Forwards {
		m.Gateway.Forwards = append(m.Gateway.Forwards, &netv1.Forward{External: uint32(fw.External), Client: addrBytes(fw.Client), Internal: uint32(fw.Internal)})
	}
	for _, i := range f.Interfaces {
		w := &netv1.Iface{Name: i.Name, Index: int32(i.Index), Tunnel: i.Tunnel} //nolint:gosec // G115: an interface index fits
		for _, a := range i.Addrs {
			w.Addrs = append(w.Addrs, &netv1.IfaceAddr{
				Addr: addrBytes(a.Prefix.Addr()), Bits: uint32(a.Prefix.Bits()), //nolint:gosec // G115: a valid prefix's bits fit
				Temporary: netv1.Tri(a.Temporary), Deprecated: netv1.Tri(a.Deprecated),
			})
		}
		m.Interfaces = append(m.Interfaces, w)
	}
	for _, s := range f.STUN {
		m.Stun = append(m.Stun, &netv1.STUNAnswer{Server: s.Server, MappedAddr: addrBytes(s.Mapped.Addr()), MappedPort: uint32(s.Mapped.Port()), Error: netv1.STUNError(s.Err)})
	}
	for _, l := range f.Listeners {
		m.Listeners = append(m.Listeners, &netv1.Listener{
			Port: uint32(l.Port), Program: l.Program.Name, Slipstream: l.Program.SlipStream,
			V4: netv1.Tri(l.V4), V6: netv1.Tri(l.V6), Loopback: l.Loopback, Docker: l.Docker,
		})
	}
	for _, r := range f.Firewall {
		m.Firewall = append(m.Firewall, &netv1.Rule{
			Tool: netv1.Tool(r.Tool), Port: uint32(r.Port), Family: FamilyOnWire(r.Family),
			Policy: netv1.Policy(r.Policy), Partial: r.Partial, Because: r.Because,
		})
	}
	for _, mp := range f.Mappings {
		w := &netv1.Mapping{
			Protocol: netv1.MapProtocol(mp.Protocol), External: uint32(mp.External), Internal: uint32(mp.Internal),
			LifetimeMs: mp.Lifetime.Milliseconds(), Refused: netv1.MapRefusal(mp.Refused),
		}
		if !mp.Expires.IsZero() {
			w.ExpiresAtMs = mp.Expires.UnixMilli()
		}
		m.Mappings = append(m.Mappings, w)
	}
	return m
}

// aheadOfNow is how far past the Controller's now a Node's observed time
// may lie. Clocks between Nodes differ; a time further ahead would keep its
// row against every later read, since a stored row is replaced only by a
// newer one.
const aheadOfNow = time.Hour

// ParseFacts is a Node's facts as the Controller judges them at now, under
// the id of the Node it asked: an id inside the message is never read.
func ParseFacts(node role.NodeID, m *netv1.NetworkFacts, r Rules, now time.Time) (NetworkFacts, error) {
	f := NetworkFacts{Node: node, At: time.UnixMilli(m.GetAtMs()), Watched: m.GetWatched()}
	var errs []error
	if m.GetAtMs() <= 0 || f.At.After(now.Add(aheadOfNow)) {
		errs = append(errs, fmt.Errorf("observed time %d is not a time this Node could have read", m.GetAtMs()))
	}
	fail := func(what string, err error) {
		if err != nil {
			errs = append(errs, fmt.Errorf("%s: %w", what, err))
		}
	}
	for _, w := range m.GetInterfaces() {
		i := Iface{Name: w.GetName(), Index: int(w.GetIndex()), Tunnel: w.GetTunnel()}
		for _, a := range w.GetAddrs() {
			addr, err := ParseAddr(a.GetAddr())
			fail("interface address", err)
			p, err := addr.Prefix(int(a.GetBits()))
			if err == nil {
				p = netip.PrefixFrom(addr, p.Bits())
			}
			fail("interface prefix", err)
			temp, err := Enum[Tri](int32(a.GetTemporary()), No)
			fail("temporary", err)
			dep, err := Enum[Tri](int32(a.GetDeprecated()), No)
			fail("deprecated", err)
			i.Addrs = append(i.Addrs, IfaceAddr{Prefix: p, Class: r.Of(addr), Temporary: temp, Deprecated: dep})
		}
		f.Interfaces = append(f.Interfaces, i)
	}
	var err error
	f.Egress.Iface = m.GetEgress().GetIface()
	f.Egress.Local, err = ParseAddr(m.GetEgress().GetLocal())
	fail("egress", err)
	gw := m.GetGateway()
	f.Gateway.Addr, err = ParseAddr(gw.GetAddr())
	fail("gateway", err)
	f.Gateway.WAN, err = ParseAddr(gw.GetWan())
	fail("gateway WAN", err)
	pcp, err1 := Enum[Tri](int32(gw.GetPcp()), No)
	natPMP, err2 := Enum[Tri](int32(gw.GetNatPmp()), No)
	upnp, err3 := Enum[Tri](int32(gw.GetUpnp()), No)
	fail("gateway protocols", errors.Join(err1, err2, err3))
	f.Gateway.Protocols = Protocols{PCP: pcp, NATPMP: natPMP, UPnP: upnp}
	for _, w := range gw.GetForwards() {
		client, err := ParseAddr(w.GetClient())
		fail("forward", err)
		ext, err1 := port(w.GetExternal())
		in, err2 := port(w.GetInternal())
		fail("forward port", errors.Join(err1, err2))
		f.Gateway.Forwards = append(f.Gateway.Forwards, Forward{External: ext, Client: client, Internal: in})
	}
	for _, w := range m.GetStun() {
		addr, err := ParseAddr(w.GetMappedAddr())
		fail("STUN answer", err)
		p, err := port(w.GetMappedPort())
		fail("STUN port", err)
		e, err := Enum[STUNError](int32(w.GetError()), STUNNoAnswer)
		fail("STUN error", err)
		a := STUNAnswer{Server: w.GetServer(), Err: e}
		if addr.IsValid() {
			a.Mapped = netip.AddrPortFrom(addr, p)
		}
		f.STUN = append(f.STUN, a)
	}
	for _, w := range m.GetListeners() {
		p, err := port(w.GetPort())
		fail("listener port", err)
		v4, err1 := Enum[Tri](int32(w.GetV4()), No)
		v6, err2 := Enum[Tri](int32(w.GetV6()), No)
		fail("listener family", errors.Join(err1, err2))
		f.Listeners = append(f.Listeners, Listener{
			Port: p, Program: Program{Name: w.GetProgram(), SlipStream: w.GetSlipstream()},
			V4: v4, V6: v6, Loopback: w.GetLoopback(), Docker: w.GetDocker(),
		})
	}
	for _, w := range m.GetFirewall() {
		p, err := port(w.GetPort())
		fail("rule port", err)
		tool, err := Enum[Tool](int32(w.GetTool()), UFW)
		fail("rule tool", err)
		policy, err := Enum[Policy](int32(w.GetPolicy()), Blocks)
		fail("rule policy", err)
		fam, err := ParseFamily(w.GetFamily())
		fail("rule family", err)
		f.Firewall = append(f.Firewall, Rule{Tool: tool, Port: p, Family: fam, Policy: policy, Partial: w.GetPartial(), Because: w.GetBecause()})
	}
	for _, w := range m.GetMappings() {
		ext, err1 := port(w.GetExternal())
		in, err2 := port(w.GetInternal())
		fail("mapping port", errors.Join(err1, err2))
		proto, err := Enum[MapProtocol](int32(w.GetProtocol()), MapUPnP)
		fail("mapping protocol", err)
		refused, err := Enum[MapRefusal](int32(w.GetRefused()), NoAnswer)
		fail("mapping refusal", err)
		mp := Mapping{Protocol: proto, External: ext, Internal: in, Lifetime: time.Duration(w.GetLifetimeMs()) * time.Millisecond, Refused: refused}
		if w.GetExpiresAtMs() != 0 {
			mp.Expires = time.UnixMilli(w.GetExpiresAtMs())
		}
		f.Mappings = append(f.Mappings, mp)
	}
	if err := errors.Join(errs...); err != nil {
		return NetworkFacts{}, err
	}
	return f, nil
}

// FamilyOnWire is f on the wire.
func FamilyOnWire(f Family) netv1.Family {
	if f == IPv6 {
		return netv1.Family_FAMILY_IPV6
	}
	return netv1.Family_FAMILY_IPV4
}

// ParseFamily is a wire family, which must be named.
func ParseFamily(f netv1.Family) (Family, error) {
	switch f {
	case netv1.Family_FAMILY_IPV4:
		return IPv4, nil
	case netv1.Family_FAMILY_IPV6:
		return IPv6, nil
	case netv1.Family_FAMILY_UNSPECIFIED:
	}
	return 0, fmt.Errorf("family %d", f)
}

func addrBytes(a netip.Addr) []byte {
	if !a.IsValid() {
		return nil
	}
	return a.AsSlice()
}

// ParseAddr is an address on the wire: 4 or 16 bytes, or none for no
// bytes.
func ParseAddr(b []byte) (netip.Addr, error) {
	if len(b) == 0 {
		return netip.Addr{}, nil
	}
	a, ok := netip.AddrFromSlice(b)
	if !ok {
		return netip.Addr{}, fmt.Errorf("address of %d bytes", len(b))
	}
	return a, nil
}

func port(p uint32) (uint16, error) {
	if p > 0xffff {
		return 0, fmt.Errorf("port %d", p)
	}
	return uint16(p), nil
}

// Enum is a wire value as T, refusing one past last.
func Enum[T ~uint8](v int32, last T) (T, error) {
	if v < 0 || v > int32(last) {
		return 0, fmt.Errorf("value %d out of range", v)
	}
	return T(v), nil
}
