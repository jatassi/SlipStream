package rootsapi_test

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/jatassi/slipstream/internal/api"
	"github.com/jatassi/slipstream/internal/api/ownertest"
	"github.com/jatassi/slipstream/internal/contenttype/contenttypetest"
	"github.com/jatassi/slipstream/internal/files"
	"github.com/jatassi/slipstream/internal/grabs"
	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/release"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/roots"
	"github.com/jatassi/slipstream/internal/rootsapi"
	"github.com/jatassi/slipstream/internal/secret"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/storage/storagetest"
	"github.com/jatassi/slipstream/internal/targets"
	"github.com/jatassi/slipstream/internal/works"
	"github.com/jatassi/slipstream/internal/worksapi"
)

type published struct{ n int }

func (p *published) Publish(context.Context, roots.Changed) { p.n++ }

type discard[T any] struct{}

func (discard[T]) Publish(context.Context, T) {}

type node struct {
	r *roots.Roots
	w *works.Works
}

func (n node) Roots() (*roots.Roots, bool) { return n.r, true }
func (n node) Works() (*works.Works, bool) { return n.w, true }

type named struct{}

func (named) Describe(t works.TypeID) (works.Description, bool) {
	return works.Description{Name: string(t)}, true
}

var self = role.NodeID{UUID: storage.NewUUID()}

type server struct {
	*ownertest.Owner
	lib     *storage.DB[storage.Library]
	fx      *contenttypetest.Fixture
	changed *published
}

// serve serves the roots' operations and addWork over a library with two
// Content Types.
func serve(t *testing.T) *server {
	t.Helper()
	lib := storagetest.Library(t, secret.Schema(), identity.Schema(), works.Schema(), targets.Schema(), roots.Schema(), grabs.Schema(), files.Schema())
	fx := contenttypetest.New()
	w, err := works.Open(t.Context(), works.Config{
		Library:      lib,
		Catalogs:     map[works.TypeID]works.Catalog{contenttypetest.ID: fx, "other": fx},
		Timings:      works.Timings{Fetch: 5 * time.Second},
		Changed:      discard[works.WorkChanged]{},
		TypesChanged: discard[works.ContentTypesChanged]{},
		Log:          slog.New(slog.DiscardHandler),
	})
	if err != nil {
		t.Fatal(err)
	}
	_, err = targets.Open(t.Context(), targets.Config{
		Library: lib, Works: named{}, Types: map[works.TypeID]targets.Facet{contenttypetest.ID: fx, "other": fx}, Changed: discard[targets.Changed]{},
	})
	if err != nil {
		t.Fatal(err)
	}
	changed := &published{}
	svc, err := roots.Open(t.Context(), roots.Config{
		Library: lib, Types: []works.TypeID{contenttypetest.ID, "other"}, Self: self, Contents: files.RootContents{}, Changed: changed,
	})
	if err != nil {
		t.Fatal(err)
	}
	n := node{svc, w}
	owner := ownertest.Serve(t, lib, func(r *api.Router) {
		rootsapi.Routes(r, n)
		worksapi.Routes(r, n)
	})
	return &server{Owner: owner, lib: lib, fx: fx, changed: changed}
}

func body(contentType, path string) string {
	return `{"contentType":"` + contentType + `","path":` + strconv.Quote(path) + `}`
}

// create makes a root and answers its id.
func (s *server) create(t *testing.T, reqBody string) string {
	t.Helper()
	code, reply := s.Call(t, http.MethodPost, "/api/v1/roots", reqBody)
	if code != http.StatusCreated {
		t.Fatalf("createRoot %s = %d %s", reqBody, code, reply)
	}
	return ownertest.FirstID(reply)
}

func (s *server) makeDefault(t *testing.T, id string) {
	t.Helper()
	if code, reply := s.Call(t, http.MethodPost, "/api/v1/roots/"+id+"/default", ""); code != http.StatusNoContent || reply != "" {
		t.Fatalf("makeDefaultRoot %s = %d %s, want 204", id, code, reply)
	}
}

// add adds Work key of the fixture type with extra fields in addWork's
// body and answers its id.
func (s *server) add(t *testing.T, key, extra string) string {
	t.Helper()
	s.fx.Put(works.Key(key), contenttypetest.Record{Tree: works.Fetched{Label: key, Children: []works.Entry{
		{Key: works.Key(key + "-level"), Label: "Level", Children: []works.Entry{{Key: works.Key(key + "-item"), Label: "Item"}}},
	}}})
	code, reply := s.Call(t, http.MethodPost, "/api/v1/works", `{"contentType":"fixture","key":"`+key+`"`+extra+`}`)
	if code != http.StatusOK {
		t.Fatalf("addWork %s = %d %s", key, code, reply)
	}
	var out struct {
		Work struct {
			ID string `json:"id"`
		} `json:"work"`
	}
	if err := json.Unmarshal([]byte(reply), &out); err != nil {
		t.Fatal(err)
	}
	return out.Work.ID
}

