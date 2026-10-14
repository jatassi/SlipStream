package fixture_test

import (
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"testing/synctest"

	"github.com/jatassi/slipstream/fakes/events"
	"github.com/jatassi/slipstream/fakes/fake"
	"github.com/jatassi/slipstream/fakes/fake/faketest"
	"github.com/jatassi/slipstream/fakes/fault"
	"github.com/jatassi/slipstream/fakes/internal/fixture"
	"github.com/jatassi/slipstream/fakes/scenario"
)

func TestWhoamiInBothTiers(t *testing.T) {
	s, err := scenario.Load("single-machine")
	if err != nil {
		t.Fatal(err)
	}
	t.Run("loopback", func(t *testing.T) {
		assertWhoami(t, faketest.Loopback(t, fixture.Fake, s))
	})
	t.Run("in memory", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			assertWhoami(t, faketest.InMemory(t, fixture.Fake, s))
		})
	})
}

func assertWhoami(t *testing.T, f faketest.Served) {
	t.Helper()
	if got, want := whoami(t, f), "200 OK fixture"; got != want {
		t.Fatalf("GET /whoami = %q, want %q", got, want)
	}
}

func whoami(t *testing.T, f faketest.Served) string {
	t.Helper()
	req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, f.Address.URL("/whoami"), http.NoBody)
	if err != nil {
		t.Fatal(err)
	}
	resp, err := f.Client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = resp.Body.Close() }()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}
	return resp.Status + " " + string(body)
}

func TestWithFaultsPlaysInBothTiers(t *testing.T) {
	s, err := scenario.Load("single-machine")
	if err != nil {
		t.Fatal(err)
	}
	second := faketest.WithFaults(fault.Rule{ID: "second", Target: "fixture", Route: "whoami", Fault: fault.Status{Code: 503}, Skip: 1, Times: 1})
	check := func(t *testing.T, f faketest.Served) {
		var got []string
		for range 3 {
			got = append(got, whoami(t, f))
		}
		if want := []string{"200 OK fixture", "503 Service Unavailable Service Unavailable\n", "200 OK fixture"}; !slices.Equal(got, want) {
			t.Fatalf("three requests answered %q, want %q", got, want)
		}
	}
	t.Run("loopback", func(t *testing.T) { check(t, faketest.Loopback(t, fixture.Fake, s, second)) })
	t.Run("in memory", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) { check(t, faketest.InMemory(t, fixture.Fake, s, second)) })
	})
}

func TestACaptureRecordsTheOriginAndKeepsItsFiles(t *testing.T) {
	s, err := scenario.Load("single-machine")
	if err != nil {
		t.Fatal(err)
	}
	f := faketest.Loopback(t, fixture.Fake, s, faketest.As("seedbox"))
	whoami(t, f)
	recs := f.Records(t)
	if len(recs) != 1 {
		t.Fatalf("records = %+v, want one", recs)
	}
	r := recs[0]
	if got := r.Kind + " " + r.Node + " " + string(r.Data); got != `fixture.visit seedbox {"path":"/whoami"}` {
		t.Fatalf("record = %s", got)
	}
	path, ok := r.Files["user-agent.txt"]
	if !ok || !strings.HasPrefix(path, "captures/visits/") {
		t.Fatalf("record files = %v, want user-agent.txt under captures/visits/", r.Files)
	}
	if got, err := os.ReadFile(filepath.Join(f.RunDir, filepath.FromSlash(path))); err != nil || string(got) != "Go-http-client/1.1" {
		t.Fatalf("user-agent.txt = %q, %v", got, err)
	}
}

func TestOnCueRefusesAnUndeclaredName(t *testing.T) {
	spec := fixture.Fake
	spec.Cues = nil
	_, err := spec.New(scenario.Scenario{}, fake.NewEnv(scenario.Scenario{}, spec, slog.New(slog.DiscardHandler), nil, discard{}))
	if want := "fake fixture declares no cue fixture-greet"; err == nil || err.Error() != want {
		t.Fatalf("New = %v, want %s", err, want)
	}
}

type discard struct{}

func (discard) Append(rec events.Record) (events.Record, error) { return rec, nil }

func TestEnvCaptureRefusesAnUndeclaredName(t *testing.T) {
	spec := fixture.Fake
	spec.Captures = nil
	_, err := spec.New(scenario.Scenario{}, fake.NewEnv(scenario.Scenario{}, spec, slog.New(slog.DiscardHandler), nil, nil))
	if want := "fake fixture declares no capture visits"; err == nil || err.Error() != want {
		t.Fatalf("New = %v, want %s", err, want)
	}
}
