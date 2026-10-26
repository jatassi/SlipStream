package worksapi_test

import (
	"context"
	"encoding/json"
	"fmt"
	"io/fs"
	"log/slog"
	"maps"
	"math/rand/v2"
	"net/http"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/jatassi/slipstream/internal/api"
	"github.com/jatassi/slipstream/internal/api/ownertest"
	"github.com/jatassi/slipstream/internal/bencode"
	"github.com/jatassi/slipstream/internal/contenttype/contenttypetest"
	"github.com/jatassi/slipstream/internal/downloadclients"
	"github.com/jatassi/slipstream/internal/files"
	"github.com/jatassi/slipstream/internal/grabs"
	"github.com/jatassi/slipstream/internal/health"
	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/indexers"
	"github.com/jatassi/slipstream/internal/library"
	"github.com/jatassi/slipstream/internal/metadata/artwork"
	"github.com/jatassi/slipstream/internal/naming"
	"github.com/jatassi/slipstream/internal/nodes"
	"github.com/jatassi/slipstream/internal/release"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/roots"
	"github.com/jatassi/slipstream/internal/roots/rootstest"
	"github.com/jatassi/slipstream/internal/secret"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/storage/storagetest"
	"github.com/jatassi/slipstream/internal/storing"
	"github.com/jatassi/slipstream/internal/targets"
	"github.com/jatassi/slipstream/internal/works"
	"github.com/jatassi/slipstream/internal/worksapi"
)

const other works.TypeID = "other"

var (
	home    = role.NodeID{UUID: storage.NewUUID()}
	seedbox = role.NodeID{UUID: storage.NewUUID()}
)

// roster holds Storage on every Node, each in step until it goes
// offline.
type roster struct {
	mu      sync.Mutex
	offline map[role.NodeID]bool
}

func (*roster) Holds(role.Name, role.NodeID) bool { return true }

func (r *roster) InStep(n role.NodeID) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	return !r.offline[n]
}

func (r *roster) drop(n role.NodeID) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.offline[n] = true
}

// events is every event an action published, one line per batch, each
// id written as the name the test gave it.
type events struct {
	mu    sync.Mutex
	names map[string]string
	lines []string
}

func (e *events) name(id, as string) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.names[id] = as
}

func (e *events) add(kind string, ids []string) {
	e.mu.Lock()
	defer e.mu.Unlock()
	named := make([]string, len(ids))
	for i, id := range ids {
		named[i] = e.names[id]
	}
	slices.Sort(named)
	e.lines = append(e.lines, kind+" "+strings.Join(named, " "))
}

// take answers the batches since the last take.
func (e *events) take() string {
	e.mu.Lock()
	defer e.mu.Unlock()
	out := strings.Join(e.lines, "\n")
	e.lines = nil
	return out
}

type publisher[T any] struct {
	e    *events
	kind string
	ids  func(T) []string
}

func (p publisher[T]) Publish(_ context.Context, v T) { p.e.add(p.kind, p.ids(v)) }

type node struct {
	w    *works.Works
	t    *targets.Targets
	lib  *library.Library
	mint artwork.Minter
}

func (n node) Works() (*works.Works, bool)       { return n.w, true }
func (n node) Targets() (*targets.Targets, bool) { return n.t, true }
func (n node) Minter() (artwork.Minter, bool)    { return n.mint, true }
func (n node) Library() (*library.Library, bool) { return n.lib, true }

// shelves is a Controller whose library holds the fixture type and other.
// home holds the fixture roots movies, the default, and films, and the
// root others; seedbox holds the fixture root boxed. home's disk queue
// runs whenever a test settles it.
type shelves struct {
	*ownertest.Owner
	lib           *storage.DB[storage.Library]
	fx            *contenttypetest.Fixture
	goals         *targets.Targets
	queue         *files.Files
	roster        *roster
	events        *events
	movies, films roots.Root
	boxed, others roots.Root
	web           targets.Target
	byKey         map[string]string
}

