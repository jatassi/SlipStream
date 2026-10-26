package world_test

import (
	"context"
	"errors"
	"net"
	"os"
	"path/filepath"
	"slices"
	"sync"
	"testing"
	"time"

	"go.yaml.in/yaml/v3"

	"github.com/jatassi/slipstream/fakes/fake"
	"github.com/jatassi/slipstream/fakes/scenario"
	"github.com/jatassi/slipstream/fakes/world"
	"github.com/jatassi/slipstream/fakes/world/worldtest"
	"github.com/jatassi/slipstream/internal/node"
	"github.com/jatassi/slipstream/internal/node/endpoint"
)

func TestEachNodesConfigNamesItsOwnFronts(t *testing.T) {
	t.Parallel()
	movies := echo
	movies.Endpoints = func(e *node.EndpointsBlock, at fake.Address) { e.TMDB.API = endpoint.MustParseURL(at.URL("/3")) }
	s := stubScenario()
	s.Nodes = append(s.Nodes, scenario.Node{Name: "seedbox"})
	ctx := worldtest.Context(t)
	run, err := world.LaunchWith(ctx, world.Options{Binary: stub{}.build(t), Scenario: s, RunDir: filepath.Join(t.TempDir(), "run"), Origin: worldtest.Origin(t)}, []fake.Spec{movies})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _, _ = run.Stop(ctx) })
	m := run.Manifest()
	for _, n := range m.Nodes {
		i := slices.IndexFunc(m.Fronts, func(f world.Front) bool { return f.Target == "echo" && f.Node == n.Name })
		if i < 0 {
			t.Fatalf("fronts %+v name none for %s", m.Fronts, n.Name)
		}
		if got, want := tmdbAPI(t, filepath.Join(n.DataDir, "config.yaml")), m.Fronts[i].URL+"/3"; got != want {
			t.Errorf("node %s's endpoints.tmdb.api = %s, want its own front %s", n.Name, got, want)
		}
	}
}

func tmdbAPI(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var config struct {
		Endpoints struct {
			TMDB struct {
				API string `yaml:"api"`
			} `yaml:"tmdb"`
		} `yaml:"endpoints"`
	}
	if err := yaml.Unmarshal(data, &config); err != nil {
		t.Fatal(err)
	}
	return config.Endpoints.TMDB.API
}

// udpEcho responds to each datagram on an origin's socket with the
// origin's name, and records it.
var udpEcho = fake.Spec{
	Name:   "udp-echo",
	Scheme: "udp",
	NewUDP: func(scenario.Scenario, fake.Env) (fake.UDPServer, error) { return udpEchoServer{}, nil },
	Endpoints: func(e *node.EndpointsBlock, at fake.Address) {
		e.Torrent.DHTRouters = []endpoint.HostPort{endpoint.MustParseHostPort(at.AddrPort.String())}
	},
	Online: fake.NoOnlineRun("a test fixture serves no real service"),
}

type udpEchoServer struct{}

func (udpEchoServer) ServeUDP(ctx context.Context, sockets map[string]*net.UDPConn) error {
	var served sync.WaitGroup
	for origin, conn := range sockets {
		served.Go(func() {
			buf := make([]byte, 64)
			for {
				_, from, err := conn.ReadFromUDPAddrPort(buf)
				if err != nil {
					return
				}
				_, _ = conn.WriteToUDPAddrPort([]byte(origin), from)
			}
		})
	}
	<-ctx.Done()
	var errs []error
	for _, conn := range sockets {
		errs = append(errs, conn.Close())
	}
	served.Wait()
	return errors.Join(errs...)
}

func TestAUDPFakeGetsASocketForEachNode(t *testing.T) {
	t.Parallel()
	s := stubScenario()
	s.Nodes = append(s.Nodes, scenario.Node{Name: "seedbox"})
	ctx := worldtest.Context(t)
	run, err := world.LaunchWith(ctx, world.Options{Binary: stub{}.build(t), Scenario: s, RunDir: filepath.Join(t.TempDir(), "run"), Origin: worldtest.Origin(t)}, []fake.Spec{udpEcho})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _, _ = run.Stop(ctx) })
	m := run.Manifest()
	for _, n := range m.Nodes {
		i := slices.IndexFunc(m.Fronts, func(f world.Front) bool { return f.Target == "udp-echo" && f.Node == n.Name })
		if i < 0 {
			t.Fatalf("fronts %+v name no socket for %s", m.Fronts, n.Name)
		}
		router := dhtRouter(t, filepath.Join(n.DataDir, "config.yaml"))
		if want := m.Fronts[i].Address; router != want || m.Fronts[i].URL != "udp://"+want {
			t.Errorf("node %s's endpoints.torrent.dht_routers = %s, want its own socket %s at %s", n.Name, router, want, m.Fronts[i].URL)
		}
		if got := echoUDP(t, router); got != n.Name {
			t.Errorf("the socket of %s responds %q, want its origin", n.Name, got)
		}
	}
}

func dhtRouter(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var config struct {
		Endpoints struct {
			Torrent struct {
				DHTRouters []string `yaml:"dht_routers"`
			} `yaml:"torrent"`
		} `yaml:"endpoints"`
	}
	if err := yaml.Unmarshal(data, &config); err != nil {
		t.Fatal(err)
	}
	if len(config.Endpoints.Torrent.DHTRouters) != 1 {
		t.Fatalf("%s names DHT routers %v, want one", path, config.Endpoints.Torrent.DHTRouters)
	}
	return config.Endpoints.Torrent.DHTRouters[0]
}

func echoUDP(t *testing.T, addr string) string {
	t.Helper()
	conn, err := net.Dial("udp", addr)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = conn.Close() }()
	_ = conn.SetDeadline(time.Now().Add(10 * time.Second))
	if _, err := conn.Write([]byte("who")); err != nil {
		t.Fatal(err)
	}
	buf := make([]byte, 64)
	n, err := conn.Read(buf)
	if err != nil {
		t.Fatal(err)
	}
	return string(buf[:n])
}
