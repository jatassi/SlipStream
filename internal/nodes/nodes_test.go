package nodes_test

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"crypto/sha256"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"math/rand/v2"
	"net/http"
	"net/netip"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"sync"
	"testing"
	"testing/synctest"
	"time"

	"github.com/jatassi/slipstream/fakes/fault"
	"github.com/jatassi/slipstream/internal/joincode"
	"github.com/jatassi/slipstream/internal/link"
	"github.com/jatassi/slipstream/internal/link/linktest"
	"github.com/jatassi/slipstream/internal/nodes"
	"github.com/jatassi/slipstream/internal/nodes/nodestest"
	linkv1 "github.com/jatassi/slipstream/internal/pb/slipstream/link/v1"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/storage/storagetest"
)

var discard = slog.New(slog.DiscardHandler)

// side is the Controller link's Trust and Pair, answered by the Service
// once it is open.
type side struct{ svc *nodes.Service }

func (s *side) Admit(key link.Key) link.Admission { return s.svc.Admit(key) }

func (s *side) Pair(ctx context.Context, peer link.Key, req *linkv1.PairRequest) (*linkv1.PairResponse, error) {
	return s.svc.Pair(ctx, peer, req)
}

// published records each nodes.changed payload.
type published struct {
	mu  sync.Mutex
	ids []string
}

func (p *published) Publish(_ context.Context, v nodes.Changed) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.ids = append(p.ids, v.NodeID)
}

func (p *published) has(id string) bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	return slices.Contains(p.ids, id)
}

// heard records each nodes.traffic sample, with an Admin always listening.
type heard struct {
	mu      sync.Mutex
	samples []nodes.Traffic
}

func (h *heard) Publish(_ context.Context, v nodes.Traffic) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.samples = append(h.samples, v)
}

func (*heard) Listening() bool { return true }

func (h *heard) all() []nodes.Traffic {
	h.mu.Lock()
	defer h.mu.Unlock()
	return slices.Clone(h.samples)
}

func keptNothing(context.Context, storage.Reader[storage.Library], nodes.ID) ([]nodes.Grabbed, error) {
	return nil, nil
}

func storesNothing(context.Context, storage.Reader[storage.Library], nodes.ID) (nodes.StorageFigures, error) {
	return nodes.StorageFigures{}, nil
}

type controller struct {
	net     *fault.Network
	node    *linktest.Node
	lib     *storage.DB[storage.Library]
	svc     *nodes.Service
	pub     *published
	traffic *heard
	stop    func()
	// edit changes the roster's Config at each open.
	edit []func(*nodes.Config)
}

// lockstep serves the Controls' Sync streams once the Service is open.
func (s *side) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	_, h := s.svc.Controls().Handler()
	h.ServeHTTP(w, r)
}

func startController(t *testing.T, edit ...func(*nodes.Config)) *controller {
	t.Helper()
	net := fault.NewNetwork()
	s := &side{}
	node := linktest.Start(t, net, "home", []netip.Addr{linktest.ControllerAddr}, linktest.Side{Trust: s, Pair: s.Pair, Lockstep: s})
	lib := storagetest.Library(t, nodes.Schema(), role.LibrarySchema())
	c := &controller{net: net, node: node, lib: lib, edit: edit}
	c.open(t, s)
	return c
}

// open opens the roster on c's library and runs it until the test ends.
func (c *controller) open(t *testing.T, s *side) {
	t.Helper()
	pub, traffic := &published{}, &heard{}
	cfg := nodes.Config{
		Library: c.lib,
		Key:     c.node.Key,
		Name:    "home",
		Link:    c.node.Link,
		Changed: pub,
		Traffic: traffic,
		Owners:  nodestest.Unowned(),
		Kept:    keptNothing,
		Storage: storesNothing,
		Facts:   func(nodes.ID, string) {},
		Timings: nodes.DefaultTimings(),
		Stall:   45 * time.Second,
		Log:     discard,
	}
	for _, e := range c.edit {
		e(&cfg)
	}
	svc, err := nodes.Open(t.Context(), cfg)
	if err != nil {
		t.Fatal(err)
	}
	s.svc = svc
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		defer close(done)
		_ = svc.Run(ctx)
	}()
	stop := sync.OnceFunc(func() {
		cancel()
		<-done
	})
	t.Cleanup(stop)
	c.svc, c.pub, c.traffic, c.stop = svc, pub, traffic, stop
}

