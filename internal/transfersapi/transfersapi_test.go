package transfersapi_test

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"sync"
	"testing"
	"time"

	"github.com/jatassi/slipstream/internal/api"
	"github.com/jatassi/slipstream/internal/api/ownertest"
	"github.com/jatassi/slipstream/internal/bencode"
	"github.com/jatassi/slipstream/internal/health"
	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/link"
	"github.com/jatassi/slipstream/internal/live"
	"github.com/jatassi/slipstream/internal/nodes"
	"github.com/jatassi/slipstream/internal/nodes/nodestest"
	tv1 "github.com/jatassi/slipstream/internal/pb/slipstream/transfer/v1"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/roots"
	"github.com/jatassi/slipstream/internal/secret"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/storage/storagetest"
	"github.com/jatassi/slipstream/internal/storing"
	"github.com/jatassi/slipstream/internal/transfers"
	"github.com/jatassi/slipstream/internal/transfers/source"
	"github.com/jatassi/slipstream/internal/transfersapi"
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

type quiet struct{}

func (quiet) Publish(context.Context, transfers.Changed) {}

type controller struct {
	tr     *transfers.Transfers
	roster *nodes.Service
}

func (c controller) Transfers() (*transfers.Transfers, bool) { return c.tr, true }
func (c controller) Nodes() (*nodes.Service, bool)           { return c.roster, true }

type board map[role.NodeID]transfers.End

func (b board) End(node role.NodeID) (transfers.End, bool) {
	e, ok := b[node]
	return e, ok
}

func (board) Changed() <-chan struct{} { return nil }

// world is a Controller, home, with a library root, and a Worker,
// seedbox, that sends into it. The test sets what home's receiver
// reports.
type world struct {
	o             *ownertest.Owner
	lib           *storage.DB[storage.Library]
	tr            *transfers.Transfers
	home, seedbox role.NodeID
	root          roots.ID

	mu    sync.Mutex
	facts *tv1.Facts
}

func serve(t *testing.T) *world {
	t.Helper()
	lib := storagetest.Library(t, secret.Schema(), identity.Schema(), nodes.Schema(), role.LibrarySchema(), works.Schema(), health.Schema(), roots.Schema(), transfers.LibrarySchema())
	side, err := nodes.Open(t.Context(), nodes.Config{Library: lib, Name: "home", Link: unlinked{}, Changed: changes{}, Owners: nodestest.Unowned(), Log: slog.New(slog.DiscardHandler)})
	if err != nil {
		t.Fatal(err)
	}
	w := &world{lib: lib, home: side.Self(), seedbox: role.NodeID{UUID: storage.NewUUID()}, root: roots.ID{UUID: storage.NewUUID()}}
	w.exec(t, "INSERT INTO nodes_node (id, public_key, name, paired_at) VALUES (?, ?, 'seedbox', 0)", w.seedbox, bytes.Repeat([]byte{1}, 32))
	w.exec(t, "INSERT INTO roots_root (id, content_type, node_id, path, is_default) VALUES (?, 'alpha', ?, '/media', 1)", w.root, w.home)
	volume := storing.Volume{ID: "v1", Name: "Media", Mount: "/media", Total: 4 << 40, Free: 2 << 40, SampledAt: time.Now()}
	w.tr = transfers.New(transfers.ControllerConfig{
		Library: lib,
		Touch:   func(...role.NodeID) {},
		Facts: func(id role.NodeID) (*tv1.Facts, bool) {
			w.mu.Lock()
			defer w.mu.Unlock()
			return w.facts, id == w.home && w.facts != nil
		},
		Board: func() transfers.Board { return board{w.home: {Controller: true}, w.seedbox: {}} },
		Destinations: func(ctx context.Context, rd storage.Reader[storage.Library], _ []roots.ID) (map[roots.ID]roots.Destination, error) {
			all, err := roots.All(ctx, rd)
			out := map[roots.ID]roots.Destination{}
			for _, root := range all {
				out[root.ID] = roots.Destination{Root: root, State: roots.Ready, Volume: &volume}
			}
			return out, err
		},
		RootsMoved: func() <-chan struct{} { return nil },
		Readiness: func(_ context.Context, _ storage.Reader[storage.Library], _ role.NodeID, refs []source.Ref) (map[source.Ref]source.Readiness, error) {
			out := map[source.Ref]source.Readiness{}
			for _, ref := range refs {
				out[ref] = source.Readiness{State: source.Ready}
			}
			return out, nil
		},
		Moved:   func() <-chan struct{} { return nil },
		Sample:  func(context.Context, role.NodeID) error { return nil },
		Hub:     live.NewHub([]live.Event{transfers.ChangedEvent}),
		Changed: quiet{},
		Timings: transfers.DefaultTimings(),
		Log:     slog.New(slog.DiscardHandler),
	})
	w.tr.Register(transfers.Purpose{
		Kind:     transfers.Grab,
		Verified: func(context.Context, *storage.Tx[storage.Library], transfers.Transfer, string) error { return nil },
		Failed: func(context.Context, *storage.Tx[storage.Library], transfers.Transfer, transfers.Failure) error {
			return nil
		},
		After: func([]string) {},
		Names: func(_ context.Context, _ storage.Reader[storage.Library], jobs []string) (map[string]string, error) {
			out := map[string]string{}
			for _, j := range jobs {
				out[j] = map[string]string{"heat": "Heat (1995)", "ronin": "Ronin (1998)"}[j]
			}
			return out, nil
		},
	})
	w.o = ownertest.Serve(t, lib, func(r *api.Router) { transfersapi.Routes(r, controller{w.tr, side}) })
	return w
}

