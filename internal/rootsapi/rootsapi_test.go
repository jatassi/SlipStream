package rootsapi_test

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
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
	"github.com/jatassi/slipstream/internal/indexers"
	"github.com/jatassi/slipstream/internal/link"
	"github.com/jatassi/slipstream/internal/metadata/artwork"
	"github.com/jatassi/slipstream/internal/nodes"
	nodev1 "github.com/jatassi/slipstream/internal/pb/slipstream/node/v1"
	storagev1 "github.com/jatassi/slipstream/internal/pb/slipstream/storage/v1"
	"github.com/jatassi/slipstream/internal/pb/slipstream/storage/v1/storagev1connect"
	"github.com/jatassi/slipstream/internal/release"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/roots"
	"github.com/jatassi/slipstream/internal/roots/rootstest"
	"github.com/jatassi/slipstream/internal/rootsapi"
	"github.com/jatassi/slipstream/internal/rpc"
	"github.com/jatassi/slipstream/internal/secret"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/storage/storagetest"
	"github.com/jatassi/slipstream/internal/storing"
	"github.com/jatassi/slipstream/internal/targets"
	"github.com/jatassi/slipstream/internal/works"
	"github.com/jatassi/slipstream/internal/worksapi"
)

type published struct{ ids [][]string }

func (p *published) Publish(_ context.Context, c roots.Changed) { p.ids = append(p.ids, c.RootIDs) }

type discard[T any] struct{}

func (discard[T]) Publish(context.Context, T) {}

type node struct {
	r *roots.Roots
	w *works.Works
}

func (n node) Roots() (*roots.Roots, bool) { return n.r, true }
func (n node) Works() (*works.Works, bool) { return n.w, true }
func (node) Minter() (artwork.Minter, bool) {
	return artwork.NewMinter([]byte("rootsapi")), true
}

type named struct{}

func (named) Describe(t works.TypeID) (works.Description, bool) {
	return works.Description{Name: string(t)}, true
}

var (
	self = role.NodeID{UUID: storage.NewUUID()}
	// away holds Storage but is out of step: no call reaches it.
	away = role.NodeID{UUID: storage.NewUUID()}
	// bare doesn't hold Storage.
	bare = role.NodeID{UUID: storage.NewUUID()}
)

type roster struct{}

func (roster) Holds(_ role.Name, n role.NodeID) bool { return n != bare }
func (roster) InStep(n role.NodeID) bool             { return n != away }

type unreached struct {
	storagev1connect.UnimplementedStorageServiceHandler
	err error
}

func (u unreached) Check(context.Context, *storagev1.CheckRequest) (*storagev1.CheckResponse, error) {
	return nil, u.err
}

func (u unreached) Browse(context.Context, *storagev1.BrowseRequest) (*storagev1.BrowseResponse, error) {
	return nil, u.err
}

// gated is self's Store, calling verifying first, when set, on each Verify.
type gated struct {
	storagev1connect.StorageServiceHandler
	s *server
}

func (g gated) Verify(ctx context.Context, req *storagev1.VerifyRequest) (*storagev1.VerifyResponse, error) {
	if g.s.verifying != nil {
		g.s.verifying()
	}
	return g.StorageServiceHandler.Verify(ctx, req)
}

type server struct {
	*ownertest.Owner
	lib             *storage.DB[storage.Library]
	fx              *contenttypetest.Fixture
	changed         *published
	data, downloads string
	verifying       func()
}

