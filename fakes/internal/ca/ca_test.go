package ca_test

import (
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"reflect"
	"testing"
	"time"

	"github.com/jatassi/slipstream/fakes/internal/ca"
)

func TestCAIsConstrainedToLoopback(t *testing.T) {
	c, err := ca.New("k3v9x2qa")
	if err != nil {
		t.Fatal(err)
	}
	block, _ := pem.Decode(c.PEM())
	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		t.Fatal(err)
	}
	got := fmt.Sprintf("%s ca=%t valid=%s critical=%t dns=%v ips=%v",
		cert.Subject.CommonName, cert.IsCA, cert.NotAfter.Sub(cert.NotBefore),
		cert.PermittedDNSDomainsCritical, cert.PermittedDNSDomains, cert.PermittedIPRanges)
	want := "SlipStream Fake world k3v9x2qa ca=true valid=169h0m0s critical=true dns=[localhost .localhost] ips=[127.0.0.0/8 ::1/128]"
	if got != want {
		t.Fatalf("CA = %s\nwant %s", got, want)
	}
	if since := time.Since(cert.NotBefore); since < time.Hour-time.Minute || since > time.Hour+time.Minute {
		t.Fatalf("CA valid from %s ago, want 1h", since)
	}
}

func TestLeafNamesLoopbackAndLocalhost(t *testing.T) {
	c, err := ca.New("k3v9x2qa")
	if err != nil {
		t.Fatal(err)
	}
	leaf, err := c.Leaf("127.0.0.1", "fixture.fake.localhost")
	if err != nil {
		t.Fatal(err)
	}
	got := fmt.Sprintf("%v %v", leaf.Leaf.IPAddresses, leaf.Leaf.DNSNames)
	if want := "[127.0.0.1] [fixture.fake.localhost]"; got != want {
		t.Fatalf("leaf names = %s, want %s", got, want)
	}
	for _, name := range []string{"127.0.0.1", "fixture.fake.localhost"} {
		if _, err := leaf.Leaf.Verify(x509.VerifyOptions{Roots: c.Pool(), DNSName: name}); err != nil {
			t.Errorf("verify for %s: %v", name, err)
		}
	}
}

func TestLeafRefusesARealName(t *testing.T) {
	c, err := ca.New("k3v9x2qa")
	if err != nil {
		t.Fatal(err)
	}
	cases := map[string]string{
		"example.com": `leaf for example.com: x509: a root or intermediate certificate is not authorized to sign for this name: DNS name "example.com" is not permitted by any constraint`,
		"10.0.0.1":    `leaf for 10.0.0.1: x509: a root or intermediate certificate is not authorized to sign for this name: IP address "10.0.0.1" is not permitted by any constraint`,
	}
	got := map[string]string{}
	for name := range cases {
		_, err := c.Leaf(name)
		got[name] = fmt.Sprint(err)
	}
	if !reflect.DeepEqual(got, cases) {
		t.Fatalf("Leaf errors = %v\nwant %v", got, cases)
	}
}
