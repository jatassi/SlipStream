package systemapi_test

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/netip"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"
	"time"

	"github.com/jatassi/slipstream/internal/api"
	"github.com/jatassi/slipstream/internal/buildinfo"
	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/identity/identitytest"
	"github.com/jatassi/slipstream/internal/identityapi"
	"github.com/jatassi/slipstream/internal/live"
	"github.com/jatassi/slipstream/internal/nodesystem"
	"github.com/jatassi/slipstream/internal/secret"
	"github.com/jatassi/slipstream/internal/storage/storagetest"
	"github.com/jatassi/slipstream/internal/system"
	"github.com/jatassi/slipstream/internal/systemapi"
)

var facts = system.Facts{
	DataDir:        "/srv/slipstream",
	LogsDir:        "/srv/slipstream/logs",
	PID:            4242,
	Seed:           "fixed",
	StartedAt:      time.Date(2026, 10, 12, 9, 30, 0, 0, time.UTC),
	HTTPConfigured: netip.MustParseAddrPort("[::]:8484"),
	Listeners: system.Listeners{
		HTTP:    netip.MustParseAddrPort("127.0.0.1:8486"),
		Torrent: netip.MustParseAddrPort("0.0.0.0:6881"),
		Link:    netip.MustParseAddrPort("127.0.0.1:8485"),
		Control: "/srv/slipstream/run/control.sock",
	},
}

type node struct {
	t    *testing.T
	h    http.Handler
	code string
}

// controller is a Controller's claim, reaching Nodes through systems.
type controller struct {
	identitytest.Node
	systems *nodesystem.Nodes
}

func (c controller) Systems() (*nodesystem.Nodes, bool) { return c.systems, true }

func newNode(t *testing.T, f system.Facts) *node { return newController(t, f, nil) }

func newController(t *testing.T, f system.Facts, systems *nodesystem.Nodes) *node {
	t.Helper()
	ctx := t.Context()
	lib := storagetest.Library(t, secret.Schema(), identity.Schema())
	keys, err := secret.Open(ctx, lib, filepath.Join(t.TempDir(), secret.KeyFile))
	if err != nil {
		t.Fatal(err)
	}
	hub := live.NewHub([]live.Event{identity.SessionsChangedEvent})
	svc, err := identity.New(ctx, lib, keys.PINPepper(), identity.SessionsChangedEvent.Bind(hub), hub)
	if err != nil {
		t.Fatal(err)
	}
	log := slog.New(slog.NewTextHandler(t.Output(), nil))
	gate, code, err := identity.NewGate(ctx, identitytest.Node{Service: svc}.Claim, log)
	if err != nil {
		t.Fatal(err)
	}
	r := api.NewRouter(api.Services{Node: identitytest.Node{Service: svc}, Gate: gate, Web: fstest.MapFS{}, Log: log})
	identityapi.Routes(r, identitytest.Node{Service: svc})
	systemapi.Routes(r, f, controller{identitytest.Node{Service: svc}, systems})
	return &node{t: t, h: r.Handler(), code: code}
}

func (n *node) do(method, target, body string, headers map[string]string) *httptest.ResponseRecorder {
	n.t.Helper()
	req := httptest.NewRequestWithContext(n.t.Context(), method, target, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	w := httptest.NewRecorder()
	n.h.ServeHTTP(w, req)
	return w
}

func (n *node) claim() (apiKey string) {
	n.t.Helper()
	w := n.do(http.MethodPost, "/api/v1/setup/controller", `{"email":"admin@example.com","displayName":"Admin","pin":"1234"}`, map[string]string{"X-Setup-Code": n.code})
	var res struct {
		APIKey string `json:"apiKey"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &res); err != nil || w.Code != http.StatusCreated {
		n.t.Fatalf("claimController = %d %s", w.Code, w.Body)
	}
	return res.APIKey
}

func expect(t *testing.T, w *httptest.ResponseRecorder, status int, body string) {
	t.Helper()
	if w.Code != status || w.Body.String() != body {
		t.Fatalf("got %d %q, want %d %q", w.Code, w.Body, status, body)
	}
}

func TestHealthAndStatusArePublic(t *testing.T) {
	n := newNode(t, facts)
	expect(t, n.do(http.MethodGet, "/api/v1/health", "", nil), 200, `{"status":"ok"}`+"\n")
	expect(t, n.do(http.MethodGet, "/api/v1/status", "", nil), 200, `{"claim":"unclaimed"}`+"\n")
	n.claim()
	expect(t, n.do(http.MethodGet, "/api/v1/status", "", nil), 200, `{"claim":"controller"}`+"\n")
}

func TestSystemReportsTheFactsToTheAPIKey(t *testing.T) {
	stamped := facts
	stamped.Build = &buildinfo.Stamp{
		Version:      "2.0.0",
		Commit:       "0123456789abcdef0123456789abcdef01234567",
		Fingerprint:  "sha256:00ff",
		Target:       "linux/amd64",
		NativeBundle: "lt-2.0.11-1",
		Libtorrent:   "2.0.11",
	}
	for _, tc := range []struct {
		facts system.Facts
		build string
	}{
		{facts, `{"stamped":false}`},
		{stamped, `{"stamped":true,"version":"2.0.0","commit":"0123456789abcdef0123456789abcdef01234567","fingerprint":"sha256:00ff","target":"linux/amd64","nativeBundle":"lt-2.0.11-1","libtorrent":"2.0.11"}`},
	} {
		n := newNode(t, tc.facts)
		key := n.claim()
		expect(t, n.do(http.MethodGet, "/api/v1/system", "", map[string]string{"X-Api-Key": key}), 200,
			`{"build":`+tc.build+`,"dataDir":"/srv/slipstream","logsDir":"/srv/slipstream/logs","pid":4242,"seed":"fixed","startedAt":"2026-10-12T09:30:00Z","http":{"configured":"[::]:8484"},"listeners":{"http":"127.0.0.1:8486","torrent":"0.0.0.0:6881","link":"127.0.0.1:8485","control":"/srv/slipstream/run/control.sock"}}`+"\n")
		expect(t, n.do(http.MethodGet, "/api/v1/system", "", nil), 401, `{"status":401,"code":"signed_out","title":"Sign in to continue"}`+"\n")
	}
}

// worker is a Worker's claim: no library.db, so no identity service.
type worker struct{}

func (worker) Claim(context.Context) (system.Claim, error) { return system.Worker, nil }
func (worker) Identity() (*identity.Service, bool)         { return nil, false }
func (worker) Systems() (*nodesystem.Nodes, bool)          { return nil, false }

func TestAWorkerReportsItsClaimAndSendsAdminsToItsController(t *testing.T) {
	log := slog.New(slog.DiscardHandler)
	gate, code, err := identity.NewGate(t.Context(), worker{}.Claim, log)
	if err != nil || code != "" {
		t.Fatalf("NewGate = %q, %v; want a closed gate", code, err)
	}
	r := api.NewRouter(api.Services{Node: worker{}, Gate: gate, Web: fstest.MapFS{}, Log: log})
	systemapi.Routes(r, facts, worker{})
	n := &node{t: t, h: r.Handler()}
	expect(t, n.do(http.MethodGet, "/api/v1/status", "", nil), 200, `{"claim":"worker"}`+"\n")
	expect(t, n.do(http.MethodGet, "/api/v1/system", "", map[string]string{"X-Api-Key": "ssk_made-up"}), 409,
		`{"status":409,"code":"worker_node","title":"This Node is a Worker. Manage it from its Controller."}`+"\n")
}
