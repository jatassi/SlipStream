package fake_test

import (
	"bytes"
	"log/slog"
	"net/netip"
	"testing"

	"github.com/jatassi/slipstream/fakes/fake"
	"github.com/jatassi/slipstream/fakes/scenario"
)

func TestAddressURL(t *testing.T) {
	at := netip.MustParseAddrPort("127.0.0.1:41234")
	got := []string{fake.Address{AddrPort: at}.URL("/3"), fake.Address{AddrPort: at, TLS: true}.URL("/3")}
	if want := []string{"http://127.0.0.1:41234/3", "https://127.0.0.1:41234/3"}; got[0] != want[0] || got[1] != want[1] {
		t.Fatalf("URL = %q, want %q", got, want)
	}
}

func TestNewEnvKeysTheFakeUnderTheScenarioSeed(t *testing.T) {
	var log bytes.Buffer
	env := fake.NewEnv(scenario.Scenario{Seed: "single-machine"}, "tmdb", slog.New(slog.NewTextHandler(&log, &slog.HandlerOptions{
		ReplaceAttr: func(_ []string, a slog.Attr) slog.Attr {
			if a.Key == slog.TimeKey {
				return slog.Attr{}
			}
			return a
		},
	})))
	env.Log.Info("served")
	if got, want := log.String(), "level=INFO msg=served fake=tmdb\n"; got != want {
		t.Fatalf("log = %q, want %q", got, want)
	}
	if got, want := env.Streams.Stream("works").Uint64(), uint64(5910655446283853086); got != want {
		t.Fatalf("first draw = %d, want %d", got, want)
	}
}
