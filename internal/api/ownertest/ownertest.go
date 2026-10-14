// Package ownertest serves one owner's operations on a claimed Controller
// for its in-process tests.
package ownertest

import (
	"bytes"
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/jatassi/slipstream/internal/api"
	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/identity/identitytest"
	"github.com/jatassi/slipstream/internal/secret"
	"github.com/jatassi/slipstream/internal/storage"
)

// Owner is one owner's operations served on a claimed Controller, called
// with its Admin's API key.
type Owner struct {
	handler http.Handler
	apiKey  string
	Log     *bytes.Buffer
}

type sessions struct{}

func (sessions) Publish(context.Context, storage.UUID, identity.SessionsChanged) {}

func (sessions) EndSession(storage.UUID) {}

// Serve claims a Controller over lib, which holds at least the secret
// and identity schemas, and mounts the owner's Routes.
func Serve(t *testing.T, lib *storage.DB[storage.Library], routes func(*api.Router)) *Owner {
	t.Helper()
	logged := &bytes.Buffer{}
	log := slog.New(slog.NewTextHandler(logged, &slog.HandlerOptions{ReplaceAttr: func(_ []string, a slog.Attr) slog.Attr {
		if a.Key == slog.TimeKey {
			return slog.Attr{}
		}
		return a
	}}))
	keys, err := secret.Open(t.Context(), lib, filepath.Join(t.TempDir(), secret.KeyFile))
	if err != nil {
		t.Fatal(err)
	}
	svc, err := identity.New(t.Context(), lib, keys.PINPepper(), sessions{}, sessions{})
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
	r := api.NewRouter(api.Services{Node: identitytest.Controller{Node: identitytest.Node{Service: svc}}, Gate: gate, Web: fstest.MapFS{"index.html": {Data: []byte("<!doctype html>")}}, Log: log})
	routes(r)
	return &Owner{handler: r.Handler(), apiKey: admin.APIKey, Log: logged}
}

func (o *Owner) Call(t *testing.T, method, path, body string) (status int, reply string) {
	t.Helper()
	req := httptest.NewRequestWithContext(t.Context(), method, path, strings.NewReader(body))
	req.Header.Set("X-Api-Key", o.apiKey)
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	w := httptest.NewRecorder()
	o.handler.ServeHTTP(w, req)
	return w.Code, w.Body.String()
}

// Expect calls and compares the reply, masked, to want.
func (o *Owner) Expect(t *testing.T, method, path, body string, status int, want string) {
	t.Helper()
	code, got := o.Call(t, method, path, body)
	if code != status || Masked(got) != want+"\n" {
		t.Fatalf("%s %s = %d %s, want %d %s", method, path, code, Masked(got), status, want)
	}
}

var (
	uuids = regexp.MustCompile(`[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}`)
	times = regexp.MustCompile(`"\d{4}-\d{2}-\d{2}T[0-9:.]+Z"`)
)

// Masked numbers each distinct UUID id1, id2, ... in order of appearance
// and replaces every time with "T".
func Masked(body string) string {
	seen := map[string]string{}
	body = uuids.ReplaceAllStringFunc(body, func(id string) string {
		if _, ok := seen[id]; !ok {
			seen[id] = fmt.Sprintf("id%d", len(seen)+1)
		}
		return seen[id]
	})
	return times.ReplaceAllString(body, `"T"`)
}

// FirstID is the first UUID in body.
func FirstID(body string) string { return uuids.FindString(body) }
