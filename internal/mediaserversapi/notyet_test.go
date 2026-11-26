package mediaserversapi_test

import (
	"go/ast"
	"go/parser"
	"go/token"
	"net/http"
	"path/filepath"
	"testing"
	"testing/synctest"
)

// notYetLedger is each operation S36 (Playback report) adds, by the
// Service method behind it.
var notYetLedger = []struct{ method, path string }{
	{"Playback", "/playback"},
	{"Plays", "/plays"},
}

// stubbed is every Service method a notyet_*.go file of mediaservers
// declares.
func stubbed(t *testing.T) map[string]bool {
	t.Helper()
	files, err := filepath.Glob("../mediaservers/notyet_*.go")
	if err != nil {
		t.Fatal(err)
	}
	out := map[string]bool{}
	for _, f := range files {
		parsed, err := parser.ParseFile(token.NewFileSet(), f, nil, parser.SkipObjectResolution)
		if err != nil {
			t.Fatal(err)
		}
		for _, d := range parsed.Decls {
			if fn, ok := d.(*ast.FuncDecl); ok && fn.Recv != nil {
				out[fn.Name.Name] = true
			}
		}
	}
	return out
}

// TestNotYetLedger drives each new operation: it answers 501
// not_built_yet exactly while its method sits in a notyet_*.go file, and
// refuses a Member either way.
func TestNotYetLedger(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		r := newRig(t)
		r.w.Jellyfin(t, host, den(floorVer))
		path := servers + "/" + r.add(t)
		sam := r.w.Member(t, "sam")
		stubs := stubbed(t)
		for _, op := range notYetLedger {
			status, body := r.o.Call(t, http.MethodGet, path+op.path, "")
			if stubs[op.method] {
				expectProblem(t, status, body, http.StatusNotImplemented, `"code":"not_built_yet"`)
			} else if status == http.StatusNotImplemented {
				t.Errorf("GET %s answers 501 though %s is built: %s", op.path, op.method, body)
			}
			if status, body := r.o.CallAs(t, http.MethodGet, path+op.path, "", sam.Session); status != http.StatusForbidden {
				t.Errorf("a Member's GET %s = %d %s, want 403", op.path, status, body)
			}
			delete(stubs, op.method)
		}
		for m := range stubs {
			t.Errorf("%s is stubbed in a notyet_*.go file but has no row in notYetLedger", m)
		}
	})
}