// hold runs w's holder half in session with the Controller until stop or
// the test's end.
func (c *controller) hold(t *testing.T, w *linktest.Node) (stop func()) {
	t.Helper()
	return holder(t, role.OverLink(w.Link, c.node.Key, linktest.Timings))
}

func holder(t *testing.T, upstream role.Upstream) (stop func()) {
	t.Helper()
	h, err := role.NewHolding(t.Context(), role.HoldingConfig{
		Local:    storagetest.Local(t, role.LocalSchema()),
		Upstream: upstream,
		Timings:  role.Timings{RestartMin: time.Second, RestartMax: time.Minute, Beat: 15 * time.Second},
		Rand:     rand.New(rand.NewPCG(1, 2)), //nolint:gosec // seeded on purpose so tests replay
		Log:      discard,
	})
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		defer close(done)
		_ = h.Run(ctx)
	}()
	stop = sync.OnceFunc(func() {
		cancel()
		<-done
	})
	t.Cleanup(stop)
	return stop
}

func (c *controller) mint(t *testing.T) nodes.JoinCode {
	t.Helper()
	code, err := c.svc.MintJoinCode(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	return code
}

func (c *controller) worker(t *testing.T, name, addr string) *linktest.Node {
	t.Helper()
	return linktest.Start(t, c.net, name, []netip.Addr{netip.MustParseAddr(addr)}, linktest.Side{})
}

// apply sets want on the rows now, planned on their basis.
func (c *controller) apply(t *testing.T, want nodes.Roles) nodes.Outcome {
	t.Helper()
	board, err := c.svc.RoleBoard(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	out, err := c.svc.Apply(t.Context(), nodes.SetRoles{Basis: board.Basis(), Want: want})
	if err != nil {
		t.Fatal(err)
	}
	return out
}

func roleNames(views []nodes.RoleView) []role.Name {
	out := []role.Name{}
	for _, v := range views {
		out = append(out, v.Role)
	}
	return out
}

func refusal(err error) linkv1.RefusedReason {
	if r, ok := errors.AsType[*link.RefusedError](err); ok {
		return r.Reason
	}
	return linkv1.RefusedReason_REFUSED_REASON_UNSPECIFIED
}

func (c *controller) pair(t *testing.T, w *linktest.Node, code nodes.JoinCode, name string) (link.Paired, role.NodeID) {
	t.Helper()
	paired, err := w.Pair(t.Context(), code.Code, name)
	if err != nil {
		t.Fatal(err)
	}
	id, err := storage.UUIDFromBytes(paired.NodeID)
	if err != nil {
		t.Fatal(err)
	}
	return paired, role.NodeID{UUID: id}
}

func TestRepeatingAPairReturnsTheSameNode(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		c := startController(t)
		code := c.mint(t)
		if got, want := code.Code.Addresses(), []netip.AddrPort{netip.MustParseAddrPort("127.0.0.2:8485")}; !slices.Equal(got, want) {
			t.Fatalf("the code names %v, want %v", got, want)
		}
		box := c.worker(t, "box", "127.0.0.3")
		first, id := c.pair(t, box, code, "box")
		if first.Name != "box" || first.ControllerName != "home" || first.Controller != c.node.Key {
			t.Fatalf("paired as %q with %q (%s), want box with home (%s)", first.Name, first.ControllerName, first.Controller.Fingerprint(), c.node.Key.Fingerprint())
		}
		synctest.Wait()
		if !c.pub.has(id.String()) {
			t.Errorf("nodes.changed was not published for %s", id)
		}
		if again, _ := c.pair(t, box, code, "box"); !bytes.Equal(again.NodeID, first.NodeID) {
			t.Fatalf("pairing again with the same code gave node %x, want %x", again.NodeID, first.NodeID)
		}
		_, err := c.worker(t, "other", "127.0.0.4").Pair(t.Context(), code.Code, "other")
		if got := refusal(err); got != linkv1.RefusedReason_REFUSED_REASON_JOIN_CODE_USED {
			t.Fatalf("another key with a used code: %v, want REFUSED_REASON_JOIN_CODE_USED", err)
		}
	})
}