// serve serves the roots' operations and addWork over a library with two
// Content Types. self's Storage is a Store on this machine.
func serve(t *testing.T) *server {
	t.Helper()
	lib := storagetest.Library(t, secret.Schema(), identity.Schema(), nodes.Schema(), works.Schema(), targets.Schema(), roots.Schema(), indexers.Schema(), grabs.Schema(), files.Schema())
	fx := contenttypetest.New()
	w, err := works.Open(t.Context(), works.Config{
		Library:      lib,
		Catalogs:     map[works.TypeID]works.Catalog{contenttypetest.ID: fx, "other": fx},
		Source:       fx.SourceAs(contenttypetest.ID, "other"),
		Artwork:      fx,
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
	s := &server{lib: lib, fx: fx, changed: &published{}, data: t.TempDir(), downloads: t.TempDir()}
	cfg := rootstest.Config(t, lib, contenttypetest.ID, "other")
	local := rootstest.Remote(t, s.data, s.downloads)
	cfg.Roster, cfg.Changed = roster{}, s.changed
	cfg.Remote = func(n role.NodeID) storagev1connect.StorageServiceHandler {
		switch n {
		case away:
			return unreached{err: fmt.Errorf("call: %w", link.ErrDisconnected)}
		case bare:
			return unreached{err: rpc.ErrorWith(rpc.CodeFailedPrecondition, &nodev1.NotHolder{Role: string(role.Storage)}, "not a holder")}
		}
		return gated{local(n), s}
	}
	svc, err := roots.Open(t.Context(), cfg)
	if err != nil {
		t.Fatal(err)
	}
	n := node{svc, w}
	s.Owner = ownertest.Serve(t, lib, func(r *api.Router) {
		rootsapi.Routes(r, n)
		worksapi.Routes(r, n)
	})
	return s
}

func body(contentType, path string) string {
	return `{"contentType":"` + contentType + `","nodeId":"` + self.String() + `","path":` + strconv.Quote(path) + `}`
}

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

func TestARootIsCreatedOnItsNodeAndListed(t *testing.T) {
	s := serve(t)
	dir := t.TempDir()
	want := `{"id":"id1","contentType":"fixture","nodeId":"id2","path":` + strconv.Quote(dir) + `,"default":true,"state":"checking","works":0,"fileBytes":0}`
	s.Expect(t, http.MethodPost, "/api/v1/roots", body("fixture", dir+string(filepath.Separator)), http.StatusCreated, want)
	_, list := s.Call(t, http.MethodGet, "/api/v1/roots", "")
	id := ownertest.FirstID(list)
	if want := `{"roots":[{"id":"` + id + `","contentType":"fixture","nodeId":"` + self.String() + `","path":` + strconv.Quote(dir) + `,"default":true,"state":"checking","works":0,"fileBytes":0}]}` + "\n"; list != want {
		t.Fatalf("listRoots = %s, want %s", list, want)
	}
	entries, err := os.ReadDir(dir)
	if err != nil || len(entries) != 0 {
		t.Fatalf("the root holds %v (%v), want nothing left behind", entries, err)
	}
	if got := fmt.Sprint(s.changed.ids); got != "[["+id+"]]" {
		t.Fatalf("roots.changed published %s, want the new root once", got)
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
		return `{"status":422,"code":"root_unusable","title":"The folder can't be a root","refusal":{"reason":"` + reason + `"}}`
	}
	s.Expect(t, http.MethodPost, "/api/v1/roots", body("ghost", dir), http.StatusNotFound,
		`{"status":404,"code":"content_type_not_found","title":"No Content Type has that id"}`)
	s.Expect(t, http.MethodPost, "/api/v1/roots", body("fixture", "media"), http.StatusUnprocessableEntity, unusable("not_absolute"))
	s.Expect(t, http.MethodPost, "/api/v1/roots", body("fixture", filepath.Join(dir, "gone")), http.StatusUnprocessableEntity, unusable("not_found"))
	s.Expect(t, http.MethodPost, "/api/v1/roots", body("fixture", afile), http.StatusUnprocessableEntity, unusable("not_a_folder"))
	s.Expect(t, http.MethodPost, "/api/v1/roots", body("fixture", s.data), http.StatusUnprocessableEntity, unusable("inside_data"))
	s.Expect(t, http.MethodPost, "/api/v1/roots", body("fixture", filepath.Dir(s.downloads)), http.StatusUnprocessableEntity, unusable("contains_data"))
	s.Expect(t, http.MethodPost, "/api/v1/roots", body("fixture", s.downloads), http.StatusUnprocessableEntity, unusable("inside_downloads"))
	s.Expect(t, http.MethodPost, "/api/v1/roots", `{"contentType":"fixture","nodeId":"`+away.String()+`","path":`+strconv.Quote(dir)+`}`,
		http.StatusUnprocessableEntity, unusable("node_offline"))
	s.Expect(t, http.MethodPost, "/api/v1/roots", `{"contentType":"fixture","nodeId":"`+bare.String()+`","path":`+strconv.Quote(dir)+`}`,
		http.StatusConflict, `{"status":409,"code":"node_not_storage","title":"The Node doesn't hold Storage"}`)
	s.Expect(t, http.MethodPost, "/api/v1/roots/"+storage.NewUUID().String()+"/default", "", http.StatusNotFound,
		`{"status":404,"code":"root_not_found","title":"No root has that id"}`)
	if len(s.changed.ids) != 0 {
		t.Fatalf("roots.changed published %v, want none", s.changed.ids)
	}
}

func TestAFolderItsNodeCannotWriteIsUnusable(t *testing.T) {
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
		`{"status":422,"code":"root_unusable","title":"The folder can't be a root","refusal":{"reason":"unwritable"}}`)
}

func TestRootsNeverOverlapAcrossTypes(t *testing.T) {
	s := serve(t)
	dir := t.TempDir()
	movies := filepath.Join(dir, "movies")
	inner := filepath.Join(movies, "inner")
	if err := os.MkdirAll(inner, 0o700); err != nil {
		t.Fatal(err)
	}
	id := s.create(t, body("fixture", movies))
	refused := func(reason string) string {
		return `{"status":422,"code":"root_unusable","title":"The folder can't be a root","refusal":{"reason":"` + reason + `","rootId":"id1"}}`
	}
	for _, c := range []struct{ path, reason string }{{movies, "inside_root"}, {inner, "inside_root"}, {dir, "contains_root"}} {
		code, reply := s.Call(t, http.MethodPost, "/api/v1/roots", body("other", c.path))
		if code != http.StatusUnprocessableEntity || ownertest.Masked(reply) != refused(c.reason)+"\n" || ownertest.FirstID(reply) != id {
			t.Errorf("createRoot %s = %d %s, want %s of %s", c.path, code, reply, c.reason, id)
		}
	}
	marked := t.TempDir()
	if err := os.WriteFile(filepath.Join(marked, storing.MarkerName), []byte(id), 0o600); err != nil {
		t.Fatal(err)
	}
	code, reply := s.Call(t, http.MethodPost, "/api/v1/roots", body("other", marked))
	if code != http.StatusUnprocessableEntity || ownertest.Masked(reply) != refused("inside_root")+"\n" || ownertest.FirstID(reply) != id {
		t.Errorf("createRoot on a folder whose marker names %s = %d %s, want inside_root", id, code, reply)
	}
	stray := t.TempDir()
	if err := os.WriteFile(filepath.Join(stray, storing.MarkerName), []byte(storage.NewUUID().String()), 0o600); err != nil {
		t.Fatal(err)
	}
	s.create(t, body("other", stray))
}

func TestCheckRootJudgesWithoutSaving(t *testing.T) {
	s := serve(t)
	dir := t.TempDir()
	check := func(nodeID, path, extra string) (int, string) {
		return s.Call(t, http.MethodPost, "/api/v1/roots/check", `{"nodeId":"`+nodeID+`","path":`+strconv.Quote(path)+extra+`}`)
	}
	code, reply := check(self.String(), dir+string(filepath.Separator), "")
	var v struct {
		Usable bool   `json:"usable"`
		Path   string `json:"path"`
		Volume *struct {
			ID    string `json:"id"`
			Total int64  `json:"total"`
		} `json:"volume"`
	}
	if err := json.Unmarshal([]byte(reply), &v); code != http.StatusOK || err != nil || !v.Usable || v.Path != dir || v.Volume == nil || v.Volume.ID == "" || v.Volume.Total <= 0 {
		t.Fatalf("checkRoot %s = %d %s, want usable on its volume", dir, code, reply)
	}
	if code, reply := s.Call(t, http.MethodGet, "/api/v1/roots", ""); code != http.StatusOK || reply != `{"roots":[]}`+"\n" {
		t.Fatalf("listRoots after a check = %d %s, want none saved", code, reply)
	}
	id := s.create(t, body("fixture", dir))
	if code, reply := check(self.String(), dir, ""); code != http.StatusOK || ownertest.Masked(reply) != `{"usable":false,"path":`+strconv.Quote(dir)+`,"volume":`+volumeOf(t, reply)+`,"refusal":{"reason":"inside_root","rootId":"id1"}}`+"\n" {
		t.Errorf("checkRoot on a root = %d %s, want inside_root", code, reply)
	}
	if code, reply := check(self.String(), dir, `,"rootId":"`+id+`"`); code != http.StatusOK || !strings.HasPrefix(reply, `{"usable":true,`) {
		t.Errorf("checkRoot on the root being re-homed = %d %s, want usable", code, reply)
	}
	s.Expect(t, http.MethodPost, "/api/v1/roots/check", `{"nodeId":"`+away.String()+`","path":`+strconv.Quote(dir)+`}`, http.StatusOK,
		`{"usable":false,"path":`+strconv.Quote(dir)+`,"refusal":{"reason":"node_offline"}}`)
	s.Expect(t, http.MethodPost, "/api/v1/roots/check", `{"nodeId":"`+bare.String()+`","path":`+strconv.Quote(dir)+`}`, http.StatusConflict,
		`{"status":409,"code":"node_not_storage","title":"The Node doesn't hold Storage"}`)
}

// volumeOf is the volume member of a verdict, masked as Expect masks it.
func volumeOf(t *testing.T, reply string) string {
	t.Helper()
	var v struct {
		Volume json.RawMessage `json:"volume"`
	}
	if err := json.Unmarshal([]byte(reply), &v); err != nil {
		t.Fatal(err)
	}
	return ownertest.Masked(string(v.Volume))
}

func TestFoldersOnAStorageNode(t *testing.T) {
	s := serve(t)
	dir := t.TempDir()
	names, hidden := []string{"Season 10", "Season 2"}, 0
	if runtime.GOOS != "windows" {
		names, hidden = append(names, ".hidden"), 1
	}
	for _, name := range names {
		if err := os.Mkdir(filepath.Join(dir, name), 0o700); err != nil {
			t.Fatal(err)
		}
	}
	folders := "/api/v1/nodes/" + self.String() + "/folders"
	code, reply := s.Call(t, http.MethodGet, folders+"?path="+url.QueryEscape(dir), "")
	var l struct {
		Path    string `json:"path"`
		Folders []struct {
			Name     string `json:"name"`
			Readable bool   `json:"readable"`
		} `json:"folders"`
		Hidden int `json:"hidden"`
	}
	if err := json.Unmarshal([]byte(reply), &l); code != http.StatusOK || err != nil || l.Path != dir || l.Hidden != hidden ||
		fmt.Sprint(l.Folders) != "[{Season 2 true} {Season 10 true}]" {
		t.Fatalf("browseFolders %s = %d %s", dir, code, reply)
	}
	code, reply = s.Call(t, http.MethodGet, folders, "")
	if code != http.StatusOK || !strings.Contains(reply, `"places":[{`) {
		t.Fatalf("browseFolders of the places = %d %s", code, reply)
	}
	code, reply = s.Call(t, http.MethodGet, folders+"/suggest?text="+url.QueryEscape(filepath.Join(dir, "season 1")), "")
	if want := `{"suggestions":[` + strconv.Quote(filepath.Join(dir, "Season 10")) + `]}` + "\n"; code != http.StatusOK || reply != want {
		t.Fatalf("suggestFolders = %d %s, want %s", code, reply, want)
	}
	made := `{"parent":` + strconv.Quote(dir) + `,"name":"Movies"}`
	s.Expect(t, http.MethodPost, folders, made, http.StatusCreated, `{"path":`+strconv.Quote(filepath.Join(dir, "Movies"))+`}`)
	s.Expect(t, http.MethodPost, folders, made, http.StatusConflict, `{"status":409,"code":"folder_exists","title":"A folder of that name is already there"}`)
	code, reply = s.Call(t, http.MethodPost, folders, `{"parent":`+strconv.Quote(dir)+`,"name":"a/b"}`)
	if code != http.StatusUnprocessableEntity || !strings.HasPrefix(reply, `{"status":422,"code":"folder_invalid",`) {
		t.Fatalf("makeFolder a/b = %d %s, want folder_invalid", code, reply)
	}
	code, reply = s.Call(t, http.MethodGet, folders+"?path="+url.QueryEscape(filepath.Join(dir, "gone")), "")
	if code != http.StatusUnprocessableEntity || !strings.HasPrefix(reply, `{"status":422,"code":"folder_unreadable","title":"The Node can't read the folder","problem":{"kind":"not_found","osText":`) {
		t.Fatalf("browseFolders of a missing folder = %d %s, want folder_unreadable not_found", code, reply)
	}
	s.Expect(t, http.MethodGet, "/api/v1/nodes/"+away.String()+"/folders", "", http.StatusServiceUnavailable,
		`{"status":503,"code":"node_offline","title":"The Node is offline"}`)
	s.Expect(t, http.MethodGet, "/api/v1/nodes/"+bare.String()+"/folders", "", http.StatusConflict,
		`{"status":409,"code":"node_not_storage","title":"The Node doesn't hold Storage"}`)
}

func TestSeveralRootsPerType(t *testing.T) {
	s := serve(t)
	first := s.create(t, body("fixture", t.TempDir()))
	second := s.create(t, body("fixture", t.TempDir()))
	other := s.create(t, body("other", t.TempDir()))
	third := s.create(t, `{"contentType":"fixture","nodeId":"`+self.String()+`","path":`+strconv.Quote(t.TempDir())+`,"makeDefault":false}`)
	want := first + " default=true works=0\n" + second + " default=false works=0\n" + third + " default=false works=0\n" + other + " default=true works=0\n"
	if got := s.works(t); got != want {
		t.Fatalf("listRoots =\n%swant\n%s", got, want)
	}
}

func TestTheFirstRootIsDefaultAndHomesRootlessWorks(t *testing.T) {
	s := serve(t)
	s.add(t, "w1", "")
	s.add(t, "w2", "")
	first := s.create(t, `{"contentType":"fixture","nodeId":"`+self.String()+`","path":`+strconv.Quote(t.TempDir())+`,"makeDefault":false}`)
	if got, want := s.works(t), first+" default=true works=2\n"; got != want {
		t.Fatalf("listRoots = %s, want %s", got, want)
	}
	second := s.create(t, `{"contentType":"fixture","nodeId":"`+self.String()+`","path":`+strconv.Quote(t.TempDir())+`,"makeDefault":true}`)
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
