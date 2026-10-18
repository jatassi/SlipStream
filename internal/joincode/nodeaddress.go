package joincode

import (
	"crypto/ed25519"
	"errors"
	"fmt"
	"net/netip"
	"slices"
)

const addressPrefix = "SLN1"

var (
	ErrAddressMistyped = errors.New("the Node address is mistyped or cut short")
	ErrNotNodeAddress  = errors.New("the text is not a SlipStream Node address")
)

// NodeAddress is an SLN1 Node address: a Node's full key and the
// addresses its Controller dials it at.
type NodeAddress struct {
	key       [ed25519.PublicKeySize]byte
	addresses []netip.AddrPort
}

// NewNodeAddress is the address of the Node holding key, reachable at
// addresses in dial order.
func NewNodeAddress(key ed25519.PublicKey, addresses []netip.AddrPort) (NodeAddress, error) {
	if len(key) != ed25519.PublicKeySize {
		return NodeAddress{}, fmt.Errorf("a Node key is %d bytes, not %d", ed25519.PublicKeySize, len(key))
	}
	if len(addresses) < 1 || len(addresses) > MaxAddresses {
		return NodeAddress{}, fmt.Errorf("a Node address holds 1 to %d addresses, not %d", MaxAddresses, len(addresses))
	}
	for _, a := range addresses {
		if !a.IsValid() || a.Addr().Zone() != "" {
			return NodeAddress{}, fmt.Errorf("address %s is not a literal IP and port", a)
		}
	}
	return NodeAddress{key: [ed25519.PublicKeySize]byte(key), addresses: slices.Clone(addresses)}, nil
}

// ParseNodeAddress reads an address as a pasted or retyped copy may
// arrive, as Parse reads a code.
func ParseNodeAddress(s string) (NodeAddress, error) {
	body, err := unpack(addressPrefix, s)
	switch {
	case errors.Is(err, ErrNotSlipStream):
		return NodeAddress{}, ErrNotNodeAddress
	case err != nil, len(body) < ed25519.PublicKeySize:
		return NodeAddress{}, ErrAddressMistyped
	}
	a := NodeAddress{key: [ed25519.PublicKeySize]byte(body[:ed25519.PublicKeySize])}
	var ok bool
	if a.addresses, ok = readAddresses(body[ed25519.PublicKeySize:]); !ok {
		return NodeAddress{}, ErrAddressMistyped
	}
	return a, nil
}

func (a NodeAddress) Key() ed25519.PublicKey { return slices.Clone(a.key[:]) }

func (a NodeAddress) Addresses() []netip.AddrPort { return slices.Clone(a.addresses) }

func (a NodeAddress) IsZero() bool { return len(a.addresses) == 0 }

func (a NodeAddress) String() string {
	if a.IsZero() {
		return ""
	}
	return pack(addressPrefix, appendAddresses(slices.Clone(a.key[:]), a.addresses))
}

func (a NodeAddress) MarshalText() ([]byte, error) { return []byte(a.String()), nil }

func (a *NodeAddress) UnmarshalText(text []byte) error {
	parsed, err := ParseNodeAddress(string(text))
	if err != nil {
		return err
	}
	*a = parsed
	return nil
}
