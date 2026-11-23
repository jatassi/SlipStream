package reconciling_test

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"slices"
	"strings"
	"sync"
	"testing"
	"testing/synctest"
	"time"

	"github.com/jatassi/slipstream/internal/bencode"
	"github.com/jatassi/slipstream/internal/contenttype/contenttypetest"
	dc "github.com/jatassi/slipstream/internal/downloadclients"
	"github.com/jatassi/slipstream/internal/health"
	"github.com/jatassi/slipstream/internal/history/entry"
	"github.com/jatassi/slipstream/internal/history/entry/entrytest"
	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/indexers"
	"github.com/jatassi/slipstream/internal/live"
	"github.com/jatassi/slipstream/internal/nodes"
	"github.com/jatassi/slipstream/internal/nodes/nodestest"
	dlv1 "github.com/jatassi/slipstream/internal/pb/slipstream/downloading/v1"
	"github.com/jatassi/slipstream/internal/reconciling"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/roots"
	"github.com/jatassi/slipstream/internal/rpc"
	"github.com/jatassi/slipstream/internal/secret"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/storage/storagetest"
	"github.com/jatassi/slipstream/internal/works"
)

// world is a Controller's library with the reconciling Service adopting
// over it, and Nodes that answer metainfo from meta.
type world struct {
	t       *testing.T
	lib     *storage.DB[storage.Library]
	keeper  *reconciling.Service
	clients *dc.Clients
	page    int

	mu sync.Mutex
	// meta is what every Node answers for a hash's metainfo, and failing
	// the error a hash's ask fails with instead.
	meta    map[bencode.InfoHash][]byte
	failing map[bencode.InfoHash]error
	asked   map[bencode.InfoHash]int
	touched map[role.NodeID]int
	// rehomed and scans are each root the re-home step saved and each
	// scan it started; ending, ended and forgot count the end's two hooks
	// and each Forget.
	rehomed []roots.ID
	scans   [][]roots.ID
	// ready is each root its Node reports Ready, and hub carries
	// roots.changed.
	ready  map[roots.ID]bool
	hub    *live.Hub
	ending int
	ended  int
	forgot int
	// onDisk is each save folder whose files every Node still has.
	onDisk  map[string]bool
	history *entrytest.Recorder
}

const holdingsMax = 100_000

// controllerNode is the Controller's own Node in every world.
var controllerNode = role.NodeID{UUID: storage.NewUUID()}

func newWorld(t *testing.T) *world {
	t.Helper()
	w := idle(t, 1000)
	w.start(t.Context())
	return w
}

// idle is a world whose Service isn't running yet.
func idle(t *testing.T, page int) *world {
	t.Helper()
	return &world{
		t:    t,
		lib:  storagetest.Library(t, secret.Schema(), identity.Schema(), nodes.Schema(), works.Schema(), health.Schema(), roots.Schema(), dc.Schema(), indexers.Schema(), reconciling.Schema()),
		page: page, meta: map[bencode.InfoHash][]byte{}, failing: map[bencode.InfoHash]error{},
		asked: map[bencode.InfoHash]int{}, touched: map[role.NodeID]int{}, onDisk: map[string]bool{},
		ready: map[roots.ID]bool{}, hub: live.NewHub([]live.Event{roots.ChangedEvent}),
	}
}

