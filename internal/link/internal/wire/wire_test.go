package wire_test

import (
	"bytes"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"io"
	"math/big"
	"net"
	"net/netip"
	"strings"
	"testing"
	"testing/synctest"

	"github.com/jatassi/slipstream/fakes/fault"
	"github.com/jatassi/slipstream/internal/link/internal/wire"
)

func ed25519Certificate(t *testing.T) (tls.Certificate, ed25519.PublicKey) {
	t.Helper()
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	cert, err := wire.Certificate(priv)
	if err != nil {
		t.Fatal(err)
	}
	return cert, pub
}

func rsaCertificate(t *testing.T) tls.Certificate {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	tmpl := &x509.Certificate{SerialNumber: big.NewInt(1)}
	der, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &key.PublicKey, key)
	if err != nil {
		t.Fatal(err)
	}
	return tls.Certificate{Certificate: [][]byte{der}, PrivateKey: key}
}

func anyKey(ed25519.PublicKey, string) error { return nil }

var lane = wire.LaneALPN("0.0.0-test")

// The release build stamps the Node's version with git describe: the
// release tag, vX.Y.Z with an optional pre-release, then for a commit
// past the tag the commits since it and the commit's hash, at most
// SHA-256's 64 hex digits. An ALPN protocol name holds at most 255 bytes,
// and Go refuses to offer a longer one.
func TestTheLongestReleaseBuildNamesALane(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		build := "v65535.65535.65535-beta.65535-4294967295-g" + strings.Repeat("f", 64)
		longest := wire.LaneALPN(build)
		clientCert, _ := ed25519Certificate(t)
		serverCert, _ := ed25519Certificate(t)
		state, clientErr, serverErr := handshake(t, wire.LaneConfig(clientCert, longest, anyKey), wire.ServerConfig(serverCert, longest, anyKey))
		if clientErr != nil || serverErr != nil || state.NegotiatedProtocol != longest {
			t.Fatalf("a lane of a %d-byte protocol name negotiated %q: %v, %v", len(longest), state.NegotiatedProtocol, clientErr, serverErr)
		}
	})
}

// conns dials 127.0.0.3 from 127.0.0.2 over an in-memory network.
func conns(t *testing.T) (client, server net.Conn) {
	t.Helper()
	n := fault.NewNetwork()
	l, err := n.Host(netip.MustParseAddr("127.0.0.3")).Listen("tcp", "127.0.0.3:8485")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = l.Close() }()
	client, err = n.Host(netip.MustParseAddr("127.0.0.2")).DialContext(t.Context(), "tcp", "127.0.0.3:8485")
	if err != nil {
		t.Fatal(err)
	}
	if server, err = l.Accept(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = client.Close(); _ = server.Close() })
	return client, server
}

// handshake runs one TLS handshake and reports each side's error. The
// server writes a byte once its handshake succeeds, and the client reads
// it, so a client certificate the server refuses after the client's
// handshake finished still reaches the client as an alert.
func handshake(t *testing.T, clientCfg, serverCfg *tls.Config) (clientState tls.ConnectionState, clientErr, serverErr error) {
	t.Helper()
	c, s := conns(t)
	done := make(chan error, 1)
	go func() {
		st := tls.Server(s, serverCfg)
		err := st.HandshakeContext(t.Context())
		if err == nil {
			_, err = st.Write([]byte("k"))
		}
		done <- err
	}()
	ct := tls.Client(c, clientCfg)
	clientErr = ct.HandshakeContext(t.Context())
	if clientErr == nil {
		_, clientErr = io.ReadFull(ct, make([]byte, 1))
	} else {
		_ = c.Close()
	}
	return ct.ConnectionState(), clientErr, <-done
}

func errText(err error) string {
	if err == nil {
		return "<nil>"
	}
	return err.Error()
}

