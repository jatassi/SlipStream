package searching_test

import (
	"context"
	"errors"
	"net"
	"net/netip"
	"testing"

	"github.com/jatassi/slipstream/internal/searching"
)

func TestASiteHostIsDialedAtItsAddress(t *testing.T) {
	var e searching.Endpoints
	for _, text := range []string{"fixture-site.fake.localhost=127.0.0.2:4431", "Other.Test=[::1]:80"} {
		var a searching.SiteAddress
		if err := a.UnmarshalText([]byte(text)); err != nil {
			t.Fatal(err)
		}
		e.Sites = append(e.Sites, a)
	}
	var dialed []string
	dial := e.Dial(func(_ context.Context, _, addr string) (net.Conn, error) {
		dialed = append(dialed, addr)
		return nil, errors.New("not dialed")
	})
	for _, addr := range []string{
		"torznab.fixture-site.fake.localhost:443",
		"fixture-site.fake.localhost:80",
		"A.B.FIXTURE-SITE.fake.localhost.:443",
		"evilfixture-site.fake.localhost:443",
		"www.other.test:443",
		"127.0.0.1:9",
		"example.org:443",
	} {
		_, _ = dial(t.Context(), "tcp", addr)
	}
	want := []string{
		"127.0.0.2:4431", "127.0.0.2:4431", "127.0.0.2:4431",
		"evilfixture-site.fake.localhost:443", "[::1]:80", "127.0.0.1:9", "example.org:443",
	}
	if len(dialed) != len(want) {
		t.Fatalf("dialed %v, want %v", dialed, want)
	}
	for i := range want {
		if dialed[i] != want[i] {
			t.Errorf("dial %d went to %s, want %s", i, dialed[i], want[i])
		}
	}
}

func TestASiteAddressReadsAndWritesItsText(t *testing.T) {
	a := searching.SiteAddress{Domain: "fixture-site.fake.localhost", At: netip.MustParseAddrPort("127.0.0.2:4431")}
	text, _ := a.MarshalText()
	var back searching.SiteAddress
	if err := back.UnmarshalText(text); err != nil || back != a {
		t.Fatalf("%s read back as %+v, %v", text, back, err)
	}
	for _, bad := range []string{"", "fixture-site.fake.localhost", "=127.0.0.1:1", "fixture-site.fake.localhost=localhost:1", "fixture-site.fake.localhost=127.0.0.1"} {
		if err := new(searching.SiteAddress).UnmarshalText([]byte(bad)); err == nil {
			t.Errorf("%q read without an error", bad)
		}
	}
}
