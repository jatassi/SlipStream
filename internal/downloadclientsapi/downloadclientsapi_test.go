package downloadclientsapi_test

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"slices"
	"testing"
	"time"

	"github.com/jatassi/slipstream/internal/api"
	"github.com/jatassi/slipstream/internal/api/ownertest"
	"github.com/jatassi/slipstream/internal/downloadclients"
	"github.com/jatassi/slipstream/internal/downloadclientsapi"
	"github.com/jatassi/slipstream/internal/health"
	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/indexers"
	"github.com/jatassi/slipstream/internal/link"
	"github.com/jatassi/slipstream/internal/nodes"
	"github.com/jatassi/slipstream/internal/nodes/nodestest"
	dlv1 "github.com/jatassi/slipstream/internal/pb/slipstream/downloading/v1"
	"github.com/jatassi/slipstream/internal/pb/slipstream/downloading/v1/downloadingv1connect"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/roots"
	"github.com/jatassi/slipstream/internal/secret"
	"github.com/jatassi/slipstream/internal/secret/secrettest"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/storage/storagetest"
	"github.com/jatassi/slipstream/internal/trackers"
	"github.com/jatassi/slipstream/internal/works"
)

type unlinked struct{}

func (unlinked) State(link.Key) link.State                    { return link.Down{} }
func (unlinked) Subscribe(context.Context) <-chan link.Change { return nil }
func (unlinked) Advertised() (link.Addresses, bool)           { return link.Addresses{}, false }
func (unlinked) Reset(link.Key)                               {}
func (unlinked) Keep(link.Target)                             {}
func (unlinked) Drop(link.Key)                                {}
func (unlinked) Invite(context.Context, link.Target) error    { return nil }
func (unlinked) Stats(link.Key) (link.Stats, bool)            { return link.Stats{}, false }
func (unlinked) LaneBytes(link.Key) link.Stats                { return link.Stats{} }

type changes struct{}

func (changes) Publish(context.Context, nodes.Changed) {}

type controller struct {
	clients  *downloadclients.Clients
	roster   *nodes.Service
	trackers *trackers.Trackers
}

func (c controller) DownloadClients() (*downloadclients.Clients, bool) { return c.clients, true }
func (c controller) Nodes() (*nodes.Service, bool)                     { return c.roster, true }
func (c controller) Trackers() (*trackers.Trackers, bool)              { return c.trackers, true }

// downloaders is the roster as downloadclients reads it: every Node
// serving Downloading, each in step unless away.
type downloaders struct {
	serving []role.NodeID
	away    map[role.NodeID]bool
}

func (d *downloaders) InStep(n role.NodeID) bool              { return !d.away[n] }
func (d *downloaders) WrittenOff(role.Name, role.NodeID) bool { return false }
func (d *downloaders) Changed() <-chan struct{}               { return nil }
func (d *downloaders) Draining(role.Name, role.NodeID) bool   { return false }
func (d *downloaders) Holds(name role.Name, n role.NodeID) bool {
	return slices.Contains(d.Serving(name), n)
}

func (d *downloaders) Holding(name role.Name) []role.NodeID { return d.Serving(name) }

func (d *downloaders) Serving(name role.Name) []role.NodeID {
	if name != role.Downloading {
		return nil
	}
	return d.serving
}

// uses is the Grabs of each Torrent as a test sets them.
type uses map[downloadclients.ID][]downloadclients.Use

func (u uses) Of(_ context.Context, _ storage.Reader[storage.Library], ids []downloadclients.ID) (map[downloadclients.ID][]downloadclients.Use, error) {
	out := map[downloadclients.ID][]downloadclients.Use{}
	for _, id := range ids {
		out[id] = u[id]
	}
	return out, nil
}

// heard is what the Clients told: the Nodes whose intents they touched
// and the download-clients.changed events.
type heard struct {
	touched []role.NodeID
	cards   [][]string
}

func (h *heard) Publish(_ context.Context, c downloadclients.ClientsChanged) {
	h.cards = append(h.cards, c.NodeIDs)
}

// blocklist records who each mark-failed's entries name, by name or
// "the API key".
type blocklist struct{ by []string }

func (b *blocklist) MarkedFailed(_ context.Context, _ *storage.Tx[storage.Library], _ []downloadclients.ID, by identity.Admin) error {
	name := "the API key"
	if a, ok := by.Account(); ok {
		name = a.Profile().DisplayName
	}
	b.by = append(b.by, name)
	return nil
}

func (*blocklist) Changed(context.Context) {}

type torrentsChanged struct{}

func (torrentsChanged) Publish(context.Context, downloadclients.Changed) {}

// world is a Controller whose only Node is itself, serving Downloading
// with a client that answers Inspect, Act and Frees as the test sets it.
type world struct {
	o         *ownertest.Owner
	lib       *storage.DB[storage.Library]
	self      role.NodeID
	roster    *downloaders
	uses      uses
	node      *node
	facts     map[role.NodeID]*dlv1.Facts
	heard     *heard
	hashes    map[downloadclients.ID]string
	trackers  *trackers.Trackers
	blocklist *blocklist
}

// late is the Trackers' seeder once they are made, after the clients.
type late struct{ x **trackers.Trackers }

func (l late) Judge(ctx context.Context, r storage.Reader[storage.Library], rows []downloadclients.Row) (map[downloadclients.ID]downloadclients.Judged, error) {
	return (*l.x).Seeder().Judge(ctx, r, rows)
}

func (l late) EndlessUnits(ctx context.Context, r storage.Reader[storage.Library], node role.NodeID) (int64, error) {
	return (*l.x).Seeder().EndlessUnits(ctx, r, node)
}

