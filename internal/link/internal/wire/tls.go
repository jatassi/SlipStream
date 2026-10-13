// Package wire is the link's frozen framing above TCP: TLS 1.3 with a
// self-signed Ed25519 certificate and a raw-key pin check, the one-byte
// stream kind, and unencrypted HTTP/2 with prior knowledge.
package wire

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"math/big"
	"time"
)

// Certificate mints key's self-signed certificate. It lives only in memory:
// peers pin the raw key, so it may change at every start without a re-pair.
func Certificate(key ed25519.PrivateKey) (tls.Certificate, error) {
	tmpl := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		NotBefore:    time.Unix(0, 0),
		NotAfter:     time.Date(9999, 12, 31, 23, 59, 59, 0, time.UTC),
	}
	der, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, key.Public(), key)
	if err != nil {
		return tls.Certificate{}, fmt.Errorf("mint link certificate: %w", err)
	}
	return tls.Certificate{Certificate: [][]byte{der}, PrivateKey: key}, nil
}

// Accept decides whether a peer holding key may finish the handshake.
type Accept func(key ed25519.PublicKey) error

// ClientConfig is the dialing side's TLS: the server's key must pass accept.
func ClientConfig(cert tls.Certificate, accept Accept) *tls.Config {
	return &tls.Config{
		Certificates:       []tls.Certificate{cert},
		MinVersion:         tls.VersionTLS13,
		InsecureSkipVerify: true, //nolint:gosec // the pin check replaces chain verification
		VerifyConnection:   verify(accept),
	}
}

// ServerConfig is the accepting side's TLS: the client must present a
// certificate and its key must pass accept.
func ServerConfig(cert tls.Certificate, accept Accept) *tls.Config {
	return &tls.Config{
		Certificates:     []tls.Certificate{cert},
		MinVersion:       tls.VersionTLS13,
		ClientAuth:       tls.RequireAnyClientCert,
		VerifyConnection: verify(accept),
	}
}

// verify runs on every handshake, resumed ones included, so a session
// ticket never skips the pin check. It ignores expiry, names and chain.
func verify(accept Accept) func(tls.ConnectionState) error {
	return func(cs tls.ConnectionState) error {
		key, err := PeerKey(cs)
		if err != nil {
			return err
		}
		return accept(key)
	}
}

// PeerKey is the raw Ed25519 key of the one certificate a peer presented.
func PeerKey(cs tls.ConnectionState) (ed25519.PublicKey, error) {
	if n := len(cs.PeerCertificates); n != 1 {
		return nil, fmt.Errorf("peer presented %d certificates; the link pins exactly one", n)
	}
	pub := cs.PeerCertificates[0].PublicKey
	key, ok := pub.(ed25519.PublicKey)
	if !ok {
		return nil, fmt.Errorf("peer certificate holds a %T; the link pins Ed25519 keys", pub)
	}
	return key, nil
}
