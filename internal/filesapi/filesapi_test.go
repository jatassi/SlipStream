package filesapi_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/jatassi/slipstream/internal/api"
	"github.com/jatassi/slipstream/internal/api/ownertest"
	"github.com/jatassi/slipstream/internal/contenttype/contenttypetest"
	"github.com/jatassi/slipstream/internal/files"
	"github.com/jatassi/slipstream/internal/filesapi"
	"github.com/jatassi/slipstream/internal/grabs"
	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/release"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/roots"
	"github.com/jatassi/slipstream/internal/secret"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/storage/storagetest"
	"github.com/jatassi/slipstream/internal/targets"
	"github.com/jatassi/slipstream/internal/works"
)

var admin identity.Admin

type discard[T any] struct{}

func (discard[T]) Publish(context.Context, T) {}

type library struct{ f *files.Files }

func (l library) Files() (*files.Files, bool) { return l.f, true }

func must[T any](t *testing.T) func(T, error) T {
	return func(v T, err error) T {
		t.Helper()
		if err != nil {
			t.Fatal(err)
		}
		return v
	}
}

func TestGetWorkCellsAnswersEachCellAndItsFile(t *testing.T) {
	ctx := t.Context()
	lib := storagetest.Library(t, secret.Schema(), identity.Schema(), works.Schema(), targets.Schema(), roots.Schema(), grabs.Schema(), files.Schema())
	fx := contenttypetest.New()
	w := must[*works.Works](t)(works.Open(ctx, works.Config{
		Library: lib, Catalogs: map[works.TypeID]works.Catalog{contenttypetest.ID: fx},
		Changed: discard[works.WorkChanged]{}, TypesChanged: discard[works.ContentTypesChanged]{},
	}))
	types := []works.TypeID{contenttypetest.ID}
	goals := targets.New(targets.Config{Library: lib, Types: types, Changed: discard[targets.Changed]{}})
	hd := must[targets.Target](t)(goals.CreateTarget(ctx, admin, targets.Spec{
		Name: "1080p", Tiers: [][]release.Quality{{release.Bluray1080p}, {release.WEBDL1080p}}, Cutoff: release.Bluray1080p,
	}))
	uhd := must[targets.Target](t)(goals.CreateTarget(ctx, admin, targets.Spec{Name: "2160p", Tiers: [][]release.Quality{{release.WEBDL2160p}}, Cutoff: release.WEBDL2160p}))
	small := must[targets.Target](t)(goals.CreateTarget(ctx, admin, targets.Spec{Name: "Small", Tiers: [][]release.Quality{{release.WEBDL1080p}}, Cutoff: release.WEBDL1080p}))
	must[targets.Profile](t)(goals.CreateProfile(ctx, admin, "Standard", contenttypetest.ID, []targets.TargetID{hd.ID, uhd.ID, small.ID}))
	root := must[roots.Root](t)(roots.New(roots.Config{Library: lib, Types: types, Self: role.NodeID{UUID: storage.NewUUID()}, Changed: discard[roots.Changed]{}}).
		Create(ctx, admin, contenttypetest.ID, t.TempDir()))
	fx.Put("w1", contenttypetest.Record{Tree: works.Fetched{Label: "The Attic", Children: []works.Entry{
		{Key: "b1", Label: "Winter", Children: []works.Entry{{Key: "c1", Label: "Scarf"}}},
	}}})
	tree, _, err := w.Add(ctx, admin, contenttypetest.ID, "w1", true)
	if err != nil {
		t.Fatal(err)
	}
	var placed files.File
	err = lib.Write(ctx, func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
		var err error
		placed, err = files.Add(ctx, tx, files.File{
			WorkID: tree.ID, ItemID: tree.Items[0].ID, Edition: files.Standard, RootID: root.ID,
			Path: "The Attic/The Attic - 1080p.mkv", Size: 4096, Probe: files.Probe{Width: 1920, Height: 1080}, Quality: release.WEBDL1080p,
		}, []targets.TargetID{hd.ID, small.ID})
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	s := ownertest.Serve(t, lib, func(r *api.Router) {
		filesapi.Routes(r, library{files.New(files.Config{Library: lib, Works: w, Grabs: files.NoGrabs{}})})
	})

	code, body := s.Call(t, http.MethodGet, "/api/v1/works/"+tree.ID.String()+"/cells", "")
	file := `"file":{"id":"` + placed.ID.String() + `","path":"The Attic/The Attic - 1080p.mkv","size":4096,"quality":"webdl-1080p","probe":{"width":1920,"height":1080}}`
	want := `{"cells":[` +
		`{"itemId":"` + tree.Items[0].ID.String() + `","edition":"standard","targetId":"` + hd.ID.String() + `","targetName":"1080p","wanted":false,"state":"upgradable",` + file + `},` +
		`{"itemId":"` + tree.Items[0].ID.String() + `","edition":"standard","targetId":"` + uhd.ID.String() + `","targetName":"2160p","wanted":true,"state":"missing"},` +
		`{"itemId":"` + tree.Items[0].ID.String() + `","edition":"standard","targetId":"` + small.ID.String() + `","targetName":"Small","wanted":false,"state":"met",` + file + `}]}` + "\n"
	if code != http.StatusOK || body != want {
		t.Fatalf("getWorkCells = %d\n%s\nwant\n%s", code, body, want)
	}
	s.Expect(t, http.MethodGet, "/api/v1/works/"+storage.NewUUID().String()+"/cells", "", http.StatusNotFound,
		`{"status":404,"code":"work_not_found","title":"No Work has that id"}`)
}
