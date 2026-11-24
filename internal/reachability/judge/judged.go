package judge

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/netip"
	"time"

	"github.com/jatassi/slipstream/internal/netmodel"
	"github.com/jatassi/slipstream/internal/role"
)

type RouteKind uint8

const (
	Direct RouteKind = iota
	OwnProxy
	ThroughRelay
	Unplaced
)

func (k RouteKind) String() string {
	return [...]string{"direct", "own-proxy", "through-relay", "unplaced"}[k]
}

// HopKind is a hop of a route; the order is the draw order.
type HopKind uint8

const (
	PlayerHop HopKind = iota
	DNSHop
	ProxyHop
	RelayHop
	TLSHop
	NodeLinkHop
	OriginHop
	MediaServerHop
)

func (k HopKind) String() string {
	return [...]string{"player", "dns", "proxy", "relay", "tls", "node-link", "origin", "media-server"}[k]
}

type Result uint8

const (
	Passes Result = iota
	Fails
	NotTested
	NotUsed
	NotReached
)

func (r Result) String() string {
	return [...]string{"passes", "fails", "not-tested", "not-used", "not-reached"}[r]
}

type Verdict uint8

const (
	Reachable Verdict = iota
	Fixable
	NeedsRelay
)

func (v Verdict) String() string { return [...]string{"reachable", "fixable", "needs-relay"}[v] }

// Params are a fix's or fact's typed values. Only the fields its id needs
// are set; words_test.go names them per id.
type Params struct {
	Node, Tester          role.NodeID
	Host                  string
	Port, LocalPort       uint16
	Addr, LocalAddr, Want netip.Addr
	Prefix                netip.Prefix
	Date                  time.Time
	Tool                  netmodel.Tool
	Class                 netmodel.Class4
	Program               string
	Family                *netmodel.Family
	URL                   string
	Raw                   string // a raw value shown under Details, such as Plex's mappingState
	Reason                string // a closed sub-kind, such as link-unconfirmed's private or other-family
}

type ActionKind uint8

const (
	PairNode ActionKind = iota + 1
	MapPorts
)

type Action struct {
	Kind ActionKind
	Node role.NodeID
}

type Fix struct {
	ID     FixID
	Params Params
	// Command is a firewall command words renders for Params.Tool.
	Command bool
	Action  *Action
}

type Fact struct {
	ID     FactID
	Family *netmodel.Family
	Params Params
}

// Value is an observed value, shown under Details. Text is an
// address, a host name, a date or a raw state, never a sentence.
type Value struct{ Text string }

type Check struct {
	ID     CheckID
	Family *netmodel.Family // nil for a check with no family
	Hop    HopKind
	Result Result
	// Fact is shown and never points: an own-proxy bind, an overruled local
	// guess, a connect-back.
	Fact     bool
	Points   Verdict // what a Fails or NotTested result points to
	Observed Value
	Fix      *Fix
	By       role.NodeID
}

type Hop struct {
	Kind   HopKind
	Worst  Result // the worst non-fact check; NotReached past a failure
	Checks []Check
	Facts  []Fact
}

type CheckRef struct {
	ID     CheckID
	Family *netmodel.Family
}

func (r CheckRef) matches(c Check) bool {
	return r.ID == c.ID && sameFamily(r.Family, c.Family)
}

func sameFamily(a, b *netmodel.Family) bool {
	return (a == nil) == (b == nil) && (a == nil || *a == *b)
}

type FamilyVerdict struct {
	Family  netmodel.Family
	Verdict Verdict
	Tested  bool
	Tester  role.NodeID // the Node whose test proved it, zero when Plex's check did
	ByPlex  bool
}

// Version is the Judged codec's version. A change to Judged's shape or
// meaning raises it, and a row of another version reads as not judged yet.
const Version = 1

type Judged struct {
	V        int
	Exposure ExposureID
	Placement
	Hops     []Hop
	Families []FamilyVerdict
	Verdict  Verdict
	Tested   bool      // every offered family was tested from outside
	Deciding *CheckRef // nil exactly when Verdict is Reachable and Tested
	Facts    []Fact    // for the Exposure: ipv6-viewers-direct
	// Forward is what "Open router ports automatically" maps on the
	// origin's router for this route; nil when mapping can't serve it.
	Forward *PortForward
}

// PortForward is a TCP forward on a router, from External on its internet
// side to Internal on the origin.
type PortForward struct{ External, Internal uint16 }

// Check is the check ref names; the zero Check when there is none.
func (j Judged) Check(ref CheckRef) Check {
	for _, h := range j.Hops {
		for _, c := range h.Checks {
			if ref.matches(c) {
				return c
			}
		}
	}
	return Check{}
}

// Fix is the deciding check's. Fix, Card and Failing all read the deciding
// check, so the chip, the card, the step and Health can't disagree.
func (j Judged) Fix() *Fix {
	if j.Deciding == nil {
		return nil
	}
	return j.Check(*j.Deciding).Fix
}

type Card uint8

const (
	NoCard Card = iota
	// Untested is the neutral line: the verdict rests on Not tested.
	Untested
	// Warn is Fixable resting on a failing check.
	Warn
	// Fail is Needs a relay resting on a failing check.
	Fail
)

func (c Card) String() string { return [...]string{"none", "untested", "warn", "fail"}[c] }

func (j Judged) Card() Card {
	switch {
	case j.Deciding == nil:
		return NoCard
	case !j.Failing():
		return Untested
	case j.Verdict == NeedsRelay:
		return Fail
	}
	return Warn
}

// Failing is the deciding check failing. Not tested never fails.
func (j Judged) Failing() bool {
	return j.Deciding != nil && j.Check(*j.Deciding).Result == Fails
}

// Folded is Reachable and tested: the route shows as one line.
func (j Judged) Folded() bool { return j.Verdict == Reachable && j.Tested }

var ErrOtherVersion = errors.New("judged by another version")

func Encode(j Judged) ([]byte, error) {
	j.V = Version
	b, err := json.Marshal(j)
	if err != nil {
		return nil, fmt.Errorf("encode the judged route: %w", err)
	}
	return b, nil
}

func Decode(b []byte) (Judged, error) {
	var j Judged
	if err := json.Unmarshal(b, &j); err != nil {
		return Judged{}, fmt.Errorf("decode the judged route: %w", err)
	}
	if j.V != Version {
		return Judged{}, ErrOtherVersion
	}
	return j, nil
}
