package host

import (
	"cmp"
	"fmt"
	"net"
	"net/netip"
	"strconv"
	"strings"

	"github.com/jatassi/slipstream/internal/netmodel"
)

// fwProfile is one of Windows Defender Firewall's network profiles, as its
// NET_FW_PROFILE2 bit.
type fwProfile int32

const (
	fwDomain  fwProfile = 1
	fwPrivate fwProfile = 2
	fwPublic  fwProfile = 4
)

var profileNames = map[fwProfile]string{fwDomain: "Domain networks", fwPrivate: "Private networks", fwPublic: "Public networks"}

func (p fwProfile) String() string { return profileNames[p] }

// fwProfileState is one profile's switches.
type fwProfileState struct {
	Profile      fwProfile
	On           bool
	BlockAll     bool
	DefaultAllow bool
}

// NET_FW_IP_PROTOCOL values.
const (
	fwTCP         = 6
	fwAnyProtocol = 256
)

// fwRule is one INetFwRule, cut to what decides inbound TCP to a port.
type fwRule struct {
	Name                    string
	Enabled, Inbound, Allow bool
	Profiles                fwProfile
	Protocol                int32
	LocalPorts              string
	RemoteAddresses         string
	// Program is the rule's ApplicationName with its environment
	// variables expanded.
	Program string
	Service string
	// Packaged is a rule with LocalAppPackageId or LocalUserOwner set.
	Packaged bool
}

// fwAsk is inbound TCP to Port on Family, to the program that listens
// there, empty when it couldn't be read.
type fwAsk struct {
	Port    uint16
	Family  netmodel.Family
	Program string
}

// reach is how far a rule's conditions cover an inbound connection from
// the internet, in increasing order.
type reach uint8

const (
	misses reach = iota
	mayHit
	hits
)

// fwDecide is Windows Defender Firewall's answer for a under every one of
// profiles: block over allow, then the default. Profiles that disagree
// read Unknown.
func fwDecide(profiles []fwProfileState, rules []fwRule, a fwAsk) netmodel.Rule {
	out := netmodel.Rule{Tool: netmodel.WindowsDefender, Port: a.Port, Family: a.Family}
	allowsIn, othersIn := fwProfile(0), fwProfile(0)
	for i, p := range profiles {
		policy, because := decideIn(p, rules, a)
		if policy == netmodel.Allows {
			allowsIn = cmp.Or(allowsIn, p.Profile)
		} else {
			othersIn = cmp.Or(othersIn, p.Profile)
		}
		if i == 0 {
			out.Policy, out.Because = policy, because
		} else if policy != out.Policy {
			out.Policy = netmodel.PolicyUnknown
		}
	}
	if out.Policy == netmodel.PolicyUnknown && allowsIn != 0 && othersIn != 0 {
		out.Because = fmt.Sprintf("Windows Defender Firewall allows port %d on %s and doesn't on %s.", a.Port, allowsIn, othersIn)
	}
	return out
}

func decideIn(p fwProfileState, rules []fwRule, a fwAsk) (policy netmodel.Policy, because string) {
	switch {
	case !p.On:
		return netmodel.Allows, fmt.Sprintf("Windows Defender Firewall is off for %s.", p.Profile)
	case p.BlockAll:
		return netmodel.Blocks, fmt.Sprintf(`"Block all incoming connections" is on for %s.`, p.Profile)
	}
	var allow, mayAllow, mayBlock string
	for _, r := range rules {
		if !r.Enabled || !r.Inbound || r.Profiles&p.Profile == 0 {
			continue
		}
		switch reach := r.reaches(a); {
		case reach == hits && !r.Allow:
			return netmodel.Blocks, fmt.Sprintf(`The rule %q blocks port %d.`, r.Name, a.Port)
		case reach == hits:
			allow = cmp.Or(allow, r.Name)
		case reach == mayHit && r.Allow:
			mayAllow = cmp.Or(mayAllow, r.Name)
		case reach == mayHit:
			mayBlock = cmp.Or(mayBlock, r.Name)
		}
	}
	switch {
	case mayBlock != "":
		return netmodel.PolicyUnknown, fmt.Sprintf(`SlipStream can't tell whether the rule %q blocks port %d.`, mayBlock, a.Port)
	case allow != "":
		return netmodel.Allows, fmt.Sprintf(`The rule %q allows port %d.`, allow, a.Port)
	case p.DefaultAllow:
		return netmodel.Allows, fmt.Sprintf("%s allow incoming connections by default.", p.Profile)
	case mayAllow != "":
		return netmodel.PolicyUnknown, fmt.Sprintf(`SlipStream can't tell whether the rule %q allows port %d.`, mayAllow, a.Port)
	}
	return netmodel.Blocks, fmt.Sprintf("No rule allows port %d, and %s block incoming connections by default.", a.Port, p.Profile)
}

