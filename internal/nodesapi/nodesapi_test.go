package nodesapi_test

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"slices"
	"testing"
	"testing/fstest"

	"github.com/jatassi/slipstream/internal/api"
	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/identity/identitytest"
	"github.com/jatassi/slipstream/internal/link"
	"github.com/jatassi/slipstream/internal/nodes"
	"github.com/jatassi/slipstream/internal/nodesapi"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/secret"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/storage/storagetest"
)

func TestTheRoleEnumIsRoleNames(t *testing.T) {
	r := api.NewRouter(api.Services{})
	nodesapi.Routes(r, nil)
	var doc struct {
		Components struct {
			Schemas map[string]struct {
				Properties map[string]struct {
					Items struct {
						Enum []string `json:"enum"`
					} `json:"items"`
				} `json:"properties"`
			} `json:"schemas"`
		} `json:"components"`
	}
	if err := json.Unmarshal(r.Document(), &doc); err != nil {
		t.Fatal(err)
	}
	var want []string
	for _, n := range role.Names() {
		want = append(want, string(n))
	}
	for _, schema := range []string{"NodesNode", "NodesRoles"} {
		if got := doc.Components.Schemas[schema].Properties["roles"].Items.Enum; !slices.Equal(got, want) {
			t.Errorf("%s.roles enum = %v, want role.Names() %v", schema, got, want)
		}
	}
}

// unadvertised is a link with no address to advertise yet.
type unadvertised struct{}

func (unadvertised) State(link.Key) link.State { return link.Down{} }

func (unadvertised) Subscribe(context.Context) <-chan link.Change { return nil }

func (unadvertised) Advertised() (link.Addresses, bool) { return link.Addresses{}, false }

type changes struct{}

func (changes) Publish(context.Context, nodes.Changed) {}

type roster struct{ svc *nodes.Service }

func (r roster) Nodes() (*nodes.Service, bool) { return r.svc, true }

// claimed is an identity service holding an Admin, and the API key.
func claimed(t *testing.T, lib *storage.DB[storage.Library], log *slog.Logger) (*identity.Service, *identity.Gate, string) {
	t.Helper()
	keys, err := secret.Open(t.Context(), lib, filepath.Join(t.TempDir(), secret.KeyFile))
	if err != nil {
		t.Fatal(err)
	}
	svc, err := identity.New(t.Context(), identity.Config{Library: lib, Pepper: keys.PINPepper(), Hub: identitytest.Hub(t)})
	if err != nil {
		t.Fatal(err)
	}
	gate, code, err := identity.NewGate(t.Context(), identitytest.Node{Service: svc}.Claim, log)
	if err != nil {
		t.Fatal(err)
	}
	pin, err := identity.ParsePIN("2468")
	if err != nil {
		t.Fatal(err)
	}
	var admin identity.Claimed
	err = gate.Run(t.Context(), identity.Caller{}, code, func(c identity.Claim) {
		admin, err = svc.ClaimController(t.Context(), c, "ada@example.test", "Ada", pin)
	})
	if err != nil {
		t.Fatal(err)
	}
	return svc, gate, admin.APIKey
}

func TestAJoinCodeNeedsALinkAddress(t *testing.T) {
	log := slog.New(slog.DiscardHandler)
	lib := storagetest.Library(t, secret.Schema(), identity.Schema(), nodes.Schema(), role.LibrarySchema())
	svc, gate, apiKey := claimed(t, lib, log)
	ns, err := nodes.Open(t.Context(), nodes.Config{Library: lib, Name: "home", Link: unadvertised{}, Changed: changes{}, Log: log})
	if err != nil {
		t.Fatal(err)
	}
	r := api.NewRouter(api.Services{Node: identitytest.Node{Service: svc}, Gate: gate, Web: fstest.MapFS{"index.html": {Data: []byte("<!doctype html>")}}, Log: log})
	nodesapi.Routes(r, roster{ns})
	req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/api/v1/nodes/join-codes", nil)
	req.Header.Set("X-Api-Key", apiKey)
	w := httptest.NewRecorder()
	r.Handler().ServeHTTP(w, req)
	if want := `{"status":409,"code":"no_link_address","title":"The link has no address to put in a Join Code"}` + "\n"; w.Body.String() != want {
		t.Fatalf("createJoinCode = %d %s, want %s", w.Code, w.Body, want)
	}
}
