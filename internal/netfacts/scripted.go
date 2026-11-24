package netfacts

import (
	"context"
	"net/netip"
	"os"
	"slices"
	"sync"
	"time"

	"github.com/jatassi/slipstream/internal/netfacts/script"
	"github.com/jatassi/slipstream/internal/netmodel"
)

// Scripted is the Fake world's scripted-network divergence: interfaces,
// the route, SlipStream's own listeners' scope, Docker's ports and the
// firewall come from the scripted readers' file at path, read again
// whenever it changes. Packets and lookups stay host's.
func Scripted(path string, host Env) Env {
	s := &scripted{path: path, host: host}
	env := host
	env.Interfaces, env.Route, env.Self, env.Sockets, env.Firewall = s.interfaces, s.route, s.self, s.sockets, s.firewall
	return env
}

type scripted struct {
	path string
	host Env
	mu   sync.Mutex
	mod  time.Time
	last script.File
}

func (s *scripted) interfaces(context.Context) ([]netmodel.Iface, error) {
	f, err := s.file()
	if err != nil {
		return nil, err
	}
	out := make([]netmodel.Iface, len(f.Interfaces))
	for i, fi := range f.Interfaces {
		out[i] = netmodel.Iface{Name: fi.Name, Index: i + 1, Tunnel: fi.Tunnel}
		for _, a := range fi.Addresses {
			out[i].Addrs = append(out[i].Addrs, netmodel.IfaceAddr{Prefix: a.Prefix, Temporary: tri(a.Temporary), Deprecated: tri(a.Deprecated)})
		}
	}
	return out, nil
}

func (s *scripted) route(context.Context, netip.Addr) (Route, error) {
	f, err := s.file()
	return Route{Iface: f.Egress, Gateway: f.Gateway}, err
}

func (s *scripted) self() []netmodel.Listener {
	f, _ := s.file()
	local := f.Bind == script.BindLocal
	own := s.host.Self()
	for i := range own {
		own[i].V4, own[i].V6 = tri(!local), tri(!local)
		own[i].Loopback = local
	}
	return own
}

func (s *scripted) sockets(ctx context.Context, ports []uint16) ([]netmodel.Listener, error) {
	f, err := s.file()
	if err != nil {
		return nil, err
	}
	listeners, err := s.host.Sockets(ctx, ports)
	for i, l := range listeners {
		listeners[i].Docker = slices.ContainsFunc(f.Firewall, func(r script.Rule) bool { return r.Port == l.Port && r.Policy == script.Docker })
		if l.Loopback {
			listeners[i].V4, listeners[i].V6, listeners[i].Loopback = netmodel.Yes, netmodel.Yes, false
		}
	}
	return listeners, err
}

func (s *scripted) firewall(_ context.Context, qs []Query) ([]netmodel.Rule, error) {
	f, err := s.file()
	if err != nil {
		return nil, err
	}
	web := map[uint16]bool{}
	for _, l := range s.host.Self() {
		web[l.Port] = true
	}
	out := make([]netmodel.Rule, len(qs))
	for i, q := range qs {
		out[i] = netmodel.Rule{Port: q.Port, Family: q.Family}
		fam := map[netmodel.Family]script.Family{netmodel.IPv4: script.IPv4, netmodel.IPv6: script.IPv6}[q.Family]
		j := slices.IndexFunc(f.Firewall, func(r script.Rule) bool {
			return r.Family == fam && (r.Port == q.Port || r.Web && web[q.Port])
		})
		if j >= 0 {
			out[i].Policy = map[script.Policy]netmodel.Policy{script.Allows: netmodel.Allows, script.Blocks: netmodel.Blocks}[f.Firewall[j].Policy]
		}
	}
	return out, nil
}

// file is the script, read again when its modification time changed.
func (s *scripted) file() (script.File, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	info, err := os.Stat(s.path)
	if err != nil {
		return script.File{}, err //nolint:wrapcheck // the path names the file
	}
	if info.ModTime().Equal(s.mod) {
		return s.last, nil
	}
	f, err := script.Read(s.path)
	if err != nil {
		return script.File{}, err
	}
	s.mod, s.last = info.ModTime(), f
	return f, nil
}

func tri(b bool) netmodel.Tri {
	if b {
		return netmodel.Yes
	}
	return netmodel.No
}
