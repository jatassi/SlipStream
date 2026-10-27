package indexersapi_test

import (
	"context"
	"io/fs"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/jatassi/slipstream/fakes/catalog"
	"github.com/jatassi/slipstream/fakes/fake/faketest"
	"github.com/jatassi/slipstream/fakes/scenario"
	"github.com/jatassi/slipstream/fakes/site/definitions"
	"github.com/jatassi/slipstream/internal/api"
	"github.com/jatassi/slipstream/internal/api/ownertest"
	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/indexers"
	"github.com/jatassi/slipstream/internal/indexers/indexerstest"
	"github.com/jatassi/slipstream/internal/indexersapi"
	"github.com/jatassi/slipstream/internal/node/endpoint"
	"github.com/jatassi/slipstream/internal/nodes"
	"github.com/jatassi/slipstream/internal/secret"
	"github.com/jatassi/slipstream/internal/secret/secrettest"
	"github.com/jatassi/slipstream/internal/storage/storagetest"
	"github.com/jatassi/slipstream/internal/tasks"
)

type catalogServed struct {
	owner *ownertest.Owner
	x     *indexers.Indexers
	fake  faketest.Served
}

func serveCatalog(t *testing.T) catalogServed {
	t.Helper()
	lib := storagetest.Library(t, secret.Schema(), identity.Schema(), nodes.Schema(), indexers.Schema())
	f := faketest.Loopback(t, catalog.Fake, scenario.Scenario{})
	x := indexers.New(indexers.Config{
		Library: lib, Sealer: secrettest.Sealer(t), Roster: indexerstest.Alone(self), HTTP: f.Client,
		Endpoints:   indexers.Endpoints{Catalog: endpoint.MustParseURL(f.Address.URL(catalog.Base))},
		CatalogNext: func(context.Context) (time.Time, error) { return time.Date(2026, 10, 28, 3, 0, 0, 0, time.UTC), nil },
		Changed:     discard[indexers.Changed]{}, Status: discard[indexers.StatusChanged]{}, Trackers: discard[indexers.TrackerChanged]{}, Catalogs: discard[indexers.CatalogChanged]{},
		Timings: indexers.DefaultTimings(), Zone: time.UTC, Log: slog.New(slog.DiscardHandler),
	})
	return catalogServed{owner: ownertest.Serve(t, lib, func(r *api.Router) { indexersapi.Routes(r, registry{x}) }), x: x, fake: f}
}

func (s catalogServed) refresh(t *testing.T) {
	t.Helper()
	if err := s.x.Tasks()[0].Run(t.Context(), tasks.Unit{}, nil); err != nil {
		t.Fatal(err)
	}
}

func yamlBody(t *testing.T, file string) string {
	t.Helper()
	src, err := fs.ReadFile(definitions.FS, file)
	if err != nil {
		t.Fatal(err)
	}
	return `{"yaml":` + strconv.Quote(string(src)) + `}`
}

func TestTheCatalogSaysWhenItTriesAgainUntilItIsDownloaded(t *testing.T) {
	s := serveCatalog(t)
	address := s.fake.Address.URL(catalog.Base)
	s.owner.Expect(t, http.MethodGet, "/api/v1/indexer-catalog", "", http.StatusOK,
		`{"refreshedAt":null,"nextRefreshAt":"T","address":"`+address+`","addressIsDefault":true,`+
			`"notice":"SlipStream hasn't downloaded the Indexer catalog yet. SlipStream retries at 03:00.","entries":[]}`)
	s.owner.Expect(t, http.MethodGet, "/api/v1/indexer-catalog/larkspur", "", http.StatusServiceUnavailable,
		`{"status":503,"code":"catalog_unavailable","title":"SlipStream hasn't downloaded the Indexer catalog yet","detail":"SlipStream hasn't downloaded the Indexer catalog yet. SlipStream retries at 03:00."}`)
	s.refresh(t)
	_, reply := s.owner.Call(t, http.MethodGet, "/api/v1/indexer-catalog", "")
	quarry := `{"id":"quarry","name":"Quarry","description":"Quarry is a fictional semi-private tracker whose API takes a key.","language":"en-US",` +
		`"privacy":"semi-private","categories":[2000,112696,5000,117804],"origin":"catalog","reason":null,"replacedByCustom":false,"version":"T"}`
	broken := `{"id":"broken","name":"Broken",`
	if !strings.Contains(ownertest.Masked(reply), quarry) || !strings.Contains(reply, `"notice":null`) || !strings.Contains(reply, broken) ||
		!strings.Contains(reply, `"reason":"it has no links"`) {
		t.Errorf("getIndexerCatalog = %s, want quarry as %s and broken with its reason", reply, quarry)
	}
}

