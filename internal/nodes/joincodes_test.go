package nodes_test

import (
	"context"
	"crypto/ed25519"
	"errors"
	"net/netip"
	"path/filepath"
	"slices"
	"testing"
	"testing/synctest"
	"time"

	"github.com/jatassi/slipstream/internal/joincode"
	"github.com/jatassi/slipstream/internal/link"
	"github.com/jatassi/slipstream/internal/link/linktest"
	"github.com/jatassi/slipstream/internal/nodes"
	"github.com/jatassi/slipstream/internal/nodes/nodestest"
	linkv1 "github.com/jatassi/slipstream/internal/pb/slipstream/link/v1"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/storage"
)

// nodeAddress is w's Node address at its listening addresses.
func nodeAddress(t *testing.T, w *linktest.Node) joincode.NodeAddress {
	t.Helper()
	a, err := joincode.NewNodeAddress(ed25519.PublicKey(w.Key[:]), w.Target().Addresses.All())
	if err != nil {
		t.Fatal(err)
	}
	return a
}

// await has w wait for the Controller's dial for code, as a Worker whose
// Join Code can't reach its Controller does, and pins the Controller once
// paired, as pairing does.
func await(t *testing.T, c *controller, w *linktest.Node, code nodes.JoinCode) <-chan error {
	t.Helper()
	done := make(chan error, 1)
	go func() {
		_, err := w.AwaitPair(t.Context(), code.Code, "box")
		if err == nil {
			w.Trust.Set(c.node.Key, link.Pinned{})
		}
		done <- err
	}()
	synctest.Wait()
	return done
}

func (c *controller) state(t *testing.T, id nodes.JoinCodeID) nodes.JoinCodeState {
	t.Helper()
	st, err := c.svc.JoinCodeState(t.Context(), id)
	if err != nil {
		t.Fatal(err)
	}
	return st
}

func TestAJoinCodeIsListedUntilItIsUsedOrExpires(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		c := startController(t)
		first := c.mint(t)
		time.Sleep(time.Minute)
		second := c.mint(t)
		open, err := c.svc.JoinCodes(t.Context())
		if err != nil {
			t.Fatal(err)
		}
		if len(open) != 2 || open[0].ID != first.ID || open[1].ID != second.ID || !open[1].ExpiresAt.Equal(second.ExpiresAt) {
			t.Fatalf("JoinCodes = %+v, want both codes oldest first", open)
		}
		if st := c.state(t, first.ID); st.State != nodes.CodeOpen || st.Dialing != nil {
			t.Fatalf("a fresh code reads %+v, want open with no dial", st)
		}
		_, box := c.pair(t, c.worker(t, "box", "127.0.0.3"), first, "box")
		if st := c.state(t, first.ID); st.State != nodes.CodeUsed || st.Node != box {
			t.Fatalf("a used code reads %+v, want used by %s", st, box)
		}
		time.Sleep(time.Hour)
		if st := c.state(t, second.ID); st.State != nodes.CodeExpired {
			t.Fatalf("a code past its lifetime reads %s, want expired", st.State)
		}
		if open, err := c.svc.JoinCodes(t.Context()); err != nil || len(open) != 0 {
			t.Fatalf("JoinCodes = %+v, %v; want none open", open, err)
		}
	})
}

func TestARevokedCodeIsGone(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		c := startController(t)
		code := c.mint(t)
		if err := c.svc.RevokeJoinCode(t.Context(), code.ID); err != nil {
			t.Fatal(err)
		}
		if _, err := c.svc.JoinCodeState(t.Context(), code.ID); !errors.Is(err, nodes.ErrJoinCodeNotFound) {
			t.Fatalf("JoinCodeState of a revoked code = %v, want %v", err, nodes.ErrJoinCodeNotFound)
		}
		if err := c.svc.RevokeJoinCode(t.Context(), code.ID); !errors.Is(err, nodes.ErrJoinCodeNotFound) {
			t.Fatalf("revoking twice = %v, want %v", err, nodes.ErrJoinCodeNotFound)
		}
		_, err := c.worker(t, "box", "127.0.0.3").Pair(t.Context(), code.Code, "box")
		if got := refusal(err); got != linkv1.RefusedReason_REFUSED_REASON_JOIN_CODE_UNKNOWN {
			t.Fatalf("Pair with a revoked code: %v, want REFUSED_REASON_JOIN_CODE_UNKNOWN", err)
		}
	})
}