// start runs a reconciling Service over w's library until ctx ends.
func (w *world) start(ctx context.Context) {
	var rec entry.Recorder
	rec, w.history = entrytest.New("reconciling", reconciling.Entries())
	w.keeper = reconciling.New(reconciling.Config{
		Library: w.lib,
		Self:    func() role.NodeID { return controllerNode },
		History: &rec,
		Touch: func(ids ...role.NodeID) {
			w.mu.Lock()
			defer w.mu.Unlock()
			for _, id := range ids {
				w.touched[id]++
			}
		},
		Metainfo: w.metainfo,
		Page:     w.page, Max: holdingsMax,
		Rehome: func(_ context.Context, _ identity.Admin, id roots.ID, _ roots.Rehome) (roots.View, error) {
			w.mu.Lock()
			defer w.mu.Unlock()
			w.rehomed = append(w.rehomed, id)
			return roots.View{}, nil
		},
		Scan: func(_ context.Context, ids []roots.ID) error {
			w.mu.Lock()
			defer w.mu.Unlock()
			w.scans = append(w.scans, ids)
			return nil
		},
		Ready: func(context.Context) ([]roots.Root, error) {
			w.mu.Lock()
			defer w.mu.Unlock()
			var out []roots.Root
			for id, ready := range w.ready {
				if ready {
					out = append(out, roots.Root{ID: id})
				}
			}
			return out, nil
		},
		Hub: w.hub,
		Ending: func(context.Context, *storage.Tx[storage.Library]) error {
			w.mu.Lock()
			defer w.mu.Unlock()
			w.ending++
			return nil
		},
		Ended: func(context.Context) {
			w.mu.Lock()
			defer w.mu.Unlock()
			w.ended++
		},
		Look: func(_ context.Context, _ role.NodeID, folder string, files []dc.LookFile) ([]bool, error) {
			w.mu.Lock()
			defer w.mu.Unlock()
			out := make([]bool, len(files))
			for i := range files {
				out[i] = w.onDisk[folder]
			}
			return out, nil
		},
		Forgot: func() {
			w.mu.Lock()
			defer w.mu.Unlock()
			w.forgot++
		},
		Changed: quiet{},
		Log:     slog.New(slog.DiscardHandler),
	})
	w.clients = dc.New(dc.Config{Library: w.lib, Keeps: w.keeper, HoldingsPage: w.page, HoldingsMax: holdingsMax, Touch: func(...role.NodeID) {}})
	go func() { _ = w.keeper.Run(ctx) }()
	synctest.Wait()
}

func (w *world) metainfo(_ context.Context, _ role.NodeID, h bencode.InfoHash, client *dc.ClientID) ([]byte, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.asked[h]++
	if err, ok := w.failing[h]; ok {
		return nil, err
	}
	if raw, ok := w.meta[h]; ok {
		return raw, nil
	}
	if client != nil {
		return nil, rpc.Errorf(rpc.CodeUnimplemented, "can't export")
	}
	return nil, rpc.Errorf(rpc.CodeNotFound, "no resume data")
}

func (w *world) write(fn func(ctx context.Context, tx *storage.Tx[storage.Library]) error) {
	w.t.Helper()
	if err := w.lib.Write(w.t.Context(), fn); err != nil {
		w.t.Fatal(err)
	}
}

// node pairs a Node named name with no keeping row.
func (w *world) node(name string) role.NodeID {
	w.t.Helper()
	id := role.NodeID{UUID: storage.NewUUID()}
	w.write(func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
		return nodestest.Insert(ctx, tx, id, name)
	})
	return id
}

// paired pairs a Node named name as nodes' Pair does.
func (w *world) paired(name string) role.NodeID {
	w.t.Helper()
	id := role.NodeID{UUID: storage.NewUUID()}
	w.write(func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
		if err := nodestest.Insert(ctx, tx, id, name); err != nil {
			return err
		}
		return reconciling.Paired(ctx, tx, id)
	})
	return id
}

// restore begins Reconciling in place for every Node paired so far.
func (w *world) restore() {
	w.t.Helper()
	w.write(func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
		return reconciling.Begin(ctx, tx, reconciling.BeginArgs{ID: "restore-1", Cause: reconciling.Restore, Backup: "nightly", TakenAt: time.Now()})
	})
}

// torrent is a torrent a client may hold: its metainfo, and the Holdings
// entry for it.
type torrent struct {
	raw  []byte
	hash bencode.InfoHash
	name string
}

func newTorrent(t *testing.T, name string) torrent {
	t.Helper()
	raw, err := bencode.Encode(map[string]any{
		"announce": "https://tracker.example/announce",
		"info":     map[string]any{"name": name, "length": int64(1 << 20), "piece length": int64(1 << 18), "pieces": strings.Repeat("p", 80)},
	})
	if err != nil {
		t.Fatal(err)
	}
	m, err := bencode.DecodeMetainfo(raw)
	if err != nil {
		t.Fatal(err)
	}
	return torrent{raw: raw, hash: m.InfoHash, name: name}
}

func (tr torrent) entry() *dlv1.Holding {
	return &dlv1.Holding{
		InfoHash: tr.hash[:], Content: &dlv1.HoldingContent{Name: tr.name, Size: 1 << 20}, Done: 1, SeedSeconds: 3600,
		Hosts: []string{"tracker.example"}, SaveFolder: "/downloads/" + tr.name,
	}
}

