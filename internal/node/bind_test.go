package node

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/netip"
	"os"
	"path/filepath"
	"slices"
	"testing"
	"testing/synctest"

	"github.com/jatassi/slipstream/fakes/fault"
	"github.com/jatassi/slipstream/internal/netmodel"
	nodehost "github.com/jatassi/slipstream/internal/node/internal/host"
)

type boundListener struct {
	net.Listener
	addr   netip.AddrPort
	closed *[]string
}

func (l boundListener) Addr() net.Addr { return net.TCPAddrFromAddrPort(l.addr) }

func (l boundListener) Close() error {
	*l.closed = append(*l.closed, l.addr.String())
	return nil
}

var errTaken = errors.New("taken")

// takenAddrs is a listen that refuses taken, records each address tried
// and each listener closed, and binds port 0 as 40000.
func takenAddrs(taken []string, tried, closed *[]string) func(network, address string) (net.Listener, error) {
	return func(_, address string) (net.Listener, error) {
		*tried = append(*tried, address)
		if slices.Contains(taken, address) {
			return nil, errTaken
		}
		ap := netip.MustParseAddrPort(address)
		if ap.Port() == 0 {
			ap = netip.AddrPortFrom(ap.Addr(), 40000)
		}
		return boundListener{addr: ap, closed: closed}, nil
	}
}

func TestBindHTTPKeepsTheFirstPortThatBinds(t *testing.T) {
	for _, c := range []struct {
		name   string
		addr   string
		tries  int
		taken  []string
		tried  []string
		bound  []string
		closed []string
		err    string
		// noIPv6 is a host without ::1.
		noIPv6 bool
	}{
		{name: "a free port", addr: "127.0.0.1:8484", tries: 10, tried: []string{"127.0.0.1:8484"}, bound: []string{"127.0.0.1:8484"}},
		{name: "two taken ports", addr: "127.0.0.1:8484", tries: 10, taken: []string{"127.0.0.1:8484", "127.0.0.1:8485"}, tried: []string{"127.0.0.1:8484", "127.0.0.1:8485", "127.0.0.1:8486"}, bound: []string{"127.0.0.1:8486"}},
		{name: "every port taken", addr: "127.0.0.1:8484", tries: 3, taken: []string{"127.0.0.1:8484", "127.0.0.1:8485", "127.0.0.1:8486"}, tried: []string{"127.0.0.1:8484", "127.0.0.1:8485", "127.0.0.1:8486"}, err: "listen for HTTP on 127.0.0.1, ports 8484 to 8486: taken"},
		{name: "one try", addr: "127.0.0.1:8484", tries: 1, taken: []string{"127.0.0.1:8484"}, tried: []string{"127.0.0.1:8484"}, err: "listen for HTTP on 127.0.0.1:8484: taken"},
		{name: "any port", addr: "127.0.0.1:0", tries: 10, taken: []string{"127.0.0.1:0"}, tried: []string{"127.0.0.1:0"}, err: "listen for HTTP on 127.0.0.1:0: taken"},
		{name: "the last port", addr: "127.0.0.1:65534", tries: 10, taken: []string{"127.0.0.1:65534", "127.0.0.1:65535"}, tried: []string{"127.0.0.1:65534", "127.0.0.1:65535"}, err: "listen for HTTP on 127.0.0.1, ports 65534 to 65535: taken"},
		{name: "all networks", addr: "[::]:8484", tries: 10, tried: []string{"[::]:8484"}, bound: []string{"[::]:8484"}},
		{name: "this machine only", addr: "[::1]:8484", tries: 10, tried: []string{"[::1]:8484", "127.0.0.1:8484"}, bound: []string{"[::1]:8484", "127.0.0.1:8484"}},
		{
			name: "this machine only with the port taken on one loopback", addr: "[::1]:8484", tries: 10, taken: []string{"127.0.0.1:8484"},
			tried: []string{"[::1]:8484", "127.0.0.1:8484", "[::1]:8485", "127.0.0.1:8485"}, bound: []string{"[::1]:8485", "127.0.0.1:8485"}, closed: []string{"[::1]:8484"},
		},
		{name: "this machine only on a host without IPv6", addr: "[::1]:8484", tries: 10, noIPv6: true, tried: []string{"127.0.0.1:8484"}, bound: []string{"127.0.0.1:8484"}},
		{name: "this machine only on any port", addr: "[::1]:0", tries: 10, tried: []string{"[::1]:0", "127.0.0.1:40000"}, bound: []string{"[::1]:40000", "127.0.0.1:40000"}},
		{
			name: "this machine only on any port taken on the other loopback", addr: "[::1]:0", tries: 10, taken: []string{"127.0.0.1:40000"},
			tried: []string{"[::1]:0", "127.0.0.1:40000"}, closed: []string{"[::1]:40000"}, err: "listen for HTTP on [::1]:0: taken",
		},
	} {
		t.Run(c.name, func(t *testing.T) {
			var tried, closed []string
			lns, err := bindHTTP(takenAddrs(c.taken, &tried, &closed), netip.MustParseAddrPort(c.addr), c.tries, !c.noIPv6)
			if !slices.Equal(tried, c.tried) || !slices.Equal(closed, c.closed) {
				t.Fatalf("tried %v and closed %v, want %v and %v", tried, closed, c.tried, c.closed)
			}
			if c.err != "" {
				if err == nil || err.Error() != c.err || !errors.Is(err, errTaken) {
					t.Fatalf("err = %v, want %s", err, c.err)
				}
				return
			}
			var bound []string
			for _, ln := range lns {
				bound = append(bound, ln.Addr().String())
			}
			if err != nil || !slices.Equal(bound, c.bound) {
				t.Fatalf("bound %v, %v; want %v", bound, err, c.bound)
			}
		})
	}
}

