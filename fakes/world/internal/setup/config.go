// Package setup is what both worlds, the process world and the simulation
// tier, do to a Scenario's Nodes: the config.yaml each runs with, the setup
// steps once they are ready, and each one's manifest entry.
package setup

import (
	"bytes"
	"fmt"
	"net/netip"
	"path"
	"strings"

	"go.yaml.in/yaml/v3"

	"github.com/jatassi/slipstream/fakes/internal/yamltree"
	"github.com/jatassi/slipstream/fakes/scenario"
	"github.com/jatassi/slipstream/internal/contenttype"
	"github.com/jatassi/slipstream/internal/install"
	"github.com/jatassi/slipstream/internal/node"
)

// Address is Node i's loopback address, 127.0.0.(i+2).
func Address(i int) netip.Addr {
	addr := netip.AddrFrom4([4]byte{127, 0, 0, 2})
	for range i {
		addr = addr.Next()
	}
	return addr
}

// ListenKeys is every listen key of the binary's config, from a probe of
// its keys.
func ListenKeys(probe node.Resolved) []string {
	var keys []string
	for _, k := range probe.Keys {
		listen, _ := path.Match("*.listen", k.Key)
		if listen && !strings.HasPrefix(k.Key, "timings.") && !strings.HasPrefix(k.Key, "endpoints.") {
			keys = append(keys, k.Key)
		}
	}
	return keys
}

// Config is Node i's config.yaml: its seed and name, every listen key on
// its own address, the timings the Scenario sets and the endpoints the
// fakes moved. A world of fronts passes the Node's link front, which the
// Node advertises. The simulation tier, which has no fronts, passes the
// zero link.
func Config(s scenario.Scenario, i int, listen []string, link netip.AddrPort, endpoints node.EndpointsBlock) ([]byte, error) {
	root := &yaml.Node{Kind: yaml.MappingNode}
	set(root, "seed", scalar(s.Seed+"/"+s.Nodes[i].Name))
	set(root, "name", scalar(s.Nodes[i].Name))
	at := scalar(netip.AddrPortFrom(Address(i), 0).String())
	for _, key := range listen {
		set(root, key, at)
	}
	if link.IsValid() {
		set(root, "link.advertise", &yaml.Node{Kind: yaml.SequenceNode, Content: []*yaml.Node{scalar(link.String())}})
	}
	if t := scenario.SetTimings(s.Timings); t != nil {
		root.Content = append(root.Content, scalar("timings"), t)
	}
	e, err := yamltree.Diff(endpoints, node.Defaults(contenttype.Set{}, install.Dev).Endpoints)
	if err != nil {
		return nil, err
	}
	if e != nil {
		root.Content = append(root.Content, scalar("endpoints"), e)
	}
	return encode(root)
}

// Pin is config with each listen key whose listener is in listeners, by
// name such as http, at the address that listener bound, so a Node
// started again on it binds the same ports.
func Pin(config []byte, listeners map[string]string) ([]byte, error) {
	var doc yaml.Node
	if err := yaml.Unmarshal(config, &doc); err != nil {
		return nil, fmt.Errorf("config.yaml: %w", err)
	}
	root := doc.Content[0]
	for i := 0; i < len(root.Content); i += 2 {
		bound, ok := listeners[root.Content[i].Value]
		if !ok || root.Content[i+1].Kind != yaml.MappingNode {
			continue
		}
		block := root.Content[i+1]
		for j := 0; j < len(block.Content); j += 2 {
			if block.Content[j].Value == "listen" {
				block.Content[j+1] = scalar(bound)
			}
		}
	}
	return encode(root)
}

func encode(root *yaml.Node) ([]byte, error) {
	var b bytes.Buffer
	enc := yaml.NewEncoder(&b)
	enc.SetIndent(2)
	if err := enc.Encode(root); err != nil {
		return nil, fmt.Errorf("config.yaml: %w", err)
	}
	return b.Bytes(), nil
}

func set(m *yaml.Node, key string, value *yaml.Node) {
	head, rest, nested := strings.Cut(key, ".")
	for i := 0; i < len(m.Content); i += 2 {
		if m.Content[i].Value == head && nested {
			set(m.Content[i+1], rest, value)
			return
		}
	}
	if !nested {
		m.Content = append(m.Content, scalar(head), value)
		return
	}
	child := &yaml.Node{Kind: yaml.MappingNode}
	m.Content = append(m.Content, scalar(head), child)
	set(child, rest, value)
}

func scalar(v string) *yaml.Node {
	var n yaml.Node
	if err := n.Encode(v); err != nil {
		panic("setup: " + err.Error())
	}
	return &n
}
