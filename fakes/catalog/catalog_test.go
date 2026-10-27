package catalog_test

import (
	"net/http"
	"testing"
	"testing/synctest"

	conformance "github.com/jatassi/slipstream/conformance/indexercatalog"
	"github.com/jatassi/slipstream/fakes/catalog"
	"github.com/jatassi/slipstream/fakes/fake/faketest"
	"github.com/jatassi/slipstream/fakes/scenario"
	"github.com/jatassi/slipstream/internal/cardigann"
	"github.com/jatassi/slipstream/internal/indexers"
)

func inBothTiers(t *testing.T, test func(t *testing.T, f faketest.Served)) {
	t.Run("loopback", func(t *testing.T) { test(t, faketest.Loopback(t, catalog.Fake, scenario.Scenario{})) })
	t.Run("in memory", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) { test(t, faketest.InMemory(t, catalog.Fake, scenario.Scenario{})) })
	})
}

func base(f faketest.Served) string { return f.Address.URL(catalog.Base) }

func TestFakeCatalogConforms(t *testing.T) {
	inBothTiers(t, func(t *testing.T, f faketest.Served) {
		r := conformance.Run(t, conformance.Target{Base: base(f), Client: f.Client})
		want := map[string]cardigann.LoginClass{"larkspur": cardigann.SessionLogin, "quarry": cardigann.StatelessLogin, "cinder": cardigann.SessionLogin}
		for id, class := range want {
			if got, ok := r.Classes[id]; !ok || got != class {
				t.Errorf("%s loads with class %v (%t), want %v", id, got, ok, class)
			}
		}
		if r.Refused["broken"] != "it has no links" {
			t.Errorf("broken is refused with %q, want it has no links", r.Refused["broken"])
		}
	})
}

func status(t *testing.T, f faketest.Served, path, since string) (int, string) {
	t.Helper()
	req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, base(f)+path, http.NoBody)
	if err != nil {
		t.Fatal(err)
	}
	if since != "" {
		req.Header.Set("If-Modified-Since", since)
	}
	res, err := f.Client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	_ = res.Body.Close()
	return res.StatusCode, res.Header.Get("Last-Modified")
}

func TestTheIndexIsNotModifiedUntilACueChangesIt(t *testing.T) {
	inBothTiers(t, func(t *testing.T, f faketest.Served) {
		code, modified := status(t, f, "/12/", "")
		if code != http.StatusOK || modified == "" {
			t.Fatalf("GET /12/ = %d with Last-Modified %q", code, modified)
		}
		if code, _ := status(t, f, "/12/", modified); code != http.StatusNotModified {
			t.Errorf("GET /12/ since its Last-Modified = %d, want 304", code)
		}
		f.Cue(t, catalog.Update.Name, nil)
		if code, _ := status(t, f, "/12/", modified); code != http.StatusOK {
			t.Errorf("GET /12/ since its Last-Modified after an update = %d, want 200", code)
		}
	})
}

func TestCuesChangeWhatTheCatalogLists(t *testing.T) {
	inBothTiers(t, func(t *testing.T, f faketest.Served) {
		read := func() map[string]string {
			index, _, err := indexers.ReadCatalog(t.Context(), f.Client, base(f))
			if err != nil {
				t.Fatalf("ReadCatalog: %v", err)
			}
			shas := map[string]string{}
			for _, e := range index {
				shas[e.ID] = e.SHA
			}
			return shas
		}
		before := read()
		f.Cue(t, catalog.Update.Name, nil)
		updated := read()
		if updated["larkspur"] == before["larkspur"] || updated["quarry"] != before["quarry"] {
			t.Errorf("an update changed the shas from %v to %v, want only larkspur's", before, updated)
		}
		f.Cue(t, catalog.Remove.Name, map[string]string{"id": "cinder"})
		if _, ok := read()["cinder"]; ok {
			t.Error("cinder is still listed after its removal")
		}
		f.Cue(t, catalog.CorruptZip.Name, nil)
		if code, _ := status(t, f, "/12/package.zip", ""); code != http.StatusOK {
			t.Errorf("GET package.zip = %d, want a 200 that isn't a zip", code)
		}
		read()
		f.Cue(t, catalog.Schema.Name, map[string]int{"schema": 13})
		if code, _ := status(t, f, "/12/", ""); code != http.StatusNotFound {
			t.Errorf("GET /12/ once the catalog serves schema 13 = %d, want 404", code)
		}
	})
}
