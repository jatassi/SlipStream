package conversionsapi_test

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"
	"testing"
	"time"

	"github.com/jatassi/slipstream/internal/api"
	"github.com/jatassi/slipstream/internal/api/ownertest"
	"github.com/jatassi/slipstream/internal/conversions"
	"github.com/jatassi/slipstream/internal/conversionsapi"
	"github.com/jatassi/slipstream/internal/converting"
	"github.com/jatassi/slipstream/internal/downloadclients"
	"github.com/jatassi/slipstream/internal/files"
	"github.com/jatassi/slipstream/internal/grabs"
	"github.com/jatassi/slipstream/internal/history/entry/entrytest"
	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/indexers"
	"github.com/jatassi/slipstream/internal/mediaservers"
	"github.com/jatassi/slipstream/internal/nodes"
	"github.com/jatassi/slipstream/internal/nodes/nodestest"
	cv1 "github.com/jatassi/slipstream/internal/pb/slipstream/converting/v1"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/roots"
	"github.com/jatassi/slipstream/internal/secret"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/storage/storagetest"
	"github.com/jatassi/slipstream/internal/targets"
	"github.com/jatassi/slipstream/internal/transfers"
	"github.com/jatassi/slipstream/internal/works"
)

type owner struct {
	c *conversions.Conversions
	n *nodes.Service
}

func (o owner) Conversions() (*conversions.Conversions, bool) { return o.c, true }
func (o owner) Nodes() (*nodes.Service, bool)                 { return o.n, true }

// sources is Files: no source file is ever found.
type sources struct{}

func (sources) Sourced(context.Context, storage.Reader[storage.Library], works.Among[works.WorkID]) (map[works.WorkID][]files.Sourced, error) {
	return nil, nil
}

func (sources) Located(context.Context, storage.Reader[storage.Library], files.ID) (files.Located, error) {
	return files.Located{}, files.ErrNoFile
}

type copies struct{}

func (copies) Want(context.Context, *storage.Tx[storage.Library], transfers.Request) (transfers.ID, error) {
	return transfers.ID{}, nil
}

func (copies) Touch() {}

// placing is the Importer: every Import's file is placing.
type placing struct{}

func (placing) Start(context.Context, *storage.Tx[storage.Library], conversions.Handoff) error {
	return nil
}
func (placing) End(context.Context, *storage.Tx[storage.Library], conversions.ID) error {
	return conversions.ErrPlacing
}
func (placing) Wake() {}

type discard[T any] struct{}

func (discard[T]) Publish(context.Context, T) {}

type world struct {
	*ownertest.Owner
	lib  *storage.DB[storage.Library]
	work works.WorkID
	self role.NodeID
	n    int
}

func newWorld(t *testing.T) *world {
	t.Helper()
	ctx := t.Context()
	lib := storagetest.Library(t, secret.Schema(), identity.Schema(), nodes.Schema(), role.LibrarySchema(), works.Schema(), targets.Schema(), roots.Schema(),
		downloadclients.Schema(), grabs.Schema(), indexers.Schema(), files.Schema(), transfers.LibrarySchema(), conversions.Schema())
	runs := converting.Def.Serve(
		func(context.Context, storage.Reader[storage.Library], role.NodeID) (*cv1.Intent, error) {
			return &cv1.Intent{}, nil
		},
		func(context.Context, *storage.Tx[storage.Library], role.NodeID, *cv1.Result) error { return nil },
	)
	roster, err := nodes.Open(ctx, nodes.Config{Library: lib, Name: "home", Owners: nodestest.Unowned(), Served: []role.Served{runs}, Log: slog.New(slog.DiscardHandler)})
	if err != nil {
		t.Fatal(err)
	}
	rec, _ := entrytest.New("conversions", conversions.Entries())
	now := time.Date(2026, 11, 28, 22, 0, 0, 0, time.UTC)
	c := conversions.New(conversions.Config{
		Library: lib, Files: sources{}, Transfers: copies{}, History: rec, Changed: discard[conversions.Changed]{}, Poke: conversions.NewPoke(),
		Board:   roster.Board,
		Control: func() *role.Control[*cv1.Intent, *cv1.Facts, *cv1.Result] { return runs },
		RootRoom: func(context.Context, storage.Reader[storage.Library], roots.ID) (int64, bool, error) {
			return 1 << 50, true, nil
		},
		TranscodingOn: func(context.Context, storage.Reader[storage.Library], role.NodeID) ([]mediaservers.Transcoding, error) {
			return nil, nil
		},
		Reconciling: func(context.Context, storage.Reader[storage.Library]) (bool, error) { return false, nil },
		Zone:        func() *time.Location { return time.UTC }, Timings: conversions.DefaultTimings(), Run: converting.DefaultTimings(),
		Clock: func() time.Time { return now }, Log: slog.New(slog.DiscardHandler),
	})
	c.Register(placing{})
	w := &world{lib: lib, work: works.WorkID{UUID: storage.NewUUID()}, self: roster.Self()}
	w.write(t, `INSERT INTO works_work (id, content_type, key, label, sort_label, added_at, refreshed_at) VALUES (?, 'movie', 'heat', 'Heat', 'Heat', 0, 0)`, w.work)
	w.Owner = ownertest.Serve(t, lib, func(r *api.Router) { conversionsapi.Routes(r, owner{c, roster}) })
	return w
}

