// Package judge is the Reachability diagnostic's verdict: pure functions
// over the facts a run gathered, with no IO and no state. Place decides
// each family's route, Testers and Vet decide who may test and what, and
// Judge folds one check table into one verdict and the check it rests on.
// It speaks in CheckID, FixID and FactID with Params; reachability's
// words.go turns them into sentences.
package judge

import (
	"net/netip"
	"net/url"
	"strconv"
	"time"

	"github.com/jatassi/slipstream/internal/mediaserverlink/link"
	"github.com/jatassi/slipstream/internal/netmodel"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/storage"
)

type ExposureID struct{ storage.UUID }

// Kind is what an Exposure makes reachable. NamedPort waits for S52
// (Relay Node).
type Kind uint8

const (
	UIPortal Kind = iota
	MediaServer
)

type Exposure struct {
	ID     ExposureID
	Kind   Kind
	Server link.ServerID // MediaServer only
	Public *url.URL      // the UI's from Address, a Media Server's from its row
	// Local is the port the service listens on: http.listen's port, or the
	// Media Server's own.
	Local uint16
}

// PublicPort is the public URL's port, the scheme's default when it names
// none.
func (e Exposure) PublicPort() uint16 {
	if p, err := strconv.ParseUint(e.Public.Port(), 10, 16); err == nil {
		return uint16(p)
	}
	if e.Public.Scheme == "https" {
		return 443
	}
	return 80
}

// OriginLabel is how the origin was chosen.
type OriginLabel uint8

const (
	// Controller: the UI and portal whose A names no Serving Node.
	Controller OriginLabel = iota
	// Serving: the UI and portal's Serving Node whose address the A names.
	Serving
	// RunsOn: the Node a Media Server runs on.
	RunsOn
	// Reaching: a Media Server on no Node; its reaching Node's class
	// stands in, with no bind, firewall or mapping.
	Reaching
)

// Origin is the unit's choice before placement: the Controller for the UI
// and portal, a Media Server's runs-on or reaching Node. Serving lists the
// UI's Serving Nodes, any of which becomes the origin when the A names it.
type Origin struct {
	Node    role.NodeID
	Label   OriginLabel
	Serving []role.NodeID
}

type DNSError uint8

const (
	DNSNone DNSError = iota
	NXDomain
	ServFail
	DNSTimeout
	NoResolver
)

func (e DNSError) String() string {
	return [...]string{"", "NXDOMAIN", "SERVFAIL", "timeout", "no resolver"}[e]
}

// DNSAnswer is one Node's resolution of the public host.
type DNSAnswer struct {
	By            role.NodeID
	A, AAAA       []netip.Addr
	ErrA, ErrAAAA DNSError
}

// Candidate is a Node the unit may ask to test, with what the Controller
// knows of it.
type Candidate struct {
	Node     role.NodeID
	Name     string
	InStep   bool
	Relaying bool
	Self     bool // the Controller
	// Serves is a Worker that serves this Exposure: its origin, runs-on,
	// reaching or Serving Node.
	Serves bool
	Link   LinkSeen // zero for the Controller
}

// LinkSeen is the Node link's remote address as the Controller sees it.
// Dialed by the Worker it is the Worker's source after any NAT; dialed by
// the Controller it is the Worker's listen address from its kept record.
// Either way a Global one must match the Worker's claim.
type LinkSeen struct {
	Addr netip.Addr
}

type ServerFacts struct {
	Kind      link.KindName
	Name      string
	Mapping   *PlexMapping // Plex only
	Custom    []string     // customConnections, Jellyfin's published URL, Emby's WAN and remote addresses
	Published []Connection // plex.tv connections, relays included and marked
}

const Plex link.KindName = "plex"

type PlexMapping struct {
	State   string // raw mappingState; only "mapped" counts
	Error   string // mappingError; must be empty to count
	Pending bool   // reachabilityPending; true doesn't count
	Public  netip.AddrPort
}

// Counts is Plex's own check: mapped, with no error and nothing pending.
func (m *PlexMapping) Counts() bool {
	return m != nil && m.State == "mapped" && m.Error == "" && !m.Pending
}

type Connection struct {
	URL          string
	Addr         netip.Addr // zero when the host is a name
	Port         uint16
	Family       netmodel.Family
	Local, Relay bool
}

type DialKind uint8

const (
	PlayerDial DialKind = iota
	ConnectBackDial
	ProxyLegDial
)

// Stage is how far a connection got.
type Stage uint8

const (
	StageNone Stage = iota
	Resolved
	Connected
	TLSDone
	HTTPDone
)

type TLSProblem uint8

const (
	TLSNone TLSProblem = iota
	Expired
	WrongName
	Untrusted
	Handshake
)

func (p TLSProblem) String() string {
	return [...]string{"", "expired", "wrong-name", "untrusted", "handshake"}[p]
}

type TLSFacts struct {
	Problem  TLSProblem
	Names    []string
	NotAfter time.Time
}

// IdentitySeen is what the Controller parsed from the response body.
type IdentitySeen uint8

const (
	Unread IdentitySeen = iota
	Matches
	OtherIdentity
	SignIn
)

type Dial struct {
	Kind     DialKind
	Tester   role.NodeID
	Family   netmodel.Family
	Addr     netip.AddrPort // the address the tester connected to, or tried
	Stage    Stage
	TLS      *TLSFacts
	Identity IdentitySeen
	Echo     *Arrival // the UI and portal only: what the echo recorded, nil when nothing arrived
}

// Arrival is the echo's record of one fetch.
type Arrival struct {
	At     role.NodeID // the Node whose listener took it
	Via    role.NodeID // the Serving Node whose Serve stream carried it, zero for a bound listener
	Source netip.Addr  // the client after http.trusted_proxies
}

type Facts struct {
	Rules      netmodel.Rules
	Exposure   Exposure
	Origin     Origin
	DNS        []DNSAnswer
	Nodes      map[role.NodeID]netmodel.NetworkFacts
	Candidates []Candidate
	// Paired is every address a paired Node holds, for placement.
	Paired map[netip.Addr]role.NodeID
	Server *ServerFacts
	Dials  []Dial
	// Skipped is testers the unit tried that couldn't test: tester-away.
	Skipped []Fact
	Pins    map[role.NodeID]netip.Addr // Stable IPv6 address
	Churn   map[role.NodeID]bool
}
