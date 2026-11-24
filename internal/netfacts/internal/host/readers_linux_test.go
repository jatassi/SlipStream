package host

import (
	"net/netip"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/jatassi/slipstream/internal/netmodel"
)

// proc writes a /proc tree under a temp root: the TCP tables, bindv6only,
// and process 100, docker-proxy, owning inode 12345.
func proc(t *testing.T, tcp, tcp6, bindv6only string) {
	t.Helper()
	root := t.TempDir()
	for name, data := range map[string]string{
		"proc/net/tcp":                 "  sl  local_address rem_address   st tx_queue rx_queue tr tm->when retrnsmt   uid  timeout inode\n" + tcp,
		"proc/net/tcp6":                "  sl  local_address rem_address   st tx_queue rx_queue tr tm->when retrnsmt   uid  timeout inode\n" + tcp6,
		"proc/sys/net/ipv6/bindv6only": bindv6only + "\n",
		"proc/100/comm":                "docker-proxy\n",
		"proc/net/route":               "Iface\tDestination\tGateway \tFlags\nwg0\t00000000\t00000000\t0001\neth0\t0001A8C0\t00000000\t0001\neth0\t00000000\t0101A8C0\t0003\n",
	} {
		p := filepath.Join(root, name)
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(data), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.MkdirAll(filepath.Join(root, "proc", "100", "fd"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink("socket:[12345]", filepath.Join(root, "proc", "100", "fd", "3")); err != nil {
		t.Fatal(err)
	}
	old := procRoot
	procRoot = root
	t.Cleanup(func() { procRoot = old })
}

func row(local, inode string) string {
	return "   0: " + local + " 00000000:0000 0A 00000000:00000000 00:00000000 00000000  1000        0 " + inode + " 1 0000000000000000 100 0 0 10 0\n"
}

func TestSocketsReadEachPortsFamilies(t *testing.T) {
	const (
		any4    = "00000000"
		local4  = "0100007F"
		any6    = "00000000000000000000000000000000"
		local6  = "00000000000000000000000001000000"
		port    = ":2124" // 8484
		plex    = ":7E90" // 32400
		torrent = ":1AE1" // 6881
	)
	yes, no, unknown := netmodel.Yes, netmodel.No, netmodel.Unknown
	for _, c := range []struct {
		name       string
		tcp, tcp6  string
		bindv6only string
		noIPv6     bool
		want       []netmodel.Listener
	}{
		{
			name: "every kind of listener", bindv6only: "0",
			tcp:  row(any4+port, "1") + row(local4+torrent, "2"),
			tcp6: row(any6+plex, "12345") + row(local6+torrent, "3"),
			want: []netmodel.Listener{
				{Port: 8484, V4: yes, V6: no},
				{Port: 32400, Program: netmodel.Program{Name: "docker-proxy"}, Docker: true, V4: yes, V6: yes},
				{Port: 6881, V4: no, V6: no, Loopback: true},
				{Port: 1, V4: no, V6: no},
			},
		},
		{
			name: "a kernel with IPv6 off", noIPv6: true,
			tcp:  row(any4+port, "1"),
			want: []netmodel.Listener{{Port: 8484, V4: yes, V6: no}, {Port: 32400, V4: no, V6: no}, {Port: 6881, V4: no, V6: no}, {Port: 1, V4: no, V6: no}},
		},
		{
			name: "an IPv6 wildcard where bindv6only is set", bindv6only: "1",
			tcp6: row(any6+plex, "7"),
			want: []netmodel.Listener{{Port: 8484, V4: no, V6: no}, {Port: 32400, V4: unknown, V6: yes}, {Port: 6881, V4: no, V6: no}, {Port: 1, V4: no, V6: no}},
		},
	} {
		t.Run(c.name, func(t *testing.T) {
			proc(t, c.tcp, c.tcp6, c.bindv6only)
			if c.noIPv6 {
				if err := os.Remove(filepath.Join(procRoot, "proc", "net", "tcp6")); err != nil {
					t.Fatal(err)
				}
			}
			got, err := Sockets(t.Context(), []uint16{8484, 32400, 6881, 1})
			if err != nil || !reflect.DeepEqual(got, c.want) {
				t.Errorf("Sockets = %+v, %v; want %+v", got, err, c.want)
			}
		})
	}
}

func TestTheDefaultGatewayIsTheInterfaces(t *testing.T) {
	proc(t, "", "", "0")
	table, err := os.ReadFile(filepath.Join(procRoot, "proc", "net", "route"))
	if err != nil {
		t.Fatal(err)
	}
	for iface, want := range map[string]netip.Addr{"eth0": netip.MustParseAddr("192.168.1.1"), "wg0": {}, "eth1": {}} {
		if got := defaultGateway(table, iface); got != want {
			t.Errorf("%s's gateway = %v, want %v", iface, got, want)
		}
	}
}

// Without privilege no Linux firewall can be read whole, so the reader
// never answers Allows.
func TestTheFirewallReadsUnknown(t *testing.T) {
	for _, fam := range []netmodel.Family{netmodel.IPv4, netmodel.IPv6} {
		if got := Firewall(t.Context(), 32400, fam); got.Policy != netmodel.PolicyUnknown {
			t.Errorf("%v reads %v, want unknown", fam, got.Policy)
		}
	}
}