// unsaved is tr's entry from a Node that never saved its resume data.
func (tr torrent) unsaved() *dlv1.Holding {
	return &dlv1.Holding{InfoHash: tr.hash[:], SaveFolder: "/downloads/" + tr.name}
}

// exports has every Node answer tr's metainfo.
func (w *world) exports(trs ...torrent) {
	w.mu.Lock()
	defer w.mu.Unlock()
	for _, tr := range trs {
		w.meta[tr.hash] = tr.raw
	}
}

// record has the library record tr on node, as a Grab adds it.
func (w *world) record(node role.NodeID, tr torrent) dc.ID {
	w.t.Helper()
	m, err := bencode.DecodeMetainfo(tr.raw)
	if err != nil {
		w.t.Fatal(err)
	}
	var id dc.ID
	w.write(func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
		want := dc.Want{Target: dc.Target{Node: node}, Metainfo: m, Raw: tr.raw, ContentType: contenttypetest.ID, WrittenOff: func(role.NodeID) bool { return false }}
		id, _, err = dc.Attach(ctx, tx, want, storage.Now())
		return err
	})
	return id
}

// writeOff parts every Torrent of node, as an Admin's write-off does.
func (w *world) writeOff(node role.NodeID) {
	w.write(func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
		_, err := tx.ExecContext(ctx, `UPDATE downloadclients_torrent SET removed_at = coalesce(removed_at, 1),
			removed_reason = 'node_written_off', parted_at = 1 WHERE node_id = ? AND parted_at IS NULL`, node)
		return err
	})
}

// report has node send one page of its round's Holdings, as its holder
// does, and lets the Service act on it.
func (w *world) report(node role.NodeID, round uint64, page, pages uint32, entries ...*dlv1.Holding) {
	w.t.Helper()
	rep := &dlv1.Report{Kind: &dlv1.Report_Holdings{Holdings: &dlv1.Holdings{Round: round, Page: page, Pages: pages, Entries: entries}}}
	w.write(func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
		return w.clients.Apply(ctx, tx, node, rep)
	})
	w.keeper.Applied(node, string(role.Downloading))
	synctest.Wait()
}

// reportAll has node send its whole round in one page.
func (w *world) reportAll(node role.NodeID, entries ...*dlv1.Holding) {
	w.t.Helper()
	w.report(node, w.round(node), 0, 1, entries...)
}

// intent is node's Downloading intent as the Controller derives it.
func (w *world) intent(node role.NodeID) *dlv1.Intent {
	w.t.Helper()
	var in *dlv1.Intent
	err := w.lib.Read(w.t.Context(), func(ctx context.Context, r storage.Reader[storage.Library]) error {
		var err error
		in, err = dc.Derive{Keeps: w.keeper, HoldingsPage: w.page}.Intent(ctx, r, node)
		return err
	})
	if err != nil {
		w.t.Fatal(err)
	}
	return in
}

// round is the Holdings round node's intent asks for, 0 for none.
func (w *world) round(node role.NodeID) uint64 { return w.intent(node).GetHoldings().GetRound() }

// keeps is whether node's intent keeps tr though it lists no Torrent for
// it.
func (w *world) keeps(node role.NodeID, tr torrent) bool {
	return slices.ContainsFunc(w.intent(node).GetKeepHashes(), func(h []byte) bool { return bytes.Equal(h, tr.hash[:]) })
}

// lists is whether node's intent lists a Torrent for tr.
func (w *world) lists(node role.NodeID, tr torrent) bool {
	return slices.ContainsFunc(w.intent(node).GetTorrents(), func(t *dlv1.Torrent) bool { return bytes.Equal(t.GetInfoHash(), tr.hash[:]) })
}

// row is what the library records for a hash.
type row struct {
	id       dc.ID
	node     role.NodeID
	adopted  bool
	live     bool
	folder   string
	withheld bool
	metainfo bool
	seeding  int64
}

