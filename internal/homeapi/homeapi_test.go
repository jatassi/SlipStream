package homeapi_test

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"slices"
	"testing"
	"time"

	"github.com/jatassi/slipstream/internal/api"
	"github.com/jatassi/slipstream/internal/api/ownertest"
	"github.com/jatassi/slipstream/internal/bencode"
	"github.com/jatassi/slipstream/internal/contenttype/contenttypetest"
	"github.com/jatassi/slipstream/internal/conversions"
	"github.com/jatassi/slipstream/internal/downloadclients"
	"github.com/jatassi/slipstream/internal/files"
	"github.com/jatassi/slipstream/internal/grabs"
	"github.com/jatassi/slipstream/internal/home"
	"github.com/jatassi/slipstream/internal/homeapi"
	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/imports"
	"github.com/jatassi/slipstream/internal/indexers"
	"github.com/jatassi/slipstream/internal/indexers/indexerstest"
	"github.com/jatassi/slipstream/internal/metadata/artwork"
	"github.com/jatassi/slipstream/internal/moves"
	"github.com/jatassi/slipstream/internal/nodes"
	"github.com/jatassi/slipstream/internal/nodes/nodestest"
	dlv1 "github.com/jatassi/slipstream/internal/pb/slipstream/downloading/v1"
	"github.com/jatassi/slipstream/internal/release"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/roots"
	"github.com/jatassi/slipstream/internal/roots/rootstest"
	"github.com/jatassi/slipstream/internal/secret"
	"github.com/jatassi/slipstream/internal/secret/secrettest"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/storage/storagetest"
	"github.com/jatassi/slipstream/internal/targets"
	"github.com/jatassi/slipstream/internal/transfers"
	"github.com/jatassi/slipstream/internal/works"
)

var admin = identity.Admin{}

type discard[T any] struct{}

func (discard[T]) Publish(context.Context, T) {}

// heard counts the events a test hears.
type heard[T any] struct{ n *int }

func (h heard[T]) Publish(context.Context, T) { *h.n++ }

const filmsID works.TypeID = "films"

// films is the Fixture as a type of one Item.
type films struct{ *contenttypetest.Fixture }

func (films) Describe() works.Description {
	d := contenttypetest.Description
	d.Name, d.Work, d.Item, d.Items = "Films", works.Noun{One: "Film", Many: "Films"}, works.Noun{One: "Film", Many: "Films"}, works.ItemsOne
	d.Levels, d.Aside = nil, works.Noun{}
	d.Fields = slices.DeleteFunc(slices.Clone(d.Fields), func(f works.FieldSpec) bool { return f.Tier != works.TierWork && f.Tier != works.TierItem })
	d.Artwork = slices.DeleteFunc(slices.Clone(d.Artwork), func(k artwork.Kind) bool { return k.Tier != works.TierWork && k.Tier != works.TierItem })
	d.Dated = nil
	return d
}

// roster is one Controller that downloads, and a seedbox out of step.
type roster struct{ self, away role.NodeID }

func (r roster) Route(role.Name, role.NodeID) (role.NodeID, error) { return r.self, nil }
func (r roster) InStep(n role.NodeID) bool                         { return n != r.away }
func (roster) Changed() <-chan struct{}                            { return nil }
func (roster) WrittenOff(role.Name, role.NodeID) bool              { return false }
func (r roster) Holds(_ role.Name, n role.NodeID) bool             { return n == r.self || n == r.away }
func (r roster) Serving(role.Name) []role.NodeID                   { return []role.NodeID{r.self, r.away} }

type noRoutes struct{}

func (noRoutes) Route(context.Context, storage.Reader[storage.Library], indexers.TrackerID, indexers.ID) (downloadclients.Target, bool, error) {
	return downloadclients.Target{}, false, nil
}

type noSeeding struct{}

func (noSeeding) Judge(context.Context, storage.Reader[storage.Library], []downloadclients.Row) (map[downloadclients.ID]downloadclients.Judged, error) {
	return nil, nil
}

func (noSeeding) EndlessUnits(context.Context, storage.Reader[storage.Library], role.NodeID) (int64, error) {
	return 0, nil
}

type node struct {
	h *home.Service
	m artwork.Minter
}

func (n node) Home() (*home.Service, bool)    { return n.h, true }
func (n node) Minter() (artwork.Minter, bool) { return n.m, true }

type world struct {
	lib        *storage.DB[storage.Library]
	fx         *contenttypetest.Fixture
	works      *works.Works
	hd         targets.Target
	root       roots.Root
	r          roster
	client     downloadclients.ClientID
	off        downloadclients.ClientID
	s          *ownertest.Owner
	h          *home.Service
	clients    *downloadclients.Clients
	found      *imports.Imports
	sealer     *secret.Sealer
	saved      int
	added      int
	grabsAdded int
}