func TestAKnownKeyKeepsItsRoles(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		c := startController(t)
		box := c.worker(t, "box", "127.0.0.3")
		_, id := c.pair(t, box, c.mint(t), "box")
		box.Drop(c.node.Key)
		synctest.Wait()
		c.apply(t, nodes.Roles{id: role.Of(role.Downloading, role.Searching)})
		if _, again := c.pair(t, box, c.mint(t), "renamed"); again != id {
			t.Fatalf("a new code paired the known key as %s, want %s", again, id)
		}
		got, err := c.svc.Get(t.Context(), id)
		if err != nil {
			t.Fatal(err)
		}
		if want := []role.Name{role.Searching, role.Downloading}; got.Name != "box" || !slices.Equal(roleNames(got.Roles), want) {
			t.Fatalf("after a new code the known key is %q holding %v, want box holding %v", got.Name, got.Roles, want)
		}
	})
}

// A Worker whose link is live but whose Sync session is not attached is
// not in step: it takes no new work until its holder half is in session.
func TestOnlyANodeWhoseSyncIsAttachedMayTakeNewWork(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		c := startController(t)
		w := c.worker(t, "box", "127.0.0.3")
		_, id := c.pair(t, w, c.mint(t), "box")
		w.Drop(c.node.Key)
		synctest.Wait()
		c.apply(t, nodes.Roles{id: role.Of(role.Searching)})
		w.Keep(c.node.Target())
		synctest.Wait()
		if _, live := w.State(c.node.Key).(link.Live); !live {
			t.Fatal("box's link to home is not live")
		}
		if got, ok := c.svc.Pick(role.Searching, id); !ok || got != c.svc.Self() || c.svc.InStep(id) {
			t.Fatalf("with box's link live and no Sync session, Pick(box) = %s, %v and box in step %v; want home, out of step", got, ok, c.svc.InStep(id))
		}
		changed := c.svc.Changed()
		c.hold(t, w)
		synctest.Wait()
		select {
		case <-changed:
		default:
			t.Fatal("the roster did not change when box's Sync session attached")
		}
		if got, ok := c.svc.Pick(role.Searching, id); !ok || got != id || !c.svc.InStep(id) {
			t.Fatalf("with box in session, Pick(box) = %s, %v and box in step %v; want box", got, ok, c.svc.InStep(id))
		}
	})
}

// seedbox in session holds Searching. Gone past the offline grace from
// its last Sync frame, it lapses and the Controller covers Searching in
// its own node scope.
func TestTheControllerCoversALapsedNodesSearching(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		c := startController(t)
		w := c.worker(t, "seedbox", "127.0.0.3")
		_, id := c.pair(t, w, c.mint(t), "seedbox")
		holder(t, role.InProcess(c.svc.Controls()))
		stop := c.hold(t, w)
		c.apply(t, nodes.Roles{id: role.Of(role.Searching), c.svc.Self(): role.Of(role.Downloading)})
		synctest.Wait()
		holding, err := c.svc.Controls().NodeVersion(t.Context(), c.svc.Self())
		if err != nil {
			t.Fatal(err)
		}
		stop()
		synctest.Wait()
		left := time.Now()
		if who := c.svc.Who(role.Searching); who.Covering() || !who.Serves(id) {
			t.Fatalf("just after seedbox left: covering %v, seedbox serves %v; want seedbox away and still serving", who.Covering(), who.Serves(id))
		}
		time.Sleep(time.Until(left.Add(nodes.OfflineGrace.Default)))
		synctest.Wait()
		who := c.svc.Who(role.Searching)
		if !who.Lapsed(id) || !who.Covering() {
			t.Fatalf("at the grace: seedbox lapsed %v, home covering %v; want both", who.Lapsed(id), who.Covering())
		}
		covering, err := c.svc.Controls().NodeVersion(t.Context(), c.svc.Self())
		if err != nil {
			t.Fatal(err)
		}
		if bytes.Equal(covering, holding) {
			t.Fatal("home's node scope did not change when it began to cover Searching")
		}
		if !c.pub.has(c.svc.Self().String()) || !c.pub.has(id.String()) {
			t.Fatal("nodes.changed was not published for home and seedbox at the lapse")
		}
	})
}

