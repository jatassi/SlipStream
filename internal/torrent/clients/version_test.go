package clients_test

import (
	"testing"

	"github.com/jatassi/slipstream/internal/torrent/clients"
)

func TestParseVersionReadsWhatEachProductReports(t *testing.T) {
	cases := map[string]string{
		"4.1.1 (a8bd3e2b)": "4.1.1",
		"v4.6.2":           "4.6.2",
		"v5.2.0beta1":      "5.2",
		"2.1.1":            "2.1.1",
		"0.9.8":            "0.9.8",
		"4":                "4.0",
		"4.6.2.10":         "4.6.2",
	}
	for raw, want := range cases {
		v, err := clients.ParseVersion(raw)
		if err != nil {
			t.Errorf("ParseVersion(%q): %v", raw, err)
			continue
		}
		if v.String() != want {
			t.Errorf("ParseVersion(%q) = %s, want %s", raw, v, want)
		}
	}
	for _, raw := range []string{"", "unknown", "4..1"} {
		if v, err := clients.ParseVersion(raw); err == nil {
			t.Errorf("ParseVersion(%q) = %s, want an error", raw, v)
		}
	}
}

func TestVersionLessOrdersByNumberNeverByText(t *testing.T) {
	ordered := []string{"0.9.8", "0.16", "4.5.2", "4.6", "4.6.10", "5.0"}
	for i := range len(ordered) - 1 {
		a, b := version(t, ordered[i]), version(t, ordered[i+1])
		if !a.Less(b) || b.Less(a) {
			t.Errorf("want %s < %s", a, b)
		}
	}
	if v := version(t, "4.6"); v.Less(v) {
		t.Errorf("%s < itself", v)
	}
}

func TestAVersionUnderAKindsFloorIsLess(t *testing.T) {
	if !version(t, "4.5.2").Less(clients.QBittorrent.Facts().Floor) {
		t.Error("qBittorrent 4.5.2 isn't under its floor")
	}
	if version(t, "4.1.1").Less(clients.Transmission.Facts().Floor) {
		t.Error("Transmission 4.1.1 is under its floor")
	}
}

func version(t *testing.T, s string) clients.Version {
	t.Helper()
	v, err := clients.ParseVersion(s)
	if err != nil {
		t.Fatal(err)
	}
	return v
}
