// Package joincode is the Join Code an Admin mints on the Controller and a
// Worker joins with: format SLP1, frozen, since any Worker must read any
// Controller's code. A later format takes SLP2 and every binary keeps
// reading SLP1.
//
// Before encoding, an SLP1 code is
//
//	16       SHA-256 of the Controller's raw Ed25519 public key, first 16 bytes
//	16       secret
//	1        address count, 1 to 8
//	7 or 19  per address: family (4 or 6), 4 or 16 IP bytes, big-endian port
//	4        SHA-256 of every byte before it, first 4 bytes
//
// Its text is "SLP1-" and upper-case unpadded Crockford base32 in
// dash-joined groups of five.
package joincode

import (
	"bytes"
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/base32"
	"encoding/binary"
	"errors"
	"fmt"
	"log/slog"
	"net/netip"
	"strings"
	"unicode"
)

const (
	prefix       = "SLP1"
	hashSize     = 16
	SecretSize   = 16
	MaxAddresses = 8
	checksumSize = 4
	groupSize    = 5
)

var (
	ErrMistyped      = errors.New("the Join Code is mistyped or cut short")
	ErrNotSlipStream = errors.New("the Join Code is not a SlipStream code")
)

var crockford = base32.NewEncoding("0123456789ABCDEFGHJKMNPQRSTVWXYZ").WithPadding(base32.NoPadding)

// Code is an SLP1 Join Code. It carries no expiry: the Controller enforces
// that from its own row. The zero Code is the unset config value.
type Code struct {
	keyHash   [hashSize]byte
	secret    [SecretSize]byte
	addresses []netip.AddrPort
}

// New mints the code naming controller, reachable at addresses in dial
// order.
func New(controller ed25519.PublicKey, secret [SecretSize]byte, addresses []netip.AddrPort) (Code, error) {
	if len(addresses) < 1 || len(addresses) > MaxAddresses {
		return Code{}, fmt.Errorf("a Join Code holds 1 to %d addresses, not %d", MaxAddresses, len(addresses))
	}
	for _, a := range addresses {
		if !a.IsValid() || a.Addr().Zone() != "" {
			return Code{}, fmt.Errorf("address %s is not a literal IP and port", a)
		}
	}
	return Code{keyHash: keyHash(controller), secret: secret, addresses: append([]netip.AddrPort(nil), addresses...)}, nil
}

func keyHash(key ed25519.PublicKey) [hashSize]byte {
	sum := sha256.Sum256(key)
	return [hashSize]byte(sum[:hashSize])
}

// Names reports whether key is the Controller this code was minted by.
func (c Code) Names(key ed25519.PublicKey) bool {
	return c.keyHash == keyHash(key)
}

func (c Code) Secret() [SecretSize]byte { return c.secret }

func (c Code) Addresses() []netip.AddrPort {
	return append([]netip.AddrPort(nil), c.addresses...)
}

func (c Code) IsZero() bool { return len(c.addresses) == 0 }

func (c Code) String() string {
	if c.IsZero() {
		return ""
	}
	b := append(c.keyHash[:], c.secret[:]...)
	b = append(b, byte(len(c.addresses))) //nolint:gosec // New caps it at MaxAddresses
	for _, a := range c.addresses {
		ip := a.Addr()
		if ip.Is4() {
			b = append(b, 4)
		} else {
			b = append(b, 6)
		}
		b = append(b, ip.AsSlice()...)
		b = binary.BigEndian.AppendUint16(b, a.Port())
	}
	sum := sha256.Sum256(b)
	text := crockford.EncodeToString(append(b, sum[:checksumSize]...))
	groups := []string{prefix}
	for len(text) > groupSize {
		groups = append(groups, text[:groupSize])
		text = text[groupSize:]
	}
	return strings.Join(append(groups, text), "-")
}

// Redacted is how a config leaf holding a code shows it, so no log or
// `slipstream config` output carries the secret.
func (c Code) Redacted() string {
	if c.IsZero() {
		return ""
	}
	return prefix + "-[redacted]"
}

// LogValue is the code as every log record shows it: Redacted.
func (c Code) LogValue() slog.Value { return slog.StringValue(c.Redacted()) }

// Parse reads a code as an emailed or retyped copy may arrive: ignoring
// case, whitespace and dashes after the prefix, and reading O as 0 and I
// and L as 1.
func Parse(s string) (Code, error) {
	s = strings.TrimSpace(s)
	if len(s) < len(prefix) || !strings.EqualFold(s[:len(prefix)], prefix) {
		return Code{}, ErrNotSlipStream
	}
	text := strings.Map(func(r rune) rune {
		switch {
		case r == '-' || unicode.IsSpace(r):
			return -1
		case r == 'O':
			return '0'
		case r == 'I' || r == 'L':
			return '1'
		}
		return r
	}, strings.ToUpper(s[len(prefix):]))
	b, err := crockford.DecodeString(text)
	// A changed last character can differ only in bits past the last byte,
	// so the text must also be the one its bytes encode to.
	if err != nil || crockford.EncodeToString(b) != text || len(b) < checksumSize {
		return Code{}, ErrMistyped
	}
	body, sum := b[:len(b)-checksumSize], b[len(b)-checksumSize:]
	if want := sha256.Sum256(body); !bytes.Equal(sum, want[:checksumSize]) {
		return Code{}, ErrMistyped
	}
	return decode(body)
}

func decode(b []byte) (Code, error) {
	if len(b) < hashSize+SecretSize+1 {
		return Code{}, ErrMistyped
	}
	c := Code{keyHash: [hashSize]byte(b[:hashSize]), secret: [SecretSize]byte(b[hashSize : hashSize+SecretSize])}
	n := int(b[hashSize+SecretSize])
	rest := b[hashSize+SecretSize+1:]
	if n < 1 || n > MaxAddresses {
		return Code{}, ErrMistyped
	}
	for range n {
		if len(rest) < 1 {
			return Code{}, ErrMistyped
		}
		size := map[byte]int{4: 4, 6: 16}[rest[0]]
		if size == 0 || len(rest) < 1+size+2 {
			return Code{}, ErrMistyped
		}
		ip, _ := netip.AddrFromSlice(rest[1 : 1+size])
		c.addresses = append(c.addresses, netip.AddrPortFrom(ip, binary.BigEndian.Uint16(rest[1+size:])))
		rest = rest[1+size+2:]
	}
	if len(rest) != 0 {
		return Code{}, ErrMistyped
	}
	return c, nil
}

func (c Code) MarshalText() ([]byte, error) { return []byte(c.String()), nil }

// UnmarshalText reads empty text as the zero Code, the unset config value.
func (c *Code) UnmarshalText(text []byte) error {
	if len(bytes.TrimSpace(text)) == 0 {
		*c = Code{}
		return nil
	}
	parsed, err := Parse(string(text))
	if err != nil {
		return err
	}
	*c = parsed
	return nil
}
