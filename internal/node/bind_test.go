package node

import (
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
)

type boundListener struct {
	net.Listener
	addr netip.AddrPort
}

var errTaken = errors.New("taken")

// takenPorts is a listen that refuses taken and records each port tried.
func takenPorts(taken []uint16, tried *[]uint16) func(network, address string) (net.Listener, error) {
	return func(_, address string) (net.Listener, error) {
		ap := netip.MustParseAddrPort(address)
		*tried = append(*tried, ap.Port())
		if slices.Contains(taken, ap.Port()) {
			return nil, errTaken
		}
		return boundListener{addr: ap}, nil
	}
}

func TestBindHTTPKeepsTheFirstPortThatBinds(t *testing.T) {
	for _, c := range []struct {
		name  string
		addr  string
		tries int
		taken []uint16
		want  []uint16
		bound uint16
		err   string
	}{
		{name: "a free port", addr: "127.0.0.1:8484", tries: 10, want: []uint16{8484}, bound: 8484},
		{name: "two taken ports", addr: "127.0.0.1:8484", tries: 10, taken: []uint16{8484, 8485}, want: []uint16{8484, 8485, 8486}, bound: 8486},
		{name: "every port taken", addr: "127.0.0.1:8484", tries: 3, taken: []uint16{8484, 8485, 8486}, want: []uint16{8484, 8485, 8486}, err: "listen for HTTP on 127.0.0.1, ports 8484 to 8486: taken"},
		{name: "one try", addr: "127.0.0.1:8484", tries: 1, taken: []uint16{8484}, want: []uint16{8484}, err: "listen for HTTP on 127.0.0.1:8484: taken"},
		{name: "any port", addr: "127.0.0.1:0", tries: 10, taken: []uint16{0}, want: []uint16{0}, err: "listen for HTTP on 127.0.0.1:0: taken"},
		{name: "the last port", addr: "127.0.0.1:65534", tries: 10, taken: []uint16{65534, 65535}, want: []uint16{65534, 65535}, err: "listen for HTTP on 127.0.0.1, ports 65534 to 65535: taken"},
	} {
		t.Run(c.name, func(t *testing.T) {
			var tried []uint16
			ln, err := bindHTTP(takenPorts(c.taken, &tried), netip.MustParseAddrPort(c.addr), c.tries)
			if !slices.Equal(tried, c.want) {
				t.Fatalf("tried %v, want %v", tried, c.want)
			}
			if c.err != "" {
				if err == nil || err.Error() != c.err || !errors.Is(err, errTaken) {
					t.Fatalf("err = %v, want %s", err, c.err)
				}
				return
			}
			if got, _ := ln.(boundListener); err != nil || got.addr.Port() != c.bound {
				t.Fatalf("bound %v, %v; want port %d", ln, err, c.bound)
			}
		})
	}
}

// fallbackEnv puts http.listen in config.yaml and the Node link on 8485,
// the port after it.
func fallbackEnv(t *testing.T, portsToTry string) Env {
	t.Helper()
	env, _ := testEnv(t)
	env.Environ = []string{"SLIPSTREAM_TORRENT_LISTEN=127.0.0.1:0", "SLIPSTREAM_LINK_LISTEN=127.0.0.1:8485"}
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
		key := decode[struct {
			APIKey string `json:"apiKey"`
		}](t, c.claim(n.Ready().SetupCode, "1234"), http.StatusCreated).APIKey
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
			if r["msg"] == "web port taken, serving on a later one" {
				warned = append(warned, fmt.Sprintf("%v %v %v", r["level"], r["configured"], r["bound"]))
			}
		}
		if want := []string{"WARN 127.0.0.1:8484 127.0.0.1:8486"}; !slices.Equal(warned, want) {
			t.Fatalf("warnings = %v, want %v", warned, want)
		}
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
