package judgetest

import (
	"time"

	"github.com/jatassi/slipstream/internal/netmodel"
	"github.com/jatassi/slipstream/internal/reachability/judge"
)

var presets = []func() *B{Open, Mappable, SingleMachine, CGNAT, DoubleNAT, VPN, OwnProxy, PairedProxy, IPv6, OpenUI}

var classes = []netmodel.Class4{netmodel.Class4Unknown, netmodel.Open, netmodel.ThroughVPN, netmodel.CGNAT, netmodel.DoubleNAT, netmodel.Mappable, netmodel.BehindRouter}

var outcomes = []Outcome{Passed, Closed, Expired, WrongName, SignInPage, Something, NoResponse}

// ops each change one fact by an argument byte.
var ops = []func(b *B, x byte) *B{
	func(b *B, x byte) *B { return b.Class(classes[int(x)%len(classes)]) },
	func(b *B, x byte) *B { return b.Firewall(netmodel.Policy(x % 3)) },
	func(b *B, x byte) *B { return b.FirewallFor(netmodel.IPv6, netmodel.Policy(x%3)) },
	func(b *B, x byte) *B { return b.ServerListens(netmodel.Tri(x%3), netmodel.Tri(x/3%3)) },
	func(b *B, _ byte) *B { return b.Loopback() },
	func(b *B, _ byte) *B { return b.Docker() },
	func(b *B, _ byte) *B { return b.FirewalldPartial() },
	func(b *B, x byte) *B { return b.STUNVaries(x%2 == 0) },
	func(b *B, _ byte) *B { return b.Caddy() },
	func(b *B, x byte) *B { return b.Mapped(time.Duration(x%2) * time.Hour) },
	func(b *B, x byte) *B { return b.Refused(netmodel.MapRefusal(1 + x%3)) },
	func(b *B, x byte) *B {
		if b.f.Server == nil {
			return b
		}
		return b.PlexMapping([]string{"mapped", "unknown", "failed"}[x%3], x&4 != 0)
	},
	func(b *B, _ byte) *B { return b.Previous4("203.0.113.7") },
	func(b *B, _ byte) *B { return b.Previous6("2001:db8:1::/64") },
	func(b *B, _ byte) *B { return b.Pin("2001:db8:2::20") },
	func(b *B, _ byte) *B { return b.Churn() },
	func(b *B, _ byte) *B { return b.V6("2001:db8:1::20/64") },
	func(b *B, x byte) *B {
		return b.Resolves([][]string{
			{"203.0.113.9"}, {"203.0.113.7"}, {"198.51.100.8"}, {"192.168.1.20"}, {"2001:db8:1::20"},
			{"203.0.113.9", "2001:db8:1::20"}, {"203.0.113.9", "fd72::20"}, {"203.0.113.9", "198.51.100.99"},
			{"198.51.100.8", "2001:db8:1::20"}, {"203.0.113.50"}, {"192.0.2.44"},
		}[int(x)%11]...)
	},
	func(b *B, x byte) *B { return b.ResolvesAt(Atlas, []string{"192.168.1.20", "203.0.113.9"}[x%2]) },
	func(b *B, x byte) *B {
		return b.DNSError(netmodel.Family([]uint8{4, 6}[x%2]), judge.DNSError(1+x/2%4))
	},
	func(b *B, _ byte) *B { return b.Outside() },
	func(b *B, _ byte) *B { return b.Inside() },
	func(b *B, _ byte) *B { return b.Proxy() },
	func(b *B, _ byte) *B { return b.Away(Seedbox) },
	func(b *B, _ byte) *B { return b.Serves(Seedbox) },
	func(b *B, x byte) *B {
		return b.LinkFrom(Seedbox, []string{"192.0.2.44", "198.51.100.20", "100.101.7.3", "2001:db8:9::44"}[x%4])
	},
	func(b *B, _ byte) *B { return b.Reaching() },
	func(b *B, x byte) *B { return b.Tested(netmodel.IPv4, outcomes[int(x)%len(outcomes)]) },
	func(b *B, x byte) *B { return b.Tested(netmodel.IPv6, outcomes[int(x)%len(outcomes)]) },
	func(b *B, x byte) *B { return b.Echo(Arrival(x % 6)) },
	func(b *B, x byte) *B { return b.ConnectBack(x%2 == 0) },
	func(b *B, x byte) *B { return b.ProxyLeg(x%2 == 0) },
}

// Gen builds Facts from fuzz bytes: the first picks a preset, and each
// following pair picks an op and its argument.
func Gen(in []byte) judge.Facts {
	if len(in) == 0 {
		return Open().Facts()
	}
	b := presets[int(in[0])%len(presets)]()
	for i := 1; i+1 < len(in); i += 2 {
		b = ops[int(in[i])%len(ops)](b, in[i+1])
	}
	return b.Facts()
}

// Seeds are inputs for Gen that reach every op from every preset.
func Seeds() [][]byte {
	var out [][]byte
	for p := byte(0); int(p) < len(presets); p++ {
		for o := byte(0); int(o) < len(ops); o++ {
			out = append(out, []byte{p, o, o * 7, o + 1, p})
		}
	}
	return out
}