// works lists each root's id, default flag and Work count.
func (s *server) works(t *testing.T) string {
	t.Helper()
	code, reply := s.Call(t, http.MethodGet, "/api/v1/roots", "")
	if code != http.StatusOK {
		t.Fatalf("listRoots = %d %s", code, reply)
	}
	var list struct {
		Roots []struct {
			ID      string `json:"id"`
			Default bool   `json:"default"`
			Works   int64  `json:"works"`
		} `json:"roots"`
	}
	if err := json.Unmarshal([]byte(reply), &list); err != nil {
		t.Fatal(err)
	}
	var out strings.Builder
	for _, r := range list.Roots {
		fmt.Fprintf(&out, "%s default=%t works=%d\n", r.ID, r.Default, r.Works)
	}
	return out.String()
}

func TestARootIsCreatedOnThisNodeAndListed(t *testing.T) {
	s := serve(t)
	dir := t.TempDir()
	want := `{"id":"id1","contentType":"fixture","nodeId":"id2","path":` + strconv.Quote(dir) + `,"default":true,"works":0,"fileBytes":0}`
	s.Expect(t, http.MethodPost, "/api/v1/roots", body("fixture", dir+string(filepath.Separator)), http.StatusCreated, want)
	_, list := s.Call(t, http.MethodGet, "/api/v1/roots", "")
	if want := `{"roots":[{"id":"` + ownertest.FirstID(list) + `","contentType":"fixture","nodeId":"` + self.String() + `","path":` + strconv.Quote(dir) + `,"default":true,"works":0,"fileBytes":0}]}` + "\n"; list != want {
		t.Fatalf("listRoots = %s, want %s", list, want)
	}
	entries, err := os.ReadDir(dir)
	if err != nil || len(entries) != 0 {
		t.Fatalf("the root holds %v (%v), want nothing left behind", entries, err)
	}
	if s.changed.n != 1 {
		t.Fatalf("roots.changed published %d times, want 1", s.changed.n)
	}
}

func TestEachRootsErrorAnswersItsProblem(t *testing.T) {
	s := serve(t)
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
	s.Expect(t, http.MethodPost, "/api/v1/roots/"+storage.NewUUID().String()+"/default", "", http.StatusNotFound,
		`{"status":404,"code":"root_not_found","title":"No root has that id"}`)
	if s.changed.n != 0 {
		t.Fatalf("roots.changed published %d times, want none", s.changed.n)
	}
}

func TestAFolderThisNodeCannotWriteIsUnusable(t *testing.T) {
	if runtime.GOOS == "windows" || os.Geteuid() == 0 {
		t.Skip("a read-only mode stops no write on Windows or as root")
	}
	s := serve(t)
	dir := t.TempDir()
	if err := os.Chmod(dir, 0o500); err != nil { //nolint:gosec // a folder it can list but not write
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(dir, 0o700) }) //nolint:gosec // as t.TempDir made it
	s.Expect(t, http.MethodPost, "/api/v1/roots", body("fixture", dir), http.StatusUnprocessableEntity,
		`{"status":422,"code":"root_unusable","title":"The folder can't be a root","detail":"SlipStream can't create a file in the folder: permission denied"}`)
}

func TestSeveralRootsPerType(t *testing.T) {
	s := serve(t)
	first := s.create(t, body("fixture", t.TempDir()))
	second := s.create(t, body("fixture", t.TempDir()))
	other := s.create(t, body("other", t.TempDir()))
	third := s.create(t, `{"contentType":"fixture","path":`+strconv.Quote(t.TempDir())+`,"makeDefault":false}`)
	want := first + " default=true works=0\n" + second + " default=false works=0\n" + third + " default=false works=0\n" + other + " default=true works=0\n"
	if got := s.works(t); got != want {
		t.Fatalf("listRoots =\n%swant\n%s", got, want)
	}
}

func TestTheFirstRootIsDefaultAndHomesRootlessWorks(t *testing.T) {
	s := serve(t)
	s.add(t, "w1", "")
	s.add(t, "w2", "")
	first := s.create(t, `{"contentType":"fixture","path":`+strconv.Quote(t.TempDir())+`,"makeDefault":false}`)
	if got, want := s.works(t), first+" default=true works=2\n"; got != want {
		t.Fatalf("listRoots = %s, want %s", got, want)
	}
	second := s.create(t, `{"contentType":"fixture","path":`+strconv.Quote(t.TempDir())+`,"makeDefault":true}`)
	if got, want := s.works(t), first+" default=false works=2\n"+second+" default=true works=0\n"; got != want {
		t.Fatalf("listRoots after a second root made default =\n%swant\n%s", got, want)
	}
}

func TestMakeDefaultMovesTheDefault(t *testing.T) {
	s := serve(t)
	first := s.create(t, body("fixture", t.TempDir()))
	second := s.create(t, body("fixture", t.TempDir()))
	s.makeDefault(t, second)
	if got, want := s.works(t), first+" default=false works=0\n"+second+" default=true works=0\n"; got != want {
		t.Fatalf("listRoots =\n%swant\n%s", got, want)
	}
	s.makeDefault(t, second)
	s.add(t, "w1", "")
	if got, want := s.works(t), first+" default=false works=0\n"+second+" default=true works=1\n"; got != want {
		t.Fatalf("listRoots after adding a Work =\n%swant\n%s", got, want)
	}
}

