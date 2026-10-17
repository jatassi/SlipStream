package joincode_test

import (
	"crypto/ed25519"
	"errors"
	"net/netip"
	"slices"
	"strings"
	"testing"

	"github.com/jatassi/slipstream/internal/joincode"
)

const goldenAddress = "SLN1-40GJ4-8S44M-K2EA1-958NJ-RB9E5-WR32C-HK6GT-KCDSR-74X3P-F1X7R-ZG216-B01RG-J895T-WRNRQ-0"

func goldenNodeAddress(t *testing.T) (joincode.NodeAddress, ed25519.PublicKey) {
	t.Helper()
	key := ed25519.PublicKey(span(0x20, 32))
	a, err := joincode.NewNodeAddress(key, []netip.AddrPort{netip.MustParseAddrPort("203.0.113.9:8485")})
	if err != nil {
		t.Fatal(err)
	}
	return a, key
}

func TestNodeAddressGoldenText(t *testing.T) {
	a, _ := goldenNodeAddress(t)
	if a.String() != goldenAddress {
		t.Errorf("String() = %s, want %s", a.String(), goldenAddress)
	}
}

func TestParseNodeAddressReadsAPastedCopy(t *testing.T) {
	_, key := goldenNodeAddress(t)
	for _, s := range []string{
		goldenAddress,
		strings.ToLower(goldenAddress),
		"  SLN1 4OGJ4 8S44M K2EA1 958NJ RB9E5\n\tWR32C HK6GT KCDSR 74X3P F1X7R ZG2I6 BO1RG J895T WRNRQ O \n",
	} {
		a, err := joincode.ParseNodeAddress(s)
		if err != nil {
			t.Fatalf("ParseNodeAddress(%q): %v", s, err)
		}
		if a.String() != goldenAddress || !slices.Equal(a.Key(), key) || !slices.Equal(a.Addresses(), []netip.AddrPort{netip.MustParseAddrPort("203.0.113.9:8485")}) {
			t.Errorf("ParseNodeAddress(%q) = %s holding %x at %v", s, a, a.Key(), a.Addresses())
		}
	}
}

func TestANodeAddressOfEightAddressesRoundTrips(t *testing.T) {
	var addrs []netip.AddrPort
	for i := range 4 {
		addrs = append(addrs,
			netip.AddrPortFrom(netip.AddrFrom16([16]byte{0x20, 0x01, 0x0d, 0xb8, 15: byte(i + 1)}), 8485),
			netip.AddrPortFrom(netip.AddrFrom4([4]byte{192, 0, 2, byte(i + 1)}), uint16(40000+i)),
		)
	}
	a, err := joincode.NewNodeAddress(ed25519.PublicKey(span(0x20, 32)), addrs)
	if err != nil {
		t.Fatal(err)
	}
	parsed, err := joincode.ParseNodeAddress(a.String())
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(parsed.Addresses(), addrs) {
		t.Errorf("Addresses() = %v, want %v in dial order", parsed.Addresses(), addrs)
	}
}

func TestParseNodeAddressRefusesDamagedText(t *testing.T) {
	for name, s := range map[string]string{
		"a changed character":      strings.Replace(goldenAddress, "8S44M", "8S44N", 1),
		"a changed last character": goldenAddress[:len(goldenAddress)-1] + "1",
		"cut short":                goldenAddress[:len(goldenAddress)-6],
		"nothing after the prefix": "SLN1-",
	} {
		if _, err := joincode.ParseNodeAddress(s); !errors.Is(err, joincode.ErrAddressMistyped) || err.Error() != "the Node address is mistyped or cut short" {
			t.Errorf("%s: err = %v", name, err)
		}
	}
	for _, s := range []string{golden, "SLN2-40GJ4", ""} {
		if _, err := joincode.ParseNodeAddress(s); !errors.Is(err, joincode.ErrNotNodeAddress) {
			t.Errorf("ParseNodeAddress(%q): err = %v, want %v", s, err, joincode.ErrNotNodeAddress)
		}
	}
	if _, err := joincode.Parse(goldenAddress); !errors.Is(err, joincode.ErrNotSlipStream) {
		t.Errorf("Parse(a Node address): err = %v, want %v", err, joincode.ErrNotSlipStream)
	}
}

func TestNewNodeAddressRefusesWhatTheFormatCannotHold(t *testing.T) {
	key := ed25519.PublicKey(span(0x20, 32))
	nine := slices.Repeat([]netip.AddrPort{netip.MustParseAddrPort("127.0.0.3:1")}, 9)
	for _, c := range []struct {
		key   ed25519.PublicKey
		addrs []netip.AddrPort
		want  string
	}{
		{key[:16], []netip.AddrPort{netip.MustParseAddrPort("127.0.0.3:1")}, "a Node key is 32 bytes, not 16"},
		{key, nil, "a Node address holds 1 to 8 addresses, not 0"},
		{key, nine, "a Node address holds 1 to 8 addresses, not 9"},
		{key, []netip.AddrPort{netip.MustParseAddrPort("[fe80::1%eth0]:1")}, "address [fe80::1%eth0]:1 is not a literal IP and port"},
	} {
		if _, err := joincode.NewNodeAddress(c.key, c.addrs); err == nil || err.Error() != c.want {
			t.Errorf("err = %v, want %s", err, c.want)
		}
	}
}
