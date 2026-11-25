package scenario

import (
	"strings"
	"testing"
)

func networkScenario() Scenario {
	s := singleMachine()
	s.Nodes = append(s.Nodes, Node{Name: "seedbox", Setup: SetupJoin})
	s.Nodes[0].Site, s.Nodes[1].Site = "home", "ams"
	s.Nodes[0].Addresses = []NodeAddress{{Prefix: "192.168.1.20/24"}, {Prefix: "2001:db8:1::20/64", Temporary: true}}
	s.Nodes[0].Firewall = []FirewallRule{{Service: "plex", Family: "ipv6", Policy: "blocks"}}
	s.Nodes[1].Addresses = []NodeAddress{{Prefix: "192.0.2.44/24"}}
	s.Network = &Network{
		Sites: []Site{
			{Name: "home", WAN: "public", Address: "203.0.113.9", Mapping: []string{"pcp", "upnp"}, Prefix6: "2001:db8:1::/64", Forwards: []Forward{{Node: "home", Service: "http", Listed: true}}},
			{Name: "ams", WAN: "none", Address: "192.0.2.44"},
		},
		Proxies: []Proxy{{Name: "the-vps", Address: "198.51.100.8", Routes: []ProxyRoute{{Host: "ui.own.localhost", Node: "home", Service: "http"}}}},
		DNS:     []Record{{Name: "slipstream.open.localhost", A: []string{"203.0.113.9"}, Home: []string{"192.168.1.20"}}},
	}
	return s
}

func TestValidateNetwork(t *testing.T) {
	cases := []struct {
		name   string
		change func(*Scenario)
		want   string
	}{
		{"a valid network", func(*Scenario) {}, ""},
		{"no network block and network fields", func(s *Scenario) { s.Network = nil }, "node home sets network fields, and the Scenario has no network"},
		{"a site named twice", func(s *Scenario) { s.Network.Sites[1].Name = "home" }, "network site home is named twice"},
		{"a node with no site", func(s *Scenario) { s.Nodes[1].Site = "" }, `node seedbox's site "" is not a network site`},
		{"an address outside the site", func(s *Scenario) { s.Nodes[1].Addresses[0].Prefix = "203.0.113.9/24" }, "node seedbox's address 203.0.113.9/24 is not inside site ams"},
		{"an IPv6 address outside the prefix", func(s *Scenario) { s.Nodes[0].Addresses[1].Prefix = "2001:db8:9::20/64" }, "node home's address 2001:db8:9::20/64 is not inside site home"},
		{"a cgnat site with a public gateway", func(s *Scenario) { s.Network.Sites[0].WAN, s.Network.Sites[0].Gateway = "cgnat", "203.0.113.1" }, "network site home is cgnat, so its gateway is inside 100.64.0.0/10"},
		{"a public site with a gateway", func(s *Scenario) { s.Network.Sites[0].Gateway = "100.64.0.1" }, "network site home is public and sets a gateway, which only cgnat and private take"},
		{"upstream on a public site", func(s *Scenario) { s.Network.Sites[0].Upstream = "closed" }, "network site home sets upstream, which only a private site's outer router takes"},
		{"a prefix outside the documentation block", func(s *Scenario) { s.Network.Sites[0].Prefix6 = "2001:db9::/64" }, `network site home's prefix6 "2001:db9::/64" is not a /64 inside 2001:db8::/32`},
		{"a WAN outside the closed set", func(s *Scenario) { s.Network.Sites[0].WAN = "fiber" }, `network site home wan "fiber" is not one of public, cgnat, private, empty, none`},
		{"a forward to no node", func(s *Scenario) { s.Network.Sites[0].Forwards[0].Node = "nas" }, "network site home forwards http to nas, which is not a node"},
		{"a firewall policy outside the closed set", func(s *Scenario) { s.Nodes[0].Firewall[0].Policy = "drop" }, `node home firewall policy "drop" is not one of allows, blocks, unknown, docker`},
		{"a proxy host outside .localhost", func(s *Scenario) { s.Network.Proxies[0].Routes[0].Host = "ui.example.org" }, "network proxy the-vps routes ui.example.org, which is not under .localhost"},
		{"a DNS name outside .localhost", func(s *Scenario) { s.Network.DNS[0].Name = "plex.example.org" }, "network dns name plex.example.org is not under .localhost"},
		{"a DNS answer that isn't an address", func(s *Scenario) { s.Network.DNS[0].A = []string{"atlas"} }, `network dns name slipstream.open.localhost's address "atlas" is not an IP address`},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			s := networkScenario()
			c.change(&s)
			err := s.Validate()
			switch {
			case c.want == "" && err != nil:
				t.Fatalf("Validate = %v, want nil", err)
			case c.want != "" && (err == nil || !strings.Contains(err.Error(), "scenario single-machine: "+c.want)):
				t.Fatalf("Validate = %v, want %s", err, c.want)
			}
		})
	}
}
