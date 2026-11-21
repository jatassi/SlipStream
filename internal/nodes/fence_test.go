package nodes_test

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"net/netip"
	"slices"
	"sync"
	"testing"
	"testing/synctest"

	"github.com/jatassi/slipstream/internal/health"
	"github.com/jatassi/slipstream/internal/history/historytest"
	"github.com/jatassi/slipstream/internal/link"
	"github.com/jatassi/slipstream/internal/link/linktest"
	"github.com/jatassi/slipstream/internal/nodes"
	"github.com/jatassi/slipstream/internal/nodes/nodestest"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/storage"
)

// secretFence proves what it minted under its current secret, as the
// Backups service proves a proof under the fence secret.
type secretFence struct {
	mu     sync.Mutex
	secret []byte
}

func (f *secretFence) mint(old, replacement link.Key) []byte {
	f.mu.Lock()
	defer f.mu.Unlock()
	mac := hmac.New(sha256.New, f.secret)
	mac.Write(old[:])
	mac.Write(replacement[:])
	return append([]byte{0, 0, 0, 1}, mac.Sum(nil)...)
}

func (f *secretFence) Proven(old, replacement, proof []byte) bool {
	return hmac.Equal(proof, f.mint(link.Key(old), link.Key(replacement)))
}

// regenerate replaces the secret, as Resume does.
func (f *secretFence) regenerate() {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.secret = append(f.secret, 'x')
}

var restored = link.NewPrivateKey([32]byte{42}).Public()

// fenced is a Controller holding fence and a Worker paired with it that
// re-paired elsewhere, so it dials no session to the Controller.
func fenced(t *testing.T, fence *secretFence) (*controller, *linktest.Node, role.NodeID) {
	t.Helper()
	c := startController(t, func(cfg *nodes.Config) { cfg.Fence = fence })
	w := c.worker(t, "vps-relay", "127.0.0.3")
	_, id := c.pair(t, w, c.mint(t), "vps-relay")
	w.Drop(c.node.Key)
	synctest.Wait()
	return c, w, id
}

func tell(t *testing.T, from *linktest.Node, to *controller, proof []byte) {
	t.Helper()
	sup := link.Superseded{By: restored, Addresses: addresses(t, "192.168.1.20:8485"), Proof: proof}
	if err := from.Tell(t.Context(), to.node.Target(), sup); err != nil {
		t.Fatal(err)
	}
	synctest.Wait()
}

func addresses(t *testing.T, list ...string) link.Addresses {
	t.Helper()
	aps := make([]netip.AddrPort, len(list))
	for i, s := range list {
		aps[i] = netip.MustParseAddrPort(s)
	}
	a, err := link.ParseAddresses(aps)
	if err != nil {
		t.Fatal(err)
	}
	return a
}

func fenceOf(t *testing.T, c *controller) (nodes.Fence, bool) {
	t.Helper()
	var (
		f  nodes.Fence
		ok bool
	)
	if err := c.lib.Read(t.Context(), func(ctx context.Context, r storage.Reader[storage.Library]) error {
		var err error
		f, ok, err = nodes.FenceIn(ctx, r)
		return err
	}); err != nil {
		t.Fatal(err)
	}
	return f, ok
}

// unprovenItems is the Nodes Health Errors that name a Worker paired with
// another Controller.
func unprovenItems(t *testing.T, c *controller) []health.Item {
	t.Helper()
	a, err := c.svc.HealthSource().Answer(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	return slices.DeleteFunc(a.Items, func(i health.Item) bool {
		return i.Level != health.Fail || i.Subject != "vps-relay is paired with another Controller"
	})
}

func TestAProvenReplyFences(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		fence := &secretFence{secret: []byte("fence secret")}
		c, w, id := fenced(t, fence)
		tell(t, w, c, fence.mint(c.node.Key, restored))

		f, ok := fenceOf(t, c)
		if !ok || f.State != nodes.Proven || f.By != restored || f.ByNode != id || f.Addresses != addresses(t, "192.168.1.20:8485") {
			t.Fatalf("the fence is %+v (%v), want proven by %s through %s at 192.168.1.20:8485", f, ok, restored.Fingerprint(), id)
		}
		if got := c.node.State(w.Key); got != (link.Down{}) {
			t.Fatalf("the Controller sees the Worker as %+v, want Down", got)
		}
		if got := unprovenItems(t, c); len(got) != 0 {
			t.Fatalf("a proven fence raised %+v", got)
		}
		if got := historytest.Entries(t, c.lib); len(got) != 0 {
			t.Fatalf("a proven fence recorded History %+v, want none", got)
		}
		if !c.pub.has(id.String()) {
			t.Fatal("no nodes.changed for the Worker that carried the proof")
		}
	})
}