// Go opens [::] dual-stack, so its families come from IPV6_V6ONLY, not
// its address. A host with no IPv6 opens it as IPv4 alone and binds
// This machine only on 127.0.0.1 alone.
func TestTheWebListenersReadTheirFamilies(t *testing.T) {
	listen := func(network, address string) (net.Listener, error) {
		return (&net.ListenConfig{}).Listen(t.Context(), network, address)
	}
	probe, err := listen("tcp", "[::1]:0")
	ipv6 := err == nil
	if ipv6 {
		_ = probe.Close()
	}
	for _, c := range []struct {
		addr string
		want []nodehost.Families
	}{
		{"[::]:0", []nodehost.Families{{IPv4: true, IPv6: ipv6}}},
		{"[::1]:0", map[bool][]nodehost.Families{true: {{IPv6: true}, {IPv4: true}}, false: {{IPv4: true}}}[ipv6]},
		{"127.0.0.1:0", []nodehost.Families{{IPv4: true}}},
	} {
		lns, err := bindHTTP(listen, netip.MustParseAddrPort(c.addr), 1, ipv6)
		if (err != nil) != (c.want == nil) {
			t.Fatalf("binding %s: %v", c.addr, err)
		}
		var got []nodehost.Families
		for _, ln := range lns {
			tcp, _ := ln.(*net.TCPListener)
			f, err := nodehost.FamiliesOf(tcp)
			if err != nil {
				t.Fatal(err)
			}
			got = append(got, f)
			if boundTCP(ln).Port() != boundTCP(lns[0]).Port() {
				t.Errorf("%s bound %s and %s, want one port", c.addr, lns[0].Addr(), ln.Addr())
			}
			_ = ln.Close()
		}
		if !slices.Equal(got, c.want) {
			t.Errorf("%s accepts %+v, want %+v", c.addr, got, c.want)
		}
	}
}

// fallbackEnv puts http.listen in config.yaml and the Node link on 8485,
// the port after it.
func fallbackEnv(t *testing.T, portsToTry string) Env {
	t.Helper()
	env, _ := testEnv(t)
	env.Environ = []string{"SLIPSTREAM_TORRENT_LISTEN=127.0.0.1:0", "SLIPSTREAM_LINK_LISTEN=127.0.0.1:8485", lowScrypt}
	writeFile(t, filepath.Join(env.DataDir.Path(), "config.yaml"), "http:\n  listen: 127.0.0.1:8484\n  ports_to_try: "+portsToTry+"\n")
	return env
}

func TestATakenWebPortFallsBackPastTheNodesOwnPorts(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		env, host := inMemory(fallbackEnv(t, "3"))
		held, err := host.Listen("tcp", "127.0.0.1:8484")
		if err != nil {
			t.Fatal(err)
		}
		defer func() { _ = held.Close() }()
		n, c := startOn(t, env, host)

		if got := n.Ready().Listeners.HTTP.String(); got != "127.0.0.1:8486" {
			t.Fatalf("listeners.http = %s, want 127.0.0.1:8486", got)
		}
		key := keyOf(t, c, c.claim(n.Ready().SetupCode, "1234"))
		system := decode[struct {
			LogsDir string `json:"logsDir"`
			HTTP    struct {
				Configured string `json:"configured"`
			} `json:"http"`
			Listeners struct {
				HTTP string `json:"http"`
			} `json:"listeners"`
		}](t, c.admin(key, http.MethodGet, "/api/v1/system", ""), http.StatusOK)
		if system.LogsDir != env.DataDir.LogsDir() || system.HTTP.Configured != "127.0.0.1:8484" || system.Listeners.HTTP != "127.0.0.1:8486" {
			t.Fatalf("getSystem = %+v, want logsDir %s, http.configured 127.0.0.1:8484, listeners.http 127.0.0.1:8486", system, env.DataDir.LogsDir())
		}

		logged, err := os.ReadFile(filepath.Join(env.DataDir.LogsDir(), "slipstream.log"))
		if err != nil {
			t.Fatal(err)
		}
		var warned []string
		for _, r := range records(t, string(logged)) {
			if r["msg"] == "the web port was taken, so the web server listens on the next free port" {
				warned = append(warned, fmt.Sprintf("%v %v %v", r["level"], r["configured"], r["bound"]))
			}
		}
		if want := []string{"WARN 127.0.0.1:8484 127.0.0.1:8486"}; !slices.Equal(warned, want) {
			t.Fatalf("warnings = %v, want %v", warned, want)
		}
	})
}

