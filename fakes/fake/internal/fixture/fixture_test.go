package fixture_test

import (
	"io"
	"net/http"
	"slices"
	"testing"
	"testing/synctest"

	"github.com/jatassi/slipstream/fakes/fake/faketest"
	"github.com/jatassi/slipstream/fakes/fake/internal/fixture"
	"github.com/jatassi/slipstream/fakes/fault"
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