func TestAnUnprovenReplyRaisesHealthOnly(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		fence := &secretFence{secret: []byte("fence secret")}
		c, w, id := fenced(t, fence)
		tell(t, w, c, nil)

		if f, ok := fenceOf(t, c); !ok || f.State != nodes.Unproven || f.ByNode != id {
			t.Fatalf("the fence is %+v (%v), want unproven through %s", f, ok, id)
		}
		items := unprovenItems(t, c)
		want := "vps-relay now connects to a different Controller and doesn't accept this one. This Controller keeps running because vps-relay sent no proof of a restore. If you restored a Backup on another machine, stop this Controller. If you didn't, remove vps-relay."
		if len(items) != 1 || items[0].Text != want || items[0].Detail[0].Text != restored.Fingerprint() {
			t.Fatalf("Health has %+v, want one Error with %q naming the other key", items, want)
		}
		if got := historytest.Entries(t, c.lib); len(got) != 0 {
			t.Fatalf("an unproven reply recorded History %+v, want none", got)
		}
	})
}

func TestAForgedProofChangesNothingButHealth(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		fence := &secretFence{secret: []byte("fence secret")}
		c, w, _ := fenced(t, fence)
		forged := (&secretFence{secret: []byte("a guess")}).mint(c.node.Key, restored)
		tell(t, w, c, forged)

		if f, _ := fenceOf(t, c); f.State != nodes.Unproven {
			t.Fatalf("a forged proof left the fence %+v, want unproven", f)
		}
		if got := unprovenItems(t, c); len(got) != 1 {
			t.Fatalf("a forged proof raised %d unproven Errors, want 1", len(got))
		}
		tell(t, w, c, fence.mint(c.node.Key, restored))
		if f, _ := fenceOf(t, c); f.State != nodes.Proven {
			t.Fatalf("the real proof after a forged one left the fence %+v, want proven", f)
		}
	})
}

func TestAReplayedProofAfterResumeChangesNothing(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		fence := &secretFence{secret: []byte("fence secret")}
		c, w, _ := fenced(t, fence)
		proof := fence.mint(c.node.Key, restored)
		tell(t, w, c, proof)
		if err := c.lib.Write(t.Context(), nodes.ClearFence); err != nil {
			t.Fatal(err)
		}
		fence.regenerate()

		tell(t, w, c, proof)
		if f, _ := fenceOf(t, c); f.State != nodes.Unproven {
			t.Fatalf("a proof replayed after Resume left the fence %+v, want unproven", f)
		}
		if got := unprovenItems(t, c); len(got) != 1 {
			t.Fatalf("a replayed proof raised %d unproven Errors, want 1", len(got))
		}
	})
}

func TestAProofFromAnUnpinnedKeyIsIgnored(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		fence := &secretFence{secret: []byte("fence secret")}
		c := startController(t, func(cfg *nodes.Config) { cfg.Fence = fence })
		stranger := c.worker(t, "stranger", "127.0.0.5")
		tell(t, stranger, c, fence.mint(c.node.Key, restored))

		if f, ok := fenceOf(t, c); ok {
			t.Fatalf("a key the Controller doesn't pin left the fence %+v", f)
		}
		if got := unprovenItems(t, c); len(got) != 0 {
			t.Fatalf("a key the Controller doesn't pin raised %+v", got)
		}
	})
}

// lists records each superseded list a holder's node-scope intent gave.
type lists struct {
	mu   sync.Mutex
	seen [][]link.SupersededProof
}

func (l *lists) remember(ps []link.SupersededProof) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.seen = append(l.seen, ps)
}

func (l *lists) last() []link.SupersededProof {
	l.mu.Lock()
	defer l.mu.Unlock()
	if len(l.seen) == 0 {
		return nil
	}
	return l.seen[len(l.seen)-1]
}

