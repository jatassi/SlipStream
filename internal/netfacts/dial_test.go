package netfacts_test

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"fmt"
	"log/slog"
	"math/big"
	"net"
	"net/http"
	"net/http/httptest"
	"net/netip"
	"strings"
	"testing"
	"time"

	"github.com/jatassi/slipstream/internal/netfacts"
	"github.com/jatassi/slipstream/internal/netmodel"
	netv1 "github.com/jatassi/slipstream/internal/pb/slipstream/network/v1"
	"github.com/jatassi/slipstream/internal/storage/storagetest"
)

// authority is a CA a test trusts and the leaves it signs.
type authority struct {
	cert *x509.Certificate
	key  *ecdsa.PrivateKey
	pool *x509.CertPool
}

func newAuthority(t *testing.T) authority {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	tmpl := &x509.Certificate{
		SerialNumber: big.NewInt(1), Subject: pkix.Name{CommonName: "test CA"}, IsCA: true, BasicConstraintsValid: true,
		KeyUsage: x509.KeyUsageCertSign, NotBefore: time.Now().Add(-time.Hour), NotAfter: time.Now().Add(time.Hour),
	}
	der, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &key.PublicKey, key)
	if err != nil {
		t.Fatal(err)
	}
	cert, _ := x509.ParseCertificate(der)
	pool := x509.NewCertPool()
	pool.AddCert(cert)
	return authority{cert: cert, key: key, pool: pool}
}

func (a authority) leaf(t *testing.T, name string, notAfter time.Time) tls.Certificate {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	tmpl := &x509.Certificate{
		SerialNumber: big.NewInt(2), DNSNames: []string{name}, NotBefore: notAfter.Add(-2 * time.Hour), NotAfter: notAfter,
		ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
	}
	der, err := x509.CreateCertificate(rand.Reader, tmpl, a.cert, &key.PublicKey, a.key)
	if err != nil {
		t.Fatal(err)
	}
	return tls.Certificate{Certificate: [][]byte{der}, PrivateKey: key}
}

func serveTLS(t *testing.T, cert *tls.Certificate, h http.Handler) netip.AddrPort {
	t.Helper()
	s := httptest.NewUnstartedServer(h)
	if cert != nil {
		s.TLS = &tls.Config{Certificates: []tls.Certificate{*cert}, MinVersion: tls.VersionTLS12}
		s.StartTLS()
	} else {
		s.Start()
	}
	t.Cleanup(s.Close)
	return netip.MustParseAddrPort(s.Listener.Addr().String())
}

// A test connection reports how far each address got, the certificate
// whatever its validity, and the first answer alone, cut at read_bytes.
func TestDialReportsHowFarEachConnectionGot(t *testing.T) {
	ca := newAuthority(t)
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/moved" {
			http.Redirect(w, r, "https://elsewhere.example/", http.StatusFound)
			return
		}
		_, _ = w.Write([]byte(strings.Repeat("x", 100)))
	})
	valid, expired := ca.leaf(t, "plex.example.org", time.Now().Add(time.Hour)), ca.leaf(t, "plex.example.org", time.Now().Add(-time.Hour))
	at := map[string]netip.AddrPort{
		"valid":   serveTLS(t, &valid, handler),
		"expired": serveTLS(t, &expired, handler),
		"plain":   serveTLS(t, nil, handler),
	}
	untrusted := httptest.NewTLSServer(handler)
	t.Cleanup(untrusted.Close)
	at["untrusted"] = netip.MustParseAddrPort(untrusted.Listener.Addr().String())
	closed, err := new(net.ListenConfig).Listen(t.Context(), "tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	at["closed"] = netip.MustParseAddrPort(closed.Addr().String())
	_ = closed.Close()

	tm := netfacts.DefaultTimings()
	tm.ReadBytes = 16
	svc := netfacts.NewService(netfacts.New(netfacts.Config{Local: storagetest.Local(t, netfacts.Schema()), Env: env(), Timings: tm, Roots: ca.pool, Log: slog.New(slog.DiscardHandler)}))
	https, plain := netv1.Scheme_SCHEME_HTTPS, netv1.Scheme_SCHEME_HTTP
	for _, c := range []struct {
		name, server, sni, path string
		scheme                  netv1.Scheme
		want                    string
	}{
		{"a valid certificate", "valid", "plex.example.org", "/identity", https, "http none 200 16 "},
		{"another name", "valid", "other.example.org", "/identity", https, "http wrong-name 200 16 "},
		{"an expired certificate", "expired", "plex.example.org", "/identity", https, "http expired 200 16 "},
		{"an untrusted certificate", "untrusted", "plex.example.org", "/identity", https, "http untrusted 200 16 "},
		{"plain HTTP", "plain", "plex.example.org", "/identity", plain, "http - 200 16 "},
		{"a redirect, not followed", "valid", "plex.example.org", "/moved", https, "http none 302 16 elsewhere.example"},
		{"TLS to a plain listener", "plain", "plex.example.org", "/identity", https, "connected handshake 0 0 "},
		{"a refused connection", "closed", "plex.example.org", "/identity", https, "resolved refused"},
	} {
		res, err := svc.Dial(t.Context(), &netv1.DialRequest{
			Target: &netv1.DialRequest_Addr{Addr: at[c.server].Addr().AsSlice()}, Port: uint32(at[c.server].Port()),
			Family: netmodel.FamilyOnWire(netmodel.IPv4), Scheme: c.scheme, Sni: c.sni, Path: c.path,
		})
		if err != nil || len(res.GetAttempts()) != 1 {
			t.Fatalf("%s: %v, %v", c.name, res, err)
		}
		if got := describe(res.GetAttempts()[0]); got != c.want {
			t.Errorf("%s: got %q, want %q", c.name, got, c.want)
		}
	}
}

// describe is how far a got: its stage, then its TLS problem, status,
// body length and redirect host, or its error once it stopped.
func describe(a *netv1.Attempt) string {
	stage := strings.ToLower(strings.TrimPrefix(a.GetStage().String(), "STAGE_"))
	if a.GetError() == netv1.DialError_DIAL_ERROR_REFUSED {
		return stage + " refused"
	}
	problem := "-"
	if a.GetTls() != nil {
		problem = strings.ReplaceAll(strings.ToLower(strings.TrimPrefix(a.GetTls().GetProblem().String(), "TLS_PROBLEM_")), "_", "-")
		if problem == "unspecified" {
			problem = "none"
		}
	}
	return fmt.Sprintf("%s %s %d %d %s", stage, problem, a.GetStatus(), len(a.GetBody()), a.GetLocationHost())
}
