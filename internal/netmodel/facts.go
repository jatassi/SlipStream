package netmodel

import (
	"net/netip"
	"time"

	"github.com/jatassi/slipstream/internal/role"
)

// Tri is a fact a reader may not know. Its zero value is Unknown, so a field
// no reader filled never reads as Yes.
type Tri uint8

const (
	Unknown Tri = iota
	Yes
	No
)

// Policy is a firewall's answer for one port and family. Zero is Unknown.
type Policy uint8

const (
	PolicyUnknown Policy = iota
	Allows
	Blocks
)

type Family uint8

const (
	IPv4 Family = 4
	IPv6 Family = 6
)

func (f Family) String() string {
	if f == IPv6 {
		return "IPv6"
	}
	return "IPv4"
}

// Class4 is the IPv4 class, in the class rule's order. Zero is Unknown.
type Class4 uint8

const (
	Class4Unknown Class4 = iota
	Open
	ThroughVPN
	CGNAT
	DoubleNAT
	Mappable
	BehindRouter
)

type Class6 uint8

const (
	Class6None Class6 = iota
	LocalOnly
	Global6
)

type Iface struct {
	Name  string
	Index int
	// Tunnel is Linux's pointtopoint flag or netlink kind, Windows' IfType
	// 53, or a macOS utun.
	Tunnel bool
	Addrs  []IfaceAddr
}

type IfaceAddr struct {
	Prefix netip.Prefix
	Class  AddrClass
	// Temporary and Deprecated stay Unknown until S60 (IPv6 direct reach);
	// the scripted reader sets them.
	Temporary, Deprecated Tri
}

// Egress is the interface the STUN socket left by.
type Egress struct {
	Iface string // empty when no STUN server was asked
	Local netip.Addr
}

type Protocols struct{ PCP, NATPMP, UPnP Tri }

// Forward is a port forward the gateway lists over UPnP IGD.
type Forward struct {
	External uint16
	Client   netip.Addr
	Internal uint16
}

type Gateway struct {
	Addr      netip.Addr // the egress interface's gateway, zero when none
	Protocols Protocols
	WAN       netip.Addr // zero when hidden, empty, or PCP-only with mapping off
	Forwards  []Forward  // deduped across WANIP and WANPPP
}

type STUNError uint8

const (
	STUNNone STUNError = iota
	STUNTimeout
	STUNRefused
	STUNNoAnswer
)

type STUNAnswer struct {
	Server string // host:port as configured, never resolved
	Mapped netip.AddrPort
	Err    STUNError
}

// Program is the owner of a listening socket. An empty Name is an owner
// the reader couldn't see.
type Program struct {
	Name string
	// SlipStream is set by the Node on its own sockets.
	SlipStream bool
}

type Listener struct {
	Port     uint16
	Program  Program
	V4, V6   Tri  // listening on the family beyond loopback
	Loopback bool // bound to loopback addresses only
	// Docker is a port docker-proxy owns: the firewall can't say whether
	// it is allowed.
	Docker bool
}

// Family is whether the listener takes the family beyond loopback.
func (l Listener) Family(f Family) Tri {
	if f == IPv6 {
		return l.V6
	}
	return l.V4
}

type Tool uint8

const (
	ToolUnknown Tool = iota
	WindowsDefender
	SocketFilter
	Firewalld
	UFW
)

type Rule struct {
	Tool   Tool
	Port   uint16
	Family Family
	Policy Policy
	// Partial is firewalld allowing the port while other rules are unread.
	Partial bool
	Because string // the reader's raw reason, under Details
}

type MapProtocol uint8

const (
	MapPCP MapProtocol = iota + 1
	MapNATPMP
	MapUPnP
)

type MapRefusal uint8

const (
	RefusedNone MapRefusal = iota
	NotAuthorized
	OtherPort
	NoAnswer
)

type Mapping struct {
	Protocol MapProtocol
	External uint16
	Internal uint16
	Lifetime time.Duration // 0 is a permanent lease
	Expires  time.Time
	Refused  MapRefusal
}

// NetworkFacts is one Node's facts as the Controller stores and judges them.
type NetworkFacts struct {
	Node       role.NodeID
	At         time.Time
	Watched    bool // false: the Node read local facts only, with no STUN and no gateway
	Interfaces []Iface
	Egress     Egress
	Gateway    Gateway
	STUN       []STUNAnswer
	// Previous4 and Previous6 are set on the Controller by Carry from the
	// stored row, never by a Node.
	Previous4 netip.Addr
	Previous6 []netip.Prefix
	Listeners []Listener
	Firewall  []Rule
	Mappings  []Mapping // SlipStream's own, granted or refused
}

// Listener is the socket on port, and whether one was read.
func (f NetworkFacts) Listener(port uint16) (Listener, bool) {
	for _, l := range f.Listeners {
		if l.Port == port {
			return l, true
		}
	}
	return Listener{Port: port}, false
}

// Rule is the firewall's answer for port and family; the zero Rule is
// Unknown.
func (f NetworkFacts) Rule(port uint16, fam Family) Rule {
	for _, r := range f.Firewall {
		if r.Port == port && r.Family == fam {
			return r
		}
	}
	return Rule{Port: port, Family: fam}
}

// Holds is whether an interface of the Node holds a.
func (f NetworkFacts) Holds(a netip.Addr) bool {
	for _, i := range f.Interfaces {
		for _, ia := range i.Addrs {
			if ia.Prefix.Addr() == a {
				return true
			}
		}
	}
	return false
}

// OnLink is whether a lies in one of the Node's interface prefixes.
func (f NetworkFacts) OnLink(a netip.Addr) bool {
	for _, i := range f.Interfaces {
		for _, ia := range i.Addrs {
			if ia.Prefix.Contains(a) {
				return true
			}
		}
	}
	return false
}

// Observed is what Classify reads: the subset a summary also carries.
type Observed struct {
	Interfaces []Iface
	Egress     Egress
	Gateway    Gateway
	STUN       []STUNAnswer
}

func (f NetworkFacts) Observed() Observed {
	return Observed{Interfaces: f.Interfaces, Egress: f.Egress, Gateway: f.Gateway, STUN: f.STUN}
}