func TestACodePastItsLifetimeIsRefused(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		c := startController(t)
		tenMinutes, err := nodes.JoinCodeLifetime.Parse(new(int64(600)))
		if err != nil {
			t.Fatal(err)
		}
		if err := c.svc.SetJoinCodeLifetime(t.Context(), tenMinutes); err != nil {
			t.Fatal(err)
		}
		code := c.mint(t)
		time.Sleep(10 * time.Minute)
		_, err = c.worker(t, "box", "127.0.0.3").Pair(t.Context(), code.Code, "box")
		if got := refusal(err); got != linkv1.RefusedReason_REFUSED_REASON_JOIN_CODE_EXPIRED {
			t.Fatalf("a code past its lifetime: %v, want REFUSED_REASON_JOIN_CODE_EXPIRED", err)
		}
	})
}

func TestAnUnknownCodeIsRefused(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		c := startController(t)
		minted := c.mint(t)
		var secret [16]byte
		forged, err := joincode.New(ed25519.PublicKey(c.node.Key[:]), secret, minted.Code.Addresses())
		if err != nil {
			t.Fatal(err)
		}
		_, err = c.worker(t, "box", "127.0.0.3").Pair(t.Context(), forged, "box")
		if got := refusal(err); got != linkv1.RefusedReason_REFUSED_REASON_JOIN_CODE_UNKNOWN {
			t.Fatalf("a code the Controller never minted: %v, want REFUSED_REASON_JOIN_CODE_UNKNOWN", err)
		}
	})
}

func TestTheControllersOwnRow(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		c := startController(t)
		list, err := c.svc.List(t.Context())
		if err != nil {
			t.Fatal(err)
		}
		if len(list) != 1 {
			t.Fatalf("listed %d Nodes, want the Controller's own row", len(list))
		}
		own := list[0]
		want := slices.DeleteFunc(role.Names(), func(r role.Name) bool { return r == role.Relaying })
		if !own.Self || own.Name != "home" || own.Key != c.node.Key || own.Link != nodes.Local || !slices.Equal(roleNames(own.Roles), want) {
			t.Fatalf("own row = self %v, %q, %s, %s, %v; want self, home, its key, local, %v", own.Self, own.Name, own.Key.Fingerprint(), own.Link, own.Roles, want)
		}
	})
}

// A nodes_address row the peer address record refuses fails the read.
func TestAZonedAddressRowIsRefused(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		c := startController(t)
		list, err := c.svc.List(t.Context())
		if err != nil {
			t.Fatal(err)
		}
		own := list[0].ID
		err = c.lib.Write(t.Context(), func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
			_, err := tx.ExecContext(ctx, "INSERT INTO nodes_address (node_id, position, address) VALUES (?, 0, '[fe80::1%eth0]:8485')", own)
			return err
		})
		if err != nil {
			t.Fatal(err)
		}
		want := "node " + own.String() + " addresses: address [fe80::1%eth0]:8485 is not a literal IP and port"
		if _, err := c.svc.List(t.Context()); err == nil || err.Error() != want {
			t.Fatalf("List = %v, want %s", err, want)
		}
		if _, err := c.svc.Get(t.Context(), own); err == nil || err.Error() != want {
			t.Fatalf("Get = %v, want %s", err, want)
		}
	})
}

// TestAJoinCodeIsStoredOnlyAsItsSecretsHash mints a code: its one row
// holds the SHA-256 of the secret, and no value in the nodes set holds
// the secret, the code's text or its prefix.
func TestAJoinCodeIsStoredOnlyAsItsSecretsHash(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		c := startController(t)
		code := c.mint(t).Code
		secret := code.Secret()
		hash := sha256.Sum256(secret[:])
		if hashes := cells(t, c.lib, "SELECT secret_hash FROM nodes_join_code"); len(hashes) != 1 || !bytes.Equal(hashes[0], hash[:]) {
			t.Fatalf("nodes_join_code holds secret hashes %x, want only %x", hashes, hash)
		}
		for _, table := range cells(t, c.lib, "SELECT name FROM sqlite_master WHERE type = 'table' AND name LIKE 'nodes!_%' ESCAPE '!'") {
			for _, v := range cells(t, c.lib, "SELECT * FROM "+string(table)) {
				if bytes.Contains(v, secret[:]) || bytes.Contains(v, []byte(code.String())) || bytes.Contains(bytes.ToUpper(v), []byte("SLP1")) {
					t.Fatalf("%s holds %q, which carries the Join Code", table, v)
				}
			}
		}
	})
}