func serveWorld(t *testing.T) *world {
	t.Helper()
	lib := storagetest.Library(t, secret.Schema(), identity.Schema(), nodes.Schema(), role.LibrarySchema(), works.Schema(), health.Schema(), roots.Schema(), downloadclients.Schema(), indexers.Schema(), trackers.Schema())
	side, err := nodes.Open(t.Context(), nodes.Config{Library: lib, Name: "home", Link: unlinked{}, Changed: changes{}, Owners: nodestest.Unowned(), Log: slog.New(slog.DiscardHandler)})
	if err != nil {
		t.Fatal(err)
	}
	sealer := secrettest.Sealer(t)
	w := &world{lib: lib, self: side.Self(), roster: &downloaders{serving: []role.NodeID{side.Self()}, away: map[role.NodeID]bool{}}, uses: uses{}, node: &node{}, facts: map[role.NodeID]*dlv1.Facts{}, heard: &heard{}, hashes: map[downloadclients.ID]string{}, blocklist: &blocklist{}}
	clients := downloadclients.New(downloadclients.Config{
		Library: lib, Touch: func(n ...role.NodeID) { w.heard.touched = append(w.heard.touched, n...) }, Changed: torrentsChanged{},
		CardsChanged: w.heard, Rerouted: func() {}, Roster: w.roster, Uses: w.uses, Blocklist: w.blocklist, Seeding: late{&w.trackers},
		Facts:  func(n role.NodeID) (*dlv1.Facts, bool) { f, ok := w.facts[n]; return f, ok },
		Remote: func(n role.NodeID) downloadingv1connect.DownloadingServiceHandler { return calledOn{w.node, n} },
		Sealer: sealer,
		Live:   downloadclients.LiveConfig{Sample: time.Second},
		Log:    slog.New(slog.DiscardHandler),
	})
	w.trackers = trackers.New(trackers.Config{Library: lib, Clients: clients, Roster: w.roster, Reconciling: trackers.NotReconciling, Routing: trackers.Routing{Roster: w.roster}})
	w.o = ownertest.Serve(t, lib, func(r *api.Router) { downloadclientsapi.Routes(r, controller{clients, side, w.trackers}) })
	return w
}

// serve is the operations on a Controller whose only Node is itself, and
// the Nodes whose intents they touched.
func serve(t *testing.T) (o *ownertest.Owner, self role.NodeID, touched *[]role.NodeID) {
	t.Helper()
	w := serveWorld(t)
	return w.o, w.self, &w.heard.touched
}

type field[T any] struct {
	Value     T    `json:"value"`
	IsDefault bool `json:"isDefault"`
}

type settings struct {
	UploadLimit field[int64] `json:"uploadLimit"`
	Connections field[int64] `json:"connections"`
	UTP         field[bool]  `json:"utp"`
	DHT         field[bool]  `json:"dht"`
}

func call(t *testing.T, o *ownertest.Owner, method, path, body string) settings {
	t.Helper()
	status, reply := o.Call(t, method, path, body)
	var s settings
	if err := json.Unmarshal([]byte(reply), &s); status != http.StatusOK || err != nil {
		t.Fatalf("%s %s %s = %d %s, want 200 and the settings", method, path, body, status, reply)
	}
	return s
}

func TestANodesClientSettingsChangeAndReachItsIntent(t *testing.T) {
	o, home, touched := serve(t)
	path := "/api/v1/download-clients/" + home.String() + "/settings"

	if got := call(t, o, http.MethodGet, path, ""); !got.UTP.Value || !got.UTP.IsDefault || !got.DHT.Value || !got.DHT.IsDefault || got.Connections.Value != 8000 || got.UploadLimit.Value != 0 {
		t.Fatalf("a Node's settings before any change = %+v, want the defaults", got)
	}
	got := call(t, o, http.MethodPatch, path, `{"utp":false,"dht":false,"uploadLimit":524288}`)
	if got.UTP.Value || got.UTP.IsDefault || got.DHT.Value || got.DHT.IsDefault || got.UploadLimit.Value != 524288 || got.Connections.Value != 8000 {
		t.Fatalf("after setting uTP and the DHT off and a 512 KB/s upload limit = %+v", got)
	}
	if again := call(t, o, http.MethodGet, path, ""); again != got {
		t.Fatalf("read back = %+v, want %+v", again, got)
	}
	if !slices.Equal(*touched, []role.NodeID{home}) {
		t.Fatalf("touched %v, want the Node's intent once", *touched)
	}
	if got := call(t, o, http.MethodPatch, path, `{"utp":null}`); !got.UTP.Value || !got.UTP.IsDefault || got.UploadLimit.Value != 524288 {
		t.Fatalf("after resetting uTP = %+v, want it on by default and the upload limit kept", got)
	}
}

func TestClientSettingsRefuse(t *testing.T) {
	o, home, touched := serve(t)
	o.Expect(t, http.MethodPatch, "/api/v1/download-clients/"+home.String()+"/settings", `{"connections":10}`, http.StatusUnprocessableEntity,
		`{"status":422,"code":"setting_out_of_range","title":"The value is outside the setting's range","detail":"Between 50 and 65535 connections."}`)
	o.Expect(t, http.MethodGet, "/api/v1/download-clients/0b0e3b9c-7c0a-4c8e-9a51-2f7d1e7f9a10/settings", "", http.StatusNotFound,
		`{"status":404,"code":"node_not_found","title":"No Node has this id"}`)
	if len(*touched) != 0 {
		t.Fatalf("refusals touched %v, want no intent", *touched)
	}
}
