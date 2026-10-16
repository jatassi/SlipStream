// Package passkey is SlipStream's only importer of go-webauthn
// (webauthn-confined). It decides whether an address may hold a passkey and
// runs ceremonies over its own types; it stores nothing but challenges in
// flight.
package passkey

import (
	"net"
	"net/netip"
	"strings"

	"github.com/go-webauthn/webauthn/protocol"
)

// Reason is why an address can't hold a passkey.
type Reason string

const (
	// IPAddress is a host that is an IP literal, which is never an RP ID
	// (ADR 0005, #27 Q37).
	IPAddress Reason = "ip_address"
	// PlainHTTP is a domain name over HTTP that isn't localhost or
	// *.localhost, which browsers refuse as an insecure context.
	PlainHTTP Reason = "plain_http"
	// NotADomain is a host WebAuthn can't take as an RP ID: one label other
	// than localhost, a numeric last label, an empty label, non-ASCII.
	NotADomain Reason = "not_a_domain"
)

// Verdict is whether an address can hold a passkey, and why not.
type Verdict struct {
	Available bool
	Reason    Reason
}

// RP is the relying party a ceremony runs as: the RP ID is the host and the
// origin is scheme://host[:port]. Only Decide builds a usable one.
type RP struct {
	id, origin string
}

func (rp RP) ID() string { return rp.id }

func (rp RP) Origin() string { return rp.origin }

// Decide is the one rule for where a passkey works. host is the request's
// Host, port optional; https is whether the client reached SlipStream over
// TLS (after trusted proxies). The host must be a domain name, and the
// scheme https or the host localhost or a name under .localhost. The RP is
// the zero RP unless the verdict is available.
func Decide(host string, https bool) (RP, Verdict) {
	name := strings.ToLower(hostname(host))
	if _, err := netip.ParseAddr(strings.TrimSuffix(strings.TrimPrefix(name, "["), "]")); err == nil {
		return RP{}, Verdict{Reason: IPAddress}
	}
	if protocol.ValidateRPID(name) != nil {
		return RP{}, Verdict{Reason: NotADomain}
	}
	scheme := "https"
	if !https {
		if name != "localhost" && !strings.HasSuffix(name, ".localhost") {
			return RP{}, Verdict{Reason: PlainHTTP}
		}
		scheme = "http"
	}
	return RP{id: name, origin: scheme + "://" + strings.ToLower(host)}, Verdict{Available: true}
}

func hostname(host string) string {
	if h, _, err := net.SplitHostPort(host); err == nil {
		return h
	}
	return host
}

// Credential is a stored passkey as the ceremonies need it.
type Credential struct {
	ID             []byte
	PublicKey      []byte
	SignCount      uint32
	AAGUID         [16]byte
	Transports     []string
	BackupEligible bool
	BackupState    bool
}

// User is the Account a ceremony is for. Handle is its id's bytes, and
// Credentials are its passkeys for the ceremony's RP ID only.
type User struct {
	Handle      []byte
	Name        string
	DisplayName string
	Credentials []Credential
}

// Asserted is a verified assertion: whose it was, and the credential with
// the sign count and backup state to save.
type Asserted struct {
	Handle     []byte
	Credential Credential
}