// rows is every Torrent the library records for tr, in added order.
func (w *world) rows(tr torrent) []row {
	w.t.Helper()
	rs, err := w.lib.QueryContext(w.t.Context(), `SELECT id, node_id, adopted, parted_at IS NULL, coalesce(save_folder, ''),
		withheld_at IS NOT NULL, metainfo IS NOT NULL, seeding_seconds FROM downloadclients_torrent WHERE info_hash = ? ORDER BY added_order`, tr.hash[:])
	if err != nil {
		w.t.Fatal(err)
	}
	defer func() { _ = rs.Close() }()
	var out []row
	for rs.Next() {
		var r row
		if err := rs.Scan(&r.id, &r.node, &r.adopted, &r.live, &r.folder, &r.withheld, &r.metainfo, &r.seeding); err != nil {
			w.t.Fatal(err)
		}
		out = append(out, r)
	}
	if err := rs.Err(); err != nil {
		w.t.Fatal(err)
	}
	return out
}

// only is the one Torrent the library records for tr.
func (w *world) only(tr torrent) row {
	w.t.Helper()
	rs := w.rows(tr)
	if len(rs) != 1 {
		w.t.Fatalf("the library records %d Torrents for %s, want 1: %+v", len(rs), tr.name, rs)
	}
	return rs[0]
}

// review is the review list's reason for Torrent id, "" when it isn't on
// it.
func (w *world) review(id dc.ID) string {
	w.t.Helper()
	var reason string
	err := w.lib.QueryRowContext(w.t.Context(), "SELECT reason FROM reconciling_torrent WHERE torrent_id = ?", id).Scan(&reason)
	if err != nil && !errors.Is(err, storage.ErrNoRows) {
		w.t.Fatal(err)
	}
	return reason
}

// roles is the Roles node is assigned.
func (w *world) roles(node role.NodeID) []string {
	w.t.Helper()
	rs, err := w.lib.QueryContext(w.t.Context(), "SELECT role FROM nodes_role WHERE node_id = ? ORDER BY role", node)
	if err != nil {
		w.t.Fatal(err)
	}
	defer func() { _ = rs.Close() }()
	var out []string
	for rs.Next() {
		var r string
		if err := rs.Scan(&r); err != nil {
			w.t.Fatal(err)
		}
		out = append(out, r)
	}
	return out
}

// pending is whether node still keeps everything, waiting for adoption.
func (w *world) pending(node role.NodeID) bool {
	w.t.Helper()
	var set map[role.NodeID]bool
	err := w.lib.Read(w.t.Context(), func(ctx context.Context, r storage.Reader[storage.Library]) error {
		var err error
		set, err = reconciling.Pending(ctx, r)
		return err
	})
	if err != nil {
		w.t.Fatal(err)
	}
	return set[node]
}

// granted is when adoption gave node Downloading, zero when it didn't.
func (w *world) granted(node role.NodeID) int64 {
	w.t.Helper()
	var at *int64
	if err := w.lib.QueryRowContext(w.t.Context(), "SELECT granted_at FROM reconciling_node WHERE node_id = ?", node).Scan(&at); err != nil {
		w.t.Fatal(err)
	}
	if at == nil {
		return 0
	}
	return *at
}

// waits is what Reconciling waits on.
func (w *world) waits() reconciling.Waits {
	w.t.Helper()
	var out reconciling.Waits
	err := w.lib.Read(w.t.Context(), func(ctx context.Context, r storage.Reader[storage.Library]) error {
		var err error
		out, err = reconciling.Waiting(ctx, r)
		return err
	})
	if err != nil {
		w.t.Fatal(err)
	}
	return out
}

// phase is the gates' State.
func (w *world) phase() reconciling.Phase {
	w.t.Helper()
	var p reconciling.Phase
	err := w.lib.Read(w.t.Context(), func(ctx context.Context, r storage.Reader[storage.Library]) error {
		var err error
		p, err = reconciling.State(ctx, r)
		return err
	})
	if err != nil {
		w.t.Fatal(err)
	}
	return p
}

func torrents(t *testing.T, prefix string, n int) []torrent {
	out := make([]torrent, n)
	for i := range out {
		out[i] = newTorrent(t, fmt.Sprintf("%s.%04d", prefix, i))
	}
	return out
}

func entries(trs []torrent) []*dlv1.Holding {
	out := make([]*dlv1.Holding, len(trs))
	for i, tr := range trs {
		out[i] = tr.entry()
	}
	return out
}

type quiet struct{}

func (quiet) Publish(context.Context, reconciling.Changed) {}
