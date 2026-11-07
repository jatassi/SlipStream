package clients_test

import (
	"errors"
	"testing"

	"github.com/jatassi/slipstream/internal/torrent/clients"
)

func TestParseAddressFillsTheKindsPath(t *testing.T) {
	cases := []struct {
		raw  string
		kind clients.Kind
		want string
	}{
		{"http://192.168.1.20:9091", clients.Transmission, "http://192.168.1.20:9091/transmission/rpc"},
		{"http://192.168.1.20:9091/", clients.Transmission, "http://192.168.1.20:9091/transmission/rpc"},
		{"https://seedbox.lan/transmission/rpc/", clients.Transmission, "https://seedbox.lan/transmission/rpc"},
		{"http://192.168.1.20:8112", clients.Deluge, "http://192.168.1.20:8112/json"},
		{"http://192.168.1.20", clients.RTorrent, "http://192.168.1.20/RPC2"},
		{"http://192.168.1.20:8080/", clients.QBittorrent, "http://192.168.1.20:8080"},
		{"http://192.168.1.20/qbt/", clients.QBittorrent, "http://192.168.1.20/qbt"},
		{"  http://[fd00::20]:8080  ", clients.QBittorrent, "http://[fd00::20]:8080"},
	}
	for _, c := range cases {
		a, err := clients.ParseAddress(c.raw, c.kind)
		if err != nil {
			t.Errorf("ParseAddress(%q, %s): %v", c.raw, c.kind, err)
			continue
		}
		if a.String() != c.want {
			t.Errorf("ParseAddress(%q, %s) = %q, want %q", c.raw, c.kind, a, c.want)
		}
	}
}

func TestParseAddressRefusesWhatTheAddressFieldDoesntHold(t *testing.T) {
	cases := []struct {
		raw  string
		want error
	}{
		{"http://jackson:hunter2@192.168.1.20:9091", clients.ErrSignInInAddress},
		{"http://jackson@192.168.1.20:9091", clients.ErrSignInInAddress},
		{"http://192.168.1.20:9091/transmission/rpc?x=1", clients.ErrQueryInAddress},
		{"http://192.168.1.20:9091/?", clients.ErrQueryInAddress},
		{"http://192.168.1.20:9091/#web", clients.ErrQueryInAddress},
		{"http://192.168.1.20:9091#", clients.ErrQueryInAddress},
		{"192.168.1.20:9091", clients.ErrAddressScheme},
		{"ftp://192.168.1.20", clients.ErrAddressScheme},
		{"", clients.ErrAddressScheme},
		{"http://:9091", clients.ErrAddressHost},
		{"http://", clients.ErrAddressHost},
		{"http://192.168.1.20:0", clients.ErrAddressPort},
		{"http://192.168.1.20:65536", clients.ErrAddressPort},
		{"http://fd00::20:8080", clients.ErrAddressNotURL},
		{"http://[fd00::20", clients.ErrAddressNotURL},
	}
	for _, c := range cases {
		if _, err := clients.ParseAddress(c.raw, clients.Transmission); !errors.Is(err, c.want) {
			t.Errorf("ParseAddress(%q) = %v, want %v", c.raw, err, c.want)
		}
	}
}

func TestAddressKeyIsOneForEveryWayToWriteOneClient(t *testing.T) {
	same := [][]string{
		{"http://192.168.1.20:9091", "HTTP://192.168.1.20:9091/transmission/rpc", "http://192.168.1.20:9091/transmission/rpc/"},
		{"http://Transmission.LAN", "http://transmission.lan:80/transmission/rpc"},
		{"https://transmission.lan", "https://transmission.lan:443/"},
		{"http://[FD00::20]:9091", "http://[fd00::20]:9091/transmission/rpc"},
	}
	for _, group := range same {
		first := key(t, group[0])
		for _, raw := range group[1:] {
			if k := key(t, raw); k != first {
				t.Errorf("Key(%q) = %q, want %q as for %q", raw, k, first, group[0])
			}
		}
	}
	apart := [][2]string{
		{"http://transmission.lan", "https://transmission.lan"},
		{"http://transmission.lan:9091", "http://transmission.lan:9092"},
		{"http://transmission.lan/a/transmission/rpc", "http://transmission.lan/b/transmission/rpc"},
		{"http://transmission.lan/Rpc", "http://transmission.lan/rpc"},
	}
	for _, pair := range apart {
		if key(t, pair[0]) == key(t, pair[1]) {
			t.Errorf("%q and %q share a Key", pair[0], pair[1])
		}
	}
}

func TestAddressHostPortNamesWhatTheNodeDials(t *testing.T) {
	cases := map[string]string{
		"http://192.168.1.20:9091": "192.168.1.20:9091",
		"http://transmission.lan":  "transmission.lan:80",
		"https://transmission.lan": "transmission.lan:443",
		"http://[fd00::20]:9091":   "[fd00::20]:9091",
	}
	for raw, want := range cases {
		a, err := clients.ParseAddress(raw, clients.Transmission)
		if err != nil {
			t.Fatal(err)
		}
		if got := a.HostPort(); got != want {
			t.Errorf("HostPort(%q) = %q, want %q", raw, got, want)
		}
	}
}

func TestAddressJoinAddsAPathAfterItsOwn(t *testing.T) {
	a, err := clients.ParseAddress("http://192.168.1.20/qbt", clients.QBittorrent)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := a.Join("/api/v2/auth/login"), "http://192.168.1.20/qbt/api/v2/auth/login"; got != want {
		t.Errorf("Join = %q, want %q", got, want)
	}
}

func TestAddressWebUIIsOnTheAddressHost(t *testing.T) {
	cases := []struct {
		raw  string
		kind clients.Kind
		want string
	}{
		{"http://192.168.1.20:9091/transmission/rpc", clients.Transmission, "http://192.168.1.20:9091/transmission/web/"},
		{"https://192.168.1.20:8080", clients.QBittorrent, "https://192.168.1.20:8080/"},
		{"http://192.168.1.20:8112/json", clients.Deluge, "http://192.168.1.20:8112/"},
	}
	for _, c := range cases {
		a, err := clients.ParseAddress(c.raw, c.kind)
		if err != nil {
			t.Fatal(err)
		}
		if got, ok := a.WebUI(); !ok || got != c.want {
			t.Errorf("WebUI(%q) = %q, %v, want %q", c.raw, got, ok, c.want)
		}
	}
	a, err := clients.ParseAddress("http://192.168.1.20/RPC2", clients.RTorrent)
	if err != nil {
		t.Fatal(err)
	}
	if got, ok := a.WebUI(); ok {
		t.Errorf("rTorrent's WebUI = %q, want none", got)
	}
}

func key(t *testing.T, raw string) string {
	t.Helper()
	a, err := clients.ParseAddress(raw, clients.Transmission)
	if err != nil {
		t.Fatalf("ParseAddress(%q): %v", raw, err)
	}
	return a.Key()
}
