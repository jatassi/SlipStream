package filesapi_test

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"strings"
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
	"github.com/jatassi/slipstream/internal/secret"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/storage/storagetest"
	"github.com/jatassi/slipstream/internal/targets"
	"github.com/jatassi/slipstream/internal/works"
)

func TestAPlanConfirmedOverHTTPQueuesItsRenamesOnce(t *testing.T) {
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
	node := role.NodeID{UUID: storage.NewUUID()}
	root := rootstest.Root(t, lib, node, contenttypetest.ID, t.TempDir())
	held := rootstest.Held(t, lib, node, nil, contenttypetest.ID)
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
			WorkID: tree.ID, Items: []works.ItemID{tree.Items[0].ID}, Cut: release.Standard, RootID: root.ID, Path: "Loft/scarf.mkv", Size: 5,
			Media: media.Record{Video: media.Video{Width: 1920, Height: 1080}}, Quality: release.WEBDL1080p,
		}, []targets.TargetID{hd.ID})
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root.Path, "Loft"), 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root.Path, "Loft", "scarf.mkv"), []byte("scarf"), 0o600); err != nil {
		t.Fatal(err)
	}
	svc := must[*files.Files](t)(files.Open(ctx, files.Config{
		Library: lib, Targets: goals, Grabs: files.NoGrabs{},
		Naming: naming.New(naming.Config{
			Library: lib, Facets: map[works.TypeID]naming.Facet{contenttypetest.ID: fx}, Describe: w.Describe, Budget: roots.SmallestBudget,
		}),
		Names: map[works.TypeID]release.Names{contenttypetest.ID: fx}, Today: time.Now, Disk: held, Moved: discard[files.Moved]{},
	}))
	s := ownertest.Serve(t, lib, func(r *api.Router) { filesapi.Routes(r, library{svc}) })
	work := tree.ID.String()

	s.Expect(t, http.MethodGet, "/api/v1/renames/unmatched", "", http.StatusOK, `{"contentTypes":[{"contentType":"fixture","files":1}]}`)
	code, body := s.Call(t, http.MethodPost, "/api/v1/renames/plan", `{"scope":{"kind":"library"}}`)
	var plan struct {
		Files int `json:"files"`
		Works []struct {
			WorkID string `json:"workId"`
			Digest string `json:"digest"`
			Rename int    `json:"rename"`
		} `json:"works"`
	}
	if err := json.Unmarshal([]byte(body), &plan); code != http.StatusOK || err != nil || plan.Files != 1 || len(plan.Works) != 1 || plan.Works[0].WorkID != work {
		t.Fatalf("planRenames = %d %s, want the Work's one file", code, body)
	}
	digest := plan.Works[0].Digest
	s.Expect(t, http.MethodPost, "/api/v1/renames/plan/"+work, "", http.StatusOK,
		`{"workId":"id1","contentType":"fixture","title":"The Attic","digest":"`+digest+`","moving":0,"rows":[`+
			`{"fileId":"id2","from":"Loft/scarf.mkv","to":"The Attic/The Attic - Scarf - HD.mkv","group":0,"companions":[]}]}`)
	refused := `{"scope":{"kind":"work","id":"` + work + `"},"draft":{"contentType":"fixture","values":{"fixture.naming.file.standard":"{Shelf} - {Thing}"}}}`
	if code, body := s.Call(t, http.MethodPost, "/api/v1/renames/plan", refused); code != http.StatusUnprocessableEntity ||
		!strings.Contains(body, `"code":"naming_template_refused"`) || !strings.Contains(body, `"location":"body.draft.values.fixture.naming.file.standard"`) {
		t.Fatalf("planRenames with a template without {Target} = %d %s, want the save check's refusal on that field", code, body)
	}
	s.Expect(t, http.MethodPost, "/api/v1/renames/plan", `{"scope":{"kind":"type","id":"other"}}`, http.StatusNotFound,
		`{"status":404,"code":"content_type_not_found","title":"No Content Type has that id"}`)

	confirm := `{"works":[{"work":"` + work + `","digest":"` + digest + `"}]}`
	s.Expect(t, http.MethodPost, "/api/v1/renames", confirm, http.StatusOK, `{"queued":1,"changed":[]}`)
	s.Expect(t, http.MethodGet, "/api/v1/renames/moving?workId="+work, "", http.StatusOK, `{"files":1}`)
	s.Expect(t, http.MethodGet, "/api/v1/renames/unmatched", "", http.StatusOK, `{"contentTypes":[]}`)
	s.Expect(t, http.MethodPost, "/api/v1/renames", confirm, http.StatusOK, `{"queued":0,"changed":["id1"]}`)
}