// reaches is how far r covers a. A packaged rule with no program, service
// or port limit belongs to a Store or system app and covers no other
// program. A rule limited to a service may hit, since the service that
// owns a socket isn't read.
func (r fwRule) reaches(a fwAsk) reach {
	if r.Packaged && r.Program == "" && r.Service == "" && anyPort(r.LocalPorts) {
		return misses
	}
	if r.Protocol != fwTCP && r.Protocol != fwAnyProtocol || !portIn(r.LocalPorts, a.Port) {
		return misses
	}
	who := hits
	switch {
	case r.Program == "":
	case a.Program == "":
		who = mayHit
	case !strings.EqualFold(r.Program, a.Program):
		return misses
	}
	if r.Service != "" {
		who = mayHit
	}
	return min(who, remoteReach(r.RemoteAddresses, a.Family))
}

func anyPort(spec string) bool { return spec == "" || spec == "*" }

// portIn is whether a LocalPorts list names port. Keywords such as RPC
// name no port a listener here has.
func portIn(spec string, port uint16) bool {
	if anyPort(spec) {
		return true
	}
	for item := range strings.SplitSeq(spec, ",") {
		lo, hi, isRange := strings.Cut(strings.TrimSpace(item), "-")
		if !isRange {
			hi = lo
		}
		l, err1 := strconv.ParseUint(lo, 10, 16)
		h, err2 := strconv.ParseUint(hi, 10, 16)
		if err1 == nil && err2 == nil && uint64(port) >= l && uint64(port) <= h {
			return true
		}
	}
	return false
}

// remoteReach is how far a RemoteAddresses list covers the internet on
// fam: all of it for * or Internet, some of it for a list that names
// global addresses, none for LocalSubnet, the other keywords and private
// ranges.
func remoteReach(spec string, fam netmodel.Family) reach {
	if anyPort(spec) {
		return hits
	}
	out := misses
	for item := range strings.SplitSeq(spec, ",") {
		item = strings.TrimSpace(item)
		if strings.EqualFold(item, "Internet") {
			return hits
		}
		first, last, ok := addrRange(item)
		if !ok || (fam == netmodel.IPv6) != first.Is6() {
			continue
		}
		if first.IsUnspecified() && isLast(last) {
			return hits
		}
		if global(first) || global(last) {
			out = mayHit
		}
	}
	return out
}

// addrRange parses one remote address entry: an address, a prefix with
// a length or a mask, or a range.
func addrRange(item string) (first, last netip.Addr, ok bool) {
	if lo, hi, isRange := strings.Cut(item, "-"); isRange {
		var err1, err2 error
		first, err1 = netip.ParseAddr(lo)
		last, err2 = netip.ParseAddr(hi)
		return first, last, err1 == nil && err2 == nil
	}
	at, mask, isPrefix := strings.Cut(item, "/")
	first, err := netip.ParseAddr(at)
	if err != nil || !isPrefix {
		return first, first, err == nil
	}
	bits, err := strconv.Atoi(mask)
	if err != nil {
		m, perr := netip.ParseAddr(mask)
		if perr != nil {
			return first, first, false
		}
		bits, _ = net.IPMask(m.AsSlice()).Size()
	}
	p, err := first.Prefix(bits)
	if err != nil {
		return first, first, false
	}
	return p.Addr(), lastIn(p), true
}

func lastIn(p netip.Prefix) netip.Addr {
	b := p.Addr().AsSlice()
	for i := p.Bits(); i < len(b)*8; i++ {
		b[i/8] |= 0x80 >> (i % 8)
	}
	a, _ := netip.AddrFromSlice(b)
	return a
}

func isLast(a netip.Addr) bool { return !a.Next().IsValid() }

func global(a netip.Addr) bool { return a.IsGlobalUnicast() && !a.IsPrivate() }