func (w *world) write(t *testing.T, query string, args ...any) {
	t.Helper()
	if err := w.lib.Write(t.Context(), func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
		_, err := tx.ExecContext(ctx, query, args...)
		return err
	}); err != nil {
		t.Fatal(err)
	}
}

// seed is a Conversion of Heat at stage, with no source file or Target,
// in backfill when it isn't zero.
func (w *world) seed(t *testing.T, stage string, backfill storage.UUID) string {
	t.Helper()
	w.n++
	id := storage.NewUUID()
	var node, finished, code, class, detail, why, bf any
	switch stage {
	case "landing":
		node = w.self
	case "done":
		finished = 0
	case "failed":
		code, class, detail = "encoder_failed", "run", "the encoder exited"
	case "canceled":
		finished, why = 0, "admin"
	}
	lane := "new"
	if !backfill.IsZero() {
		bf, lane = backfill, "backfill"
	}
	w.write(t, `INSERT INTO conversions_conversion (id, work_id, type, items, edition, version, lane, position, backfill_id, stage, node_id,
		not_before, failure_code, failure_class, failure_detail, cancel_why, snapshot, queued_at, finished_at)
		VALUES (?, ?, 'movie', '[]', 'any', 1, ?, ?, ?, ?, ?, 0, ?, ?, ?, ?, '{"work":"Heat","target":"720p mobile","sourceName":"Heat.mkv"}', 0, ?)`,
		id, w.work, lane, strconv.Itoa(w.n), bf, stage, node, code, class, detail, why, finished)
	return id.String()
}

type row struct {
	ID    string `json:"id"`
	Stage string `json:"stage"`
	Lane  string `json:"lane"`
	Work  struct {
		Title string `json:"title"`
	} `json:"work"`
	Target struct {
		Name string `json:"name"`
	} `json:"target"`
	Wait *struct {
		Code string `json:"code"`
	} `json:"wait"`
	Failure *struct {
		Code  string `json:"code"`
		Class string `json:"class"`
	} `json:"failure"`
}

type entry struct {
	row
	Kind      string `json:"kind"`
	GroupKind string `json:"groupKind"`
	Count     int    `json:"count"`
	Label     string `json:"label"`
}

type list struct {
	Counts struct {
		Converting, Paused, Waiting, Failed int
	} `json:"counts"`
	Attention []row `json:"attention"`
	Next      []struct {
		Lane string  `json:"lane"`
		Rows []entry `json:"rows"`
	} `json:"next"`
}

func decode[T any](t *testing.T, status int, reply string) T {
	t.Helper()
	if status != http.StatusOK {
		t.Fatalf("status %d: %s", status, reply)
	}
	var v T
	if err := json.Unmarshal([]byte(reply), &v); err != nil {
		t.Fatal(err)
	}
	return v
}

func problemCode(t *testing.T, reply string) string {
	t.Helper()
	var p struct{ Code string }
	if err := json.Unmarshal([]byte(reply), &p); err != nil {
		t.Fatal(err)
	}
	return p.Code
}