// cells is every value query returns, row by row, as bytes: a BLOB as
// stored, anything else as its text.
func cells(t *testing.T, lib *storage.DB[storage.Library], query string) [][]byte {
	t.Helper()
	r, err := lib.QueryContext(t.Context(), query)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = r.Close() }()
	cols, err := r.Columns()
	if err != nil {
		t.Fatal(err)
	}
	var out [][]byte
	for r.Next() {
		row := make([]sql.RawBytes, len(cols))
		ptrs := make([]any, len(cols))
		for i := range row {
			ptrs[i] = &row[i]
		}
		if err := r.Scan(ptrs...); err != nil {
			t.Fatal(err)
		}
		for _, v := range row {
			out = append(out, bytes.Clone(v))
		}
	}
	if err := r.Err(); err != nil {
		t.Fatal(err)
	}
	return out
}

func TestTheNodesSetPassesTheAudit(t *testing.T) {
	storagetest.AuditLibrary(t, []storage.Schema[storage.Library]{nodes.Schema()})
}

func TestTheLifetimeMigrationKeepsOnlyAnOverride(t *testing.T) {
	for _, tc := range []struct {
		name    string
		seconds int64
		want    []byte
	}{
		{"a changed lifetime", 7200, []byte("7200")},
		{"the default lifetime", 3600, nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), storage.LibraryFile)
			log := slog.New(slog.DiscardHandler)
			old, err := storage.OpenLibrary(t.Context(), path, []storage.Schema[storage.Library]{nodes.Schema().Through(1)}, nil, storage.DefaultTimings(), log)
			if err != nil {
				t.Fatal(err)
			}
			err = old.Write(t.Context(), func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
				_, err := tx.ExecContext(ctx, "UPDATE nodes_settings SET join_code_lifetime_seconds = ?", tc.seconds)
				return err
			})
			if err != nil {
				t.Fatal(err)
			}
			if err := old.Close(); err != nil {
				t.Fatal(err)
			}
			lib, err := storage.OpenLibrary(t.Context(), path, []storage.Schema[storage.Library]{nodes.Schema()}, nil, storage.DefaultTimings(), log)
			if err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() { _ = lib.Close() })
			if got := cells(t, lib, "SELECT join_code_lifetime_seconds FROM nodes_settings"); len(got) != 1 || !bytes.Equal(got[0], tc.want) {
				t.Fatalf("join_code_lifetime_seconds = %q, want %q", got, tc.want)
			}
		})
	}
}

// A Role with no owner fails every test that opens a roster, before the
// roster writes anything.
func TestOpenPanicsOnARoleWithNoOwner(t *testing.T) {
	lib := storagetest.Library(t, nodes.Schema(), role.LibrarySchema())
	for _, tc := range []struct {
		name   string
		owners []nodes.Owner
		want   string
	}{
		{"none", slices.DeleteFunc(nodestest.Unowned(), func(o nodes.Owner) bool { return o.Role == role.Storage }), "nodes: the storage Role has no owner"},
		{"two", append(nodestest.Unowned(), nodes.Unowned(role.Searching)), "nodes: the searching Role has two owners"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			defer func() {
				if got := recover(); got == nil || !strings.HasPrefix(fmt.Sprint(got), tc.want) {
					t.Errorf("Open panicked with %v, want %s", got, tc.want)
				}
				if rows := cells(t, lib, "SELECT id FROM nodes_node"); len(rows) != 0 {
					t.Errorf("Open wrote %d Nodes before it panicked", len(rows))
				}
			}()
			_, _ = nodes.Open(t.Context(), nodes.Config{Library: lib, Name: "home", Link: unlinked{}, Owners: tc.owners, Log: discard})
		})
	}
}

type unlinked struct{}

func (unlinked) State(link.Key) link.State { return link.Down{} }

func (unlinked) Subscribe(context.Context) <-chan link.Change { return nil }

func (unlinked) Advertised() (link.Addresses, bool) { return link.Addresses{}, false }

func (unlinked) Reset(link.Key) {}

func (unlinked) Keep(link.Target) {}

func (unlinked) Drop(link.Key) {}

func (unlinked) Invite(context.Context, link.Target) error { return nil }
func (unlinked) Stats(link.Key) (link.Stats, bool)         { return link.Stats{}, false }

