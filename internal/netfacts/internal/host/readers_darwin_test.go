package host

import (
	"net"
	"net/netip"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/jatassi/slipstream/internal/netmodel"
)

func TestNetstatListening(t *testing.T) {
	const header = "Active Internet connections (including servers)\n" +
		"Proto Recv-Q Send-Q  Local Address          Foreign Address        (state)      rhiwat  shiwat    pid   epid state  options           gencnt    flags   flags1 usscnt rtncnt fltrs\n"
	const named = "Active Internet connections (including servers)\n" +
		"Proto Recv-Q Send-Q  Local Address          Foreign Address        (state)       rxbytes      txbytes  rhiwat  shiwat    process:pid   state  options\n"
	for _, c := range []struct {
		name, out string
		want      []netstatRow
	}{
		{
			name: "pid column",
			out: header +
				"tcp46      0      0  *.32400                *.*                    LISTEN       131072  131072    412      0 0x0100 0x00000006 0000000000000a1b 00000000 00000800      1      0 000001\n" +
				"tcp4       0      0  127.0.0.1.8484         *.*                    LISTEN       131072  131072    977      0 0x0100 0x00000006 0000000000000a1c 00000000 00000800      1      0 000001\n" +
				"tcp6       0      0  *.8484                 *.*                    LISTEN       131072  131072    977      0 0x0100 0x00000006 0000000000000a1d 00000000 00000800      1      0 000001\n" +
				"tcp6       0      0  fe80::1%lo0.631        *.*                    LISTEN       131072  131072    55       0 0x0100 0x00000006 0000000000000a1e 00000000 00000800      1      0 000001\n" +
				"tcp4       0      0  10.0.0.5.52144         17.57.144.20.5223      ESTABLISHED  131072  131072    300      0 0x0102 0x00000020 0000000000000a1f 00000000 00000800      1      0 000001\n",
			want: []netstatRow{
				{socket{local: netip.MustParseAddrPort("[::]:32400"), v4: netmodel.Yes}, 412},
				{socket{local: netip.MustParseAddrPort("127.0.0.1:8484"), v4: netmodel.No}, 977},
				{socket{local: netip.MustParseAddrPort("[::]:8484"), v4: netmodel.No}, 977},
				{socket{local: netip.MustParseAddrPort("[fe80::1]:631"), v4: netmodel.No}, 55},
			},
		},
		{
			name: "process:pid column",
			out:  named + "tcp4       0      0  *.8484                 *.*                    LISTEN              0            0  131072  131072  slipstream:977 00000 00000\n",
			want: []netstatRow{{socket{local: netip.MustParseAddrPort("0.0.0.0:8484"), v4: netmodel.No}, 977}},
		},
	} {
		t.Run(c.name, func(t *testing.T) {
			if got := netstatListening([]byte(c.out)); !reflect.DeepEqual(got, c.want) {
				t.Errorf("netstatListening = %+v, want %+v", got, c.want)
			}
		})
	}
}

func TestApplicationFirewall(t *testing.T) {
	const (
		on     = "Firewall is enabled. (State = 1)\n"
		signed = "Automatically allow built-in signed software ENABLED.\nAutomatically allow downloaded signed software ENABLED.\n"
		apple  = "Automatically allow built-in signed software ENABLED.\nAutomatically allow downloaded signed software DISABLED.\n"
		open   = "Firewall has block all state set to disabled.\n"
		app    = "/Applications/SlipStream.app/Contents/MacOS/slipstream"
	)
	allows, blocks, unknown := netmodel.Allows, netmodel.Blocks, netmodel.PolicyUnknown
	for _, c := range []struct {
		name                     string
		state, blockAll, allowed string
		path, appBlocked         string
		builtIn, downloaded      bool
		want                     netmodel.Policy
	}{
		{name: "off", state: "Firewall is disabled. (State = 0)\n", blockAll: open, allowed: signed, path: app, want: allows},
		{name: "block all by state", state: "Firewall is enabled. (State = 2)\n", blockAll: open, allowed: signed, path: app, want: blocks},
		{name: "block all by setting", state: on, blockAll: "Firewall has block all state set to enabled.\n", allowed: signed, path: app, want: blocks},
		{name: "a permitted entry", state: on, blockAll: open, allowed: apple, path: app, appBlocked: "The application " + app + " is permitted\n", want: allows},
		{name: "a blocked entry", state: on, blockAll: open, allowed: signed, path: app, appBlocked: "The application " + app + " is blocked\n", downloaded: true, want: blocks},
		{name: "no entry, Developer ID signed", state: on, blockAll: open, allowed: signed, path: app, appBlocked: "The application " + app + " is not part of the firewall\n", downloaded: true, want: allows},
		{name: "no entry, signed, downloaded software not allowed", state: on, blockAll: open, allowed: apple, path: app, downloaded: true, want: blocks},
		{name: "no entry, unsigned", state: on, blockAll: open, allowed: signed, path: app, want: blocks},
		{name: "the listener unread", state: on, blockAll: open, allowed: signed, want: unknown},
	} {
		t.Run(c.name, func(t *testing.T) {
			a := parseALFApp(c.path, c.appBlocked)
			a.BuiltIn, a.Downloaded = c.builtIn, c.downloaded
			if c.path == "" {
				a = alfApp{}
			}
			if got, _ := alfDecide(parseALFGlobal(c.state, c.blockAll, c.allowed), a, 8484); got != c.want {
				t.Errorf("alfDecide = %v, want %v", got, c.want)
			}
		})
	}
}

func TestMacOSReaders(t *testing.T) {
	exe, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	ln, err := (&net.ListenConfig{}).Listen(t.Context(), "tcp4", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = ln.Close() })
	port := netip.MustParseAddrPort(ln.Addr().String()).Port()
	got, err := Sockets(t.Context(), []uint16{port})
	if err != nil || len(got) != 1 {
		t.Fatalf("Sockets = %+v, %v", got, err)
	}
	if exe, err = filepath.EvalSymlinks(exe); err != nil {
		t.Fatal(err)
	}
	owner, _ := filepath.EvalSymlinks(got[0].Program.Name)
	if !got[0].Loopback || owner != exe {
		t.Errorf("Sockets = %+v, want a loopback listener owned by %s", got, exe)
	}
	rules, err := Firewall(t.Context(), "", []netmodel.Rule{{Port: port, Family: netmodel.IPv4}})
	if err != nil || len(rules) != 1 || rules[0].Tool != netmodel.SocketFilter || rules[0].Because == "" {
		t.Errorf("Firewall = %+v, %v; want an answer from socketfilterfw", rules, err)
	}
}
