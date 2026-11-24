package netfacts_test

import (
	"context"
	"log/slog"
	"net/netip"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"github.com/jatassi/slipstream/internal/netfacts"
	"github.com/jatassi/slipstream/internal/netfacts/script"
	"github.com/jatassi/slipstream/internal/netmodel"
)

// A scripted Node reads its network from the script, and again each time
// the script changes. The world binds every listener to a loopback
// address, so a socket there listens on every network.
func TestAScriptedNodeReadsItsScript(t *testing.T) {
	path := filepath.Join(t.TempDir(), "atlas.json")
	write := func(f script.File, at time.Time) {
		if err := script.Write(path, f); err != nil {
			t.Fatal(err)
		}
		if err := os.Chtimes(path, at, at); err != nil {
			t.Fatal(err)
		}
	}
	host := env()
	host.Self = func() []netmodel.Listener {
		return []netmodel.Listener{{Port: 8484, Program: netmodel.Program{Name: "SlipStream", SlipStream: true}, V4: netmodel.Yes, V6: netmodel.Yes}}
	}
	host.Sockets = func(context.Context, []uint16) ([]netmodel.Listener, error) {
		return []netmodel.Listener{{Port: 32400, V4: netmodel.No, V6: netmodel.No, Loopback: true}}, nil
	}
	n := netfacts.New(netfacts.Config{Env: netfacts.Scripted(path, host), Timings: timings(), Rules: netmodel.Rules{DocumentationGlobal: true}, Log: slog.New(slog.DiscardHandler)})
	yes, no := netmodel.Yes, netmodel.No
	for i, c := range []struct {
		name      string
		file      script.File
		ifaces    []netmodel.Iface
		listeners []netmodel.Listener
		policies  []netmodel.Policy
	}{
		{
			name: "this machine only, behind a tunnel",
			file: script.File{
				Interfaces: []script.Interface{
					{Name: "eth0", Addresses: []script.Address{{Prefix: netip.MustParsePrefix("192.168.1.20/24")}, {Prefix: netip.MustParsePrefix("2001:db8:1::20/64"), Temporary: true}}},
					{Name: "wg0", Tunnel: true, Addresses: []script.Address{{Prefix: netip.MustParsePrefix("10.64.0.2/32")}}},
				},
				Egress: "wg0", Bind: script.BindLocal,
				Firewall: []script.Rule{{Web: true, Family: script.IPv4, Policy: script.Allows}, {Port: 32400, Family: script.IPv6, Policy: script.Blocks}, {Port: 32400, Family: script.IPv4, Policy: script.Docker}},
			},
			ifaces: []netmodel.Iface{
				{Name: "eth0", Index: 1, Addrs: []netmodel.IfaceAddr{{Prefix: netip.MustParsePrefix("192.168.1.20/24"), Class: netmodel.Private, Temporary: no, Deprecated: no}, {Prefix: netip.MustParsePrefix("2001:db8:1::20/64"), Class: netmodel.Global, Temporary: yes, Deprecated: no}}},
				{Name: "wg0", Index: 2, Tunnel: true, Addrs: []netmodel.IfaceAddr{{Prefix: netip.MustParsePrefix("10.64.0.2/32"), Class: netmodel.Private, Temporary: no, Deprecated: no}}},
			},
			listeners: []netmodel.Listener{{Port: 32400, V4: yes, V6: yes, Docker: true}, {Port: 8484, Program: netmodel.Program{Name: "SlipStream", SlipStream: true}, V4: no, V6: no, Loopback: true}},
			policies:  []netmodel.Policy{netmodel.Allows, netmodel.PolicyUnknown, netmodel.PolicyUnknown, netmodel.Blocks},
		},
		{
			name:      "every network after a change",
			file:      script.File{Interfaces: []script.Interface{{Name: "eth0", Addresses: []script.Address{{Prefix: netip.MustParsePrefix("203.0.113.9/24")}}}}, Egress: "eth0", Bind: script.BindAll},
			ifaces:    []netmodel.Iface{{Name: "eth0", Index: 1, Addrs: []netmodel.IfaceAddr{{Prefix: netip.MustParsePrefix("203.0.113.9/24"), Class: netmodel.Global, Temporary: no, Deprecated: no}}}},
			listeners: []netmodel.Listener{{Port: 32400, V4: yes, V6: yes}, {Port: 8484, Program: netmodel.Program{Name: "SlipStream", SlipStream: true}, V4: yes, V6: yes}},
			policies:  []netmodel.Policy{netmodel.PolicyUnknown, netmodel.PolicyUnknown, netmodel.PolicyUnknown, netmodel.PolicyUnknown},
		},
	} {
		write(c.file, time.Now().Add(time.Duration(i+1)*time.Second))
		f, err := n.Facts(t.Context(), []uint16{8484, 32400}, 0, true)
		if err != nil {
			t.Fatal(err)
		}
		var policies []netmodel.Policy
		for _, r := range f.Firewall {
			policies = append(policies, r.Policy)
		}
		if !reflect.DeepEqual(f.Interfaces, c.ifaces) || !reflect.DeepEqual(f.Listeners, c.listeners) || !reflect.DeepEqual(policies, c.policies) {
			t.Errorf("%s: read %+v, %+v and %v; want %+v, %+v and %v", c.name, f.Interfaces, f.Listeners, policies, c.ifaces, c.listeners, c.policies)
		}
	}
}