// TestTheControllerDialsAPastedNodeAddress pastes a waiting Worker's Node
// address for its code: the Controller dials it, the Worker pairs over
// that session, and the Controller keeps dialing it at that address.
func TestTheControllerDialsAPastedNodeAddress(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		c := startController(t)
		code := c.mint(t)
		w := c.worker(t, "box", "127.0.0.3")
		done := await(t, c, w, code)
		if err := c.svc.DialForCode(t.Context(), code.ID, nodeAddress(t, w)); err != nil {
			t.Fatal(err)
		}
		if err := <-done; err != nil {
			t.Fatal(err)
		}
		synctest.Wait()
		st := c.state(t, code.ID)
		if st.State != nodes.CodeUsed || st.Dialing != nil {
			t.Fatalf("the code reads %+v after the Worker paired, want used with no dial", st)
		}
		row, err := c.svc.Get(t.Context(), st.Node)
		if err != nil {
			t.Fatal(err)
		}
		want := []netip.AddrPort{netip.MustParseAddrPort("127.0.0.3:8485")}
		if row.Dial.By != nodes.ByController || !slices.Equal(row.Dial.ControllerDials, want) {
			t.Fatalf("the Node's dial reads %s at %v, want controller at %v", row.Dial.By, row.Dial.ControllerDials, want)
		}
		c.node.Reset(w.Key)
		time.Sleep(time.Minute)
		if live, ok := c.node.State(w.Key).(link.Live); !ok || !live.Dialed {
			t.Fatalf("after its session ended the Controller sees %+v, want a session it dialed again", c.node.State(w.Key))
		}
		if err := c.svc.SetDial(t.Context(), st.Node, nil); err != nil {
			t.Fatal(err)
		}
		c.node.Reset(w.Key)
		time.Sleep(time.Minute)
		if got := c.node.State(w.Key); got != (link.Down{}) {
			t.Fatalf("with no dial set the Controller sees %+v, want Down", got)
		}
	})
}

func TestADialForAnUnreachableAddressIsReportedAfterDialReport(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		c := startController(t)
		code := c.mint(t)
		w := c.worker(t, "box", "127.0.0.3")
		pasted := nodeAddress(t, w)
		start := time.Now()
		if err := c.svc.DialForCode(t.Context(), code.ID, pasted); err != nil {
			t.Fatal(err)
		}
		time.Sleep(29 * time.Second)
		if st := c.state(t, code.ID); st.Dialing == nil || st.Dialing.Address.String() != pasted.String() || !st.Dialing.Since.Equal(start) || st.Dialing.Unreachable {
			t.Fatalf("29s after the paste the code reads %+v, want dialing %s since %s", st.Dialing, pasted, start)
		}
		time.Sleep(time.Second)
		if err := c.svc.DialForCode(t.Context(), code.ID, pasted); err != nil {
			t.Fatal(err)
		}
		if st := c.state(t, code.ID); st.Dialing == nil || !st.Dialing.Since.Equal(start) || !st.Dialing.Unreachable {
			t.Fatalf("30s after the paste, pasted again, the code reads %+v, want unreachable since %s", st.Dialing, start)
		}
		done := await(t, c, w, code)
		if err := <-done; err != nil {
			t.Fatalf("the Worker's await after the Controller kept dialing: %v", err)
		}
	})
}