func sameProofs(got []link.SupersededProof, want ...link.SupersededProof) bool {
	return slices.EqualFunc(got, want, func(a, b link.SupersededProof) bool { return a.Key == b.Key && bytes.Equal(a.Proof, b.Proof) })
}

func TestAKeyThatPairsLeavesTheList(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		c := startController(t)
		old := c.worker(t, "media-pc", "127.0.0.6")
		proof := bytes.Repeat([]byte{9}, 36)
		if err := c.lib.Write(t.Context(), func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
			return nodestest.Supersede(ctx, tx, c.svc.Self(), old.Key[:], proof)
		}); err != nil {
			t.Fatal(err)
		}
		w := c.worker(t, "vps-relay", "127.0.0.3")
		paired, _ := c.pair(t, w, c.mint(t), "vps-relay")
		want := link.SupersededProof{Key: old.Key, Proof: proof}
		if !sameProofs(paired.Superseded, want) {
			t.Fatalf("Paired carried %+v, want the old key with its proof", paired.Superseded)
		}
		heard := &lists{}
		w.Keep(c.node.Target())
		holder(t, role.OverLink(w.Link, c.node.Key, linktest.Timings), func(cfg *role.HoldingConfig) { cfg.Superseded = heard.remember })
		synctest.Wait()
		if got := heard.last(); !sameProofs(got, want) {
			t.Fatalf("the node-scope intent listed %+v, want the old key with its proof", got)
		}

		c.pair(t, old, c.mint(t), "media-pc")
		synctest.Wait()
		if got := heard.last(); len(got) != 0 {
			t.Fatalf("once the old key paired, the node-scope intent listed %+v, want none", got)
		}
		if again, _ := c.pair(t, c.worker(t, "laptop", "127.0.0.7"), c.mint(t), "laptop"); len(again.Superseded) != 0 {
			t.Fatalf("once the old key paired, Paired carried %+v, want none", again.Superseded)
		}
	})
}

// TestRekeyGivesTheControllerRowTheNewKey re-keys a Controller row, as the
// restore's Write does on a new machine whose key a Worker row already
// held: the Controller keeps its id, the Worker row is removed, and a Pair
// hands out the old key with its proof.
func TestRekeyGivesTheControllerRowTheNewKey(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		c := startController(t)
		w := c.worker(t, "new-machine", "127.0.0.3")
		_, id := c.pair(t, w, c.mint(t), "new-machine")
		self := c.svc.Self()
		c.stop()
		proof := bytes.Repeat([]byte{9}, 36)
		var removed nodes.ID
		if err := c.lib.Write(t.Context(), func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
			var err error
			removed, err = nodes.Rekey(ctx, tx, w.Key, c.node.Key, proof)
			return err
		}); err != nil {
			t.Fatal(err)
		}
		if removed != id {
			t.Fatalf("Rekey removed %s, want the row that held the new key, %s", removed, id)
		}
		svc, err := nodes.Open(t.Context(), nodes.Config{Library: c.lib, Key: w.Key, Name: "home", Link: unlinked{}, Owners: nodestest.Unowned(), Log: discard})
		if err != nil {
			t.Fatal(err)
		}
		if svc.Self() != self {
			t.Fatalf("under the new key the Controller is %s, want %s", svc.Self(), self)
		}
		names, err := svc.Names(t.Context())
		if err != nil {
			t.Fatal(err)
		}
		if !names[id].Removed {
			t.Fatalf("the row that held the new key is %+v, want removed", names[id])
		}
	})
}

func TestOnlyANewRowIsHeardByPaired(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		var mu sync.Mutex
		var heard []role.NodeID
		c := startController(t, func(cfg *nodes.Config) {
			cfg.Paired = func(_ context.Context, _ *storage.Tx[storage.Library], node role.NodeID) error {
				mu.Lock()
				defer mu.Unlock()
				heard = append(heard, node)
				return nil
			}
		})
		w := c.worker(t, "vps-relay", "127.0.0.3")
		_, id := c.pair(t, w, c.mint(t), "vps-relay")
		w.Drop(c.node.Key)
		synctest.Wait()
		c.pair(t, w, c.mint(t), "vps-relay")
		mu.Lock()
		defer mu.Unlock()
		if !slices.Equal(heard, []role.NodeID{id}) {
			t.Fatalf("Paired heard %v, want only the new row %s", heard, id)
		}
	})
}