func shelve(t *testing.T) *shelves {
	t.Helper()
	lib := storagetest.Library(t, secret.Schema(), identity.Schema(), nodes.Schema(), works.Schema(), targets.Schema(), health.Schema(), roots.Schema(),
		indexers.Schema(), downloadclients.Schema(), grabs.Schema(), files.Schema())
	s := &shelves{lib: lib, fx: contenttypetest.New(), roster: &roster{offline: map[role.NodeID]bool{}}, events: &events{names: map[string]string{}}, byKey: map[string]string{}}
	s.exec(t, `INSERT INTO nodes_node (id, public_key, name, paired_at) VALUES (?, randomblob(32), 'home', 0), (?, randomblob(32), 'seedbox', 0)`, home, seedbox)
	byWorks := func(v works.Changed) []string { return v.WorkIDs }
	w, err := works.Open(t.Context(), works.Config{
		Library:      lib,
		Catalogs:     map[works.TypeID]works.Catalog{contenttypetest.ID: s.fx, other: s.fx},
		Source:       s.fx.SourceAs(contenttypetest.ID, other),
		Artwork:      s.fx,
		Timings:      works.Timings{Fetch: 5 * time.Second},
		Changed:      publisher[works.Changed]{s.events, "works.changed", byWorks},
		Removed:      publisher[works.Removed]{s.events, "works.removed", func(v works.Removed) []string { return v.WorkIDs }},
		TypesChanged: discard[works.ContentTypesChanged]{},
		TypeChanged:  discard[works.TypeChanged]{},
		Log:          slog.New(slog.DiscardHandler),
	})
	if err != nil {
		t.Fatal(err)
	}
	s.goals, err = targets.Open(t.Context(), targets.Config{
		Library: lib, Works: w, Types: map[works.TypeID]targets.Facet{contenttypetest.ID: s.fx, other: s.fx},
		Changed: publisher[targets.Changed]{s.events, "targets.changed", func(v targets.Changed) []string { return v.WorkIDs }},
	})
	if err != nil {
		t.Fatal(err)
	}
	s.movies = s.root(t, "movies", home, contenttypetest.ID)
	s.films = s.root(t, "films", home, contenttypetest.ID)
	s.others = s.root(t, "others", home, other)
	s.boxed = s.root(t, "boxed", seedbox, contenttypetest.ID)
	cfg := rootstest.Config(t, lib, contenttypetest.ID, other)
	cfg.Roster = s.roster
	cfg.Changed = publisher[roots.Changed]{s.events, "roots.changed", func(v roots.Changed) []string { return v.RootIDs }}
	homes, err := roots.Open(t.Context(), cfg)
	if err != nil {
		t.Fatal(err)
	}
	s.queue = files.New(files.Config{
		Library: lib, Targets: s.goals, Grabs: grabs.GrabCells{},
		Naming:  map[works.TypeID]naming.Facet{contenttypetest.ID: s.fx, other: s.fx},
		Today:   func() time.Time { return time.Date(2030, time.January, 1, 9, 0, 0, 0, time.UTC) },
		Disk:    rootstest.Held(t, lib, home, nil, contenttypetest.ID, other),
		Timings: files.Timings{RetryMin: time.Millisecond, RetryMax: 4 * time.Millisecond},
		Rand:    rand.New(rand.NewPCG(1, 2)), //nolint:gosec // seeded on purpose so tests replay
		Log:     slog.New(slog.DiscardHandler),
	})
	shelf := node{w: w, t: s.goals, mint: artwork.NewMinter([]byte("actions")), lib: library.New(library.Config{
		Library: lib, Works: w, Targets: s.goals, Roots: homes, Files: s.queue,
	})}
	s.Owner = ownertest.Serve(t, lib, func(r *api.Router) { worksapi.Routes(r, shelf) })
	s.web, err = s.goals.CreateTarget(t.Context(), identity.Admin{}, targets.Spec{Name: "Web", Tiers: [][]release.Quality{{release.WEBDL1080p}}, Cutoff: release.WEBDL1080p})
	if err != nil {
		t.Fatal(err)
	}
	s.events.take()
	return s
}

func (s *shelves) root(t *testing.T, name string, on role.NodeID, typ works.TypeID) roots.Root {
	t.Helper()
	r := rootstest.Root(t, s.lib, on, typ, t.TempDir())
	s.events.name(r.ID.String(), name)
	return r
}

