package host

import (
	"testing"

	"github.com/jatassi/slipstream/internal/netmodel"
)

// The recorded run on a Windows 11 Plex server: Private and Public both
// active, each on, blocking inbound by default.
func TestWindowsFirewallRules(t *testing.T) {
	const (
		plex   = `C:\Program Files\Plex\Plex Media Server\Plex Media Server.exe`
		v1     = `C:\users\jacks\appdata\local\slipstream\slipstream.exe`
		v2     = `C:\Program Files\SlipStream\slipstream.exe`
		all    = fwProfile(2147483647)
		recent = fwPrivate | fwPublic
	)
	recorded := []fwProfileState{{Profile: fwPrivate, On: true}, {Profile: fwPublic, On: true}}
	plexRule := fwRule{Name: "Plex Media Server", Enabled: true, Inbound: true, Allow: true, Profiles: all, Protocol: fwAnyProtocol, RemoteAddresses: "*", Program: plex}
	v1Rule := fwRule{Name: "slipstream.exe", Enabled: true, Inbound: true, Allow: true, Profiles: recent, Protocol: fwTCP, LocalPorts: "*", RemoteAddresses: "*", Program: v1}
	allow := func(edit func(*fwRule)) fwRule {
		r := fwRule{Name: "rule", Enabled: true, Inbound: true, Allow: true, Profiles: all, Protocol: fwTCP, RemoteAddresses: "*"}
		edit(&r)
		return r
	}
	var packaged []fwRule
	for range 32 {
		packaged = append(packaged, allow(func(r *fwRule) { r.Protocol, r.Packaged = fwAnyProtocol, true }))
	}
	recordedRules := append([]fwRule{plexRule, v1Rule}, packaged...)
	plexOn := fwAsk{Port: 32400, Family: netmodel.IPv4, Program: plex}
	v2On := fwAsk{Port: 8484, Family: netmodel.IPv4, Program: v2}
	allows, blocks, unknown := netmodel.Allows, netmodel.Blocks, netmodel.PolicyUnknown

	for _, c := range []struct {
		name     string
		profiles []fwProfileState
		rules    []fwRule
		ask      fwAsk
		want     netmodel.Policy
	}{
		{"Plex's program rules allow Plex", recorded, recordedRules, plexOn, allows},
		{"Plex's program rules allow Plex on IPv6", recorded, recordedRules, fwAsk{Port: 32400, Family: netmodel.IPv6, Program: plex}, allows},
		{"the 32 package-owned rules don't count", recorded, recordedRules, v2On, blocks},
		{"v1's program rule doesn't cover v2 at another path", recorded, []fwRule{v1Rule}, v2On, blocks},
		{"a program rule matches the path in any case", recorded, []fwRule{v1Rule}, fwAsk{Port: 8484, Family: netmodel.IPv4, Program: `C:\Users\Jacks\AppData\Local\SlipStream\slipstream.exe`}, allows},
		{"a package rule limited to a port counts", recorded, []fwRule{allow(func(r *fwRule) { r.Packaged, r.LocalPorts = true, "8484" })}, v2On, allows},
		{"LocalSubnet never counts", recorded, []fwRule{allow(func(r *fwRule) { r.RemoteAddresses = "LocalSubnet" })}, v2On, blocks},
		{"private ranges never count", recorded, []fwRule{allow(func(r *fwRule) { r.RemoteAddresses = "10.0.0.0/255.0.0.0,fd00::/8,LocalSubnet" })}, v2On, blocks},
		{"the Internet keyword counts", recorded, []fwRule{allow(func(r *fwRule) { r.RemoteAddresses = "LocalSubnet,Internet" })}, v2On, allows},
		{"0.0.0.0/0 counts", recorded, []fwRule{allow(func(r *fwRule) { r.RemoteAddresses = "0.0.0.0/0.0.0.0" })}, v2On, allows},
		{"one global address may not be the tester", recorded, []fwRule{allow(func(r *fwRule) { r.RemoteAddresses = "5.161.67.8" })}, v2On, unknown},
		{"an IPv4 range doesn't cover IPv6", recorded, []fwRule{allow(func(r *fwRule) { r.RemoteAddresses = "1.0.0.0-9.255.255.255" })}, fwAsk{Port: 8484, Family: netmodel.IPv6, Program: v2}, blocks},
		{"a port range covers the port", recorded, []fwRule{allow(func(r *fwRule) { r.LocalPorts = "80,8000-9000" })}, v2On, allows},
		{"a port keyword names no listener's port", recorded, []fwRule{allow(func(r *fwRule) { r.LocalPorts = "RPC" })}, v2On, blocks},
		{"a UDP rule doesn't count", recorded, []fwRule{allow(func(r *fwRule) { r.Protocol = 17 })}, v2On, blocks},
		{"a disabled rule doesn't count", recorded, []fwRule{allow(func(r *fwRule) { r.Enabled = false })}, v2On, blocks},
		{"an outbound rule doesn't count", recorded, []fwRule{allow(func(r *fwRule) { r.Inbound = false })}, v2On, blocks},
		{"a program rule with the listener unread may apply", recorded, []fwRule{plexRule}, fwAsk{Port: 32400, Family: netmodel.IPv4}, unknown},
		{"a service rule may apply", recorded, []fwRule{allow(func(r *fwRule) { r.Service = "PlexUpdateService" })}, v2On, unknown},
		{"block over allow", recorded, []fwRule{plexRule, allow(func(r *fwRule) { r.Allow, r.LocalPorts = false, "32400" })}, plexOn, blocks},
		{"a block that may apply leaves an allow unknown", recorded, []fwRule{plexRule, allow(func(r *fwRule) { r.Allow, r.RemoteAddresses = false, "203.0.113.0/24" })}, plexOn, unknown},
		{"block all overrides every rule", []fwProfileState{{Profile: fwPrivate, On: true, BlockAll: true}}, recordedRules, plexOn, blocks},
		{"a profile that is off allows", []fwProfileState{{Profile: fwPrivate}}, nil, v2On, allows},
		{"a profile that allows by default allows", []fwProfileState{{Profile: fwPrivate, On: true, DefaultAllow: true}}, nil, v2On, allows},
		{"a rule outside the egress profile doesn't count", []fwProfileState{{Profile: fwPublic, On: true}}, []fwRule{allow(func(r *fwRule) { r.Profiles = fwPrivate })}, v2On, blocks},
		{"profiles that disagree read unknown", recorded, []fwRule{allow(func(r *fwRule) { r.Profiles = fwPrivate })}, v2On, unknown},
	} {
		t.Run(c.name, func(t *testing.T) {
			got := fwDecide(c.profiles, c.rules, c.ask)
			if got.Policy != c.want || got.Tool != netmodel.WindowsDefender || got.Port != c.ask.Port || got.Family != c.ask.Family {
				t.Errorf("fwDecide = %+v, want %v", got, c.want)
			}
		})
	}
}
