package importsapi_test

import (
	"context"
	"encoding/json"
	"net/http"
	"path"
	"slices"
	"strconv"
	"strings"
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
	"github.com/jatassi/slipstream/internal/importsapi"
	"github.com/jatassi/slipstream/internal/indexers"
	"github.com/jatassi/slipstream/internal/library"
	"github.com/jatassi/slipstream/internal/media"
	"github.com/jatassi/slipstream/internal/metadata/artwork"
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

type discard[T any] struct{}

func (discard[T]) Publish(context.Context, T) {}

type noGrabs struct{}

func (noGrabs) CancelTransfersIn(context.Context, *storage.Tx[storage.Library], []works.WorkID) error {
	return nil
}

func (noGrabs) TouchTransfers() {}

// shelfFacet places a file at the Item its name, without extension,
// labels.
type shelfFacet struct {
	scanning.None
	release.TitleOnly
}

func (shelfFacet) Hint(name string) scanning.Hint { return scanning.ReadName(name) }

func (shelfFacet) Place(ctx context.Context, r storage.Reader[storage.Library], work works.WorkID) (scanning.Placer, error) {
	items, err := works.ItemsOf(ctx, r, works.Only([]works.WorkID{work}))
	if err != nil {
		return nil, err
	}
	return func(rel string) scanning.Placement {
		label := strings.TrimSuffix(path.Base(rel), path.Ext(rel))
		if i := slices.IndexFunc(items[work], func(it works.Item) bool { return it.Label == label }); i >= 0 {
			return scanning.Placed(items[work][i].ID)
		}
		return scanning.Unplaced(scanning.Reason{Code: scanning.NoSuchItem, Names: label})
	}, nil
}

type node struct {
	im *imports.Imports
	r  *roots.Roots
}

func (n node) Imports() (*imports.Imports, bool) { return n.im, true }
func (n node) Roots() (*roots.Roots, bool)       { return n.r, true }
func (node) Minter() (artwork.Minter, bool)      { return artwork.NewMinter([]byte("importsapi")), true }

// server serves Pending imports over a library of contenttypetest Works on
// one Node, "Media server", with a root to hold their files.
type server struct {
	*ownertest.Owner
	lib     *storage.DB[storage.Library]
	fx      *contenttypetest.Fixture
	library *library.Library
	roots   *roots.Roots
	root    roots.Root
	node    role.NodeID
	nas     role.NodeID
}

func serve(t *testing.T) *server {
	t.Helper()
	ctx := t.Context()
	s := &server{
		lib: storagetest.Library(t, secret.Schema(), identity.Schema(), nodes.Schema(), works.Schema(), targets.Schema(), roots.Schema(), indexers.Schema(),
			downloadclients.Schema(), grabs.Schema(), files.Schema(), naming.Schema(), imports.Schema()),
		fx:   contenttypetest.New(),
		node: role.NodeID{UUID: storage.NewUUID()},
		nas:  role.NodeID{UUID: storage.NewUUID()},
	}
	w, err := works.Open(ctx, works.Config{
		Library: s.lib, Catalogs: map[works.TypeID]works.Catalog{contenttypetest.ID: s.fx}, Source: s.fx.Source(), Artwork: s.fx,
		Changed: discard[works.Changed]{}, TypesChanged: discard[works.ContentTypesChanged]{}, Removed: discard[works.Removed]{},
	})
	if err != nil {
		t.Fatal(err)
	}
	cfg := rootstest.Config(t, s.lib, contenttypetest.ID)
	cfg.Names = func(context.Context) (map[role.NodeID]nodes.Named, error) {
		return map[role.NodeID]nodes.Named{s.node: {Name: "Media server"}, s.nas: {Name: "NAS"}}, nil
	}
	cfg.Describe = w.Describe
	if s.roots, err = roots.Open(ctx, cfg); err != nil {
		t.Fatal(err)
	}
	s.root = s.newRoot(t, s.node)
	goals, err := targets.Open(ctx, targets.Config{Library: s.lib, Works: w, Types: map[works.TypeID]targets.Facet{contenttypetest.ID: s.fx}, Changed: discard[targets.Changed]{}})
	if err != nil {
		t.Fatal(err)
	}
	hd, err := goals.CreateTarget(ctx, identity.Admin{}, targets.Spec{Scores: targets.Scores{Increment: 1}, Name: "HD", Tiers: [][]release.Quality{{release.Bluray1080p}}, Cutoff: release.Bluray1080p})
	if err != nil {
		t.Fatal(err)
	}
	profile, err := goals.CreateProfile(ctx, identity.Admin{}, targets.ProfileSpec{Name: "HD", ContentType: contenttypetest.ID, Entries: []targets.Entry{{Target: hd.ID, Sourcing: targets.Download}}})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := goals.MakeDefault(ctx, identity.Admin{}, profile.ID); err != nil {
		t.Fatal(err)
	}
	fs, err := files.Open(ctx, files.Config{
		Library: s.lib, Targets: goals, Grabs: grabs.GrabCells{InStep: func(role.NodeID) bool { return true }},
		Naming: naming.New(naming.Config{Facets: map[works.TypeID]naming.Facet{contenttypetest.ID: s.fx}}), Names: map[works.TypeID]release.Names{contenttypetest.ID: s.fx}, Today: time.Now,
	})
	if err != nil {
		t.Fatal(err)
	}
	im := imports.New(imports.Config{
		Library: s.lib, Roots: s.roots, Files: fs, Works: w,
		Facets: map[works.TypeID]scanning.Facet{contenttypetest.ID: shelfFacet{}},
		Add: func(ctx context.Context, by identity.Admin, a library.Adding) (library.Added, error) {
			return s.library.Add(ctx, by, a)
		},
		Changed: discard[imports.Changed]{}, FilesChanged: discard[files.Changed]{},
	})
	s.library = library.New(library.Config{Library: s.lib, Works: w, Roots: s.roots, Files: fs, Grabs: noGrabs{}, Leaver: im})
	s.Owner = ownertest.Serve(t, s.lib, func(r *api.Router) { importsapi.Routes(r, node{im, s.roots}) })
	return s
}

func (s *server) newRoot(t *testing.T, on role.NodeID) roots.Root {
	t.Helper()
	v, err := s.roots.Create(t.Context(), identity.Admin{}, roots.NewRoot{ContentType: contenttypetest.ID, Node: on, Path: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	return v.Root
}

// stock puts Work key in the fixture with one box of three clips.
func (s *server) stock(key works.Key, label string) {
	s.fx.Put(key, contenttypetest.Record{Tree: works.Fetched{Label: label, Children: []works.Entry{{Key: "b1", Label: "Box", Children: []works.Entry{
		{Key: "c1", Label: "Clip 1"}, {Key: "c2", Label: "Clip 2"}, {Key: "c3", Label: "Clip 3"},
	}}}}})
}

// add puts Work key in the library, homed in root.
func (s *server) add(t *testing.T, key works.Key, root roots.ID) works.Tree {
	t.Helper()
	s.stock(key, "Shelf "+string(key))
	added, err := s.library.Add(t.Context(), identity.Admin{}, library.Adding{Type: contenttypetest.ID, Key: key, Root: &root})
	if err != nil {
		t.Fatal(err)
	}
	return added.Tree
}

// waiting is a file of a root Import, which ffprobe read unless its reason
// is unreadable.
type waiting struct {
	path   string
	reason scanning.Reason
}

// pend writes a root Import of folder in the server's root with
// suggestion, ignored or not, its files at positions in order, and
// answers its id.
func (s *server) pend(t *testing.T, folder string, suggestion *imports.Suggestion, ignored bool, fs ...waiting) string {
	t.Helper()
	id := storage.NewUUID()
	hint, err := json.Marshal(scanning.ReadName(folder))
	if err != nil {
		t.Fatal(err)
	}
	var suggested *string
	if suggestion != nil {
		b, err := json.Marshal(suggestion)
		if err != nil {
			t.Fatal(err)
		}
		suggested = new(string(b))
	}
	var ignoredAt *storage.Time
	if ignored {
		ignoredAt = new(storage.Now())
	}
	err = s.lib.Write(t.Context(), func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
		if _, err := tx.ExecContext(ctx, `INSERT INTO imports_import (id, source, root_id, folder, content_type, hint, suggestion, created_at, ignored_at)
			VALUES (?, 'root', ?, ?, ?, ?, ?, ?, ?)`, id, s.root.ID, folder, contenttypetest.ID, string(hint), suggested, storage.Now(), ignoredAt); err != nil {
			return err
		}
		for i, f := range fs {
			var probe *media.Record
			var quality *release.Quality
			if f.reason.Code != scanning.Unreadable {
				probe = &media.Record{Version: media.Version, Container: "matroska,webm", Video: media.Video{Codec: "hevc", Width: 1920, Height: 1080}}
				quality = new(release.Bluray1080p)
			}
			if _, err := tx.ExecContext(ctx, `INSERT INTO imports_file (import_id, position, path, size, modified_at, probe, quality, cut, release_name, source_name, outcome, reason)
				VALUES (?, ?, ?, 1073741824, 1700000000000000, ?, ?, 'standard', ?, ?, 'waiting', ?)`, id, i, f.path, probe, quality, path.Base(f.path), path.Base(f.path), reasonOf(t, f.reason)); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return id.String()
}

func reasonOf(t *testing.T, r scanning.Reason) string {
	t.Helper()
	b, err := json.Marshal(imports.Reason{Code: imports.ReasonCode(r.Code), Scan: &r})
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func noMatch(p string) waiting { return waiting{p, scanning.Reason{Code: scanning.NoMatch}} }

func (s *server) entries(t *testing.T, query string) string {
	t.Helper()
	code, reply := s.Call(t, http.MethodGet, "/api/v1/imports"+query, "")
	if code != http.StatusOK {
		t.Fatalf("listImports%s = %d %s", query, code, reply)
	}
	return ownertest.Masked(reply)
}

func TestPendingImportsLeadWithEachScannedRootThenListTheEntries(t *testing.T) {
	s := serve(t)
	s.pend(t, "Heat (1995)", &imports.Suggestion{Key: "w1", Label: "Heat", Year: 1995, Confident: true}, false,
		noMatch("Heat (1995)/Heat.mkv"),
		waiting{"Heat (1995)/Heat.sample.mkv", scanning.Reason{Code: scanning.Unreadable, Line: "Invalid data found when processing input"}})
	other := s.newRoot(t, s.nas)
	s.pend(t, "Halloween", nil, false, waiting{"Halloween/Halloween.mkv", scanning.Reason{Code: scanning.Elsewhere, Title: "Halloween", Year: 1978, Root: other.ID.String()}})
	s.pend(t, "Brazil (1985)", nil, true, noMatch("Brazil (1985)/Brazil.mkv"))
	err := s.lib.Write(t.Context(), func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
		_, err := tx.ExecContext(ctx, `INSERT INTO imports_root (root_id, place, compared_at, found, added) VALUES (?, ?, 1700000000000000, 401, 398)`, s.root.ID, s.root.Place)
		return err
	})
	if err != nil {
		t.Fatal(err)
	}

	want := `{"leads":[{"root":{"id":"id1","label":"Shelves library root on Media server"},"contentType":"fixture","added":398,"needsYou":2}],"imports":[` +
		`{"id":"id2","source":"root","stage":"waiting","root":{"id":"id1","label":"Shelves library root on Media server"},"folder":"Halloween","contentType":"fixture","hint":{"title":"Halloween","year":null,"ids":[]},"suggestion":null,"createdAt":"T","ignoredAt":null,"finishedAt":null,"files":[` +
		`{"position":0,"path":"Halloween/Halloween.mkv","size":1073741824,"quality":"bluray-1080p","cut":"standard","itemIds":[],"outcome":"waiting","reason":{"code":"elsewhere","title":"Halloween","year":1978,"root":{"id":"id3","label":"Shelves library root on NAS"}}}]},` +
		`{"id":"id4","source":"root","stage":"waiting","root":{"id":"id1","label":"Shelves library root on Media server"},"folder":"Heat (1995)","contentType":"fixture","hint":{"title":"Heat","year":1995,"ids":[]},` +
		`"suggestion":{"key":"w1","label":"Heat","year":1995,"posterUrl":null,"workId":null,"confident":true},"createdAt":"T","ignoredAt":null,"finishedAt":null,"files":[` +
		`{"position":0,"path":"Heat (1995)/Heat.mkv","size":1073741824,"quality":"bluray-1080p","cut":"standard","itemIds":[],"outcome":"waiting","reason":{"code":"no_match"}},` +
		`{"position":1,"path":"Heat (1995)/Heat.sample.mkv","size":1073741824,"quality":null,"cut":"standard","itemIds":[],"outcome":"waiting","reason":{"code":"unreadable","line":"Invalid data found when processing input"}}]}]}` + "\n"
	if got := s.entries(t, ""); got != want {
		t.Errorf("listImports = %s\nwant %s", got, want)
	}
	ignored := `{"leads":[{"root":{"id":"id1","label":"Shelves library root on Media server"},"contentType":"fixture","added":398,"needsYou":2}],"imports":[` +
		`{"id":"id2","source":"root","stage":"waiting","root":{"id":"id1","label":"Shelves library root on Media server"},"folder":"Brazil (1985)","contentType":"fixture","hint":{"title":"Brazil","year":1985,"ids":[]},"suggestion":null,"createdAt":"T","ignoredAt":"T","finishedAt":null,"files":[` +
		`{"position":0,"path":"Brazil (1985)/Brazil.mkv","size":1073741824,"quality":"bluray-1080p","cut":"standard","itemIds":[],"outcome":"waiting","reason":{"code":"no_match"}}]}]}` + "\n"
	if got := s.entries(t, "?ignored=true"); got != ignored {
		t.Errorf("listImports?ignored=true = %s\nwant %s", got, ignored)
	}
	if got, want := s.entries(t, "?rootId="+other.ID.String()), `{"leads":[],"imports":[]}`+"\n"; got != want {
		t.Errorf("listImports of a root with none = %s, want %s", got, want)
	}
}

func TestAddAddsEachSuggestedWorkAndRefusesTheRestOnTheirOwn(t *testing.T) {
	s := serve(t)
	s.stock("w1", "Shelf w1")
	heat := s.pend(t, "Shelf w1", &imports.Suggestion{Key: "w1", Label: "Shelf w1"}, false, noMatch("Shelf w1/Clip 1.mkv"), noMatch("Shelf w1/Clip 2.mkv"))
	none := s.pend(t, "Halloween", nil, false, noMatch("Halloween/Halloween.mkv"))
	unknown := storage.NewUUID().String()

	code, reply := s.Call(t, http.MethodPost, "/api/v1/imports/add", `{"importIds":["`+heat+`","`+none+`","`+unknown+`"]}`)
	var out struct {
		Outcomes []struct {
			ID      string `json:"id"`
			Outcome string `json:"outcome"`
			WorkID  string `json:"workId"`
			Problem struct {
				Code string `json:"code"`
			} `json:"problem"`
		} `json:"outcomes"`
	}
	if err := json.Unmarshal([]byte(reply), &out); code != http.StatusOK || err != nil || len(out.Outcomes) != 3 {
		t.Fatalf("addRootImports = %d %s", code, reply)
	}
	if o := out.Outcomes[0]; o.ID != heat || o.Outcome != "added" || o.WorkID == "" {
		t.Errorf("the entry with a suggestion = %+v, want its Work added", o)
	}
	if o := out.Outcomes[1]; o.ID != none || o.Outcome != "failed" || o.Problem.Code != "import_no_suggestion" {
		t.Errorf("the entry with no suggestion = %+v, want import_no_suggestion", o)
	}
	if o := out.Outcomes[2]; o.ID != unknown || o.Outcome != "failed" || o.Problem.Code != "import_not_found" {
		t.Errorf("an id no entry has = %+v, want import_not_found", o)
	}
	if got := s.entries(t, ""); strings.Contains(got, "Shelf w1") || !strings.Contains(got, "Halloween") {
		t.Errorf("after Add Pending imports lists %s, want only the entry it refused", got)
	}
}

func TestMatchMakesARecordTheSuggestion(t *testing.T) {
	s := serve(t)
	s.stock("w2", "Heat")
	id := s.pend(t, "Heat (1995)", nil, false, noMatch("Heat (1995)/Heat.mkv"))
	s.Expect(t, http.MethodPost, "/api/v1/imports/"+id, `{"action":"match","key":"w2"}`, http.StatusOK,
		`{"id":"id1","source":"root","stage":"waiting","root":{"id":"id2","label":"Shelves library root on Media server"},"folder":"Heat (1995)","contentType":"fixture","hint":{"title":"Heat","year":1995,"ids":[]},`+
			`"suggestion":{"key":"w2","label":"Heat","year":null,"posterUrl":null,"workId":null,"confident":false},"createdAt":"T","ignoredAt":null,"finishedAt":null,"files":[`+
			`{"position":0,"path":"Heat (1995)/Heat.mkv","size":1073741824,"quality":"bluray-1080p","cut":"standard","itemIds":[],"outcome":"waiting","reason":{"code":"no_match"}}]}`)
	s.Expect(t, http.MethodPost, "/api/v1/imports/"+id, `{"action":"match","key":"nowhere"}`, http.StatusNotFound,
		`{"status":404,"code":"metadata_not_found","title":"The metadata source has no record with that key"}`)
	s.Expect(t, http.MethodPost, "/api/v1/imports/"+storage.NewUUID().String(), `{"action":"match","key":"w2"}`, http.StatusNotFound,
		`{"status":404,"code":"import_not_found","title":"No Import has that id"}`)
}

func TestGetImportAnswersOneImportWithItsFiles(t *testing.T) {
	s := serve(t)
	id := s.pend(t, "Heat (1995)", nil, false, noMatch("Heat (1995)/Heat.mkv"))
	s.Expect(t, http.MethodGet, "/api/v1/imports/"+id, "", http.StatusOK,
		`{"id":"id1","source":"root","stage":"waiting","root":{"id":"id2","label":"Shelves library root on Media server"},"folder":"Heat (1995)","contentType":"fixture","hint":{"title":"Heat","year":1995,"ids":[]},`+
			`"suggestion":null,"createdAt":"T","ignoredAt":null,"finishedAt":null,"files":[`+
			`{"position":0,"path":"Heat (1995)/Heat.mkv","size":1073741824,"quality":"bluray-1080p","cut":"standard","itemIds":[],"outcome":"waiting","reason":{"code":"no_match"}}]}`)
	s.Expect(t, http.MethodGet, "/api/v1/imports/"+storage.NewUUID().String(), "", http.StatusNotFound,
		`{"status":404,"code":"import_not_found","title":"No Import has that id"}`)
}

func TestImportIntoAttachesTheChosenFilesToAWorkInTheImportsRoot(t *testing.T) {
	s := serve(t)
	work := s.add(t, "w1", s.root.ID)
	id := s.pend(t, "Shelf", nil, false, noMatch("Shelf/a.mkv"), noMatch("Shelf/b.mkv"))
	into := func(position int, w works.WorkID, items ...works.ItemID) string {
		ids := make([]string, len(items))
		for i, it := range items {
			ids[i] = `"` + it.String() + `"`
		}
		return `{"action":"into","workId":"` + w.String() + `","files":[{"position":` + strconv.Itoa(position) + `,"itemIds":[` + strings.Join(ids, ",") + `]}]}`
	}
	url := "/api/v1/imports/" + id

	s.Expect(t, http.MethodPost, url, into(2, work.ID, work.Items[0].ID), http.StatusConflict,
		`{"status":409,"code":"import_changed","title":"The Import no longer has a chosen file waiting"}`)
	s.Expect(t, http.MethodPost, url, into(0, work.ID, works.ItemID{UUID: storage.NewUUID()}), http.StatusUnprocessableEntity,
		`{"status":422,"code":"import_items","title":"A chosen Item isn't in the Work"}`)
	away := s.add(t, "w2", s.newRoot(t, s.nas).ID)
	s.Expect(t, http.MethodPost, url, into(0, away.ID, away.Items[0].ID), http.StatusConflict,
		`{"status":409,"code":"import_elsewhere","title":"The Work lives in another library root"}`)

	s.Expect(t, http.MethodPost, url, into(0, work.ID, work.Items[0].ID), http.StatusOK,
		`{"id":"id1","source":"root","stage":"waiting","root":{"id":"id2","label":"Shelves library root on Media server"},"folder":"Shelf","contentType":"fixture","hint":{"title":"Shelf","year":null,"ids":[]},`+
			`"suggestion":{"key":"w1","label":"Shelf w1","year":null,"posterUrl":null,"workId":"id3","confident":false},"createdAt":"T","ignoredAt":null,"finishedAt":null,"files":[`+
			`{"position":0,"path":"Shelf/a.mkv","size":1073741824,"quality":"bluray-1080p","cut":"standard","itemIds":["id4"],"outcome":"placed","reason":null},`+
			`{"position":1,"path":"Shelf/b.mkv","size":1073741824,"quality":"bluray-1080p","cut":"standard","itemIds":[],"outcome":"waiting","reason":{"code":"no_match"}}]}`)
	s.Expect(t, http.MethodPost, url, into(1, work.ID, work.Items[2].ID, work.Items[1].ID), http.StatusOK,
		`{"id":"id1","source":"root","stage":"done","root":{"id":"id2","label":"Shelves library root on Media server"},"folder":"Shelf","contentType":"fixture","hint":{"title":"Shelf","year":null,"ids":[]},`+
			`"suggestion":{"key":"w1","label":"Shelf w1","year":null,"posterUrl":null,"workId":"id3","confident":false},"createdAt":"T","ignoredAt":null,"finishedAt":"T","files":[`+
			`{"position":0,"path":"Shelf/a.mkv","size":1073741824,"quality":"bluray-1080p","cut":"standard","itemIds":["id4"],"outcome":"placed","reason":null},`+
			`{"position":1,"path":"Shelf/b.mkv","size":1073741824,"quality":"bluray-1080p","cut":"standard","itemIds":["id5","id6"],"outcome":"placed","reason":null}]}`)
	s.Expect(t, http.MethodPost, url, into(1, work.ID, work.Items[1].ID), http.StatusConflict,
		`{"status":409,"code":"import_changed","title":"The Import no longer has a chosen file waiting"}`)
	if got := s.entries(t, ""); got != `{"leads":[],"imports":[]}`+"\n" {
		t.Errorf("after Into placed every file Pending imports lists %s, want nothing", got)
	}
	var recorded []string
	err := s.lib.Read(t.Context(), func(ctx context.Context, r storage.Reader[storage.Library]) error {
		on, err := files.OnRoot(ctx, r, s.root.ID)
		for _, f := range on.Files {
			recorded = append(recorded, f.Path.String())
		}
		return err
	})
	if err != nil || !slices.Equal(recorded, []string{"Shelf/a.mkv", "Shelf/b.mkv"}) {
		t.Errorf("the root records %v (%v), want both chosen files", recorded, err)
	}
}

func (s *server) noContent(t *testing.T, url, body string) {
	t.Helper()
	if code, reply := s.Call(t, http.MethodPost, url, body); code != http.StatusNoContent || reply != "" {
		t.Fatalf("POST %s = %d %s, want 204", url, code, reply)
	}
}

func TestIgnoreMovesEntriesToIgnoredUntilUnignored(t *testing.T) {
	s := serve(t)
	id := s.pend(t, "Halloween", nil, false, noMatch("Halloween/Halloween.mkv"))
	body := `{"importIds":["` + id + `"]}`
	s.noContent(t, "/api/v1/imports/ignore", body)
	if got := s.entries(t, ""); got != `{"leads":[],"imports":[]}`+"\n" {
		t.Errorf("after Ignore the entries that need the Admin are %s, want none", got)
	}
	if got := s.entries(t, "?ignored=true"); !strings.Contains(got, `"folder":"Halloween"`) {
		t.Errorf("after Ignore the ignored entries are %s, want Halloween", got)
	}
	s.noContent(t, "/api/v1/imports/unignore", body)
	if got := s.entries(t, ""); !strings.Contains(got, `"folder":"Halloween","contentType":"fixture","hint":{"title":"Halloween","year":null,"ids":[]},"suggestion":null,"createdAt":"T","ignoredAt":null`) {
		t.Errorf("after Unignore the entries that need the Admin are %s, want Halloween", got)
	}
}

func TestLibraryScanSettingsStoreTheStopShareAndProbes(t *testing.T) {
	s := serve(t)
	code, reply := s.Call(t, http.MethodGet, "/api/v1/imports/scan-settings", "")
	if code != http.StatusOK || !strings.Contains(reply, `"key":"library.scan.stopShare"`) || !strings.Contains(reply, `"key":"library.scan.probes"`) {
		t.Fatalf("getLibraryScanSettings = %d %s, want the stop share and probes", code, reply)
	}
	code, reply = s.Call(t, http.MethodPatch, "/api/v1/imports/scan-settings", `{"values":{"library.scan.stopShare":40,"library.scan.probes":4}}`)
	if code != http.StatusOK || !strings.Contains(reply, `"value":40`) || !strings.Contains(reply, `"value":4,`) {
		t.Fatalf("updateLibraryScanSettings = %d %s, want 40 and 4 stored", code, reply)
	}
	code, reply = s.Call(t, http.MethodPatch, "/api/v1/imports/scan-settings", `{"values":{"library.scan.probes":9}}`)
	if code != http.StatusUnprocessableEntity || !strings.Contains(reply, `"code":"setting_out_of_range"`) {
		t.Errorf("9 probes = %d %s, want setting_out_of_range", code, reply)
	}
	s.Expect(t, http.MethodPatch, "/api/v1/imports/scan-settings", `{"values":{"library.scan.depth":1}}`, http.StatusUnprocessableEntity,
		`{"status":422,"code":"unknown_setting","title":"No setting has this key","detail":"No setting has the key library.scan.depth."}`)
}
