package namingapi_test

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
	"github.com/jatassi/slipstream/internal/grabs"
	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/imports"
	"github.com/jatassi/slipstream/internal/indexers"
	"github.com/jatassi/slipstream/internal/media"
	"github.com/jatassi/slipstream/internal/naming"
	"github.com/jatassi/slipstream/internal/namingapi"
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

type ignore[T any] struct{}

func (ignore[T]) Publish(context.Context, T) {}

func must[T any](t *testing.T) func(T, error) T {
	return func(v T, err error) T {
		t.Helper()
		if err != nil {
			t.Fatal(err)
		}
		return v
	}
}

// attic serves the naming of a library holding one Fixture Work, The
// Attic, whose one file fills HD at Loft/scarf.mkv, and answers the
// Work's id.
func attic(t *testing.T) (o *ownertest.Owner, work string) {
	t.Helper()
	ctx := t.Context()
	admin := identity.Admin{}
	lib := storagetest.Library(t, secret.Schema(), identity.Schema(), nodes.Schema(), works.Schema(), targets.Schema(), roots.Schema(), downloadclients.Schema(), grabs.Schema(), indexers.Schema(), files.Schema(), naming.Schema(), imports.Schema())
	fx := contenttypetest.New()
	w := must[*works.Works](t)(works.Open(ctx, works.Config{
		Library: lib, Catalogs: map[works.TypeID]works.Catalog{contenttypetest.ID: fx}, Source: fx.Source(), Artwork: fx,
		Changed: ignore[works.Changed]{}, TypesChanged: ignore[works.ContentTypesChanged]{},
	}))
	goals := must[*targets.Targets](t)(targets.Open(ctx, targets.Config{
		Library: lib, Works: w, Types: map[works.TypeID]targets.Facet{contenttypetest.ID: fx}, Changed: ignore[targets.Changed]{},
	}))
	hd := must[targets.Target](t)(goals.CreateTarget(ctx, admin, targets.Spec{Scores: targets.Scores{Increment: 1}, Name: "HD", Tiers: [][]release.Quality{{release.WEBDL1080p}}, Cutoff: release.WEBDL1080p}))
	p := must[targets.Profile](t)(goals.CreateProfile(ctx, admin, targets.ProfileSpec{Name: "HD", ContentType: contenttypetest.ID, Entries: []targets.Entry{{Target: hd.ID, Sourcing: targets.Download}}}))
	must[targets.Profile](t)(goals.MakeDefault(ctx, admin, p.ID))
	root := rootstest.Root(t, lib, role.NodeID{UUID: storage.NewUUID()}, contenttypetest.ID, t.TempDir())
	fx.Put("w1", contenttypetest.Record{Tree: works.Fetched{Label: "The Attic", Children: []works.Entry{
		{Key: "b1", Label: "Winter", Children: []works.Entry{{Key: "c1", Label: "Scarf"}}},
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
	err = lib.Write(ctx, func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
		_, err := files.Add(ctx, tx, files.File{
			WorkID: tree.ID, ItemID: tree.Items[0].ID, Cut: release.Standard, RootID: root.ID, Path: "Loft/scarf.mkv", Size: 5,
			Media: media.Record{Video: media.Video{Width: 1920, Height: 1080}}, Quality: release.WEBDL1080p,
		}, []targets.TargetID{hd.ID})
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	nm := naming.New(naming.Config{
		Library: lib, Facets: map[works.TypeID]naming.Facet{contenttypetest.ID: fx}, Describe: w.Describe, Budget: roots.SmallestBudget, Changed: discard{},
	})
	svc := must[*files.Files](t)(files.Open(ctx, files.Config{
		Library: lib, Targets: goals, Grabs: files.NoGrabs{}, Naming: nm,
		Names: map[works.TypeID]release.Names{contenttypetest.ID: fx}, Today: time.Now, Moved: ignore[files.Moved]{},
	}))
	im := imports.New(imports.Config{Library: lib, Files: svc, Targets: goals, Facets: map[works.TypeID]scanning.Facet{contenttypetest.ID: fx}})
	return ownertest.Serve(t, lib, func(r *api.Router) { namingapi.Routes(r, node{n: nm, f: svc, im: im}) }), tree.ID.String()
}

type fieldProblems struct {
	Key      string
	Problems []string
}

type previewed struct {
	Fields []fieldProblems
	Work   []struct{ From, To, Detail string }
	Tested []struct {
		Name, Path, Detail string
		Unplaced           *struct{ Code, Names string }
	}
}

func TestAPreviewNamesAWorksFilesAndTestedNamesUnderTheDraft(t *testing.T) {
	o, work := attic(t)
	body := func(template string) string {
		return `{"draft":{"fixture.naming.file.standard":` + template + `},"workId":"` + work + `","names":[{"name":"Scarf.1080p.WEB-DL.mkv","workId":"` + work + `"}]}`
	}

	got := call[previewed](t, o, http.MethodPost, "/api/v1/naming/fixture/preview", body(`"{Shelf} - {Thing} - {Edition Tag} [{Target}]"`), http.StatusOK)
	if len(got.Work) != 1 || got.Work[0].From != "Loft/scarf.mkv" || got.Work[0].To != "The Attic/The Attic - Scarf - [HD].mkv" {
		t.Fatalf("work = %+v, want the Work's file under the draft", got.Work)
	}
	if len(got.Tested) != 1 || got.Tested[0].Path != "" || got.Tested[0].Unplaced == nil || got.Tested[0].Unplaced.Code != "no_rule" {
		t.Fatalf("tested = %+v, want the name unplaced, since the Fixture places no file", got.Tested)
	}

	refused := call[previewed](t, o, http.MethodPost, "/api/v1/naming/fixture/preview", body(`"{Shelf} - {Thing}"`), http.StatusOK)
	if len(refused.Work) != 0 || len(refused.Tested) != 0 || !slices.ContainsFunc(refused.Fields, func(f fieldProblems) bool {
		return f.Key == fileKey && slices.Contains(f.Problems, noTarget)
	}) {
		t.Fatalf("preview of a template without {Target} = %+v, want its problem and no paths", refused)
	}

	gone := storage.NewUUID().String()
	o.Expect(t, http.MethodPost, "/api/v1/naming/fixture/preview", `{"draft":{},"names":[{"name":"Scarf.mkv","workId":"`+gone+`"}]}`, http.StatusNotFound,
		`{"status":404,"code":"work_not_found","title":"No Work has that id"}`)
}