// The tab lists a failed Conversion in Needs attention and the rest in
// Up next, a backfill's rows folded into one group, and every count.
func TestTheTabBandsFoldsAndCounts(t *testing.T) {
	w := newWorld(t)
	failed := w.seed(t, "failed", storage.UUID{})
	queued := w.seed(t, "queued", storage.UUID{})
	backfill := storage.NewUUID()
	w.write(t, `INSERT INTO conversions_backfill (id, kind, created_at, count) VALUES (?, 'backfill', 0, 2)`, backfill)
	w.seed(t, "queued", backfill)
	w.seed(t, "queued", backfill)
	w.seed(t, "done", storage.UUID{})

	status, reply := w.Call(t, http.MethodGet, "/api/v1/conversions", "")
	l := decode[list](t, status, reply)
	if l.Counts.Failed != 1 || l.Counts.Waiting != 3 || l.Counts.Converting != 0 || l.Counts.Paused != 0 {
		t.Errorf("counts = %+v, want 1 failed and 3 waiting", l.Counts)
	}
	if len(l.Attention) != 1 || l.Attention[0].ID != failed || l.Attention[0].Failure == nil || l.Attention[0].Failure.Code != "encoder_failed" || l.Attention[0].Failure.Class != "run" {
		t.Errorf("attention = %+v, want the failed Conversion with its fault", l.Attention)
	}
	if len(l.Next) != 2 || l.Next[0].Lane != "new" || l.Next[1].Lane != "backfill" {
		t.Fatalf("next = %+v, want the new lane then the backfill lane", l.Next)
	}
	if r := l.Next[0].Rows; len(r) != 1 || r[0].Kind != "conversion" || r[0].ID != queued || r[0].Work.Title != "Heat" || r[0].Wait == nil || r[0].Wait.Code != "no_source" {
		t.Errorf("the new lane = %+v, want the queued Conversion waiting for its source", r)
	}
	g := l.Next[1].Rows
	if len(g) != 1 || g[0].Kind != "group" || g[0].GroupKind != "backfill" || g[0].Count != 2 || g[0].Label != "Backfill of 720p mobile" {
		t.Fatalf("the backfill lane = %+v, want one group of 2", g)
	}

	status, reply = w.Call(t, http.MethodGet, "/api/v1/conversions/groups/"+g[0].ID, "")
	if page := decode[struct{ Rows []row }](t, status, reply); len(page.Rows) != 2 {
		t.Errorf("the group's page = %s, want its 2 rows", reply)
	}
	status, reply = w.Call(t, http.MethodGet, "/api/v1/conversions?band=attention", "")
	if l := decode[list](t, status, reply); len(l.Attention) != 1 || len(l.Next) != 0 {
		t.Errorf("band=attention = %s, want only Needs attention", reply)
	}
	status, reply = w.Call(t, http.MethodPost, "/api/v1/conversions/groups/"+g[0].ID+"/cancel", "")
	if n := decode[struct{ Canceled int }](t, status, reply); n.Canceled != 2 {
		t.Errorf("canceling the group = %s, want 2 canceled", reply)
	}
	status, reply = w.Call(t, http.MethodPost, "/api/v1/conversions/groups/"+g[0].ID+"/cancel", "")
	if status != http.StatusNotFound || problemCode(t, reply) != "conversion_not_found" {
		t.Errorf("canceling the canceled group = %d %s, want 404 conversion_not_found", status, reply)
	}
}

