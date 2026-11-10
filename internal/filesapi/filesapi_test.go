package filesapi_test

import (
	"context"
	"net/http"
	"slices"
	"testing"
	"time"

	"github.com/jatassi/slipstream/internal/api"
	"github.com/jatassi/slipstream/internal/api/ownertest"
	"github.com/jatassi/slipstream/internal/contenttype/contenttypetest"
	"github.com/jatassi/slipstream/internal/downloadclients"
	"github.com/jatassi/slipstream/internal/files"
	"github.com/jatassi/slipstream/internal/filesapi"
	"github.com/jatassi/slipstream/internal/grabs"
	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/indexers"
	"github.com/jatassi/slipstream/internal/media"
	"github.com/jatassi/slipstream/internal/naming"
	"github.com/jatassi/slipstream/internal/nodes"
	"github.com/jatassi/slipstream/internal/release"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/roots"
	"github.com/jatassi/slipstream/internal/roots/rootstest"
	"github.com/jatassi/slipstream/internal/scanning"
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
	lib := storagetest.Library(t, secret.Schema(), identity.Schema(), nodes.Schema(), works.Schema(), targets.Schema(), roots.Schema(), downloadclients.Schema(), grabs.Schema(), indexers.Schema(), files.Schema(), naming.Schema())
	fx := contenttypetest.New()
	w := must[*works.Works](t)(works.Open(ctx, works.Config{
		Library: lib, Catalogs: map[works.TypeID]works.Catalog{contenttypetest.ID: fx}, Source: fx.Source(), Artwork: fx,
		Changed: discard[works.Changed]{}, TypesChanged: discard[works.ContentTypesChanged]{},
	}))
	goals := must[*targets.Targets](t)(targets.Open(ctx, targets.Config{
		Library: lib, Works: w, Types: map[works.TypeID]targets.Facet{contenttypetest.ID: fx}, Changed: discard[targets.Changed]{},
	}))
	hd := must[targets.Target](t)(goals.CreateTarget(ctx, admin, targets.Spec{
		Scores: targets.Scores{Increment: 1},
		Name:   "HD", Tiers: [][]release.Quality{{release.Bluray1080p}, {release.WEBDL1080p}}, Cutoff: release.Bluray1080p,
	}))
	uhd := must[targets.Target](t)(goals.CreateTarget(ctx, admin, targets.Spec{Scores: targets.Scores{Increment: 1}, Name: "2160p", Tiers: [][]release.Quality{{release.WEBDL2160p}}, Cutoff: release.WEBDL2160p}))
	small := must[targets.Target](t)(goals.CreateTarget(ctx, admin, targets.Spec{Scores: targets.Scores{Increment: 1}, Name: "Small", Tiers: [][]release.Quality{{release.WEBDL1080p}}, Cutoff: release.WEBDL1080p}))
	standard := must[targets.Profile](t)(goals.CreateProfile(ctx, admin, targets.ProfileSpec{Name: "Standard", ContentType: contenttypetest.ID, Entries: []targets.Entry{
		{Target: hd.ID, Sourcing: targets.Download}, {Target: uhd.ID, Sourcing: targets.Download}, {Target: small.ID, Sourcing: targets.Download},
	}}))
	must[targets.Profile](t)(goals.MakeDefault(ctx, admin, standard.ID))
	root := rootstest.Root(t, lib, role.NodeID{UUID: storage.NewUUID()}, contenttypetest.ID, t.TempDir())
	fx.Put("w1", contenttypetest.Record{Tree: works.Fetched{Label: "The Attic", Children: []works.Entry{
		{Key: "b1", Label: "Winter", Children: []works.Entry{{Key: "c1", Label: "Scarf"}, {Key: "c2", Label: "Mittens"}}},
	}}})
	tree, _, err := w.Add(ctx, admin, contenttypetest.ID, "w1", works.Adding{
		Monitor: true,
		Also: func(ctx context.Context, tx *storage.Tx[storage.Library], w works.Summary) error {
			if err := targets.Assign(ctx, tx, w, nil); err != nil {
				return err
			}
			return roots.Assign(ctx, tx, w, nil)
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	fx.SetTiming(tree.Items[1].ID, targets.Timing{})
	var placed files.File
	err = lib.Write(ctx, func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
		var err error
		placed, err = files.Add(ctx, tx, files.File{
			WorkID: tree.ID, ItemID: tree.Items[0].ID, Cut: release.Standard, RootID: root.ID,
			Path: "The Attic/The Attic - 1080p.mkv", Size: 4096, Media: media.Record{Video: media.Video{Width: 1920, Height: 1080}}, Quality: release.WEBDL1080p,
		}, []targets.TargetID{hd.ID, small.ID})
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	shelf, err := files.Open(t.Context(), files.Config{Library: lib, Targets: goals, Grabs: files.NoGrabs{}, Today: time.Now})
	if err != nil {
		t.Fatal(err)
	}
	s := ownertest.Serve(t, lib, func(r *api.Router) { filesapi.Routes(r, library{shelf}) })

	code, body := s.Call(t, http.MethodGet, "/api/v1/works/"+tree.ID.String()+"/cells", "")
	file := `"file":{"id":"` + placed.ID.String() + `","path":"The Attic/The Attic - 1080p.mkv","size":4096,"quality":"webdl-1080p","probe":{"width":1920,"height":1080}}`
	scarf, mittens := tree.Items[0].ID.String(), tree.Items[1].ID.String()
	want := `{"cells":[` +
		`{"itemId":"` + scarf + `","edition":"any","targetId":"` + hd.ID.String() + `","targetName":"HD","wanted":true,"state":"upgradable",` + file + `},` +
		`{"itemId":"` + scarf + `","edition":"any","targetId":"` + uhd.ID.String() + `","targetName":"2160p","wanted":true,"state":"missing"},` +
		`{"itemId":"` + scarf + `","edition":"any","targetId":"` + small.ID.String() + `","targetName":"Small","wanted":false,"state":"met",` + file + `},` +
		`{"itemId":"` + mittens + `","edition":"any","targetId":"` + hd.ID.String() + `","targetName":"HD","wanted":false,"state":"unreleased"},` +
		`{"itemId":"` + mittens + `","edition":"any","targetId":"` + uhd.ID.String() + `","targetName":"2160p","wanted":false,"state":"unreleased"},` +
		`{"itemId":"` + mittens + `","edition":"any","targetId":"` + small.ID.String() + `","targetName":"Small","wanted":false,"state":"unreleased"}],` +
		`"items":[{"itemId":"` + scarf + `","status":"missing"},{"itemId":"` + mittens + `","status":"unreleased"}],` +
		`"summary":{"status":"unreleased","cells":{"met":1,"missing":1,"unreleased":3,"upgradable":1},"items":2,"itemStatus":{"missing":1,"unreleased":1}},` +
		`"needsYou":[{"kind":"upgrade","itemIds":["` + scarf + `"],"edition":"any","upgrades":["` + hd.ID.String() + `"]}]}` + "\n"
	if code != http.StatusOK || body != want {
		t.Fatalf("getWorkCells = %d\n%s\nwant\n%s", code, body, want)
	}
	empty := must[targets.Profile](t)(goals.CreateProfile(ctx, admin, targets.ProfileSpec{Name: "Empty", ContentType: contenttypetest.ID}))
	fx.Put("w2", contenttypetest.Record{Tree: works.Fetched{Label: "The Cellar", Children: []works.Entry{{Key: "b1", Label: "Top", Children: []works.Entry{{Key: "c1", Label: "Jar"}}}}}})
	bare, _, err := w.Add(ctx, admin, contenttypetest.ID, "w2", works.Adding{
		Monitor: true,
		Also: func(ctx context.Context, tx *storage.Tx[storage.Library], w works.Summary) error {
			if err := targets.Assign(ctx, tx, w, &empty.ID); err != nil {
				return err
			}
			return roots.Assign(ctx, tx, w, nil)
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	s.Expect(t, http.MethodGet, "/api/v1/works/"+bare.ID.String()+"/cells", "", http.StatusOK,
		`{"cells":[],"items":[{"itemId":"id1","status":null}],"summary":{"status":null,"cells":{},"items":1,"itemStatus":{}},"needsYou":[]}`)
	s.Expect(t, http.MethodGet, "/api/v1/works/"+storage.NewUUID().String()+"/cells", "", http.StatusNotFound,
		`{"status":404,"code":"work_not_found","title":"No Work has that id"}`)
}

type told []files.Changed

func (t *told) Publish(_ context.Context, c files.Changed) { *t = append(*t, c) }

func TestForgetFilesForgetsGoneFilesAllOrNone(t *testing.T) {
	ctx := t.Context()
	lib := storagetest.Library(t, secret.Schema(), identity.Schema(), nodes.Schema(), works.Schema(), targets.Schema(), roots.Schema(), downloadclients.Schema(), grabs.Schema(), indexers.Schema(), files.Schema(), naming.Schema())
	fx := contenttypetest.New()
	w := must[*works.Works](t)(works.Open(ctx, works.Config{
		Library: lib, Catalogs: map[works.TypeID]works.Catalog{contenttypetest.ID: fx}, Source: fx.Source(), Artwork: fx,
		Changed: discard[works.Changed]{}, TypesChanged: discard[works.ContentTypesChanged]{},
	}))
	goals := must[*targets.Targets](t)(targets.Open(ctx, targets.Config{
		Library: lib, Works: w, Types: map[works.TypeID]targets.Facet{contenttypetest.ID: fx}, Changed: discard[targets.Changed]{},
	}))
	hd := must[targets.Target](t)(goals.CreateTarget(ctx, admin, targets.Spec{Scores: targets.Scores{Increment: 1}, Name: "HD", Tiers: [][]release.Quality{{release.WEBDL1080p}}, Cutoff: release.WEBDL1080p}))
	p := must[targets.Profile](t)(goals.CreateProfile(ctx, admin, targets.ProfileSpec{Name: "HD", ContentType: contenttypetest.ID, Entries: []targets.Entry{{Target: hd.ID, Sourcing: targets.Download}}}))
	must[targets.Profile](t)(goals.MakeDefault(ctx, admin, p.ID))
	root := rootstest.Root(t, lib, role.NodeID{UUID: storage.NewUUID()}, contenttypetest.ID, t.TempDir())
	fx.Put("w1", contenttypetest.Record{Tree: works.Fetched{Label: "The Attic", Children: []works.Entry{
		{Key: "b1", Label: "Winter", Children: []works.Entry{{Key: "c1", Label: "Scarf"}, {Key: "c2", Label: "Mittens"}}},
	}}})
	tree, _, err := w.Add(ctx, admin, contenttypetest.ID, "w1", works.Adding{
		Also: func(ctx context.Context, tx *storage.Tx[storage.Library], w works.Summary) error {
			if err := targets.Assign(ctx, tx, w, nil); err != nil {
				return err
			}
			return roots.Assign(ctx, tx, w, nil)
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	var changed told
	svc, err := files.Open(ctx, files.Config{
		Library: lib, Targets: goals, Grabs: files.NoGrabs{}, Naming: naming.New(naming.Config{Facets: map[works.TypeID]naming.Facet{contenttypetest.ID: fx}}),
		Names: map[works.TypeID]release.Names{contenttypetest.ID: fx}, Today: time.Now, Changed: &changed,
	})
	if err != nil {
		t.Fatal(err)
	}
	ids := map[string]string{}
	err = lib.Write(ctx, func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
		for i, name := range []string{"Scarf", "Mittens"} {
			f, err := files.Add(ctx, tx, files.File{
				WorkID: tree.ID, ItemID: tree.Items[i].ID, Cut: release.Standard, RootID: root.ID, Path: "The Attic/" + name + ".mkv", Size: 4096, Quality: release.WEBDL1080p,
			}, []targets.TargetID{hd.ID})
			if err != nil {
				return err
			}
			ids[name] = f.ID.String()
		}
		on, err := files.OnRoot(ctx, tx, root.ID)
		if err != nil {
			return err
		}
		gone := slices.DeleteFunc(on.Files, func(r scanning.Recorded) bool { return r.Path.String() != "The Attic/Scarf.mkv" })
		_, err = svc.Scan(ctx, tx, files.Scan{Work: tree.ID, Root: root.ID, Share: scanning.Share{Gone: gone}})
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	s := ownertest.Serve(t, lib, func(r *api.Router) { filesapi.Routes(r, library{svc}) })
	listed := func() int {
		listing := must[files.Listing](t)(svc.WorkFiles(ctx, admin, tree.ID))
		return len(listing.Files)
	}

	s.Expect(t, http.MethodPost, "/api/v1/files/forget", `{"fileIds":["`+ids["Scarf"]+`","`+ids["Mittens"]+`"]}`, http.StatusConflict,
		`{"status":409,"code":"files_not_gone","title":"Only gone files can be forgotten","detail":"The Attic/Mittens.mkv is on disk."}`)
	s.Expect(t, http.MethodPost, "/api/v1/files/forget", `{"fileIds":["`+ids["Scarf"]+`","`+storage.NewUUID().String()+`"]}`, http.StatusNotFound,
		`{"status":404,"code":"file_not_found","title":"No file has that id"}`)
	if n := listed(); n != 2 || len(changed) != 0 {
		t.Fatalf("after two refused forgets the Work lists %d files and %v changed, want both files and nothing told", n, changed)
	}
	if code, body := s.Call(t, http.MethodPost, "/api/v1/files/forget", `{"fileIds":["`+ids["Scarf"]+`"]}`); code != http.StatusNoContent || body != "" {
		t.Fatalf("forgetting the gone file = %d %s, want 204", code, body)
	}
	if n := listed(); n != 1 || !slices.Equal(changed, told{{WorkID: tree.ID.String()}}) {
		t.Fatalf("after forgetting the gone file the Work lists %d files and %v changed, want the live file alone and its Work told", n, changed)
	}
}