func (w *world) exec(t *testing.T, query string, args ...any) {
	t.Helper()
	if err := w.lib.Write(t.Context(), func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
		_, err := tx.ExecContext(ctx, query, args...)
		return err
	}); err != nil {
		t.Fatal(err)
	}
}

// start asks for job's Transfer from seedbox into home's root, one file
// of size from torrent hash for each size.
func (w *world) start(t *testing.T, job string, hash bencode.InfoHash, sizes ...int64) transfers.ID {
	t.Helper()
	req := transfers.Request{Purpose: transfers.Grab, Job: job, Source: w.seedbox, Destination: transfers.Root{ID: w.root}}
	for i, size := range sizes {
		req.Files = append(req.Files, transfers.FileRequest{Name: []string{"Heat (1995).mkv", "Heat (1995).en.srt"}[i], Source: source.Torrent{Hash: hash, File: i}, Size: size})
	}
	var id transfers.ID
	if err := w.lib.Write(t.Context(), func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
		var err error
		id, err = w.tr.Start(ctx, tx, req)
		return err
	}); err != nil {
		t.Fatal(err)
	}
	return id
}

// receive has home report it receives id on the route it would run now.
func (w *world) receive(t *testing.T, id transfers.ID, verified, rate int64) {
	t.Helper()
	v, err := w.tr.Get(t.Context(), id)
	if err != nil {
		t.Fatal(err)
	}
	w.mu.Lock()
	defer w.mu.Unlock()
	w.facts = &tv1.Facts{Receiving: []*tv1.Receiving{{
		TransferId: id.Bytes(), Route: v.Route, Active: true, Moving: true,
		VerifiedBytes: verified, Rate: rate, Lanes: 2, RttMs: 30, Retried: 1,
	}}}
}

func hashOf(b byte) bencode.InfoHash {
	var h bencode.InfoHash
	h[0] = b
	return h
}

