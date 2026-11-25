package mediaserversapi_test

import (
	"go/ast"
	"go/parser"
	"go/token"
	"net/http"
	"path/filepath"
	"testing"
	"testing/synctest"

	"github.com/jatassi/slipstream/internal/identity/identitytest"
	"github.com/jatassi/slipstream/internal/storage"
)

// notYet is every function and method the Media Servers declare in a
// notyet_*.go file: the parts of S43 (Member access to Media Servers)
// whose lanes haven't landed.
func notYet(t *testing.T) map[string]bool {
	t.Helper()
	paths, err := filepath.Glob(filepath.Join("..", "mediaservers", "notyet_*.go"))
	if err != nil {
		t.Fatal(err)
	}
	fset := token.NewFileSet()
	out := map[string]bool{}
	for _, p := range paths {
		f, err := parser.ParseFile(fset, p, nil, parser.SkipObjectResolution)
		if err != nil {
			t.Fatal(err)
		}
		for _, d := range f.Decls {
			if fn, ok := d.(*ast.FuncDecl); ok {
				out[fn.Name.Name] = true
			}
		}
	}
	return out
}

// TestNotYetLedger drives every operation S43 (Member access to Media
// Servers) adds once, and asserts each answers 501 exactly while the
// function behind it sits in a notyet_*.go file.
func TestNotYetLedger(t *testing.T) {
	stubs := notYet(t)
	synctest.Test(t, func(t *testing.T) {
		r := newRig(t)
		r.w.Jellyfin(t, host, den(floorVer))
		server := r.add(t)
		member := identitytest.Session(t, r.w.Lib, identitytest.Member(t, r.w.Lib, "member@example.com"))
		row := storage.NewUUID().String()
		mine := "/api/v1/account/media-servers"
		for _, c := range []struct {
			method, path, body string
			member             bool
			stub               string
		}{
			{http.MethodGet, mine, "", true, "Mine"},
			{http.MethodPost, mine + "/plex/pin", "", true, "StartMemberPin"},
			{http.MethodGet, mine + "/plex/pin/1", "", true, "MemberPin"},
			{http.MethodDelete, mine + "/plex", "", true, "AnotherPlex"},
			{http.MethodPost, mine + "/" + row + "/account", `{"username":"sam","password":"sam-password"}`, true, "MakeAccount"},
			{http.MethodPost, mine + "/" + row + "/check", "", true, "CheckMine"},
			{http.MethodGet, servers + "/access", "", false, "Access"},
			{http.MethodGet, servers + "/" + server + "/users", "", false, "LinkableUsers"},
			{http.MethodPut, "/api/v1/accounts/" + storage.NewUUID().String() + "/media-servers/" + server, `{}`, false, "Offer"},
			{http.MethodPost, servers + "/access/" + row + "/share", `{"as":"friend"}`, false, "Share"},
			{http.MethodPost, servers + "/access/" + row + "/mark-added", "", false, "MarkAdded"},
			{http.MethodPost, servers + "/access/" + row + "/link", `{"userId":"u1"}`, false, "LinkUser"},
			{http.MethodPost, servers + "/access/" + row + "/check", "", false, "CheckShare"},
			{http.MethodDelete, servers + "/access/" + row, "", false, "RemoveAccess"},
			{http.MethodPost, servers + "/plex/pin/1/replace", "", false, "ReplacePlexAccount"},
			{http.MethodPatch, servers + "/" + server, `{"membersSee":true}`, false, "patchMembers"},
		} {
			var status int
			var body string
			if c.member {
				status, body = r.o.CallAs(t, c.method, c.path, c.body, member)
			} else {
				status, body = r.o.CallSignedIn(t, c.method, c.path, c.body)
			}
			if unbuilt := status == http.StatusNotImplemented; unbuilt != stubs[c.stub] {
				t.Errorf("%s %s = %d %s; %s in a notyet file: %t", c.method, c.path, status, body, c.stub, stubs[c.stub])
			}
		}
	})
}
