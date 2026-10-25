package link

import (
	"errors"
	"fmt"
	"net/netip"

	linkv1 "github.com/jatassi/slipstream/internal/pb/slipstream/link/v1"
)

// Trust is what this Node admits a peer key as. The link never sees Node
// records or pairing files, only this answer.
type Trust interface {
	Admit(key Key) Admission
}

// Admission is one of Refused, Pinned, Pairing, Superseded or Removed.
type Admission interface{ admission() }

type (
	// Refused fails the TLS handshake.
	Refused struct{}
	// Pinned may say Hello and, once in step, call everything.
	Pinned struct{}
	// Pairing comes only from a Controller, and the session may call only
	// Pair.
	Pairing struct{}
	// Superseded is an old Controller's key: its Hello is answered with
	// the key that replaced it.
	Superseded struct {
		By        Key
		Addresses Addresses
	}
	// Removed is a key a Controller removed: its Hello is refused as
	// removed, and it may Pair again.
	Removed struct{}
)

func (Refused) admission()    {}
func (Pinned) admission()     {}
func (Pairing) admission()    {}
func (Superseded) admission() {}
func (Removed) admission()    {}

// State is Down or Live.
type State interface{ state() }

type (
	// Down is no live session. Refused is a kept peer's refusal of the
	// last Hello this Node sent it, zero when it answered otherwise or
	// was never reached.
	Down struct {
		Refused linkv1.RefusedReason
	}
	Live struct {
		Peer NodeInfo
		// Matched is false when the peer runs another version, so only
		// slipstream.link.v1 is served either way.
		Matched       bool
		Local, Remote netip.AddrPort
		// Dialed is true on the side that dialed the session.
		Dialed bool
	}
)

func (Down) state() {}
func (Live) state() {}

type Change struct {
	Key   Key
	State State
}

type Target struct {
	Key       Key
	Addresses Addresses
}

// NodeInfo is a Node's build and advertised addresses, as Hello carries it.
type NodeInfo struct {
	Version     string
	OS          string
	Arch        string
	InstallKind linkv1.InstallKind
	// Addresses is zero when the Node advertises none.
	Addresses Addresses
}

func (n NodeInfo) proto() *linkv1.NodeInfo {
	return &linkv1.NodeInfo{
		Version:     n.Version,
		Os:          n.OS,
		Arch:        n.Arch,
		InstallKind: n.InstallKind,
		Addresses:   n.Addresses.Proto(),
	}
}

func parseNodeInfo(p *linkv1.NodeInfo) (NodeInfo, error) {
	addrs, err := AddressesFromProto(p.GetAddresses())
	if err != nil {
		return NodeInfo{}, fmt.Errorf("peer's addresses: %w", err)
	}
	return NodeInfo{
		Version:     p.GetVersion(),
		OS:          p.GetOs(),
		Arch:        p.GetArch(),
		InstallKind: p.GetInstallKind(),
		Addresses:   addrs,
	}, nil
}

var (
	// ErrDisconnected is a call refused before sending: no live session.
	ErrDisconnected = errors.New("no live link session to the peer")
	// ErrMismatched is a lockstep call refused before sending: the peer
	// runs another version.
	ErrMismatched = errors.New("the peer runs another version, so lockstep services are closed")
	// ErrKeyMismatch is a Pair whose peer's key is not the Join Code's.
	ErrKeyMismatch = errors.New("the Controller's key doesn't match the Join Code")
)

// RefusedError is a peer's Refused answer to Pair or Hello.
type RefusedError struct {
	Reason linkv1.RefusedReason
}

func (e *RefusedError) Error() string { return "the peer refused: " + e.Reason.String() }

// SupersededError is a Hello answered with the key that replaced this
// Node's.
type SupersededError struct {
	By        Key
	Addresses Addresses
}

func (e *SupersededError) Error() string {
	return "the peer reported that key " + e.By.Fingerprint() + " superseded this Node"
}