func newWorld(t *testing.T) *world {
	ctx := t.Context()
	lib := storagetest.Library(t, secret.Schema(), identity.Schema(), nodes.Schema(), role.LibrarySchema(), works.Schema(), targets.Schema(),
		roots.Schema(), downloadclients.Schema(), grabs.Schema(), indexers.Schema(), files.Schema(), conversions.Schema(), transfers.LibrarySchema(), imports.Schema(), home.Schema())
	fx := contenttypetest.New()
	film := films{fx}
	w := &world{lib: lib, fx: fx, r: roster{self: role.NodeID{UUID: storage.NewUUID()}, away: role.NodeID{UUID: storage.NewUUID()}}}
	w.works = must[*works.Works](t)(works.Open(ctx, works.Config{
		Library: lib, Catalogs: map[works.TypeID]works.Catalog{contenttypetest.ID: fx, filmsID: film},
		Source: fx.SourceAs(contenttypetest.ID, filmsID), Artwork: fx,
		Changed: discard[works.Changed]{}, TypesChanged: discard[works.ContentTypesChanged]{},
	}))
	goals := must[*targets.Targets](t)(targets.Open(ctx, targets.Config{
		Library: lib, Works: w.works, Types: map[works.TypeID]targets.Facet{contenttypetest.ID: fx, filmsID: film}, Changed: discard[targets.Changed]{},
	}))
	w.hd = must[targets.Target](t)(goals.CreateTarget(ctx, admin, targets.Spec{
		Scores: targets.Scores{Increment: 1}, Name: "HD", Tiers: [][]release.Quality{{release.WEBDL1080p}}, Cutoff: release.WEBDL1080p,
	}))
	err := lib.Write(ctx, func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
		if err := nodestest.Insert(ctx, tx, w.r.self, "home"); err != nil {
			return err
		}
		return nodestest.Insert(ctx, tx, w.r.away, "seedbox")
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, ct := range []works.TypeID{contenttypetest.ID, filmsID} {
		p := must[targets.Profile](t)(goals.CreateProfile(ctx, admin, targets.ProfileSpec{Name: "Standard " + string(ct), ContentType: ct, Entries: []targets.Entry{{Target: w.hd.ID, Sourcing: targets.Download}}}))
		must[targets.Profile](t)(goals.MakeDefault(ctx, admin, p.ID))
		r := rootstest.Root(t, lib, w.r.self, ct, t.TempDir())
		if ct == contenttypetest.ID {
			w.root = r
		}
	}
	w.client, w.off = downloadclients.ClientID{UUID: storage.NewUUID()}, downloadclients.ClientID{UUID: storage.NewUUID()}
	w.exec(t, `INSERT INTO downloadclients_client (id, node_id, kind, name, address, address_key, label, is_on, settings_version,
		poll_state, poll_since, poll_settings_version, created_at, updated_at)
		VALUES (?, ?, 'transmission', 'Transmission', 'http://192.0.2.9:9091', '192.0.2.9:9091', 'slipstream', 1, 1, 'responding', 0, 1, 0, 0),
		(?, ?, 'qbittorrent', 'qBittorrent', 'http://192.0.2.10:8080', '192.0.2.10:8080', 'slipstream', 0, 1, NULL, NULL, NULL, 0, 0)`,
		w.client, w.r.self, w.off, w.r.self)

	router := downloadclients.Router{Roster: w.r, Routes: noRoutes{}}
	cells := grabs.GrabCells{InStep: w.r.InStep, Router: router}
	shelf := must[*files.Files](t)(files.Open(ctx, files.Config{Library: lib, Targets: goals, Grabs: imports.Cells{Grabs: cells}, Today: time.Now}))
	clients := downloadclients.New(downloadclients.Config{
		Library: lib, Roster: w.r, Uses: imports.Uses{TorrentUses: grabs.TorrentUses{}}, Seeding: noSeeding{},
		Touch: func(...role.NodeID) {}, Changed: discard[downloadclients.Changed]{}, CardsChanged: discard[downloadclients.ClientsChanged]{},
		Volumes: discard[roots.VolumesChanged]{}, Log: slog.New(slog.DiscardHandler),
	})
	w.sealer = secrettest.Sealer(t)
	xs := indexers.New(indexers.Config{
		Library: lib, Sealer: w.sealer, Roster: indexerstest.Alone(w.r.self), Works: w.works,
		Changed: discard[indexers.Changed]{}, Status: discard[indexers.StatusChanged]{}, Trackers: discard[indexers.TrackerChanged]{},
		Timings: indexers.DefaultTimings(), Log: slog.New(slog.DiscardHandler),
	})
	g := grabs.New(grabs.Config{
		Library: lib, Sealer: w.sealer, Indexers: xs, Works: w.works, Targets: goals, Files: shelf, Roster: w.r, Router: router, Imports: imports.GrabImports{}, Downloads: clients,
		Names:   map[works.TypeID]release.Names{contenttypetest.ID: fx, filmsID: fx},
		Timings: grabs.DefaultTimings(), Today: time.Now, Log: slog.New(slog.DiscardHandler),
	})
	found := imports.New(imports.Config{Library: lib, Works: w.works, Files: shelf, Transfers: transfers.New(transfers.ControllerConfig{Library: lib})})
	h := home.New(home.Config{
		Library: lib, Moves: noMoves{}, Grabs: g, Imports: found, Works: w.works, Files: shelf, Clients: clients, Changed: heard[home.SettingsChanged]{&w.saved},
	})
	w.h, w.clients, w.found = h, clients, found
	w.s = ownertest.Serve(t, lib, func(r *api.Router) {
		homeapi.Routes(r, node{h, artwork.NewMinter([]byte("home test key, 32 bytes long ok!"))})
	})
	return w
}

func must[T any](t *testing.T) func(T, error) T {
	return func(v T, err error) T {
		t.Helper()
		if err != nil {
			t.Fatal(err)
		}
		return v
	}
}

func (w *world) exec(t *testing.T, query string, args ...any) {
	t.Helper()
	err := w.lib.Write(t.Context(), func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
		_, err := tx.ExecContext(ctx, query, args...)
		return err
	})
	if err != nil {
		t.Fatalf("%s: %v", query, err)
	}
}

