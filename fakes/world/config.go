package world

import (
	"bytes"
	"context"
	"fmt"
	"net"
	"net/netip"
	"path"
	"strings"

	"go.yaml.in/yaml/v3"

	"github.com/jatassi/slipstream/fakes/internal/yamltree"
	"github.com/jatassi/slipstream/fakes/scenario"
	"github.com/jatassi/slipstream/internal/node"
)

// nodeAddress is Node i's own loopback address. Fakes take 127.0.0.1.
func nodeAddress(i int) netip.Addr {
	addr := netip.AddrFrom4([4]byte{127, 0, 0, 2})
	for range i {
		addr = addr.Next()
	}
	return addr
}

// CheckAddresses binds and closes the address of each of n Nodes, and
// returns them. Past the 31 addresses setup-loopback aliases, it refuses
// without trying.
func CheckAddresses(ctx context.Context, n int) ([]netip.Addr, error) {
	addrs := make([]netip.Addr, n)
	for i := range n {
		addr := nodeAddress(i)
		refuse := fmt.Errorf(`%s is not usable on this machine; on macOS run "sudo fakeworld setup-loopback" once`, addr)
		if i >= scenario.MaxNodes {
			return nil, refuse
		}
		l, err := new(net.ListenConfig).Listen(ctx, "tcp", netip.AddrPortFrom(addr, 0).String())
		if err != nil {
			return nil, refuse
		}
		_ = l.Close()
		addrs[i] = addr
	}
	return addrs, nil
}

// listenKeys are the probe's *.listen keys outside timings and endpoints.
func listenKeys(probe node.Resolved) []string {
	var keys []string
	for _, k := range probe.Keys {
		listen, _ := path.Match("*.listen", k.Key)
		if listen && !strings.HasPrefix(k.Key, "timings.") && !strings.HasPrefix(k.Key, "endpoints.") {
			keys = append(keys, k.Key)
		}
	}
	return keys
}

// nodeConfig is the config.yaml of Node i: its seed, every listener on its
// own address at port 0, and the timings and endpoints that differ from
// production.
func nodeConfig(s scenario.Scenario, i int, listen []string, endpoints node.EndpointsBlock) ([]byte, error) {
	root := &yaml.Node{Kind: yaml.MappingNode}
	set(root, "seed", s.Seed+"/"+s.Nodes[i].Name)
	at := netip.AddrPortFrom(nodeAddress(i), 0).String()
	for _, key := range listen {
		set(root, key, at)
	}
	if t := scenario.SetTimings(s.Timings); t != nil {
		root.Content = append(root.Content, scalar("timings"), t)
	}
	e, err := yamltree.Diff(endpoints, node.Defaults().Endpoints)
	if err != nil {
		return nil, err
	}
	if e != nil {
		root.Content = append(root.Content, scalar("endpoints"), e)
	}
	var b bytes.Buffer
	enc := yaml.NewEncoder(&b)
	enc.SetIndent(2)
	if err := enc.Encode(root); err != nil {
		return nil, fmt.Errorf("config.yaml: %w", err)
	}
	return b.Bytes(), nil
}

// set puts value at a dotted key, nesting a mapping for each dot.
func set(m *yaml.Node, key, value string) {
	head, rest, nested := strings.Cut(key, ".")
	for i := 0; i < len(m.Content); i += 2 {
		if m.Content[i].Value == head && nested {
			set(m.Content[i+1], rest, value)
			return
		}
	}
	if !nested {
		m.Content = append(m.Content, scalar(head), scalar(value))
		return
	}
	child := &yaml.Node{Kind: yaml.MappingNode}
	m.Content = append(m.Content, scalar(head), child)
	set(child, rest, value)
}

func scalar(v string) *yaml.Node {
	var n yaml.Node
	if err := n.Encode(v); err != nil {
		panic("world: " + err.Error())
	}
	return &n
}