func TestWhereANodeListensWaitsForNoRestart(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		env, host := inMemory(fallbackEnv(t, "3"))
		held, err := host.Listen("tcp", "127.0.0.1:8484")
		if err != nil {
			t.Fatal(err)
		}
		defer func() { _ = held.Close() }()
		n, c := startOn(t, env, host)
		key := claimKey(t, n, c)
		path := "/api/v1/nodes/" + selfID(t, c, key) + "/config"

		for _, step := range []struct{ body, want string }{
			{"", "127.0.0.1:8484 127.0.0.1:8486 false restart"},
			{`{"http.listen":8486}`, "127.0.0.1:8486 127.0.0.1:8486 false restart"},
			{`{"http.listen":9000}`, "127.0.0.1:9000 127.0.0.1:8486 true restart"},
			{`{"http.listen":"[::1]:9000"}`, "[::1]:9000 127.0.0.1:8486 true restart"},
			{`{"http.listen":9001}`, "[::1]:9001 127.0.0.1:8486 true restart"},
			{`{"http.listen":"[::]:9001"}`, "[::]:9001 127.0.0.1:8486 true restart"},
		} {
			r := c.admin(key, http.MethodGet, path, "")
			if step.body != "" {
				r = c.admin(key, http.MethodPatch, path, step.body)
			}
			if got := fieldLine(fieldOf(t, r, "http.listen")); got != step.want {
				t.Fatalf("http.listen after %q = %s, want %s", step.body, got, step.want)
			}
		}
	})
}

func TestARestartAfterAFallbackMovesToTheConfiguredPort(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		env, host := inMemory(fallbackEnv(t, "3"))
		env.Restart = func() error { return nil }
		held, err := host.Listen("tcp", "127.0.0.1:8484")
		if err != nil {
			t.Fatal(err)
		}
		defer func() { _ = held.Close() }()
		n, c := startOn(t, env, host)
		key := claimKey(t, n, c)

		r := c.admin(key, http.MethodPost, "/api/v1/nodes/"+selfID(t, c, key)+"/restart", "")
		accepted := decode[struct {
			BoundPort uint16 `json:"boundPort"`
			NextPort  uint16 `json:"nextPort"`
		}](t, r, http.StatusAccepted)
		if accepted.BoundPort != 8486 || accepted.NextPort != 8484 {
			t.Fatalf("restartNode = %s, want boundPort 8486 and nextPort 8484", r.body)
		}
		<-n.Done()
	})
}

func TestOnePortToTryFailsTheStart(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		env, host := inMemory(fallbackEnv(t, "1"))
		held, err := host.Listen("tcp", "127.0.0.1:8484")
		if err != nil {
			t.Fatal(err)
		}
		defer func() { _ = held.Close() }()
		_, err = Start(t.Context(), env)
		want := "listen for HTTP on 127.0.0.1:8484: listen tcp 127.0.0.1:8484: address already in use"
		if err == nil || err.Error() != want || ExitCode(err) != ExitFailed {
			t.Fatalf("start = %v, want %s with exit %d", err, want, ExitFailed)
		}
	})
}

// This machine only binds ::1 beside 127.0.0.1 only on a host that has
// it, so a host with IPv6 turned off still starts.
func TestThisMachineOnlyBindsTheLoopbacksTheHostHas(t *testing.T) {
	for _, c := range []struct {
		name  string
		addrs []netip.Addr
		want  string
	}{
		{"a host with IPv6", []netip.Addr{netip.MustParseAddr("127.0.0.1"), netip.IPv6Loopback()}, "[::1]:8484"},
		{"a host with IPv6 turned off", []netip.Addr{netip.MustParseAddr("127.0.0.1")}, "127.0.0.1:8484"},
	} {
		t.Run(c.name, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				env := fallbackEnv(t, "3")
				writeFile(t, filepath.Join(env.DataDir.Path(), "config.yaml"), "http:\n  listen: \"[::1]:8484\"\n")
				host := fault.NewNetwork().Host(c.addrs...)
				env = inMemoryOn(env, host)
				env.Network.Interfaces = func(context.Context) ([]netmodel.Iface, error) {
					lo := netmodel.Iface{Name: "lo"}
					for _, a := range c.addrs {
						lo.Addrs = append(lo.Addrs, netmodel.IfaceAddr{Prefix: netip.PrefixFrom(a, a.BitLen())})
					}
					return []netmodel.Iface{lo}, nil
				}
				n, _ := startOn(t, env, host)
				if got := n.Ready().Listeners.HTTP.String(); got != c.want {
					t.Errorf("listeners.http = %s, want %s", got, c.want)
				}
			})
		})
	}
}
