package api

import (
	"bytes"
	"context"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"slices"
	"sync/atomic"
	"testing"

	"github.com/danielgtaylor/huma/v2"

	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/identity/identitytest"
	"github.com/jatassi/slipstream/internal/secret"
	"github.com/jatassi/slipstream/internal/storage/storagetest"
)

// TestASupersededControllerRefusesAllButItsOwnPage registers an Admin
// operation, one that says WhileSuperseded and an Account one, then
// supersedes the Controller: only the Admin operation answers superseded,
// and it documents the code while the exempt one doesn't.
func TestASupersededControllerRefusesAllButItsOwnPage(t *testing.T) {
	lib := storagetest.Library(t, secret.Schema(), identity.Schema())
	svc, err := identity.New(t.Context(), identity.Config{Library: lib, Pepper: make([]byte, 32), Hub: identitytest.Hub(t)})
	if err != nil {
		t.Fatal(err)
	}
	var superseded atomic.Bool
	r := NewRouter(Services{
		Node:       identitytest.Controller{Node: identitytest.Node{Service: svc}},
		Superseded: superseded.Load,
		Log:        slog.New(slog.NewTextHandler(&bytes.Buffer{}, nil)),
	})
	o := r.Owner("fixture", Account)
	ok := func(context.Context, identity.Admin, *struct{}) (*struct{}, error) { return &struct{}{}, nil }
	Register(o, Admin, Op{ID: "getFixture", Method: http.MethodGet, Path: "/fixture", Status: http.StatusOK}, ok)
	Register(o, Admin, Op{ID: "getPage", Method: http.MethodGet, Path: "/page", Status: http.StatusOK, WhileSuperseded: true}, ok)
	Register(o, Account, Op{ID: "getMine", Method: http.MethodGet, Path: "/mine", Status: http.StatusOK},
		func(context.Context, identity.Account, *struct{}) (*struct{}, error) { return &struct{}{}, nil })
	h := r.Handler()
	session := identitytest.Session(t, lib, identitytest.Admin(t, lib, "ada@example.com"))
	status := func(path string) (int, string) {
		req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/api/v1"+path, http.NoBody)
		req.Header.Set("Cookie", identity.CookieName+"="+session)
		w := httptest.NewRecorder()
		h.ServeHTTP(w, req)
		return w.Code, w.Body.String()
	}
	for _, path := range []string{"/fixture", "/page", "/mine"} {
		if code, body := status(path); code != http.StatusOK {
			t.Fatalf("%s before the fence = %d %s, want 200", path, code, body)
		}
	}
	superseded.Store(true)
	want := `{"status":409,"code":"superseded","title":"This Controller was replaced"}` + "\n"
	if code, body := status("/fixture"); code != http.StatusConflict || body != want {
		t.Fatalf("an Admin operation on a Superseded Controller = %d %s, want 409 %s", code, body, want)
	}
	for _, path := range []string{"/page", "/mine"} {
		if code, body := status(path); code != http.StatusOK {
			t.Fatalf("%s on a Superseded Controller = %d %s, want 200", path, code, body)
		}
	}
	paths := r.huma.OpenAPI().Paths
	if !documents(paths["/fixture"].Get, "superseded") || documents(paths["/page"].Get, "superseded") {
		t.Fatal("want superseded documented on the Admin operation only")
	}
}

func documents(op *huma.Operation, code string) bool {
	r, ok := op.Responses["409"]
	return ok && slices.Contains(r.Content[problemType].Schema.AllOf[1].Properties["code"].Enum, any(code))
}
