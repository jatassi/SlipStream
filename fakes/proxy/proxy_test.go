package proxy_test

import (
	"io"
	"net/http"
	"net/http/httptest"
	"net/netip"
	"strconv"
	"strings"
	"testing"

	"github.com/jatassi/slipstream/fakes/fake"
	"github.com/jatassi/slipstream/fakes/fake/faketest"
	"github.com/jatassi/slipstream/fakes/fault"
	"github.com/jatassi/slipstream/fakes/proxy"
	"github.com/jatassi/slipstream/fakes/scenario"
	"github.com/jatassi/slipstream/internal/node"
)

func echo(t *testing.T, what string) netip.AddrPort {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.WriteString(w, what+" host="+r.Host+" xff="+r.Header.Get("X-Forwarded-For")) //nolint:gosec // G705: a test server answering a test client
	}))
	t.Cleanup(srv.Close)
	return netip.MustParseAddrPort(srv.Listener.Addr().String())
}

func TestTheProxyRoutesEachHostForEachTester(t *testing.T) {
	s := scenario.Scenario{
		Name: "proxy",
		Nodes: []scenario.Node{
			{Name: "atlas", Setup: scenario.SetupController, Site: "home"},
			{Name: "den", Setup: scenario.SetupJoin, Site: "home"},
			{Name: "seedbox", Setup: scenario.SetupJoin, Site: "ams"},
			{Name: "pi", Setup: scenario.SetupJoin, Site: "ams"},
			{Name: "box", Setup: scenario.SetupJoin, Site: "ams"},
		},
		Network: &scenario.Network{
			Sites: []scenario.Site{
				{Name: "home", WAN: "public", Address: "203.0.113.9"},
				{Name: "ams", WAN: "none", Address: "192.0.2.44"},
			},
			Proxies: []scenario.Proxy{{Name: "the-vps", Address: "198.51.100.8", Routes: []scenario.ProxyRoute{
				{Host: "slipstream.proxy.fake.localhost", Node: "atlas", Service: "http"},
				{Host: "plex.proxy.fake.localhost", Node: "atlas", Service: "plex"},
			}}},
		},
	}
	httpAt, plexAt := echo(t, "http@atlas"), echo(t, "plex@atlas")
	srv := faketest.InMemory(t, proxy.Fake, s,
		faketest.WithOrigins("den", "seedbox", "pi", "box"),
		faketest.WithListeners(func(string) (node.Listeners, error) { return node.Listeners{HTTP: httpAt}, nil }),
		faketest.WithFronts(fake.Fronts{{Target: "plex", Node: "atlas", Address: fake.Address{AddrPort: plexAt}}}),
		faketest.WithFaults(
			fault.Rule{ID: "expired", Target: "proxy", Origin: "pi", Fault: fault.BadTLS(fault.Expired)},
			fault.Rule{ID: "wrong-name", Target: "proxy", Origin: "box", Fault: fault.BadTLS(fault.WrongName)},
			fault.Rule{ID: "sign-in", Target: "proxy", Origin: "den", Route: "proxy", Fault: fault.Status{Code: http.StatusUnauthorized}},
		))
	for _, c := range []struct {
		origin, host, want string
	}{
		{"seedbox", "slipstream.proxy.fake.localhost", "200 http@atlas host=slipstream.proxy.fake.localhost xff=192.0.2.44"},
		{"seedbox", "plex.proxy.fake.localhost", "200 plex@atlas host=plex.proxy.fake.localhost xff=192.0.2.44"},
		{"seedbox", "other.proxy.fake.localhost", "404"},
		{"den", "plex.proxy.fake.localhost", "401"},
		{"pi", "plex.proxy.fake.localhost", "certificate has expired"},
		{"box", "plex.proxy.fake.localhost", "certificate is valid for wrong-name.fake.localhost"},
	} {
		if got := get(t, srv.From(t, c.origin), "https://"+c.host+"/identity"); !strings.Contains(got, c.want) {
			t.Errorf("%s asking %s = %q, want %q", c.origin, c.host, got, c.want)
		}
	}
}

// get answers the status and body of a GET of url, only the status when
// it isn't 200, or the error that stopped it.
func get(t *testing.T, client *http.Client, url string) string {
	t.Helper()
	req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, url, http.NoBody)
	if err != nil {
		t.Fatal(err)
	}
	res, err := client.Do(req)
	if err != nil {
		return err.Error()
	}
	defer func() { _ = res.Body.Close() }()
	if res.StatusCode != http.StatusOK {
		return strconv.Itoa(res.StatusCode)
	}
	body, err := io.ReadAll(res.Body)
	if err != nil {
		t.Fatal(err)
	}
	return "200 " + string(body)
}
