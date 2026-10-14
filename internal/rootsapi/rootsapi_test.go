package rootsapi_test

import (
	"context"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"testing"

	"github.com/jatassi/slipstream/internal/api"
	"github.com/jatassi/slipstream/internal/api/ownertest"
	"github.com/jatassi/slipstream/internal/contenttype/contenttypetest"
	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/roots"
	"github.com/jatassi/slipstream/internal/rootsapi"
	"github.com/jatassi/slipstream/internal/secret"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/storage/storagetest"
	"github.com/jatassi/slipstream/internal/works"
)

type published struct{ n int }

func (p *published) Publish(context.Context, roots.Changed) { p.n++ }

type folders struct{ r *roots.Roots }

func (f folders) Roots() (*roots.Roots, bool) { return f.r, true }

var self = role.NodeID{UUID: storage.NewUUID()}

func serve(t *testing.T) (*ownertest.Owner, *published) {
	t.Helper()
	lib := storagetest.Library(t, secret.Schema(), identity.Schema(), works.Schema(), roots.Schema())
	changed := &published{}
	svc := roots.New(roots.Config{Library: lib, Types: []works.TypeID{contenttypetest.ID}, Self: self, Changed: changed})
	return ownertest.Serve(t, lib, func(r *api.Router) { rootsapi.Routes(r, folders{svc}) }), changed
}

func body(contentType, path string) string {
	return `{"contentType":"` + contentType + `","path":` + strconv.Quote(path) + `}`
}

func TestARootIsCreatedOnThisNodeAndListed(t *testing.T) {
	s, changed := serve(t)
	dir := t.TempDir()
	want := `{"id":"id1","contentType":"fixture","nodeId":"id2","path":` + strconv.Quote(dir) + `}`
	s.Expect(t, http.MethodPost, "/api/v1/roots", body("fixture", dir+string(filepath.Separator)), http.StatusCreated, want)
	_, list := s.Call(t, http.MethodGet, "/api/v1/roots", "")
	if want := `{"roots":[{"id":"` + ownertest.FirstID(list) + `","contentType":"fixture","nodeId":"` + self.String() + `","path":` + strconv.Quote(dir) + `}]}` + "\n"; list != want {
		t.Fatalf("listRoots = %s, want %s", list, want)
	}
	entries, err := os.ReadDir(dir)
	if err != nil || len(entries) != 0 {
		t.Fatalf("the root holds %v (%v), want nothing left behind", entries, err)
	}
	if changed.n != 1 {
		t.Fatalf("roots.changed published %d times, want 1", changed.n)
	}
}

func TestEachRootsErrorAnswersItsProblem(t *testing.T) {
	s, changed := serve(t)
	dir := t.TempDir()
	afile := filepath.Join(dir, "a.txt")
	if err := os.WriteFile(afile, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	unusable := func(reason string) string {
		return `{"status":422,"code":"root_unusable","title":"The folder can't be a root","detail":"` + reason + `"}`
	}
	s.Expect(t, http.MethodPost, "/api/v1/roots", body("ghost", dir), http.StatusNotFound,
		`{"status":404,"code":"content_type_not_found","title":"No Content Type has that id"}`)
	s.Expect(t, http.MethodPost, "/api/v1/roots", body("fixture", "media"), http.StatusUnprocessableEntity, unusable("the path is not absolute"))
	s.Expect(t, http.MethodPost, "/api/v1/roots", body("fixture", filepath.Join(dir, "gone")), http.StatusUnprocessableEntity, unusable("no folder is at the path"))
	s.Expect(t, http.MethodPost, "/api/v1/roots", body("fixture", afile), http.StatusUnprocessableEntity, unusable("the path is a file, not a folder"))
	if code, reply := s.Call(t, http.MethodPost, "/api/v1/roots", body("fixture", dir)); code != http.StatusCreated {
		t.Fatalf("createRoot = %d %s", code, reply)
	}
	s.Expect(t, http.MethodPost, "/api/v1/roots", body("fixture", t.TempDir()), http.StatusConflict,
		`{"status":409,"code":"root_exists","title":"The Content Type already has a root"}`)
	if changed.n != 1 {
		t.Fatalf("roots.changed published %d times, want 1", changed.n)
	}
}

func TestAFolderThisNodeCannotWriteIsUnusable(t *testing.T) {
	if runtime.GOOS == "windows" || os.Geteuid() == 0 {
		t.Skip("a read-only mode stops no write on Windows or as root")
	}
	s, _ := serve(t)
	dir := t.TempDir()
	if err := os.Chmod(dir, 0o500); err != nil { //nolint:gosec // a folder it can list but not write
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(dir, 0o700) }) //nolint:gosec // as t.TempDir made it
	s.Expect(t, http.MethodPost, "/api/v1/roots", body("fixture", dir), http.StatusUnprocessableEntity,
		`{"status":422,"code":"root_unusable","title":"The folder can't be a root","detail":"SlipStream can't create a file in the folder: permission denied"}`)
}