func TestAddWorkTakesARootOrTheDefault(t *testing.T) {
	s := serve(t)
	s.add(t, "rootless", "")
	first := s.create(t, body("fixture", t.TempDir()))
	second := s.create(t, body("fixture", t.TempDir()))
	other := s.create(t, body("other", t.TempDir()))
	s.add(t, "w1", "")
	s.add(t, "w2", `,"rootId":"`+second+`"`)
	want := first + " default=true works=2\n" + second + " default=false works=1\n" + other + " default=true works=0\n"
	if got := s.works(t); got != want {
		t.Fatalf("listRoots =\n%swant\n%s", got, want)
	}
	s.fx.Put("w3", contenttypetest.Record{Tree: works.Fetched{Label: "w3"}})
	s.Expect(t, http.MethodPost, "/api/v1/works", `{"contentType":"fixture","key":"w3","rootId":"`+other+`"}`, http.StatusUnprocessableEntity,
		`{"status":422,"code":"root_wrong_type","title":"The root holds another Content Type's Works"}`)
	s.Expect(t, http.MethodPost, "/api/v1/works", `{"contentType":"fixture","key":"w3","rootId":"`+storage.NewUUID().String()+`"}`, http.StatusNotFound,
		`{"status":404,"code":"root_not_found","title":"No root has that id"}`)
	if got := s.works(t); got != want {
		t.Fatalf("listRoots after refused adds =\n%swant\n%s", got, want)
	}
}

func TestSetWorkRootRefusesAWorkWithFiles(t *testing.T) {
	s := serve(t)
	first := s.create(t, body("fixture", t.TempDir()))
	second := s.create(t, body("fixture", t.TempDir()))
	other := s.create(t, body("other", t.TempDir()))
	empty := s.add(t, "empty", "")
	held := s.add(t, "held", "")

	set := func(root string) string { return `{"rootId":"` + root + `"}` }
	s.Expect(t, http.MethodPatch, "/api/v1/works/"+empty, set(second), http.StatusOK, `{"workId":"id1","rootId":"id2"}`)
	if got, want := s.works(t), first+" default=true works=1\n"+second+" default=false works=1\n"+other+" default=true works=0\n"; got != want {
		t.Fatalf("listRoots after moving the empty Work =\n%swant\n%s", got, want)
	}

	err := s.lib.Write(t.Context(), func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
		w, err := works.SummaryOf(ctx, tx, works.WorkID{UUID: must(storage.ParseUUID(held))})
		if err != nil {
			return err
		}
		items, err := works.ItemsOf(ctx, tx, []works.WorkID{w.ID})
		if err != nil {
			return err
		}
		root := roots.ID{UUID: must(storage.ParseUUID(first))}
		_, err = files.Add(ctx, tx, files.File{
			WorkID: w.ID, ItemID: items[w.ID][0].ID, Cut: release.Standard, RootID: root, Path: "held/held.mkv", Size: 2048,
			Probe: files.Probe{Width: 1920, Height: 1080}, Quality: release.WEBDL1080p,
		}, nil)
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	s.Expect(t, http.MethodPatch, "/api/v1/works/"+held, set(second), http.StatusConflict,
		`{"status":409,"code":"work_has_files","title":"The Work has files, so its root can't change"}`)
	s.Expect(t, http.MethodPatch, "/api/v1/works/"+held, set(first), http.StatusOK, `{"workId":"id1","rootId":"id2"}`)
	s.Expect(t, http.MethodPatch, "/api/v1/works/"+empty, set(other), http.StatusUnprocessableEntity,
		`{"status":422,"code":"root_wrong_type","title":"The root holds another Content Type's Works"}`)
	s.Expect(t, http.MethodPatch, "/api/v1/works/"+storage.NewUUID().String(), set(first), http.StatusNotFound,
		`{"status":404,"code":"work_not_found","title":"No Work has that id"}`)
	s.Expect(t, http.MethodPatch, "/api/v1/works/"+empty, set(storage.NewUUID().String()), http.StatusNotFound,
		`{"status":404,"code":"root_not_found","title":"No root has that id"}`)
	_, list := s.Call(t, http.MethodGet, "/api/v1/roots", "")
	var listed struct {
		Roots []struct {
			Works     int64 `json:"works"`
			FileBytes int64 `json:"fileBytes"`
		} `json:"roots"`
	}
	if err := json.Unmarshal([]byte(list), &listed); err != nil {
		t.Fatal(err)
	}
	if r := listed.Roots[0]; r.Works != 1 || r.FileBytes != 2048 {
		t.Fatalf("the first root holds %d Works and %d bytes, want the held Work's 2048", r.Works, r.FileBytes)
	}
}

func must[T any](v T, err error) T {
	if err != nil {
		panic(err)
	}
	return v
}
