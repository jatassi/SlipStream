package fault_test

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"io"
	"math/big"
	"net"
	"testing"
	"testing/synctest"
	"time"

	"github.com/jatassi/slipstream/fakes/fault"
)

func certificate(t *testing.T) tls.Certificate {
	t.Helper()
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	tmpl := &x509.Certificate{SerialNumber: big.NewInt(1), NotBefore: time.Now(), NotAfter: time.Now().Add(time.Hour)}
	der, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, pub, priv)
	if err != nil {
		t.Fatal(err)
	}
	return tls.Certificate{Certificate: [][]byte{der}, PrivateKey: priv}
}

// TestMutualTLSThenEcho runs the link's shape: a TLS 1.3 handshake with
// client certificates, then the client writes first while the server may
// still be sending its session tickets.
func TestMutualTLSThenEcho(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		n := fault.NewNetwork()
		client, server := pair(t, n)
		n.Path(home, seedbox).Set(fault.Latency(10 * time.Millisecond))
		cfg := &tls.Config{
			Certificates:       []tls.Certificate{certificate(t)},
			MinVersion:         tls.VersionTLS13,
			ClientAuth:         tls.RequireAnyClientCert,
			InsecureSkipVerify: true, //nolint:gosec // the test checks transport, not trust
		}
		echoed := make(chan error, 1)
		go func() {
			s := tls.Server(server, cfg)
			_, err := io.Copy(s, s)
			echoed <- err
		}()
		c := tls.Client(client, cfg)
		if err := c.HandshakeContext(t.Context()); err != nil {
			t.Fatal(err)
		}
		if c.ConnectionState().Version != tls.VersionTLS13 {
			t.Fatalf("version %x", c.ConnectionState().Version)
		}
		write(t, c, "hello")
		b := make([]byte, 5)
		if _, err := io.ReadFull(c, b); err != nil || string(b) != "hello" {
			t.Fatalf("echo %q, %v", b, err)
		}
		if err := c.CloseWrite(); err != nil {
			t.Fatal(err)
		}
		if err := <-echoed; err != nil {
			t.Fatalf("server copy: %v", err)
		}
	})
}

// TestBothSidesWriteFirst writes from both ends before either reads, as
// two peers exchanging greetings do. TCP's buffers absorb this, and a bare
// net.Pipe deadlocks on it.
func TestBothSidesWriteFirst(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		client, server := pair(t, fault.NewNetwork())
		greet := func(c net.Conn, out chan<- string) {
			_, _ = c.Write([]byte("hi"))
			b := make([]byte, 2)
			_, _ = io.ReadFull(c, b)
			out <- string(b)
		}
		a, b := make(chan string, 1), make(chan string, 1)
		go greet(client, a)
		go greet(server, b)
		if got := [2]string{<-a, <-b}; got != [2]string{"hi", "hi"} {
			t.Fatalf("greetings %v", got)
		}
	})
}
