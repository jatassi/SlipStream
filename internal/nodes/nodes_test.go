package nodes_test

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"crypto/sha256"
	"database/sql"
	"errors"
	"log/slog"
	"net/netip"
	"path/filepath"
	"slices"
	"strconv"
	"sync"
	"testing"
	"testing/synctest"
	"time"

	"github.com/jatassi/slipstream/fakes/fault"
	"github.com/jatassi/slipstream/internal/joincode"
	"github.com/jatassi/slipstream/internal/link"
	"github.com/jatassi/slipstream/internal/link/linktest"
	"github.com/jatassi/slipstream/internal/nodes"
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

type controller struct {
	net  *fault.Network
	node *linktest.Node
	lib  *storage.DB[storage.Library]
	svc  *nodes.Service
	pub  *published
}

func startController(t *testing.T) *controller {
	t.Helper()
	net := fault.NewNetwork()
	s := &side{}
	node := linktest.Start(t, net, "home", []netip.Addr{linktest.ControllerAddr}, linktest.Side{Trust: s, Pair: s.Pair})
	pub := &published{}
	lib := storagetest.Library(t, nodes.Schema(), role.LibrarySchema())
	svc, err := nodes.Open(t.Context(), nodes.Config{
		Library: lib,
		Key:     node.Key,
		Name:    "home",
		Link:    node.Link,
		Changed: pub,
		Log:     discard,
	})
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
	t.Cleanup(func() {
		cancel()
		<-done
	})
	return &controller{net: net, node: node, lib: lib, svc: svc, pub: pub}
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
		if _, err := c.svc.SetRoles(t.Context(), id, []role.Name{role.Downloading, role.Searching, role.Searching}); err != nil {
			t.Fatal(err)
		}
		if _, again := c.pair(t, box, c.mint(t), "renamed"); again != id {
			t.Fatalf("a new code paired the known key as %s, want %s", again, id)
		}
		got, err := c.svc.Get(t.Context(), id)
		if err != nil {
			t.Fatal(err)
		}
		if want := []role.Name{role.Searching, role.Downloading}; got.Name != "box" || !slices.Equal(got.Roles, want) {
			t.Fatalf("after a new code the known key is %q holding %v, want box holding %v", got.Name, got.Roles, want)
		}
	})
}

func TestHoldersByPairingListTheFirstPairedFirst(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		c := startController(t)
		var ids []role.NodeID
		for i, name := range []string{"box", "crate"} {
			w := c.worker(t, name, "127.0.0."+strconv.Itoa(3+i))
			_, id := c.pair(t, w, c.mint(t), name)
			w.Drop(c.node.Key)
			synctest.Wait()
			ids = append(ids, id)
			time.Sleep(time.Minute)
		}
		for _, id := range []role.NodeID{ids[1], ids[0]} {
			if _, err := c.svc.SetRoles(t.Context(), id, []role.Name{role.Searching}); err != nil {
				t.Fatal(err)
			}
		}
		holders, err := c.svc.HoldersByPairing(t.Context(), role.Searching)
		if want := []role.NodeID{c.svc.Self(), ids[0], ids[1]}; err != nil || !slices.Equal(holders, want) {
			t.Fatalf("searching holders = %v, %v; want the Controller, box, crate %v", holders, err, want)
		}
		holders, err = c.svc.HoldersByPairing(t.Context(), role.Downloading)
		if want := []role.NodeID{c.svc.Self()}; err != nil || !slices.Equal(holders, want) {
			t.Fatalf("downloading holders = %v, %v; want the Controller alone", holders, err)
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
		if !own.Self || own.Name != "home" || own.Key != c.node.Key || own.Link != nodes.Local || !slices.Equal(own.Roles, want) {
			t.Fatalf("own row = self %v, %q, %s, %s, %v; want self, home, its key, local, %v", own.Self, own.Name, own.Key.Fingerprint(), own.Link, own.Roles, want)
		}
		if _, err := c.svc.SetRoles(t.Context(), role.NodeID{UUID: storage.NewUUID()}, nil); !errors.Is(err, nodes.ErrNotFound) {
			t.Fatalf("roles for an unknown Node: %v, want ErrNotFound", err)
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

func TestAwaitInStepReturnsOnceTheNodesLinkIsLive(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		c := startController(t)
		w := c.worker(t, "box", "127.0.0.3")
		_, id := c.pair(t, w, c.mint(t), "box")
		w.Drop(c.node.Key)
		synctest.Wait()
		if err := c.svc.AwaitInStep(t.Context(), c.svc.Self()); err != nil {
			t.Fatalf("AwaitInStep(self) = %v, want nil at once", err)
		}
		ctx, cancel := context.WithTimeout(t.Context(), time.Minute)
		err := c.svc.AwaitInStep(ctx, id)
		cancel()
		if !errors.Is(err, context.DeadlineExceeded) {
			t.Fatalf("AwaitInStep(box) while down = %v, want the deadline", err)
		}
		done := make(chan error, 1)
		go func() { done <- c.svc.AwaitInStep(t.Context(), id) }()
		synctest.Wait()
		select {
		case err := <-done:
			t.Fatalf("AwaitInStep(box) returned %v before box connected", err)
		default:
		}
		w.Keep(c.node.Target())
		if err := <-done; err != nil {
			t.Fatalf("AwaitInStep(box) = %v once box connected, want nil", err)
		}
	})
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

func TestReachIsEachNodesLinkAndWhenItEnteredIt(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		c := startController(t)
		opened := time.Now()
		w := c.worker(t, "box", "127.0.0.3")
		_, id := c.pair(t, w, c.mint(t), "box")
		reach := func(node role.NodeID, link nodes.LinkState, since time.Time) {
			t.Helper()
			if got, at := c.svc.Reach(node); got != link || !at.Equal(since) {
				t.Errorf("Reach(%s) = %s since %s, want %s since %s", node, got, at, link, since)
			}
		}
		synctest.Wait()
		reach(c.svc.Self(), nodes.Local, opened)
		reach(id, nodes.Up, opened)
		reach(role.NodeID{UUID: storage.NewUUID()}, nodes.Down, opened)
		time.Sleep(time.Minute)
		w.Drop(c.node.Key)
		synctest.Wait()
		reach(id, nodes.Down, opened.Add(time.Minute))
		time.Sleep(time.Minute)
		w.Keep(c.node.Target())
		if err := c.svc.AwaitInStep(t.Context(), id); err != nil {
			t.Fatal(err)
		}
		synctest.Wait()
		reach(id, nodes.Up, opened.Add(2*time.Minute))
	})
}