func TestATransferReadsWhatItWaitsForThenItsPace(t *testing.T) {
	w := serve(t)
	id := w.start(t, "heat", hashOf(1), 3_000_000_000, 1_000_000_000)
	path := "/api/v1/transfers/" + id.String()

	files := `"files":[{"position":0,"name":"Heat (1995).mkv","size":3000000000,"verified":false,"placed":false},` +
		`{"position":1,"name":"Heat (1995).en.srt","size":1000000000,"verified":false,"placed":false}]`
	head := `{"id":"id1","purpose":"grab","job":"heat","jobName":"Heat (1995)","source":{"nodeId":"id2","nodeName":"seedbox"},` +
		`"destination":{"kind":"root","rootId":"id3","nodeId":"id4","nodeName":"home"},`
	w.o.Expect(t, http.MethodGet, path, "", http.StatusOK, head+
		`"state":"waiting","reason":"Connecting to seedbox.","wait":{"kind":"connecting","nodeId":"id2"},`+
		`"verified":0,"total":4000000000,"rate":0,"estimateSeconds":0,"route":"receiver_dials",`+files+`,"details":{"lanes":0,"rttMs":0,"retried":0}}`)

	w.receive(t, id, 1_000_000_000, 50_000_000)
	w.o.Expect(t, http.MethodGet, path, "", http.StatusOK, head+
		`"state":"copying","verified":1000000000,"total":4000000000,"rate":50000000,"estimateSeconds":60,"route":"receiver_dials",`+
		files+`,"details":{"lanes":2,"rttMs":30,"retried":1}}`)
}

func TestASecondTransferOnThePairQueuesBehindTheFirst(t *testing.T) {
	w := serve(t)
	w.exec(t, "INSERT INTO transfers_pair (source_node, destination_node, rate, rate_at) VALUES (?, ?, 40000000, ?)", w.seedbox, w.home, storage.Now())
	w.start(t, "heat", hashOf(1), 1000)
	w.start(t, "ronin", hashOf(2), 2000)

	_, reply := w.o.Call(t, http.MethodGet, "/api/v1/transfers", "")
	var got struct {
		Transfers []struct {
			JobName, State, Reason string
			Rate                   int64
		} `json:"transfers"`
	}
	if err := json.Unmarshal([]byte(reply), &got); err != nil {
		t.Fatal(err)
	}
	if len(got.Transfers) != 2 {
		t.Fatalf("listed %s, want both Transfers", reply)
	}
	if second := got.Transfers[1]; second.JobName != "Ronin (1998)" || second.State != "queued" || second.Reason != "Queued behind 1 other Transfer from seedbox." || second.Rate != 40_000_000 {
		t.Fatalf("the second Transfer reads %+v, want Ronin queued behind Heat at the pair's last speed", second)
	}
}

func TestTransfersFilterByNodeAndTorrent(t *testing.T) {
	w := serve(t)
	id := w.start(t, "heat", hashOf(1), 1000)
	for _, tc := range []struct {
		query string
		want  int
	}{
		{"", 1},
		{"?nodeId=" + w.seedbox.String(), 1},
		{"?nodeId=" + w.home.String(), 1},
		{"?nodeId=" + storage.NewUUID().String(), 0},
		{"?infoHash=" + hashOf(1).String(), 1},
		{"?infoHash=" + hashOf(2).String(), 0},
		{"?nodeId=" + w.seedbox.String() + "&infoHash=" + hashOf(2).String(), 0},
	} {
		status, reply := w.o.Call(t, http.MethodGet, "/api/v1/transfers"+tc.query, "")
		var got struct {
			Transfers []struct{ ID string } `json:"transfers"`
		}
		if err := json.Unmarshal([]byte(reply), &got); status != http.StatusOK || err != nil {
			t.Fatalf("GET %s = %d %s", tc.query, status, reply)
		}
		if len(got.Transfers) != tc.want || tc.want == 1 && got.Transfers[0].ID != id.String() {
			t.Fatalf("GET %s listed %s, want %d", tc.query, reply, tc.want)
		}
	}
	w.o.Expect(t, http.MethodGet, "/api/v1/transfers?infoHash=nothex", "", http.StatusUnprocessableEntity,
		`{"status":422,"code":"invalid","title":"Some fields have values SlipStream can't accept","errors":[{"location":"query.infoHash","message":"expected string to match pattern ^[0-9a-f]{40}$"}]}`)
}

