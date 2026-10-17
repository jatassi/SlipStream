package joincode_test

import (
	"crypto/ed25519"
	"errors"
	"log/slog"
	"net/netip"
	"slices"
	"strings"
	"testing"

	"github.com/jatassi/slipstream/internal/joincode"
)

const golden = "SLP1-CC6WT-AB6RG-SPD48-JAH4B-QCJV9-XJ6AS-K7D1M-PMTVC-DNQ6Y-W3HE9-SG213-Z0000-58D1S-QG852-G"

func span(from byte, n int) []byte {
	b := make([]byte, n)
	for i := range b {
		b[i] = from + byte(i)
	}
	return b
}

func goldenCode(t *testing.T) (joincode.Code, ed25519.PublicKey) {
	t.Helper()
	key := ed25519.PublicKey(span(0x00, 32))
	c, err := joincode.New(key, [16]byte(span(0x64, 16)), []netip.AddrPort{netip.MustParseAddrPort("127.0.0.2:41377")})
	if err != nil {
		t.Fatal(err)
	}
	return c, key
}

func TestGoldenText(t *testing.T) {
	c, _ := goldenCode(t)
	if c.String() != golden {
		t.Errorf("String() = %s, want %s", c.String(), golden)
	}
	text, err := c.MarshalText()
	if err != nil || string(text) != golden {
		t.Errorf("MarshalText() = %s, %v", text, err)
	}
}

func TestParseReadsAnEmailedCopy(t *testing.T) {
	for _, s := range []string{
		golden,
		"slp1-cc6wt-ab6rg-spd48-jah4b-qcjv9-xj6as-k7d1m-pmtvc-dnq6y-w3he9-sg213-zoooo-58d1s-qg852-g",
		"  SLP1 CC6WT AB6RG SPD48 JAH4B QCJV9\n\tXJ6AS K7D1M PMTVC DNQ6Y W3HE9 SG2I3 ZOOOO 58DLS QG852 G \n",
	} {
		c, err := joincode.Parse(s)
		if err != nil {
			t.Fatalf("Parse(%q): %v", s, err)
		}
		if c.String() != golden {
			t.Errorf("Parse(%q) = %s, want %s", s, c.String(), golden)
		}
	}
}

func TestParseKeepsEveryField(t *testing.T) {
	c, err := joincode.Parse(golden)
	if err != nil {
		t.Fatal(err)
	}
	_, key := goldenCode(t)
	if !c.Names(key) {
		t.Error("the code does not name its Controller's key")
	}
	if c.Names(ed25519.PublicKey(span(0x01, 32))) {
		t.Error("the code names another key")
	}
	if c.Secret() != [16]byte(span(0x64, 16)) {
		t.Errorf("Secret() = %x", c.Secret())
	}
	if want := []netip.AddrPort{netip.MustParseAddrPort("127.0.0.2:41377")}; !slices.Equal(c.Addresses(), want) {
		t.Errorf("Addresses() = %v, want %v", c.Addresses(), want)
	}
}

func TestBothFamiliesRoundTrip(t *testing.T) {
	addrs := []netip.AddrPort{
		netip.MustParseAddrPort("[2001:db8::7]:8485"),
		netip.MustParseAddrPort("192.0.2.7:8485"),
	}
	c, err := joincode.New(ed25519.PublicKey(span(0x00, 32)), [16]byte(span(0x64, 16)), addrs)
	if err != nil {
		t.Fatal(err)
	}
	if len(c.String()) != 126 {
		t.Errorf("len = %d, want 126", len(c.String()))
	}
	parsed, err := joincode.Parse(c.String())
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(parsed.Addresses(), addrs) {
		t.Errorf("Addresses() = %v, want %v", parsed.Addresses(), addrs)
	}
}

func TestParseRefusesADamagedCode(t *testing.T) {
	for name, s := range map[string]string{
		"a changed character":        strings.Replace(golden, "AB6RG", "AB6RH", 1),
		"a changed last character":   golden[:len(golden)-1] + "H",
		"cut short":                  golden[:len(golden)-6],
		"a character outside base32": strings.Replace(golden, "AB6RG", "AB6RU", 1),
		"nothing after the prefix":   "SLP1-",
	} {
		_, err := joincode.Parse(s)
		if err == nil || err.Error() != "the Join Code is mistyped or cut short" || !errors.Is(err, joincode.ErrMistyped) {
			t.Errorf("%s: err = %v", name, err)
		}
	}
	for _, s := range []string{"SLP2-CC6WT-AB6RG", "hello", ""} {
		_, err := joincode.Parse(s)
		if err == nil || err.Error() != "the Join Code is not a SlipStream code" {
			t.Errorf("Parse(%q): err = %v", s, err)
		}
	}
}

func TestTheUnsetConfigValue(t *testing.T) {
	var c joincode.Code
	if err := c.UnmarshalText([]byte("")); err != nil || !c.IsZero() {
		t.Fatalf("UnmarshalText(\"\") = %v, zero %v", err, c.IsZero())
	}
	text, err := c.MarshalText()
	if err != nil || len(text) != 0 {
		t.Errorf("MarshalText() = %q, %v", text, err)
	}
	if c.Redacted() != "" {
		t.Errorf("Redacted() = %q", c.Redacted())
	}
	if err := c.UnmarshalText([]byte(golden)); err != nil {
		t.Fatal(err)
	}
	if c.Redacted() != "SLP1-[redacted]" {
		t.Errorf("Redacted() = %q", c.Redacted())
	}
	if err := c.UnmarshalText([]byte(golden[:40])); err == nil || err.Error() != "the Join Code is mistyped or cut short" {
		t.Errorf("damaged UnmarshalText err = %v", err)
	}
}

func TestNewRefusesAnAddressListOutsideTheRecord(t *testing.T) {
	key := ed25519.PublicKey(span(0x00, 32))
	nine := slices.Repeat([]netip.AddrPort{netip.MustParseAddrPort("127.0.0.2:1")}, 9)
	for want, addrs := range map[string][]netip.AddrPort{
		"a Join Code holds 1 to 8 addresses, not 0":             nil,
		"a Join Code holds 1 to 8 addresses, not 9":             nine,
		"address [fe80::1%eth0]:1 is not a literal IP and port": {netip.MustParseAddrPort("[fe80::1%eth0]:1")},
	} {
		_, err := joincode.New(key, [16]byte{}, addrs)
		if err == nil || err.Error() != want {
			t.Errorf("err = %v, want %s", err, want)
		}
	}
}

func TestALoggedCodeIsRedacted(t *testing.T) {
	code, err := joincode.Parse(golden)
	if err != nil {
		t.Fatal(err)
	}
	var b strings.Builder
	noTime := func(_ []string, a slog.Attr) slog.Attr {
		if a.Key == slog.TimeKey {
			return slog.Attr{}
		}
		return a
	}
	slog.New(slog.NewJSONHandler(&b, &slog.HandlerOptions{ReplaceAttr: noTime})).Info("join", "code", code)
	if want := `{"level":"INFO","msg":"join","code":"SLP1-[redacted]"}` + "\n"; b.String() != want {
		t.Fatalf("record = %s, want %s", b.String(), want)
	}
}