func TestPinCheckRefusals(t *testing.T) {
	good, _ := ed25519Certificate(t)
	other, _ := ed25519Certificate(t)
	rsaCert := rsaCertificate(t)
	two := tls.Certificate{Certificate: [][]byte{good.Certificate[0], other.Certificate[0]}, PrivateKey: good.PrivateKey}
	noCert := wire.ClientConfig(good, anyKey)
	noCert.Certificates = nil
	for _, tc := range []struct {
		name               string
		client, server     *tls.Config
		clientErr, servErr string
	}{
		{
			name:      "the server presents an RSA certificate",
			client:    wire.ClientConfig(good, anyKey),
			server:    wire.ServerConfig(rsaCert, lane, anyKey),
			clientErr: "peer certificate holds a *rsa.PublicKey, and the link pins only Ed25519 keys",
			servErr:   "remote error: tls: bad certificate",
		},
		{
			name:      "the client presents an RSA certificate",
			client:    wire.ClientConfig(rsaCert, anyKey),
			server:    wire.ServerConfig(good, lane, anyKey),
			clientErr: "remote error: tls: bad certificate",
			servErr:   "peer certificate holds a *rsa.PublicKey, and the link pins only Ed25519 keys",
		},
		{
			name:      "the server presents two certificates",
			client:    wire.ClientConfig(good, anyKey),
			server:    wire.ServerConfig(two, lane, anyKey),
			clientErr: "peer presented 2 certificates, and the link pins exactly one",
			servErr:   "remote error: tls: bad certificate",
		},
		{
			name:      "the client presents no certificate",
			client:    noCert,
			server:    wire.ServerConfig(good, lane, anyKey),
			clientErr: "remote error: tls: certificate required",
			servErr:   "tls: client didn't provide a certificate",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				_, clientErr, serverErr := handshake(t, tc.client, tc.server)
				if got := [2]string{errText(clientErr), errText(serverErr)}; got != [2]string{tc.clientErr, tc.servErr} {
					t.Fatalf("errors (client, server) = %q", got)
				}
			})
		})
	}
}

// seen records each key a pin check is shown, refusing once refuse is set.
type seen struct {
	keys   []ed25519.PublicKey
	refuse bool
}

func (s *seen) accept(key ed25519.PublicKey, _ string) error {
	s.keys = append(s.keys, key)
	if s.refuse {
		return errors.New("refused by the pin check")
	}
	return nil
}

func TestResumedSessionsStillRunThePinCheck(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		clientCert, clientKey := ed25519Certificate(t)
		serverCert, serverKey := ed25519Certificate(t)
		var atClient, atServer seen
		clientCfg := wire.ClientConfig(clientCert, atClient.accept)
		clientCfg.ClientSessionCache = tls.NewLRUClientSessionCache(1)
		serverCfg := wire.ServerConfig(serverCert, lane, atServer.accept)

		if _, clientErr, serverErr := handshake(t, clientCfg, serverCfg); clientErr != nil || serverErr != nil {
			t.Fatalf("first handshake: %v, %v", clientErr, serverErr)
		}
		state, clientErr, serverErr := handshake(t, clientCfg, serverCfg)
		if clientErr != nil || serverErr != nil || !state.DidResume {
			t.Fatalf("second handshake: resumed %v, %v, %v", state.DidResume, clientErr, serverErr)
		}
		if len(atClient.keys) != 2 || !bytes.Equal(atClient.keys[1], serverKey) {
			t.Fatalf("the client's pin check saw %d keys on two handshakes", len(atClient.keys))
		}
		if len(atServer.keys) != 2 || !bytes.Equal(atServer.keys[1], clientKey) {
			t.Fatalf("the server's pin check saw %d keys on two handshakes", len(atServer.keys))
		}

		atClient.refuse = true
		state, clientErr, _ = handshake(t, clientCfg, serverCfg)
		if errText(clientErr) != "refused by the pin check" || !state.DidResume {
			t.Fatalf("a resumed handshake the pin check refuses: resumed %v, %v", state.DidResume, clientErr)
		}
	})
}
