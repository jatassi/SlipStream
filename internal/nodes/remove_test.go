package nodes_test

import (
	"bytes"
	"context"
	"errors"
	"math/rand/v2"
	"net/netip"
	"slices"
	"sync"
	"testing"
	"testing/synctest"
	"time"

	"github.com/jatassi/slipstream/internal/link"
	"github.com/jatassi/slipstream/internal/nodes"
	linkv1 "github.com/jatassi/slipstream/internal/pb/slipstream/link/v1"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/storage/storagetest"
)

// TestRemovingANodeLeavesATombstone removes box while it alone downloads:
// Downloading hands back to home, and box's row stays as a tombstone that
// keeps its old key and frees its name. Home refuses the key's Hello as
// removed and lets it pair again as a new row.
func TestRemovingANodeLeavesATombstone(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		c := startController(t)
		w := c.worker(t, "box", "127.0.0.3")
		_, box := c.pair(t, w, c.mint(t), "box")
		c.hold(t, w)
		holder(t, role.InProcess(c.svc.Controls()))
		synctest.Wait()
		own := role.Of(role.Names()...).Without(role.Relaying)
		c.apply(t, nodes.Roles{box: role.Of(role.Downloading), c.svc.Self(): own.Without(role.Downloading)})

		out, err := c.svc.Apply(t.Context(), nodes.Remove{Node: box})
		if err != nil {
			t.Fatal(err)
		}
		wantWords(t, out, "home takes over Downloading, since SlipStream always needs a Node that downloads.",
			"box is removed. It can join again with a new Join Code.")
		if got := c.rolesOf(t, c.svc.Self()); !slices.Contains(got, role.Downloading) {
			t.Fatalf("home holds %v after the removal, want Downloading", got)
		}
		expectTombstone(t, c, box, w.Key)
		if c.svc.Admit(w.Key) != (link.Removed{}) {
			t.Fatalf("home admits box's key as %#v, want Removed", c.svc.Admit(w.Key))
		}
		w.Keep(link.Target{Key: c.node.Key, Addresses: mustAddresses(t, "127.0.0.2:8485")})
		time.Sleep(time.Minute)
		synctest.Wait()
		if got := w.State(c.node.Key); got != (link.Down{Refused: linkv1.RefusedReason_REFUSED_REASON_REMOVED}) {
			t.Fatalf("box's kept link to home reads %#v, want refused as removed", got)
		}
		w.Drop(c.node.Key)

		other := c.worker(t, "other", "127.0.0.4")
		if paired, _ := c.pair(t, other, c.mint(t), "box"); paired.Name != "box" {
			t.Fatalf("a new Node asking for box was named %q, want box, which the tombstone freed", paired.Name)
		}
		_, again := c.pair(t, w, c.mint(t), "box")
		if again == box || c.svc.Admit(w.Key) != (link.Pinned{}) {
			t.Fatalf("box paired again as %s and is admitted as %#v, want a new row, pinned", again, c.svc.Admit(w.Key))
		}
	})
}

// TestARenameReachesTheNodeInItsIntent renames box while it is in step:
// its holder half has the new name when Rename answers. A name another
// Node holds or one that isn't a name is refused, and a renamed
// Controller pairs Nodes under its new name.
func TestARenameReachesTheNodeInItsIntent(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		c := startController(t)
		w := c.worker(t, "box", "127.0.0.3")
		_, box := c.pair(t, w, c.mint(t), "box")
		h := named(t, role.OverLink(w.Link, c.node.Key))
		holder(t, role.InProcess(c.svc.Controls()))
		synctest.Wait()
		if h.Name() != "box" {
			t.Fatalf("box's intent names it %q, want box", h.Name())
		}
		n, err := c.svc.Rename(t.Context(), box, "seedbox")
		if err != nil || n.Name != "seedbox" || h.Name() != "seedbox" {
			t.Fatalf("Rename = %+v, %v with box's intent naming it %q; want seedbox everywhere", n.Name, err, h.Name())
		}
		if _, err := c.svc.Rename(t.Context(), box, "home"); !errors.Is(err, nodes.ErrNameTaken) {
			t.Fatalf("renaming box to home = %v, want ErrNameTaken", err)
		}
		if _, err := c.svc.Rename(t.Context(), box, "seedbox"); err != nil {
			t.Fatalf("renaming box to its own name = %v, want it kept", err)
		}
		if _, err := c.svc.Rename(t.Context(), box, "seedbox "); !errors.Is(err, nodes.ErrNameInvalid) {
			t.Fatalf("a name ending in a space = %v, want ErrNameInvalid", err)
		}
		if _, err := c.svc.Rename(t.Context(), c.svc.Self(), "attic"); err != nil {
			t.Fatal(err)
		}
		if paired, _ := c.pair(t, c.worker(t, "other", "127.0.0.4"), c.mint(t), "other"); paired.ControllerName != "attic" {
			t.Fatalf("a Node paired with the renamed Controller as %q, want attic", paired.ControllerName)
		}
	})
}

// expectTombstone checks box's row is a tombstone that holds key and no
// Roles, named only in the history of Nodes, with its work written off.
func expectTombstone(t *testing.T, c *controller, box nodes.ID, key link.Key) {
	t.Helper()
	if _, err := c.svc.Get(t.Context(), box); !errors.Is(err, nodes.ErrNotFound) {
		t.Fatalf("Get(box) = %v, want ErrNotFound", err)
	}
	row := cells(t, c.lib, "SELECT public_key IS NULL, removed_at IS NOT NULL, removed_key, (SELECT count(*) FROM nodes_role WHERE node_id = nodes_node.id) FROM nodes_node WHERE name = 'box'")
	if len(row) != 4 || string(row[0]) != "1" || string(row[1]) != "1" || !bytes.Equal(row[2], key[:]) || string(row[3]) != "0" {
		t.Fatalf("box's row reads %q, want a tombstone holding its key and no Roles", row)
	}
	names, err := nodes.NamesOf(t.Context(), c.lib)
	if err != nil {
		t.Fatal(err)
	}
	if n := names[box]; n.Name != "box" || !n.Removed {
		t.Fatalf("NamesOf has box as %+v, want it named and removed", n)
	}
	live, err := nodes.Names(t.Context(), c.lib)
	if err != nil || len(live) != 1 {
		t.Fatalf("Names = %+v, %v; want only home", live, err)
	}
	if !c.svc.WrittenOff(role.Downloading, box) {
		t.Fatal("box's Downloading reads as kept once box is removed")
	}

}

// named runs a holder half toward upstream until the test ends and returns
// it.
func named(t *testing.T, upstream role.Upstream) *role.Holding {
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
	var wg sync.WaitGroup
	wg.Go(func() { _ = h.Run(ctx) })
	t.Cleanup(func() {
		cancel()
		wg.Wait()
	})
	return h
}

func mustAddresses(t *testing.T, addrs ...string) link.Addresses {
	t.Helper()
	aps := make([]netip.AddrPort, len(addrs))
	for i, a := range addrs {
		aps[i] = netip.MustParseAddrPort(a)
	}
	out, err := link.ParseAddresses(aps)
	if err != nil {
		t.Fatal(err)
	}
	return out
}
