package node

import (
	"net/netip"
	"testing"
)

func TestSetupURL(t *testing.T) {
	const code = "ABCD-EFGH-JKMN"
	for listen, want := range map[string]string{
		"0.0.0.0:8484":           "http://localhost:8484/setup#code=ABCD-EFGH-JKMN",
		"[::]:8484":              "http://localhost:8484/setup#code=ABCD-EFGH-JKMN",
		"127.0.0.1:8484":         "http://localhost:8484/setup#code=ABCD-EFGH-JKMN",
		"[::1]:8484":             "http://localhost:8484/setup#code=ABCD-EFGH-JKMN",
		"192.168.1.20:8484":      "http://192.168.1.20:8484/setup#code=ABCD-EFGH-JKMN",
		"[::ffff:10.0.0.5]:8484": "http://10.0.0.5:8484/setup#code=ABCD-EFGH-JKMN",
		"[fd00::20]:8484":        "http://[fd00::20]:8484/setup#code=ABCD-EFGH-JKMN",
		"127.0.0.1:9100":         "http://localhost:9100/setup#code=ABCD-EFGH-JKMN",
	} {
		if got := setupURL(netip.MustParseAddrPort(listen), code); got != want {
			t.Errorf("setupURL(%s) = %q, want %q", listen, got, want)
		}
	}
}