func TestDialForCodeRefusals(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		c := startController(t)
		w := c.worker(t, "box", "127.0.0.3")
		used := c.mint(t)
		c.pair(t, w, used, "box")
		pasted := nodeAddress(t, w)
		if err := c.svc.DialForCode(t.Context(), used.ID, pasted); !errors.Is(err, nodes.ErrJoinCodeClosed) {
			t.Fatalf("a dial for a used code = %v, want %v", err, nodes.ErrJoinCodeClosed)
		}
		if err := c.svc.DialForCode(t.Context(), nodes.JoinCodeID{UUID: storage.NewUUID()}, pasted); !errors.Is(err, nodes.ErrJoinCodeNotFound) {
			t.Fatalf("a dial for no code = %v, want %v", err, nodes.ErrJoinCodeNotFound)
		}
		zero, err := joincode.NewNodeAddress(ed25519.PublicKey(w.Key[:]), []netip.AddrPort{netip.MustParseAddrPort("127.0.0.3:0")})
		if err != nil {
			t.Fatal(err)
		}
		if err := c.svc.DialForCode(t.Context(), c.mint(t).ID, zero); !errors.Is(err, nodes.ErrAddressUnusable) {
			t.Fatalf("a dial at port 0 = %v, want %v", err, nodes.ErrAddressUnusable)
		}
	})
}

func TestSetDialRefusesAnotherNodesAddress(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		c := startController(t)
		w := c.worker(t, "box", "127.0.0.3")
		_, id := c.pair(t, w, c.mint(t), "box")
		other := nodeAddress(t, c.worker(t, "other", "127.0.0.4"))
		if err := c.svc.SetDial(t.Context(), id, &other); !errors.Is(err, nodes.ErrNodeKeyMismatch) {
			t.Fatalf("SetDial with another Node's address = %v, want %v", err, nodes.ErrNodeKeyMismatch)
		}
		own := nodeAddress(t, w)
		if err := c.svc.SetDial(t.Context(), id, &own); err != nil {
			t.Fatal(err)
		}
		row, err := c.svc.Get(t.Context(), id)
		if err != nil {
			t.Fatal(err)
		}
		if want := own.Addresses(); !slices.Equal(row.Dial.ControllerDials, want) {
			t.Fatalf("the Controller dials %v, want %v", row.Dial.ControllerDials, want)
		}
		if err := c.svc.SetDial(t.Context(), role.NodeID{UUID: storage.NewUUID()}, nil); !errors.Is(err, nodes.ErrNotFound) {
			t.Fatalf("SetDial on no Node = %v, want %v", err, nodes.ErrNotFound)
		}
	})
}

// TestTheJoinCodeMigrationKeepsEveryCode migrates a library holding a
// used and an open code: each gets an id and keeps what Pair reads.
func TestTheJoinCodeMigrationKeepsEveryCode(t *testing.T) {
	path := filepath.Join(t.TempDir(), storage.LibraryFile)
	box := role.NodeID{UUID: storage.NewUUID()}
	populate(t, path, 3, []stmt{
		{"INSERT INTO nodes_node (id, public_key, name, paired_at) VALUES (?, ?, 'box', 1)", []any{box, slices.Repeat([]byte{2}, 32)}},
		{"INSERT INTO nodes_join_code (secret_hash, created_at, expires_at, used_by, used_at) VALUES (?, 1, 2, ?, 2), (?, 3, 4, NULL, NULL)", []any{slices.Repeat([]byte{3}, 32), box, slices.Repeat([]byte{4}, 32)}},
	})
	lib, err := storage.OpenLibrary(t.Context(), path, []storage.Schema[storage.Library]{nodes.Schema(), role.LibrarySchema()}, nil, storage.DefaultTimings(), discard)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = lib.Close() })
	query := "SELECT hex(substr(c.secret_hash, 1, 1)) || ' ' || length(c.id) || ' ' || c.created_at || ' ' || ifnull(n.name, '-') || ' ' || ifnull(c.dial_address, '-') FROM nodes_join_code c LEFT JOIN nodes_node n ON n.id = c.used_by ORDER BY c.created_at"
	if got, want := joined(t, lib, query), "03 16 1 box -,04 16 3 - -"; got != want {
		t.Errorf("%s\n= %q, want %q", query, got, want)
	}
	if ids := cells(t, lib, "SELECT DISTINCT id FROM nodes_join_code"); len(ids) != 2 {
		t.Errorf("the codes hold %d distinct ids, want 2", len(ids))
	}
	svc, err := nodes.Open(t.Context(), nodes.Config{Library: lib, Key: link.Key(slices.Repeat([]byte{1}, 32)), Name: "home", Link: unlinked{}, Owners: nodestest.Unowned(), Log: discard})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.JoinCodes(context.Background()); err != nil {
		t.Fatal(err)
	}
}