// TestTheRosterMigrationKeepsEveryNode migrates a library holding two
// Nodes with Roles, an address and a used Join Code: every row and
// reference survives the rebuild, and a tombstone may then share a live
// Node's name but not stand in for its key.
func TestTheRosterMigrationKeepsEveryNode(t *testing.T) {
	path := filepath.Join(t.TempDir(), storage.LibraryFile)
	homeKey, boxKey := bytes.Repeat([]byte{1}, 32), bytes.Repeat([]byte{2}, 32)
	home, box := role.NodeID{UUID: storage.NewUUID()}, role.NodeID{UUID: storage.NewUUID()}
	populate(t, path, 2, []stmt{
		{"INSERT INTO nodes_node (id, public_key, name, paired_at) VALUES (?, ?, 'home', 1), (?, ?, 'box', 2)", []any{home, homeKey, box, boxKey}},
		{"INSERT INTO nodes_role (node_id, role, assigned_at) VALUES (?, 'searching', 1), (?, 'downloading', 2)", []any{home, box}},
		{"INSERT INTO nodes_address (node_id, position, address) VALUES (?, 0, '127.0.0.3:8485')", []any{box}},
		{"INSERT INTO nodes_join_code (secret_hash, created_at, expires_at, used_by, used_at) VALUES (?, 1, 2, ?, 2)", []any{bytes.Repeat([]byte{3}, 32), box}},
	})
	lib, err := storage.OpenLibrary(t.Context(), path, []storage.Schema[storage.Library]{nodes.Schema(), role.LibrarySchema()}, nil, storage.DefaultTimings(), discard)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = lib.Close() })

	for query, want := range map[string]string{
		"SELECT name || ' ' || hex(public_key) || ' ' || paired_at || ' ' || ifnull(seen_at, '-') || ' ' || ifnull(removed_at, '-') FROM nodes_node ORDER BY paired_at": "home " + strings.Repeat("01", 32) + " 1 - -,box " + strings.Repeat("02", 32) + " 2 - -",
		"SELECT n.name || ' ' || r.role FROM nodes_role r JOIN nodes_node n ON n.id = r.node_id ORDER BY r.assigned_at":                                                 "home searching,box downloading",
		"SELECT n.name || ' ' || a.address FROM nodes_address a JOIN nodes_node n ON n.id = a.node_id":                                                                  "box 127.0.0.3:8485",
		"SELECT n.name FROM nodes_join_code c JOIN nodes_node n ON n.id = c.used_by":                                                                                    "box",
		"SELECT ifnull(offline_grace_seconds, '-') FROM nodes_settings":                                                                                                 "-",
	} {
		if got := joined(t, lib, query); got != want {
			t.Errorf("%s\n= %q, want %q", query, got, want)
		}
	}

	insert := func(query string, args ...any) error {
		return lib.Write(t.Context(), func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
			_, err := tx.ExecContext(ctx, query, args...)
			return err
		})
	}
	if err := insert("INSERT INTO nodes_node (id, name, paired_at, removed_at, removed_key) VALUES (?, 'box', 3, 4, ?)", role.NodeID{UUID: storage.NewUUID()}, boxKey); err != nil {
		t.Fatalf("a tombstone named like a live Node: %v, want it stored", err)
	}
	if err := insert("INSERT INTO nodes_node (id, public_key, name, paired_at) VALUES (?, ?, 'box', 5)", role.NodeID{UUID: storage.NewUUID()}, bytes.Repeat([]byte{4}, 32)); err == nil {
		t.Error("a second live Node named box was stored, want the name refused")
	}
	if err := insert("INSERT INTO nodes_node (id, public_key, name, paired_at) VALUES (?, ?, 'crate', 5)", role.NodeID{UUID: storage.NewUUID()}, boxKey); err == nil {
		t.Error("a second live Node with box's key was stored, want the key refused")
	}
	if err := insert("INSERT INTO nodes_node (id, name, paired_at) VALUES (?, 'crate', 5)", role.NodeID{UUID: storage.NewUUID()}); err == nil {
		t.Error("a live Node with no key was stored, want it refused")
	}

	svc, err := nodes.Open(t.Context(), nodes.Config{Library: lib, Key: link.Key(homeKey), Name: "home", Link: unlinked{}, Owners: nodestest.Unowned(), Log: discard})
	if err != nil {
		t.Fatal(err)
	}
	list, err := svc.List(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	var got []string
	for _, n := range list {
		got = append(got, fmt.Sprintf("%s %v %v", n.Name, n.Self, roleNames(n.Roles)))
	}
	if want := []string{"home true [searching]", "box false [downloading]"}; !slices.Equal(got, want) {
		t.Fatalf("the migrated roster lists %q, want %q", got, want)
	}
}