func TestTheFormOfACatalogDefinition(t *testing.T) {
	s := serveCatalog(t)
	s.refresh(t)
	_, reply := s.owner.Call(t, http.MethodGet, "/api/v1/indexer-catalog/larkspur", "")
	for _, want := range []string{
		`"links":["https://larkspur.fixture-site.fake.localhost/"]`,
		`{"name":"username","label":"Username","type":"text","default":null,"options":[],"help":null,"sealed":false}`,
		`{"name":"password","label":"Password","type":"password","default":null,"options":[],"help":null,"sealed":true}`,
		`{"name":"sort","label":"Sort requested from site","type":"select","default":"added","options":[{"value":"added","label":"created"},{"value":"seeders","label":"seeders"},{"value":"size","label":"size"}],"help":null,"sealed":false}`,
		`"session":true,"captcha":true,"solver":false}`,
	} {
		if !strings.Contains(reply, want) {
			t.Errorf("getCatalogDefinition(larkspur) = %s, want it to hold %s", reply, want)
		}
	}
	s.owner.Expect(t, http.MethodGet, "/api/v1/indexer-catalog/nowhere", "", http.StatusNotFound,
		`{"status":404,"code":"definition_not_found","title":"No definition has that id"}`)
}

func TestTheCatalogAddressIsTestedBeforeItIsSaved(t *testing.T) {
	s := serveCatalog(t)
	s.owner.Expect(t, http.MethodPut, "/api/v1/indexer-catalog/address", `{"address":"`+s.fake.Address.URL("/elsewhere")+`"}`, http.StatusUnprocessableEntity,
		`{"status":422,"code":"catalog_address_unreachable","title":"SlipStream can't read an Indexer catalog at that address","detail":"Check the address and that it serves schema 12."}`)
	address := s.fake.Address.URL(catalog.Base)
	_, reply := s.owner.Call(t, http.MethodPut, "/api/v1/indexer-catalog/address", `{"address":"`+address+`/"}`)
	if !strings.Contains(reply, `"address":"`+address+`","addressIsDefault":false`) {
		t.Errorf("setCatalogAddress = %s, want %s saved without its trailing slash", reply, address)
	}
	_, reply = s.owner.Call(t, http.MethodPut, "/api/v1/indexer-catalog/address", `{"address":null}`)
	if !strings.Contains(reply, `"addressIsDefault":true`) {
		t.Errorf("setCatalogAddress(null) = %s, want the default", reply)
	}
}

func TestCustomDefinitionsAreAddedAndDeleted(t *testing.T) {
	s := serveCatalog(t)
	s.refresh(t)
	s.owner.Expect(t, http.MethodPost, "/api/v1/indexer-catalog/custom", yamlBody(t, "larkspur-v3.yml"), http.StatusUnprocessableEntity,
		`{"status":422,"code":"definition_unloadable","title":"SlipStream can't run this definition yet","detail":"It logs in with \"twofactor\", which SlipStream doesn't run."}`)
	s.owner.Expect(t, http.MethodPost, "/api/v1/indexer-catalog/custom", yamlBody(t, "quarry.yml"), http.StatusConflict,
		`{"status":409,"code":"definition_id_taken","title":"The Indexer catalog already has a definition with this id"}`)
	meadow := strings.ReplaceAll(yamlBody(t, "quarry.yml"), "quarry", "meadow")
	s.owner.Expect(t, http.MethodPost, "/api/v1/indexer-catalog/custom", meadow, http.StatusCreated,
		`{"id":"meadow","name":"Quarry","description":"Quarry is a fictional semi-private tracker whose API takes a key.","language":"en-US",`+
			`"privacy":"semi-private","categories":[2000,112696,5000,117804],"origin":"custom","reason":null,"replacedByCustom":false,"version":"T"}`)
	if code, reply := s.owner.Call(t, http.MethodDelete, "/api/v1/indexer-catalog/custom/meadow", ""); code != http.StatusNoContent {
		t.Fatalf("deleteCustomDefinition = %d %s, want 204", code, reply)
	}
	s.owner.Expect(t, http.MethodDelete, "/api/v1/indexer-catalog/custom/meadow", "", http.StatusNotFound,
		`{"status":404,"code":"definition_not_found","title":"No definition has that id"}`)
}