// add adds a Work of type ct labeled label with things Items.
func (w *world) add(t *testing.T, ct works.TypeID, label string, things int) works.Tree {
	t.Helper()
	w.added++
	key := works.Key(fmt.Sprintf("w%d", w.added))
	tree := works.Fetched{Label: label}
	if ct == filmsID {
		tree.Children = []works.Entry{{Key: "f", Label: label}}
	} else {
		box := works.Entry{Key: "b1", Label: "Box 1"}
		for i := range things {
			box.Children = append(box.Children, works.Entry{Key: works.Key(fmt.Sprintf("t%d", i)), Label: fmt.Sprintf("Thing %d", i+1)})
		}
		tree.Children = []works.Entry{box}
	}
	w.fx.Put(key, contenttypetest.Record{Tree: tree})
	assign := func(ctx context.Context, tx *storage.Tx[storage.Library], s works.Summary) error {
		if err := targets.Assign(ctx, tx, s, nil); err != nil {
			return err
		}
		return roots.Assign(ctx, tx, s, nil)
	}
	out, _, err := w.works.Add(t.Context(), admin, ct, key, works.Adding{Monitor: true, Also: assign})
	if err != nil {
		t.Fatal(err)
	}
	return out
}

// grabbing is a Grab as a test seeds it.
type grabbing struct {
	item    works.ItemID
	stage   grabs.Stage
	node    role.NodeID
	torrent downloadclients.ID
	failure grabs.Code
	// sent is the client a Manual Grab is pinned to.
	sent *downloadclients.ClientID
	// created and entered are minutes after the test's epoch.
	created, entered int
}

var epoch = time.Date(2026, 11, 22, 9, 0, 0, 0, time.UTC)

func at(minutes int) storage.Time {
	return storage.FromTime(epoch.Add(time.Duration(minutes) * time.Minute))
}

func (w *world) grab(t *testing.T, work works.Tree, g grabbing) grabs.GrabID {
	t.Helper()
	w.grabsAdded++
	id := grabs.GrabID{UUID: storage.NewUUID()}
	if g.node.IsZero() {
		g.node = w.r.self
	}
	if g.item.IsZero() {
		g.item = work.Items[0].ID
	}
	var torrent, failure, failedStage any
	if !g.torrent.IsZero() {
		torrent = g.torrent
	}
	if g.failure != "" {
		failure, failedStage = string(g.failure), "downloading"
	}
	var sent any
	if g.sent != nil {
		sent = *g.sent
	}
	w.exec(t, `INSERT INTO grabs_grab (id, work_id, item_id, edition, source, node_id, release_title, release_size, release_magnet,
		stage, torrent_id, failure_code, failed_stage, failure_message, sent_client_id, created_at, updated_at)
		VALUES (?, ?, ?, 'any', iif(? IS NULL, 'search', 'manual'), ?, ?, 4096, 'magnet:?xt=urn:btih:0123456789abcdef0123456789abcdef01234567', ?, ?, ?, ?, iif(? IS NULL, NULL, ''), ?, ?, ?)`,
		id, work.ID, g.item, sent, g.node, fmt.Sprintf("%s.%d.1080p.WEB-DL", work.Label, w.grabsAdded), string(g.stage), torrent, failure, failedStage, failure, sent, at(g.created), at(g.created))
	w.exec(t, `INSERT INTO grabs_target (grab_id, position, item_id, target_id) VALUES (?, 0, ?, ?)`, id, g.item, w.hd.ID)
	stage := g.stage
	if g.failure != "" {
		stage = grabs.Downloading
	}
	w.exec(t, `INSERT INTO grabs_stage (grab_id, stage, entered_at) VALUES (?, ?, ?)`, id, string(stage), at(g.entered))
	return id
}

