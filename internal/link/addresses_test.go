package link_test

import (
	"net/netip"
	"slices"
	"testing"

	"github.com/jatassi/slipstream/internal/link"
	linkv1 "github.com/jatassi/slipstream/internal/pb/slipstream/link/v1"
)

func parse(list ...string) (link.Addresses, error) {
	var aps []netip.AddrPort
	for _, s := range list {
		aps = append(aps, netip.MustParseAddrPort(s))
	}
	return link.ParseAddresses(aps)
}

func TestAddressesOrderIPv6FirstAndDropRepeats(t *testing.T) {
	a, err := parse("192.0.2.1:8485", "[2001:db8::1]:8485", "[::ffff:192.0.2.1]:8485", "198.51.100.7:8485", "[2001:db8::2]:9000")
	if err != nil {
		t.Fatal(err)
	}
	want := []netip.AddrPort{
		netip.MustParseAddrPort("[2001:db8::1]:8485"),
		netip.MustParseAddrPort("[2001:db8::2]:9000"),
		netip.MustParseAddrPort("192.0.2.1:8485"),
		netip.MustParseAddrPort("198.51.100.7:8485"),
	}
	if got := a.All(); !slices.Equal(got, want) {
		t.Fatalf("All() = %v, want %v", got, want)
	}
	back, err := link.AddressesFromProto(a.Proto())
	if err != nil || back != a {
		t.Fatalf("the PeerAddresses round trip gave %v, %v", back.All(), err)
	}
}

func TestAddressesRefusals(t *testing.T) {
	nine := []string{"192.0.2.1:1", "192.0.2.2:1", "192.0.2.3:1", "192.0.2.4:1", "192.0.2.5:1", "192.0.2.6:1", "192.0.2.7:1", "192.0.2.8:1", "192.0.2.9:1"}
	for _, tc := range []struct {
		in   []string
		want string
	}{
		{nil, "a peer address record holds 1 to 8 addresses, not 0"},
		{nine, "a peer address record holds 1 to 8 addresses, not 9"},
		{[]string{"[fe80::1%eth0]:8485"}, "address [fe80::1%eth0]:8485 is not a literal IP and port"},
		{[]string{"192.0.2.1:0"}, "address 192.0.2.1:0 is not a literal IP and port"},
	} {
		if _, err := parse(tc.in...); err == nil || err.Error() != tc.want {
			t.Errorf("ParseAddresses(%v) = %v, want %s", tc.in, err, tc.want)
		}
	}
	_, err := link.AddressesFromProto(&linkv1.PeerAddresses{Addresses: []*linkv1.Address{{Ip: []byte{1, 2, 3}, Port: 8485}}})
	if want := "address of 3 bytes and port 8485 is not an IP and port"; err == nil || err.Error() != want {
		t.Errorf("AddressesFromProto of a 3-byte address = %v, want %s", err, want)
	}
}
