package fixture_test

import (
	"io"
	"net/http"
	"testing"
	"testing/synctest"

	"github.com/jatassi/slipstream/fakes/fake/faketest"
	"github.com/jatassi/slipstream/fakes/fake/internal/fixture"
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
	if got, want := resp.Status+" "+string(body), "200 OK fixture"; got != want {
		t.Fatalf("GET /whoami = %q, want %q", got, want)
	}
}