func TestAFinishedOrUnknownTransferIsNotFound(t *testing.T) {
	w := serve(t)
	w.o.Expect(t, http.MethodGet, "/api/v1/transfers/"+storage.NewUUID().String(), "", http.StatusNotFound,
		`{"status":404,"code":"transfer_not_found","title":"No Transfer has that id"}`)
}

func TestSpeedsListEachPairsLastSpeedAndRoute(t *testing.T) {
	w := serve(t)
	w.o.Expect(t, http.MethodGet, "/api/v1/transfers/speeds", "", http.StatusOK, `{"speeds":[]}`)
	w.exec(t, "INSERT INTO transfers_pair (source_node, destination_node, rate, rate_at, route, route_at) VALUES (?, ?, 48000000, ?, 2, ?)", w.seedbox, w.home, storage.Now(), storage.Now())
	w.o.Expect(t, http.MethodGet, "/api/v1/transfers/speeds", "", http.StatusOK,
		`{"speeds":[{"sourceNodeId":"id1","destinationNodeId":"id2","rate":48000000,"rateAt":"T","route":"sender_dials","routeAt":"T"}]}`)
}

type sendLimit struct {
	SendLimit struct {
		Value     int64 `json:"value"`
		IsDefault bool  `json:"isDefault"`
	} `json:"sendLimit"`
}

func call(t *testing.T, o *ownertest.Owner, method, path, body string) sendLimit {
	t.Helper()
	status, reply := o.Call(t, method, path, body)
	var s sendLimit
	if err := json.Unmarshal([]byte(reply), &s); status != http.StatusOK || err != nil {
		t.Fatalf("%s %s %s = %d %s, want 200 and the settings", method, path, body, status, reply)
	}
	return s
}

func TestANodesSendLimitChangesAndResets(t *testing.T) {
	w := serve(t)
	path := "/api/v1/transfers/nodes/" + w.seedbox.String() + "/settings"
	if got := call(t, w.o, http.MethodGet, path, ""); got.SendLimit.Value != 0 || !got.SendLimit.IsDefault {
		t.Fatalf("seedbox's send limit before any change = %+v, want no limit by default", got)
	}
	got := call(t, w.o, http.MethodPatch, path, `{"sendLimit":5242880}`)
	if got.SendLimit.Value != 5242880 || got.SendLimit.IsDefault {
		t.Fatalf("after a 5 MB/s send limit = %+v", got)
	}
	if again := call(t, w.o, http.MethodGet, path, ""); again != got {
		t.Fatalf("read back = %+v, want %+v", again, got)
	}
	if home := call(t, w.o, http.MethodGet, "/api/v1/transfers/nodes/"+w.home.String()+"/settings", ""); !home.SendLimit.IsDefault {
		t.Fatalf("home's send limit = %+v, want seedbox's change to leave it at its default", home)
	}
	if reset := call(t, w.o, http.MethodPatch, path, `{"sendLimit":null}`); reset.SendLimit.Value != 0 || !reset.SendLimit.IsDefault {
		t.Fatalf("after a reset = %+v, want no limit by default", reset)
	}
}

func TestSendLimitRefuses(t *testing.T) {
	w := serve(t)
	w.o.Expect(t, http.MethodPatch, "/api/v1/transfers/nodes/"+w.seedbox.String()+"/settings", `{"sendLimit":1000}`, http.StatusUnprocessableEntity,
		`{"status":422,"code":"setting_out_of_range","title":"The value is outside the setting's range","detail":"In whole KB/s."}`)
	w.o.Expect(t, http.MethodGet, "/api/v1/transfers/nodes/"+storage.NewUUID().String()+"/settings", "", http.StatusNotFound,
		`{"status":404,"code":"node_not_found","title":"No Node has this id"}`)
}