type stmt struct {
	query string
	args  []any
}

// populate makes a library at path on the nodes schema before the roster
// migration and runs stmts in it.
func populate(t *testing.T, path string, through int64, stmts []stmt) {
	t.Helper()
	old, err := storage.OpenLibrary(t.Context(), path, []storage.Schema[storage.Library]{nodes.Schema().Through(through)}, nil, storage.DefaultTimings(), discard)
	if err != nil {
		t.Fatal(err)
	}
	err = old.Write(t.Context(), func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
		for _, st := range stmts {
			if _, err := tx.ExecContext(ctx, st.query, st.args...); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := old.Close(); err != nil {
		t.Fatal(err)
	}
}

// joined is query's single column, rows joined by commas.
func joined(t *testing.T, lib *storage.DB[storage.Library], query string) string {
	t.Helper()
	var out []string
	for _, v := range cells(t, lib, query) {
		out = append(out, string(v))
	}
	return strings.Join(out, ",")
}

func TestPruneJoinCodesKeepsAnExpiredCodeOneLifetime(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		c := startController(t)
		c.mint(t)
		time.Sleep(90 * time.Minute)
		c.mint(t)
		expiries := func() []string {
			var out []string
			for _, v := range cells(t, c.lib, "SELECT datetime(expires_at / 1000000, 'unixepoch') FROM nodes_join_code ORDER BY expires_at") {
				out = append(out, string(v))
			}
			return out
		}
		for _, step := range []struct {
			at   string
			want []string
		}{
			{"2000-01-01T02:00:00Z", []string{"2000-01-01 01:00:00", "2000-01-01 02:30:00"}},
			{"2000-01-01T02:00:01Z", []string{"2000-01-01 02:30:00"}},
		} {
			at, err := time.Parse(time.RFC3339, step.at)
			if err != nil {
				t.Fatal(err)
			}
			time.Sleep(time.Until(at))
			if err := c.svc.PruneJoinCodes(t.Context()); err != nil {
				t.Fatal(err)
			}
			if got := expiries(); !slices.Equal(got, step.want) {
				t.Fatalf("codes left after a prune at %s expire at %q, want %q", step.at, got, step.want)
			}
		}
	})
}

func TestANodesLinkIsItsSyncSessionSinceItWasLastSeen(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		c := startController(t)
		opened := time.Now()
		w := c.worker(t, "box", "127.0.0.3")
		_, id := c.pair(t, w, c.mint(t), "box")
		reach := func(node role.NodeID, link nodes.LinkState, since time.Time) {
			t.Helper()
			n, err := c.svc.Get(t.Context(), node)
			if err != nil || n.Link != link || !n.Since.Equal(since) {
				t.Errorf("node %s's link is %s since %s (%v), want %s since %s", node, n.Link, n.Since, err, link, since)
			}
		}
		synctest.Wait()
		reach(c.svc.Self(), nodes.Local, opened)
		reach(id, nodes.Down, opened)
		time.Sleep(time.Minute)
		stop := c.hold(t, w)
		synctest.Wait()
		reach(id, nodes.Up, opened.Add(time.Minute))
		time.Sleep(20 * time.Second)
		stop()
		synctest.Wait()
		lastBeat := opened.Add(time.Minute + 15*time.Second)
		reach(id, nodes.Down, lastBeat)
		if got := cells(t, c.lib, "SELECT seen_at FROM nodes_node WHERE name = 'box'"); len(got) != 1 || string(got[0]) != strconv.FormatInt(lastBeat.UnixMicro(), 10) {
			t.Fatalf("box's seen_at is %s, want its last Beat at %d", got, lastBeat.UnixMicro())
		}
		c.stop()
		time.Sleep(time.Hour)
		c.open(t, &side{})
		reach(id, nodes.Down, lastBeat)
	})
}