// Each action answers the row as it stands after, or refuses with the
// code of the stage it found.
func TestEachActionAnswersTheRowOrItsCode(t *testing.T) {
	w := newWorld(t)
	for _, c := range []struct {
		stage, action string
		status        int
		code, want    string
	}{
		{"queued", "move-up", http.StatusOK, "", "moved_up"},
		{"failed", "move-up", http.StatusConflict, "conversion_not_waiting", ""},
		{"done", "move-up", http.StatusConflict, "conversion_finished", ""},
		{"queued", "retry", http.StatusConflict, "conversion_not_failed", ""},
		{"failed", "retry", http.StatusOK, "", "queued"},
		{"queued", "cancel", http.StatusOK, "", "canceled"},
		{"failed", "cancel", http.StatusOK, "", "canceled"},
		{"done", "cancel", http.StatusConflict, "conversion_finished", ""},
		{"canceled", "cancel", http.StatusConflict, "conversion_finished", ""},
		{"landing", "cancel", http.StatusConflict, "conversion_placing", ""},
	} {
		id := w.seed(t, c.stage, storage.UUID{})
		status, reply := w.Call(t, http.MethodPost, "/api/v1/conversions/"+id+"/"+c.action, "")
		if status != c.status {
			t.Errorf("%s of a %s Conversion = %d %s, want %d", c.action, c.stage, status, reply, c.status)
			continue
		}
		if c.code != "" {
			if got := problemCode(t, reply); got != c.code {
				t.Errorf("%s of a %s Conversion answered %s, want %s", c.action, c.stage, got, c.code)
			}
			continue
		}
		r := decode[row](t, status, reply)
		if got := map[bool]string{true: r.Lane, false: r.Stage}[c.action == "move-up"]; r.ID != id || got != c.want {
			t.Errorf("%s of a %s Conversion answered %+v, want %s", c.action, c.stage, r, c.want)
		}
	}
	none := storage.NewUUID().String()
	for _, path := range []string{"/move-up", "/retry", "/cancel"} {
		if status, reply := w.Call(t, http.MethodPost, "/api/v1/conversions/"+none+path, ""); status != http.StatusNotFound || problemCode(t, reply) != "conversion_not_found" {
			t.Errorf("%s of no Conversion = %d %s, want 404 conversion_not_found", path, status, reply)
		}
	}
	after := w.seed(t, "queued", storage.UUID{})
	moved := w.seed(t, "queued", storage.UUID{})
	status, reply := w.Call(t, http.MethodPost, "/api/v1/conversions/"+moved+"/move", `{"after":"`+after+`"}`)
	if r := decode[row](t, status, reply); r.Lane != "new" {
		t.Errorf("moving after a new Conversion answered %+v, want the new lane", r)
	}
}

// A Node's Converting section starts at its defaults and keeps what an
// Admin saves; a Node no one has is 404.
func TestANodesConvertingSectionSavesItsSettings(t *testing.T) {
	w := newWorld(t)
	path := "/api/v1/nodes/" + w.self.String() + "/converting"
	defaults := `{"atOnce":1,"atOnceDefault":1,"hours":{"on":false,"window":{"start":"01:00","end":"07:00"}},"processor":true,` +
		`"sentence":{"api":null,"processor":false,"none":true},"proofs":[],"scratch":null,"override":false,"running":0}`
	w.Expect(t, http.MethodGet, path, "", http.StatusOK, defaults)
	w.Expect(t, http.MethodPut, path, `{"atOnce":3,"hours":{"on":true,"window":{"start":"22:00","end":"04:30"}},"processor":false}`, http.StatusOK,
		`{"atOnce":3,"atOnceDefault":1,"hours":{"on":true,"window":{"start":"22:00","end":"04:30"}},"processor":false,`+
			`"sentence":{"api":null,"processor":false,"none":true},"proofs":[],"scratch":null,"override":false,"running":0}`)
	if status, reply := w.Call(t, http.MethodPut, path, `{"atOnce":null,"hours":{"on":true,"window":{"start":"04:00","end":"04:00"}},"processor":true}`); status != http.StatusUnprocessableEntity {
		t.Errorf("a window that starts and ends at once = %d %s, want 422", status, reply)
	}
	if status, reply := w.Call(t, http.MethodPost, path+"/rerun", ""); status != http.StatusAccepted {
		t.Errorf("rerun = %d %s, want 202", status, reply)
	}
	none := "/api/v1/nodes/" + storage.NewUUID().String() + "/converting"
	for _, c := range []struct{ method, path, body string }{
		{http.MethodGet, none, ""},
		{http.MethodPut, none, `{"atOnce":null,"hours":{"on":false,"window":{"start":"01:00","end":"07:00"}},"processor":true}`},
		{http.MethodPost, none + "/rerun", ""},
	} {
		if status, reply := w.Call(t, c.method, c.path, c.body); status != http.StatusNotFound || problemCode(t, reply) != "node_not_found" {
			t.Errorf("%s %s = %d %s, want 404 node_not_found", c.method, c.path, status, reply)
		}
	}
}
