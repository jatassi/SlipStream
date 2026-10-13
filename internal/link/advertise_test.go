package link

import (
	"net/netip"
	"slices"
	"testing"
)

func TestAdvertise(t *testing.T) {
	host := []netip.Addr{
		netip.MustParseAddr("127.0.0.1"),
		netip.MustParseAddr("::1"),
		netip.MustParseAddr("fe80::1"),
		netip.MustParseAddr("192.168.1.20"),
		netip.MustParseAddr("2001:db8::20"),
		netip.MustParseAddr("169.254.0.9"),
	}
	for _, tc := range []struct {
		bound string
		want  []string
	}{
		{"127.0.0.2:41377", []string{"127.0.0.2:41377"}},
		{"[::]:8485", []string{"[2001:db8::20]:8485", "192.168.1.20:8485"}},
		{"0.0.0.0:8485", []string{"192.168.1.20:8485"}},
	} {
		a, ok := advertise(netip.MustParseAddrPort(tc.bound), host)
		var got []string
		for _, ap := range a.All() {
			got = append(got, ap.String())
		}
		if !ok || !slices.Equal(got, tc.want) {
			t.Errorf("advertise(%s) = %v, %v; want %v", tc.bound, got, ok, tc.want)
		}
	}
	if _, ok := advertise(netip.MustParseAddrPort("[::]:8485"), host[:3]); ok {
		t.Error("a host with only loopback and link-local addresses advertised a record")
	}
}