// torrent attaches a Torrent named name to client, nil for the built-in
// one, and applies report when there is one.
func (w *world) torrent(t *testing.T, name string, client *downloadclients.ClientID, report *dlv1.TorrentReport) downloadclients.ID {
	t.Helper()
	raw, err := bencode.Encode(map[string]any{
		"announce": "https://beacon.test/announce",
		"info": map[string]any{
			"name": name, "piece length": int64(16384), "pieces": "aaaaaaaaaaaaaaaaaaaa",
			"files": []any{map[string]any{"length": int64(1000), "path": []any{name + ".mkv"}}},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	m, err := bencode.DecodeMetainfo(raw)
	if err != nil {
		t.Fatal(err)
	}
	to := downloadclients.Target{Node: w.r.self, Client: client}
	var id downloadclients.ID
	err = w.lib.Write(t.Context(), func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
		id, _, err = downloadclients.Attach(ctx, tx, downloadclients.Want{Target: to, Metainfo: m, Raw: raw, ContentType: contenttypetest.ID, WrittenOff: func(role.NodeID) bool { return false }}, storage.Now())
		if err != nil || report == nil {
			return err
		}
		report.Id = id.Bytes()
		c := downloadclients.New(downloadclients.Config{Touch: func(...role.NodeID) {}, Changed: discard[downloadclients.Changed]{}})
		return c.Apply(ctx, tx, to.Node, &dlv1.Report{Kind: &dlv1.Report_Torrents{Torrents: &dlv1.Torrents{Reports: []*dlv1.TorrentReport{report}}}})
	})
	if err != nil {
		t.Fatal(err)
	}
	return id
}

// importing seeds grab's unfinished Import with one file at outcome,
// held by hold when it is set.
func (w *world) importing(t *testing.T, grab grabs.GrabID, outcome, reason, hold string) {
	t.Helper()
	imp := storage.NewUUID()
	w.exec(t, `INSERT INTO imports_import (id, source, grab_id, node_id, info_hash, content_type, hint, created_at)
		VALUES (?, 'grab', ?, ?, randomblob(20), 'fixture', '{}', 0)`, imp, grab, w.r.self)
	var r, h, root, dest any
	if reason != "" {
		r = reason
	}
	if hold != "" {
		h, root, dest = hold, w.root.ID, "Attic/Attic.mkv"
	}
	w.exec(t, `INSERT INTO imports_file (import_id, position, path, source_index, size, cut, release_name, source_name, outcome, reason, hold, root_id, dest)
		VALUES (?, 0, 'Attic.mkv', 0, 1000, 'standard', 'Attic', 'Attic.mkv', ?, ?, ?, ?, ?)`, imp, outcome, r, h, root, dest)
}

type reply struct {
	Jobs struct {
		Rows []struct {
			Grab struct {
				ID    string `json:"id"`
				Stage string `json:"stage"`
			} `json:"grab"`
			Work struct {
				Label string `json:"label"`
			} `json:"work"`
			Torrent *struct {
				ID string `json:"id"`
			} `json:"torrent"`
		} `json:"rows"`
		Moving int `json:"moving"`
	} `json:"jobs"`
	NeedsYou struct {
		Rows []struct {
			Grab struct {
				ID string `json:"id"`
			} `json:"grab"`
			Wait *struct {
				Code       string `json:"code"`
				ImportWait *struct {
					RootID string `json:"rootId"`
				} `json:"importWait"`
				ClientError *struct {
					Client string `json:"client"`
				} `json:"clientError"`
			} `json:"wait"`
		} `json:"rows"`
		Failed  int `json:"failed"`
		Waiting int `json:"waiting"`
	} `json:"needsYou"`
	Recent struct {
		Rows []struct {
			ImportID string `json:"importId"`
			Work     struct {
				Label string `json:"label"`
			} `json:"work"`
			Words string `json:"words"`
		} `json:"rows"`
		Today struct {
			Imported int `json:"imported"`
			Upgraded int `json:"upgraded"`
		} `json:"today"`
	} `json:"recent"`
}

func (w *world) home(t *testing.T, since time.Time) reply {
	t.Helper()
	status, body := w.s.Call(t, http.MethodGet, "/api/v1/home?since="+url.QueryEscape(since.Format(time.RFC3339)), "")
	var out reply
	if err := json.Unmarshal([]byte(body), &out); status != http.StatusOK || err != nil {
		t.Fatalf("GET /home = %d %s, want 200", status, body)
	}
	return out
}

func (r reply) jobs() []string {
	var out []string
	for _, j := range r.Jobs.Rows {
		out = append(out, j.Grab.ID)
	}
	return out
}

func (r reply) needs() map[string]string {
	out := map[string]string{}
	for _, n := range r.NeedsYou.Rows {
		code := "failed"
		if n.Wait != nil {
			code = n.Wait.Code
		}
		out[n.Grab.ID] = code
	}
	return out
}

func ids(gs ...grabs.GrabID) []string {
	out := make([]string, len(gs))
	for i, g := range gs {
		out[i] = g.String()
	}
	return out
}

func TestJobsRunNearestAvailableFirstAndEarliestWithinAStop(t *testing.T) {
	w := newWorld(t)
	attic := w.add(t, contenttypetest.ID, "The Attic", 1)
	fetch := w.grab(t, attic, grabbing{stage: grabs.Fetching, created: 0, entered: 0})
	download := w.grab(t, attic, grabbing{stage: grabs.Downloading, torrent: w.torrent(t, "Attic.Down", nil, nil), created: 1, entered: 1})
	sending := w.grab(t, attic, grabbing{stage: grabs.Importing, torrent: w.torrent(t, "Attic.Send", nil, nil), created: 2, entered: 2})
	w.importing(t, sending, "sending", "", "")
	later := w.grab(t, attic, grabbing{stage: grabs.Importing, torrent: w.torrent(t, "Attic.Late", nil, nil), created: 3, entered: 9})
	earlier := w.grab(t, attic, grabbing{stage: grabs.Importing, torrent: w.torrent(t, "Attic.Early", nil, nil), created: 4, entered: 5})

	got := w.home(t, epoch)
	want := ids(earlier, later, sending, download, fetch)
	if !slices.Equal(got.jobs(), want) || got.Jobs.Moving != 5 {
		t.Fatalf("jobs = %v, %d moving; want %v: importing by the time each entered, then the transfer, downloading and fetching", got.jobs(), got.Jobs.Moving, want)
	}
	if got.Jobs.Rows[0].Torrent == nil || got.Jobs.Rows[4].Torrent != nil || got.Jobs.Rows[0].Work.Label != "The Attic" {
		t.Fatalf("jobs = %+v, want each with its Work and its Torrent, none for the fetch", got.Jobs.Rows)
	}
	if again := w.home(t, epoch); !slices.Equal(again.jobs(), want) {
		t.Fatalf("a second read = %v, want the same order %v", again.jobs(), want)
	}
}

func TestAGrabThatWaitsOnAnAdminIsInNeedsYouAndNotJobs(t *testing.T) {
	w := newWorld(t)
	attic := w.add(t, contenttypetest.ID, "The Attic", 1)
	away := w.grab(t, attic, grabbing{stage: grabs.Downloading, node: w.r.away, torrent: w.torrent(t, "Away", nil, nil), created: 1})
	w.exec(t, `UPDATE downloadclients_client SET poll_state = 'not_responding', poll_problem = '{"cause":"no_response"}' WHERE id = ?`, w.client)
	clientAway := w.grab(t, attic, grabbing{stage: grabs.Downloading, torrent: w.torrent(t, "ClientAway", &w.client, nil), created: 2})
	off := w.grab(t, attic, grabbing{stage: grabs.Fetching, sent: &w.off, created: 3})
	held := map[string]grabs.GrabID{}
	for i, code := range []string{"no_space", "root_not_ready", "probe_timeout", "probe_error"} {
		g := w.grab(t, attic, grabbing{stage: grabs.Importing, torrent: w.torrent(t, code, nil, nil), created: 4 + i})
		w.importing(t, g, "pending", "", `{"code":"`+code+`","attempts":1,"free":10,"needs":20,"since":""}`)
		held[code] = g
	}
	confirm := w.grab(t, attic, grabbing{stage: grabs.Importing, torrent: w.torrent(t, "Confirm", nil, nil), created: 9})
	w.importing(t, confirm, "waiting", `{"code":"confirm"}`, "")
	moving := w.grab(t, attic, grabbing{stage: grabs.Downloading, torrent: w.torrent(t, "Moving", nil, nil), created: 10})
	delayed := w.grab(t, attic, grabbing{stage: grabs.Downloading, created: 11, torrent: w.torrent(t, "Delayed", nil, &dlv1.TorrentReport{
		AtUnixMs: time.Now().UnixMilli(), Engine: dlv1.EngineState_ENGINE_STATE_SEEDING, Complete: true, ImportAtUnixMs: time.Now().Add(time.Hour).UnixMilli(),
	})})

	got := w.home(t, epoch)
	want := map[string]string{
		away.String(): "node_away", clientAway.String(): "client_away", off.String(): "client_off",
		held["no_space"].String(): "no_space", held["root_not_ready"].String(): "root_not_ready",
		held["probe_timeout"].String(): "probe_timeout", held["probe_error"].String(): "probe_error",
	}
	if g := got.needs(); fmt.Sprint(g) != fmt.Sprint(want) || got.NeedsYou.Waiting != 7 || got.NeedsYou.Failed != 0 {
		t.Fatalf("needs you = %v, %d waiting, %d failed; want %v and 7 waiting", g, got.NeedsYou.Waiting, got.NeedsYou.Failed, want)
	}
	if !slices.Equal(got.jobs(), ids(moving, delayed)) || got.Jobs.Moving != 2 {
		t.Fatalf("jobs = %v, %d moving; want the moving Grab and the one in its client's Import delay, and the one whose file waits in neither list", got.jobs(), got.Jobs.Moving)
	}
	if first := got.NeedsYou.Rows[0]; first.Grab.ID != held["probe_error"].String() {
		t.Fatalf("needs you starts with %s, want the newest Grab first", first.Grab.ID)
	}
	for _, n := range got.NeedsYou.Rows {
		if n.Grab.ID == held["no_space"].String() && (n.Wait.ImportWait == nil || n.Wait.ImportWait.RootID != w.root.ID.String()) {
			t.Fatalf("no_space wait = %+v, want it to name its root", n.Wait)
		}
	}
}

func TestAGrabWhoseFileWaitsIsInNeitherListAndPendingImportsCountsIt(t *testing.T) {
	w := newWorld(t)
	attic := w.add(t, contenttypetest.ID, "The Attic", 1)
	confirm := w.grab(t, attic, grabbing{stage: grabs.Importing, torrent: w.torrent(t, "Confirm", nil, nil), created: 1})
	w.importing(t, confirm, "waiting", `{"code":"confirm"}`, "")

	got := w.home(t, epoch)
	if len(got.Jobs.Rows) != 0 || got.Jobs.Moving != 0 || len(got.NeedsYou.Rows) != 0 || got.NeedsYou.Waiting != 0 {
		t.Fatalf("jobs = %v, %d moving; needs you = %v, %d waiting; want the Grab in neither", got.jobs(), got.Jobs.Moving, got.needs(), got.NeedsYou.Waiting)
	}
	if n := must[int](t)(w.found.Count(t.Context())); n != 1 {
		t.Fatalf("pending imports = %d, want 1: the Import whose file waits", n)
	}
}

func TestAFetchThatWaitsOnItsIndexerStaysInJobs(t *testing.T) {
	w := newWorld(t)
	attic := w.add(t, contenttypetest.ID, "The Attic", 1)
	site := indexers.ID{UUID: storage.NewUUID()}
	err := w.lib.Write(t.Context(), func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
		return indexerstest.Torznab(ctx, tx, w.sealer, site, "Quick Site", w.r.self, "https://site.test/api", "key")
	})
	if err != nil {
		t.Fatal(err)
	}
	w.exec(t, `INSERT INTO indexers_status (indexer_id, state, rung, until_at, since_at, solver_tried, revision) VALUES (?, 'failing', 1, ?, 0, 0, 1)`,
		site, storage.FromTime(time.Now().Add(time.Hour)))
	fetch := w.grab(t, attic, grabbing{stage: grabs.Fetching, created: 1})
	w.exec(t, `UPDATE grabs_grab SET indexer_id = ?, release_link_sealed = x'00', release_magnet = NULL WHERE id = ?`, site, fetch)

	status, body := w.s.Call(t, http.MethodGet, "/api/v1/home?since="+url.QueryEscape(epoch.Format(time.RFC3339)), "")
	var got struct {
		Jobs struct {
			Rows []struct {
				Grab struct {
					ID        string `json:"id"`
					FetchWait *struct {
						Sentence string `json:"sentence"`
					} `json:"fetchWait"`
				} `json:"grab"`
			} `json:"rows"`
		} `json:"jobs"`
		NeedsYou struct {
			Waiting int `json:"waiting"`
		} `json:"needsYou"`
	}
	if err := json.Unmarshal([]byte(body), &got); status != http.StatusOK || err != nil {
		t.Fatalf("GET /home = %d %s, want 200", status, body)
	}
	if len(got.Jobs.Rows) != 1 || got.Jobs.Rows[0].Grab.ID != fetch.String() || got.Jobs.Rows[0].Grab.FetchWait == nil || got.NeedsYou.Waiting != 0 {
		t.Fatalf("GET /home = %s, want the fetch in Jobs with Quick Site's wait, and nothing waiting on an Admin", body)
	}
}

func TestADownloadWhoseClientReportsAnErrorWaitsOnIt(t *testing.T) {
	w := newWorld(t)
	attic := w.add(t, contenttypetest.ID, "The Attic", 1)
	erroring := w.torrent(t, "Erroring", &w.client, &dlv1.TorrentReport{
		AtUnixMs: time.Now().UnixMilli(), Engine: dlv1.EngineState_ENGINE_STATE_DOWNLOADING, Hold: dlv1.Hold_HOLD_CLIENT_ERROR, Error: "No space left on device",
	})
	g := w.grab(t, attic, grabbing{stage: grabs.Downloading, torrent: erroring})

	got := w.home(t, epoch)
	if len(got.NeedsYou.Rows) != 1 || got.NeedsYou.Rows[0].Grab.ID != g.String() || got.NeedsYou.Rows[0].Wait.Code != "client_error" ||
		got.NeedsYou.Rows[0].Wait.ClientError == nil || got.NeedsYou.Rows[0].Wait.ClientError.Client != "Transmission" {
		t.Fatalf("needs you = %+v, want the Grab waiting on Transmission's error", got.NeedsYou.Rows)
	}
}

func TestAFailureNeedsYouUntilALaterGrabReplacesIt(t *testing.T) {
	w := newWorld(t)
	attic := w.add(t, contenttypetest.ID, "The Attic", 3)
	failed := w.grab(t, attic, grabbing{item: attic.Items[0].ID, stage: grabs.Failed, failure: grabs.ImportFailed, created: 1})
	w.grab(t, attic, grabbing{item: attic.Items[1].ID, stage: grabs.Failed, failure: grabs.Canceled, created: 2})
	w.grab(t, attic, grabbing{item: attic.Items[2].ID, stage: grabs.Failed, failure: grabs.ImportFailed, created: 3})
	w.grab(t, attic, grabbing{item: attic.Items[2].ID, stage: grabs.Downloading, torrent: w.torrent(t, "Again", nil, nil), created: 4})

	got := w.home(t, epoch)
	if fmt.Sprint(got.needs()) != fmt.Sprint(map[string]string{failed.String(): "failed"}) || got.NeedsYou.Failed != 1 {
		t.Fatalf("needs you = %v, %d failed; want only the failure no later Grab replaced, not the canceled one", got.needs(), got.NeedsYou.Failed)
	}
}

func TestRowsOnHomeCapsEachListAndTheCountsStayFull(t *testing.T) {
	w := newWorld(t)
	crates := w.add(t, contenttypetest.ID, "Crates", 101)
	for i, it := range crates.Items {
		w.grab(t, crates, grabbing{item: it.ID, stage: grabs.Failed, failure: grabs.ImportFailed, created: i})
	}
	attic := w.add(t, contenttypetest.ID, "The Attic", 1)
	for i := range 4 {
		w.grab(t, attic, grabbing{stage: grabs.Downloading, torrent: w.torrent(t, fmt.Sprintf("Attic.%d", i), nil, nil), created: 200 + i})
		w.grab(t, attic, grabbing{stage: grabs.Downloading, node: w.r.away, torrent: w.torrent(t, fmt.Sprintf("Away.%d", i), nil, nil), created: 300 + i})
	}
	for i := range 4 {
		w.landing(t, attic, landed{at: i, items: []works.ItemID{attic.Items[0].ID}})
	}

	status, body := w.s.Call(t, http.MethodPatch, "/api/v1/home-settings", `{"values":{"home.rows":3}}`)
	if status != http.StatusOK || w.saved != 1 {
		t.Fatalf("PATCH /home-settings = %d %s, %d events; want 200 and one home.settings-changed", status, body, w.saved)
	}
	got := w.home(t, epoch)
	if len(got.Jobs.Rows) != 3 || len(got.NeedsYou.Rows) != 3 || len(got.Recent.Rows) != 3 {
		t.Fatalf("rows = %d jobs, %d needs, %d recent; want 3 each", len(got.Jobs.Rows), len(got.NeedsYou.Rows), len(got.Recent.Rows))
	}
	if got.Jobs.Moving != 4 || got.NeedsYou.Waiting != 4 || got.NeedsYou.Failed != 100 {
		t.Fatalf("counts = %d moving, %d waiting, %d failed; want 4, 4 and the failures capped at 100", got.Jobs.Moving, got.NeedsYou.Waiting, got.NeedsYou.Failed)
	}
}

func TestHomeSettingsAnswerRowsOnHomeAndRefuseOutOfRange(t *testing.T) {
	w := newWorld(t)
	want := `{"settings":[{"key":"home.rows","label":"Rows on Home","source":"database","applies":"live","isDefault":true,"pendingRestart":false,` +
		`"kind":"count","value":8,"running":8,"default":8,"min":3,"max":50,"unit":{"one":"entry","many":"entries"}}]}`
	if status, body := w.s.Call(t, http.MethodGet, "/api/v1/home-settings", ""); status != http.StatusOK || body != want+"\n" {
		t.Fatalf("GET /home-settings = %d %s\nwant 200 %s", status, body, want)
	}
	status, body := w.s.Call(t, http.MethodPatch, "/api/v1/home-settings", `{"values":{"home.rows":51}}`)
	var p struct {
		Code string `json:"code"`
	}
	if err := json.Unmarshal([]byte(body), &p); status != http.StatusUnprocessableEntity || err != nil || p.Code != "setting_out_of_range" || w.saved != 0 {
		t.Fatalf("PATCH 51 = %d %s, %d events; want 422 setting_out_of_range and no event", status, body, w.saved)
	}
}

// landed is one landing as a test seeds it.
type landed struct {
	imp      storage.UUID
	at       int
	items    []works.ItemID
	quality  release.Quality
	replaced []release.Quality
}

func (w *world) landing(t *testing.T, work works.Tree, l landed) storage.UUID {
	t.Helper()
	if l.imp.IsZero() {
		l.imp = storage.NewUUID()
	}
	if l.quality == 0 {
		l.quality = release.WEBDL1080p
	}
	replaced := make([]files.Replaced, len(l.replaced))
	for i, q := range l.replaced {
		replaced[i] = files.Replaced{Quality: q, Label: "HD", Items: l.items, Path: "old.mkv"}
	}
	items, err := json.Marshal(l.items)
	if err != nil {
		t.Fatal(err)
	}
	was, err := json.Marshal(replaced)
	if err != nil {
		t.Fatal(err)
	}
	w.exec(t, `INSERT INTO files_imported (id, import_id, work_id, placed_by, items, cells, replaced, quality, at)
		VALUES (?, ?, ?, 'move', ?, '[]', ?, ?, ?)`, storage.NewUUID(), l.imp, work.ID, string(items), string(was), l.quality, at(l.at))
	return l.imp
}

// record seeds Import imp's record, from source, finished unless open.
func (w *world) record(t *testing.T, imp storage.UUID, source string, open bool) {
	t.Helper()
	var finished any = at(0)
	if open {
		finished = nil
	}
	switch source {
	case "folder":
		w.exec(t, `INSERT INTO imports_import (id, source, node_id, folder, content_type, hint, created_at, finished_at)
			VALUES (?, 'folder', ?, '/downloads/Attic', 'fixture', '{}', 0, ?)`, imp, w.r.self, finished)
	case "root":
		w.exec(t, `INSERT INTO imports_import (id, source, root_id, folder, content_type, hint, created_at, finished_at)
			VALUES (?, 'root', ?, 'Attic', 'fixture', '{}', 0, ?)`, imp, w.root.ID, finished)
	default:
		w.exec(t, `INSERT INTO imports_import (id, source, node_id, info_hash, content_type, hint, created_at, finished_at)
			VALUES (?, 'grab', ?, randomblob(20), 'fixture', '{}', 0, ?)`, imp, w.r.self, finished)
	}
}

func TestRecentListsFinishedImportsNewestFirstInWords(t *testing.T) {
	w := newWorld(t)
	attic := w.add(t, contenttypetest.ID, "The Attic", 10)
	lantern := w.add(t, filmsID, "Lantern Fall", 0)
	gone := w.add(t, filmsID, "Gone Film", 0)
	item := func(i int) []works.ItemID { return []works.ItemID{attic.Items[i].ID} }

	pack := storage.NewUUID()
	for i := range 10 {
		var was []release.Quality
		if i < 2 {
			was = []release.Quality{release.HDTV720p}
		}
		w.landing(t, attic, landed{imp: pack, at: 10 + i, items: item(i), replaced: was})
	}
	w.record(t, pack, "grab", false)
	upgrade := storage.NewUUID()
	w.landing(t, attic, landed{imp: upgrade, at: 30, items: item(0), quality: release.Remux2160p, replaced: []release.Quality{release.WEBDL1080p}})
	w.landing(t, attic, landed{imp: upgrade, at: 31, items: item(1), quality: release.Remux2160p, replaced: []release.Quality{release.WEBDL1080p}})
	film := w.landing(t, lantern, landed{at: 40, items: []works.ItemID{lantern.Items[0].ID}, quality: release.Remux2160p})
	folder := w.landing(t, attic, landed{at: 50, items: item(3)})
	w.record(t, folder, "folder", false)
	root := w.landing(t, attic, landed{at: 60, items: item(4)})
	w.record(t, root, "root", false)
	running := w.landing(t, attic, landed{at: 70, items: item(5)})
	w.record(t, running, "grab", true)
	w.landing(t, gone, landed{at: 80, items: []works.ItemID{gone.Items[0].ID}})
	w.exec(t, `DELETE FROM works_work WHERE id = ?`, gone.ID)

	got := w.home(t, epoch.Add(15*time.Minute))
	type row struct{ id, label, words string }
	var rows []row
	for _, r := range got.Recent.Rows {
		rows = append(rows, row{r.ImportID, r.Work.Label, r.Words})
	}
	want := []row{
		{root.String(), "The Attic", "Imported 1 thing"},
		{folder.String(), "The Attic", "Imported 1 thing"},
		{film.String(), "Lantern Fall", "Imported in Remux-2160p"},
		{upgrade.String(), "The Attic", "Upgraded to Remux-2160p from WEBDL-1080p"},
		{pack.String(), "The Attic", "Imported 10 things · 2 upgrades"},
	}
	if fmt.Sprint(rows) != fmt.Sprint(want) {
		t.Fatalf("recent = %v\nwant %v: finished Imports only, newest first, none of a deleted Work", rows, want)
	}
	if got.Recent.Today.Imported != 4 || got.Recent.Today.Upgraded != 1 {
		t.Fatalf("today = %+v, want 4 imported (the pack straddling since among them) and 1 upgraded", got.Recent.Today)
	}
}

type noMoves struct{}

func (noMoves) NeedsYou(context.Context, storage.Reader[storage.Library], int) ([]moves.Need, error) {
	return nil, nil
}