func (s *shelves) exec(t *testing.T, query string, args ...any) {
	t.Helper()
	err := s.lib.Write(t.Context(), func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
		_, err := tx.ExecContext(ctx, query, args...)
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
}

func (s *shelves) count(t *testing.T, query string, args ...any) int {
	t.Helper()
	var n int
	if err := s.lib.QueryRowContext(t.Context(), query, args...).Scan(&n); err != nil {
		t.Fatal(err)
	}
	return n
}

// profile makes a Profile of the fixture type wanting from of.
func (s *shelves) profile(t *testing.T, name string, of ...targets.Target) targets.ProfileID {
	t.Helper()
	spec := targets.ProfileSpec{Name: name, ContentType: contenttypetest.ID}
	for _, target := range of {
		spec.Entries = append(spec.Entries, targets.Entry{Target: target.ID, Sourcing: targets.Download})
	}
	p, err := s.goals.CreateProfile(t.Context(), identity.Admin{}, spec)
	if err != nil {
		t.Fatal(err)
	}
	return p.ID
}

// add adds the Work key of typ, of one Item and unmonitored, through the API, into root
// and wanting from profile when they are set.
func (s *shelves) add(t *testing.T, typ works.TypeID, key string, root *roots.Root, profile *targets.ProfileID) string {
	t.Helper()
	s.fx.Put(works.Key(key), contenttypetest.Record{Tree: works.Fetched{Label: "Shelf " + key, Children: []works.Entry{
		{Key: "b1", Label: "Winter", Children: []works.Entry{{Key: "c1", Label: "Scarf"}}},
	}}})
	body := `{"contentType":"` + string(typ) + `","key":"` + key + `"`
	if root != nil {
		body += `,"rootId":"` + root.ID.String() + `"`
	}
	if profile != nil {
		body += `,"profileId":"` + profile.String() + `"`
	}
	code, reply := s.Call(t, http.MethodPost, "/api/v1/works", body+`,"monitor":false}`)
	if code != http.StatusOK {
		t.Fatalf("addWork %s = %d %s", key, code, reply)
	}
	id := ownertest.FirstID(reply)
	s.byKey[key] = id
	s.events.name(id, key)
	s.events.take()
	return id
}

func (s *shelves) item(t *testing.T, work string) works.ItemID {
	t.Helper()
	var item works.ItemID
	err := s.lib.Read(t.Context(), func(ctx context.Context, r storage.Reader[storage.Library]) error {
		id := workID(t, work)
		items, err := works.ItemsOf(ctx, r, []works.WorkID{id})
		item = items[id][0].ID
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	return item
}

// download imports a file of work's Item into home's root as a Grab's
// import does, then settles the disk queue so its bytes sit at its path.
func (s *shelves) download(t *testing.T, work string, root roots.Root, content string) {
	t.Helper()
	holding := files.HoldingOf(storage.NewUUID(), ".mkv")
	at := filepath.Join(root.Path, filepath.FromSlash(holding))
	if err := os.MkdirAll(filepath.Dir(at), 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(at, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	err := s.lib.Write(t.Context(), func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
		_, _, err := s.queue.Import(ctx, tx, files.Incoming{
			WorkID: workID(t, work), ItemID: s.item(t, work), Cut: release.Standard, RootID: root.ID,
			Holding: holding, Size: int64(len(content)), Probe: files.Probe{Width: 1920, Height: 1080}, Quality: release.WEBDL1080p,
		})
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	s.settle(t)
	s.events.take()
}

// record adds a file of work's Item at path in root, whose bytes the
// test never writes.
func (s *shelves) record(t *testing.T, work string, root roots.Root, path string) {
	t.Helper()
	err := s.lib.Write(t.Context(), func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
		_, err := files.Add(ctx, tx, files.File{
			WorkID: workID(t, work), ItemID: s.item(t, work), Cut: release.Standard, RootID: root.ID,
			Path: path, Size: 2048, Probe: files.Probe{Width: 1920, Height: 1080}, Quality: release.WEBDL1080p,
		}, nil)
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
}

func (s *shelves) queued(t *testing.T) int {
	t.Helper()
	return s.count(t, `SELECT (SELECT count(*) FROM files_move) + (SELECT count(*) FROM files_doomed)`)
}

// settle runs the disk queue until it holds no row.
func (s *shelves) settle(t *testing.T) {
	t.Helper()
	ctx, cancel := context.WithCancel(t.Context())
	done := make(chan error, 1)
	go func() { done <- s.queue.Run(ctx) }()
	s.queue.Poke()
	deadline := time.Now().Add(10 * time.Second)
	for s.queued(t) > 0 {
		if time.Now().After(deadline) {
			cancel()
			t.Fatalf("the disk queue still holds %d rows", s.queued(t))
		}
		time.Sleep(time.Millisecond)
	}
	cancel()
	if err := <-done; err != nil {
		t.Fatal(err)
	}
}

// onDisk is every file under root and what it holds.
func onDisk(t *testing.T, root roots.Root) map[string]string {
	t.Helper()
	out := map[string]string{}
	err := filepath.WalkDir(root.Path, func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || d.Name() == storing.MarkerName {
			return err
		}
		data, err := os.ReadFile(p)
		if err != nil {
			return err //nolint:wrapcheck // the test reports it as is
		}
		rel, err := filepath.Rel(root.Path, p)
		out[filepath.ToSlash(rel)] = string(data)
		return err //nolint:wrapcheck // the test reports it as is
	})
	if err != nil {
		t.Fatal(err)
	}
	return out
}

type answered struct {
	Outcomes []struct {
		WorkID  string `json:"workId"`
		Outcome string `json:"outcome"`
		Reason  string `json:"reason"`
		Problem *struct {
			Code   string `json:"code"`
			Title  string `json:"title"`
			Detail string `json:"detail"`
		} `json:"problem"`
	} `json:"outcomes"`
	Impact *targetsImpact `json:"impact"`
	Plan   *plan          `json:"plan"`
}

type targetsImpact struct {
	Works    int `json:"works"`
	OwnList  int `json:"ownList"`
	Unwanted struct {
		Files int   `json:"files"`
		Bytes int64 `json:"bytes"`
	} `json:"unwanted"`
	NewlyWanted int `json:"newlyWanted"`
}

type plan struct {
	Works       int   `json:"works"`
	Files       int   `json:"files"`
	Kept        int   `json:"kept"`
	Bytes       int64 `json:"bytes"`
	Downloading int   `json:"downloading"`
}

// act calls an action that answers 200 and its outcomes, one line per
// Work by the key the test added it under, or nobody.
func (s *shelves) act(t *testing.T, method, path, body string) (string, answered) {
	t.Helper()
	code, reply := s.Call(t, method, path, body)
	if code != http.StatusOK {
		t.Fatalf("%s %s = %d %s", method, path, code, reply)
	}
	var a answered
	if err := json.Unmarshal([]byte(reply), &a); err != nil {
		t.Fatal(err)
	}
	keys := map[string]string{}
	for k, id := range s.byKey {
		keys[id] = k
	}
	var lines []string
	for _, o := range a.Outcomes {
		line := cmpOr(keys[o.WorkID], "nobody") + " " + o.Outcome
		switch {
		case o.Reason != "":
			line += " " + o.Reason
		case o.Problem != nil:
			line += " " + o.Problem.Code + ": " + o.Problem.Title
			if o.Problem.Detail != "" {
				line += ". " + o.Problem.Detail
			}
		}
		lines = append(lines, line)
	}
	return strings.Join(lines, "\n"), a
}

func cmpOr(a, b string) string {
	if a == "" {
		return b
	}
	return a
}

func ids(ws ...string) string {
	return `{"workIds":["` + strings.Join(ws, `","`) + `"]`
}

func expectOutcomes(t *testing.T, what, got, want string) {
	t.Helper()
	if got != want {
		t.Fatalf("%s answers\n%s\nwant\n%s", what, got, want)
	}
}

func expectEvents(t *testing.T, s *shelves, what, want string) {
	t.Helper()
	if got := s.events.take(); got != want {
		t.Fatalf("%s published\n%s\nwant\n%s", what, got, want)
	}
}

func TestMonitorAnswersEachWorkOnceAndPublishesOneBatch(t *testing.T) {
	s := shelve(t)
	w1 := s.add(t, contenttypetest.ID, "w1", nil, nil)
	w2 := s.add(t, contenttypetest.ID, "w2", nil, nil)
	nobody := storage.NewUUID().String()

	got, _ := s.act(t, http.MethodPut, "/api/v1/works/monitored", ids(w1, w2, nobody, w1)+`,"on":true}`)
	expectOutcomes(t, "monitoring", got, "w1 done\nw2 done\nnobody failed work_not_found: No Work has that id")
	expectEvents(t, s, "monitoring", "works.changed w1 w2")

	got, _ = s.act(t, http.MethodPut, "/api/v1/works/monitored", ids(w2, w1)+`,"on":true}`)
	expectOutcomes(t, "monitoring again", got, "w2 skipped already\nw1 skipped already")
	expectEvents(t, s, "monitoring again", "")

	got, _ = s.act(t, http.MethodPut, "/api/v1/works/monitored", ids(w1)+`,"on":false}`)
	expectOutcomes(t, "unmonitoring", got, "w1 done")
	expectEvents(t, s, "unmonitoring", "works.changed w1")
	if on := s.count(t, `SELECT count(*) FROM works_item WHERE monitored = 1`); on != 1 {
		t.Fatalf("%d Items are monitored, want w2's alone", on)
	}
}

func TestSetProfileGivesEveryWorkOneImpactAndKeepsOwnLists(t *testing.T) {
	s := shelve(t)
	uhd, err := s.goals.CreateTarget(t.Context(), identity.Admin{}, targets.Spec{Name: "UHD", Tiers: [][]release.Quality{{release.Remux2160p}}, Cutoff: release.Remux2160p})
	if err != nil {
		t.Fatal(err)
	}
	web, big := s.profile(t, "Web", s.web), s.profile(t, "Big", uhd)
	w1 := s.add(t, contenttypetest.ID, "w1", nil, &web)
	w2 := s.add(t, contenttypetest.ID, "w2", nil, &web)
	w3 := s.add(t, contenttypetest.ID, "w3", nil, &big)
	o1 := s.add(t, other, "o1", nil, nil)
	nobody := storage.NewUUID().String()
	s.download(t, w1, s.movies, "w1 web")
	own, err := targets.NamedCuts(release.Standard, "directors-cut")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.queue.Change(t.Context(), identity.Admin{}, targets.SetWants{Work: workID(t, w2), Profile: web, Own: &own}, files.Mode{Unwanted: files.Delete}); err != nil {
		t.Fatal(err)
	}
	s.events.take()
	body := ids(w1, w2, w3, o1, nobody) + `,"profileId":"` + big.String() + `"}`
	want := "w1 done\nw2 done\nw3 skipped already\n" +
		"o1 failed profile_wrong_type: The Profile is for another Content Type\nnobody failed work_not_found: No Work has that id"

	got, a := s.act(t, http.MethodPut, "/api/v1/works/profile?preview=true", body)
	expectOutcomes(t, "a preview", got, want)
	if i := a.Impact; i == nil || i.Works != 2 || i.OwnList != 1 || i.Unwanted.Files != 1 || i.Unwanted.Bytes != 6 {
		t.Fatalf("the preview's Impact is %+v, want 2 Works, 1 own list kept and w1's 6-byte file unwanted", i)
	}
	expectEvents(t, s, "a preview", "")

	s.Expect(t, http.MethodPut, "/api/v1/works/profile?confirm=0", body, http.StatusConflict,
		`{"status":409,"code":"impact_changed","title":"The change now does more than its preview showed",`+
			`"detail":"The save now leaves 1 file unwanted, and the preview showed 0. Preview the change again."}`)
	expectEvents(t, s, "a refused save", "")

	got, _ = s.act(t, http.MethodPut, "/api/v1/works/profile?confirm=1", body)
	expectOutcomes(t, "the save", got, want)
	expectEvents(t, s, "the save", "targets.changed w1 w2")
	for _, w := range []string{w1, w2} {
		g, err := s.goals.Goals(t.Context(), identity.Admin{}, workID(t, w))
		if err != nil {
			t.Fatal(err)
		}
		if g.Profile.ID != big || g.Own != (w == w2) {
			t.Errorf("Work %s wants from %s, own list %t, want Big with only w2 keeping its own", s.events.names[w], g.Profile.Name, g.Own)
		}
	}
	if left := s.count(t, `SELECT count(*) FROM files_file`); left != 0 {
		t.Fatalf("%d files stay, want w1's file deleted as the save chose", left)
	}

	s.Expect(t, http.MethodPut, "/api/v1/works/profile", ids(w1)+`,"profileId":"`+storage.NewUUID().String()+`"}`, http.StatusUnprocessableEntity,
		`{"status":422,"code":"profile_not_found","title":"No Profile has that id"}`)
}

func TestSetRootSkipsAWorkWithFiles(t *testing.T) {
	s := shelve(t)
	w1 := s.add(t, contenttypetest.ID, "w1", nil, nil)
	w2 := s.add(t, contenttypetest.ID, "w2", nil, nil)
	o1 := s.add(t, other, "o1", nil, nil)
	nobody := storage.NewUUID().String()
	s.download(t, w2, s.movies, "w2 web")
	body := ids(w1, w2, o1, nobody) + `,"rootId":"` + s.films.ID.String() + `"}`

	got, _ := s.act(t, http.MethodPut, "/api/v1/works/root", body)
	expectOutcomes(t, "rehoming", got, "w1 done\nw2 skipped work_has_files\n"+
		"o1 failed root_wrong_type: The root holds another Content Type's Works\nnobody failed work_not_found: No Work has that id")
	expectEvents(t, s, "rehoming", "roots.changed films movies\nworks.changed w1")
	if in := s.count(t, `SELECT count(*) FROM roots_work WHERE root_id = ?`, s.films.ID); in != 1 {
		t.Fatalf("films holds %d Works, want w1 alone", in)
	}

	got, _ = s.act(t, http.MethodPut, "/api/v1/works/root", body)
	expectOutcomes(t, "rehoming again", got, "w1 skipped already\nw2 skipped work_has_files\n"+
		"o1 failed root_wrong_type: The root holds another Content Type's Works\nnobody failed work_not_found: No Work has that id")
	expectEvents(t, s, "rehoming again", "")

	s.Expect(t, http.MethodPut, "/api/v1/works/root", ids(w1)+`,"rootId":"`+storage.NewUUID().String()+`"}`, http.StatusNotFound,
		`{"status":404,"code":"root_not_found","title":"No root has that id"}`)
}

func TestRefreshSkipsWorksGoneUpstreamWithoutCallingTheirSource(t *testing.T) {
	s := shelve(t)
	changes := s.add(t, contenttypetest.ID, "w1", nil, nil)
	same := s.add(t, contenttypetest.ID, "w2", nil, nil)
	gone := s.add(t, contenttypetest.ID, "w3", nil, nil)
	going := s.add(t, contenttypetest.ID, "w4", nil, nil)
	away := s.add(t, contenttypetest.ID, "w5", nil, nil)
	nobody := storage.NewUUID().String()
	s.fx.Fail("w3", fmt.Errorf("lookup: %w", works.ErrNotFound))
	s.act(t, http.MethodPost, "/api/v1/works/refresh", ids(gone)+"}")
	s.events.take()
	fetched := s.fx.Fetches("w3")
	s.fx.Put("w1", contenttypetest.Record{Tree: works.Fetched{Label: "Shelf w1, revised", Children: []works.Entry{
		{Key: "b1", Label: "Winter", Children: []works.Entry{{Key: "c1", Label: "Scarf"}}},
	}}})
	s.fx.Fail("w4", fmt.Errorf("lookup: %w", works.ErrNotFound))
	s.fx.Fail("w5", works.ErrUnavailable)

	got, _ := s.act(t, http.MethodPost, "/api/v1/works/refresh", ids(changes, same, gone, going, away, nobody)+"}")
	expectOutcomes(t, "refreshing", got, "w1 done\nw2 done\nw3 skipped work_gone_upstream\nw4 skipped work_gone_upstream\n"+
		"w5 failed metadata_unavailable: The metadata source didn't respond. Try again later.\nnobody failed work_not_found: No Work has that id")
	expectEvents(t, s, "refreshing", "works.changed w1 w4")
	if n := s.fx.Fetches("w3"); n != fetched {
		t.Fatalf("w3, already gone upstream, was fetched %d more times, want none", n-fetched)
	}
}

func TestDeleteWithFilesRemovesTheBytesThroughTheDiskQueue(t *testing.T) {
	s := shelve(t)
	web := s.profile(t, "Web", s.web)
	w1 := s.add(t, contenttypetest.ID, "w1", nil, &web)
	w2 := s.add(t, contenttypetest.ID, "w2", nil, &web)
	s.download(t, w1, s.movies, "w1 web")
	s.download(t, w1, s.movies, "w1 kept")
	s.download(t, w2, s.movies, "w2 web")
	nobody := storage.NewUUID().String()

	got, a := s.act(t, http.MethodPost, "/api/v1/works/delete", ids(w1, nobody)+`,"files":"delete","preview":true}`)
	expectOutcomes(t, "a preview", got, "w1 done\nnobody skipped work_not_found")
	if want := (plan{Works: 1, Files: 2, Kept: 1, Bytes: 13}); a.Plan == nil || *a.Plan != want {
		t.Fatalf("the preview's plan is %+v, want %+v", a.Plan, want)
	}
	expectEvents(t, s, "a preview", "")

	got, a = s.act(t, http.MethodPost, "/api/v1/works/delete", ids(w1, nobody)+`,"files":"delete"}`)
	expectOutcomes(t, "the delete", got, "w1 done\nnobody skipped work_not_found")
	if want := (plan{Works: 1, Files: 2, Kept: 1, Bytes: 13}); a.Plan == nil || *a.Plan != want {
		t.Fatalf("the delete's plan is %+v, want %+v", a.Plan, want)
	}
	expectEvents(t, s, "the delete", "works.removed w1\nroots.changed movies")
	s.settle(t)
	if got, want := onDisk(t, s.movies), map[string]string{"Shelf w2/Shelf w2 - Scarf - Web.mkv": "w2 web"}; !maps.Equal(got, want) {
		t.Fatalf("movies holds %v once the disk queue ran, want w2's file alone", got)
	}
	s.Expect(t, http.MethodGet, "/api/v1/works/"+w1, "", http.StatusNotFound, `{"status":404,"code":"work_not_found","title":"No Work has that id"}`)
	for _, table := range []string{"works_item", "files_file", "targets_wants", "roots_work"} {
		if n := s.count(t, `SELECT count(*) FROM `+table+` WHERE work_id = ?`, workID(t, w1)); n != 0 {
			t.Errorf("%s keeps %d rows of the deleted Work", table, n)
		}
	}

	got, _ = s.act(t, http.MethodPost, "/api/v1/works/delete", ids(w1)+`,"files":"delete"}`)
	expectOutcomes(t, "a repeat", got, "w1 skipped work_not_found")
	expectEvents(t, s, "a repeat", "")
}

func TestDeleteLeavingFilesKeepsTheBytes(t *testing.T) {
	s := shelve(t)
	web := s.profile(t, "Web", s.web)
	w1 := s.add(t, contenttypetest.ID, "w1", nil, &web)
	s.download(t, w1, s.movies, "w1 web")
	before := onDisk(t, s.movies)

	got, a := s.act(t, http.MethodPost, "/api/v1/works/delete", ids(w1)+`,"files":"leave"}`)
	expectOutcomes(t, "the delete", got, "w1 done")
	if want := (plan{Works: 1, Files: 1, Bytes: 6}); a.Plan == nil || *a.Plan != want {
		t.Fatalf("the delete's plan is %+v, want %+v", a.Plan, want)
	}
	expectEvents(t, s, "the delete", "works.removed w1\nroots.changed movies")
	if n := s.queued(t); n != 0 {
		t.Fatalf("leaving the files queued %d rows for the disk queue, want none", n)
	}
	s.settle(t)
	if got := onDisk(t, s.movies); !maps.Equal(got, before) || len(got) != 1 {
		t.Fatalf("movies holds %v after the delete, want the file left as it was: %v", got, before)
	}
}

func TestDeleteRefusesAWorkWhoseFilesSitOnAnOfflineNode(t *testing.T) {
	s := shelve(t)
	boxedWork := s.add(t, contenttypetest.ID, "w1", &s.boxed, nil)
	homeWork := s.add(t, contenttypetest.ID, "w2", nil, nil)
	s.record(t, boxedWork, s.boxed, "Shelf w1/Shelf w1.mkv")
	refused := "w1 failed storage_unreachable: The Work's files can't be deleted. " +
		"seedbox is offline, so this Work's files there can't be deleted. Delete the Work and leave its files, or try again once seedbox is back."

	got, _ := s.act(t, http.MethodPost, "/api/v1/works/delete", ids(boxedWork)+`,"files":"delete","preview":true}`)
	expectOutcomes(t, "a preview while seedbox is online", got, "w1 done")
	s.roster.drop(seedbox)
	got, a := s.act(t, http.MethodPost, "/api/v1/works/delete", ids(boxedWork, homeWork)+`,"files":"delete"}`)
	expectOutcomes(t, "the delete once seedbox dropped", got, refused+"\nw2 done")
	if want := (plan{Works: 1}); a.Plan == nil || *a.Plan != want {
		t.Fatalf("the delete's plan is %+v, want w2 alone", a.Plan)
	}
	expectEvents(t, s, "the delete", "works.removed w2")
	code, reply := s.Call(t, http.MethodGet, "/api/v1/works/"+boxedWork, "")
	if code != http.StatusOK {
		t.Fatalf("getWork of the refused Work = %d %s, want it whole", code, reply)
	}
	if n := s.count(t, `SELECT count(*) FROM files_file WHERE work_id = ?`, workID(t, boxedWork)); n != 1 || s.queued(t) != 0 {
		t.Fatalf("the refused Work keeps %d files and the queue holds %d rows, want its file and nothing queued", n, s.queued(t))
	}

	got, _ = s.act(t, http.MethodPost, "/api/v1/works/delete", ids(boxedWork)+`,"files":"leave"}`)
	expectOutcomes(t, "leaving its files", got, "w1 done")
	expectEvents(t, s, "leaving its files", "works.removed w1\nroots.changed boxed")
}

func TestADeleteThatFailsInItsWriteLeavesEveryRow(t *testing.T) {
	s := shelve(t)
	w1 := s.add(t, contenttypetest.ID, "w1", nil, nil)
	w2 := s.add(t, contenttypetest.ID, "w2", nil, nil)
	s.record(t, w1, s.movies, "Shelf w1/Shelf w1.mkv")
	s.record(t, w2, s.movies, "Shelf w2/Shelf w2.mkv")
	var doomed storage.UUID
	err := s.lib.QueryRowContext(t.Context(), `SELECT id FROM files_file WHERE work_id = ?`, workID(t, w2)).Scan(&doomed)
	if err != nil {
		t.Fatal(err)
	}
	s.exec(t, `INSERT INTO files_doomed (file_id, root_id, path) VALUES (?, ?, 'Elsewhere/Shelf w2.mkv')`, doomed, s.movies.ID)
	rows := func() string {
		var b strings.Builder
		for _, table := range []string{"works_work", "works_item", "files_file", "targets_wants", "roots_work", "files_doomed"} {
			fmt.Fprintf(&b, "%s %d\n", table, s.count(t, `SELECT count(*) FROM `+table))
		}
		return b.String()
	}
	before := rows()

	s.Expect(t, http.MethodPost, "/api/v1/works/delete", ids(w1, w2)+`,"files":"delete"}`, http.StatusInternalServerError,
		`{"status":500,"code":"internal","title":"SlipStream had an internal error. This Node's log has the details."}`)
	if after := rows(); after != before {
		t.Fatalf("after a delete whose Write failed the library holds\n%swant every row as before\n%s", after, before)
	}
	expectEvents(t, s, "the failed delete", "")
}

func TestDeletingAWorkMidDownloadLeavesItsTorrentSeeding(t *testing.T) {
	s := shelve(t)
	w1 := s.add(t, contenttypetest.ID, "w1", nil, nil)
	indexer, torrent, grab := storage.NewUUID(), storage.NewUUID(), storage.NewUUID()
	hash := []byte("aaaaaaaaaaaaaaaaaaaa")
	s.exec(t, `INSERT INTO indexers_indexer VALUES (?, 'Fixture Site', 'torznab', 'https://site.test/api', NULL, ?, '{}', 0)`, indexer, seedbox)
	s.exec(t, `INSERT INTO downloadclients_torrent (id, node_id, info_hash, metainfo, name, size, private, trackers, content_type, wanted,
		added_order, added_at) VALUES (?, ?, ?, x'00', 'Shelf.w1.1080p.WEB', 4096, 1, '[]', 'fixture', 'started', 1, 0)`, torrent, seedbox, hash)
	s.exec(t, `INSERT INTO grabs_grab (id, work_id, item_id, edition, source, indexer_id, node_id, release_title, release_size, release_link,
		stage, torrent_id, created_at, updated_at)
		SELECT ?, work_id, id, 'any', 'search', ?, ?, 'Shelf.w1.1080p.WEB', 4096, 'https://site.test/dl/1', 'downloading', ?, 0, 0
		FROM works_item WHERE work_id = ?`, grab, indexer, seedbox, torrent, workID(t, w1))

	got, a := s.act(t, http.MethodPost, "/api/v1/works/delete", ids(w1)+`,"files":"delete"}`)
	expectOutcomes(t, "the delete", got, "w1 done")
	if a.Plan == nil || a.Plan.Downloading != 1 {
		t.Fatalf("the delete's plan is %+v, want its one download counted", a.Plan)
	}
	if n := s.count(t, `SELECT count(*) FROM grabs_grab`); n != 0 {
		t.Fatalf("%d Grabs outlive their Work, want none", n)
	}
	var seeding []bencode.InfoHash
	err := s.lib.Read(t.Context(), func(ctx context.Context, r storage.Reader[storage.Library]) error {
		in, err := downloadclients.Intent(ctx, r, seedbox)
		for _, tor := range in.GetTorrents() {
			seeding = append(seeding, bencode.InfoHash(tor.GetInfoHash()))
		}
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(seeding, []bencode.InfoHash{bencode.InfoHash(hash)}) {
		t.Fatalf("seedbox's intent is %v after the delete, want its Torrent still seeding", seeding)
	}
}

func workID(t *testing.T, s string) works.WorkID {
	t.Helper()
	id, err := storage.ParseUUID(s)
	if err != nil {
		t.Fatal(err)
	}
	return works.WorkID{UUID: id}
}

// TestEveryActionTakesTheMostWorksARequestMayName drives each action over
// 20,000 Works, the most a request may name, so no read on its path binds
// more ids than SQLite takes.
func TestEveryActionTakesTheMostWorksARequestMayName(t *testing.T) {
	s := shelve(t)
	web, big := s.profile(t, "Web", s.web), s.profile(t, "Big")
	ws := s.seed(t, 20000, web)
	all := ids(ws...)
	done := func(what, path, body string) {
		t.Helper()
		_, a := s.act(t, http.MethodPost, path, body)
		for _, o := range a.Outcomes {
			if o.Outcome != "done" {
				t.Fatalf("%s answers %s for a Work, want every one done", what, o.Outcome)
			}
		}
		if len(a.Outcomes) != len(ws) {
			t.Fatalf("%s answers %d outcomes, want %d", what, len(a.Outcomes), len(ws))
		}
	}
	put := func(what, path, body string) {
		t.Helper()
		_, a := s.act(t, http.MethodPut, path, body)
		if len(a.Outcomes) != len(ws) || a.Outcomes[len(ws)-1].Outcome != "done" {
			t.Fatalf("%s answers %d outcomes, the last %+v, want %d done", what, len(a.Outcomes), a.Outcomes[len(a.Outcomes)-1], len(ws))
		}
	}
	put("monitoring", "/api/v1/works/monitored", all+`,"on":true}`)
	put("rehoming", "/api/v1/works/root", all+`,"rootId":"`+s.films.ID.String()+`"}`)
	put("a Profile change", "/api/v1/works/profile?confirm=0", all+`,"profileId":"`+big.String()+`"}`)
	done("a delete", "/api/v1/works/delete", all+`,"files":"delete"}`)
	if n := s.count(t, `SELECT count(*) FROM works_work`); n != 0 {
		t.Fatalf("%d Works stay after the delete, want none", n)
	}
}

// seed writes n Works of one Item each, without files, into movies and
// wanting from profile.
func (s *shelves) seed(t *testing.T, n int, profile targets.ProfileID) []string {
	t.Helper()
	ws := make([]string, n)
	err := s.lib.Write(t.Context(), func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
		for i := range ws {
			id := storage.NewUUID()
			ws[i] = id.String()
			for _, q := range []struct {
				query string
				args  []any
			}{
				{`INSERT INTO works_work (id, content_type, key, label, sort_label, added_at, refreshed_at) VALUES (?, ?, ?, 'Shelf', 'Shelf', 0, 0)`, []any{id, contenttypetest.ID, fmt.Sprintf("k%d", i)}},
				{`INSERT INTO works_item (id, work_id, key, label, position) VALUES (?, ?, 'c1', 'Scarf', 1)`, []any{storage.NewUUID(), id}},
				{`INSERT INTO targets_wants (work_id, profile_id) VALUES (?, ?)`, []any{id, profile}},
				{`INSERT INTO roots_work (work_id, root_id) VALUES (?, ?)`, []any{id, s.movies.ID}},
			} {
				if _, err := tx.ExecContext(ctx, q.query, q.args...); err != nil {
					return err
				}
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return ws
}
