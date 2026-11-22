package nodes_test

import (
	"bytes"
	"context"
	"net/netip"
	"testing"
	"testing/synctest"
	"time"

	"github.com/jatassi/slipstream/internal/link"
	"github.com/jatassi/slipstream/internal/link/linktest"
	"github.com/jatassi/slipstream/internal/nodes/nodestest"
	"github.com/jatassi/slipstream/internal/pairing"
	linkv1 "github.com/jatassi/slipstream/internal/pb/slipstream/link/v1"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/system"
)

type unclaimed struct{}

func (unclaimed) Claim(context.Context) (system.Claim, error) { return system.Unclaimed, nil }

// pairingSide is a Worker link's Trust and Pair, answered by its pairing
// once it is open.
type pairingSide struct{ p *pairing.Pairing }

func (s *pairingSide) Admit(key link.Key) link.Admission { return s.p.Admit(key) }

func (s *pairingSide) Pair(ctx context.Context, peer link.Key, req *linkv1.PairRequest) (*linkv1.PairResponse, error) {
	return s.p.Pair(ctx, peer, req)
}

// join is vps-relay on version, joined to c through its own pairing.json.
func (c *controller) join(t *testing.T, version string) (w *linktest.Node, p *pairing.Pairing, dir string) {
	t.Helper()
	s := &pairingSide{}
	w = linktest.Start(t, c.net, "vps-relay", []netip.Addr{netip.MustParseAddr("127.0.0.3")}, linktest.Side{Version: version, Trust: s, Pair: s.Pair})
	dir = t.TempDir()
	p, err := pairing.Open(pairing.Config{DataDir: dir, Key: w.Key, Link: w.Link, Name: "vps-relay", Claims: unclaimed{}, AwaitDial: time.Minute, Log: discard})
	if err != nil {
		t.Fatal(err)
	}
	s.p = p
	if _, err := p.Claim(t.Context(), c.mint(t).Code, func(pairing.Pending) {}); err != nil {
		t.Fatal(err)
	}
	return w, p, dir
}

func (c *controller) supersede(t *testing.T, old link.Key, proof []byte) {
	t.Helper()
	if err := c.lib.Write(t.Context(), func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
		return nodestest.Supersede(ctx, tx, c.svc.Self(), old[:], proof)
	}); err != nil {
		t.Fatal(err)
	}
}

func (l *lists) count() int {
	l.mu.Lock()
	defer l.mu.Unlock()
	return len(l.seen)
}

var replaced = link.NewPrivateKey([32]byte{7}).Public()

// holdsProof is whether rec lists old as replaced by by with proof, and p
// answers old's dial with that proof.
func holdsProof(p *pairing.Pairing, old, by link.Key, proof []byte) bool {
	rec, _ := p.Record()
	for _, s := range rec.Superseded {
		if s.Key == old && s.By == by && bytes.Equal(s.Proof, proof) {
			sup, ok := p.Admit(old).(link.Superseded)
			return ok && sup.By == by && bytes.Equal(sup.Proof, proof)
		}
	}
	return false
}

// TestTheNodeScopeIntentCarriesEachSupersededKeyToTheWorkersRecord
// supersedes a key after vps-relay paired, so only the node-scope intent
// names it: the Worker's holder hands the list to its pairing, which
// writes it into pairing.json and answers the old key with the proof.
func TestTheNodeScopeIntentCarriesEachSupersededKeyToTheWorkersRecord(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		c := startController(t)
		w, p, dir := c.join(t, "")
		proof := bytes.Repeat([]byte{9}, 36)
		c.supersede(t, replaced, proof)
		if holdsProof(p, replaced, c.node.Key, proof) {
			t.Fatal("the Worker holds the proof before any intent named it")
		}
		holder(t, role.OverLink(w.Link, c.node.Key, linktest.Timings), func(cfg *role.HoldingConfig) { cfg.Superseded = p.Remember })
		synctest.Wait()
		if !holdsProof(p, replaced, c.node.Key, proof) {
			rec, _ := p.Record()
			t.Fatalf("after the node-scope intent pairing.json lists %+v, want the superseded key with its proof", rec.Superseded)
		}
		reopened, err := pairing.Open(pairing.Config{DataDir: dir, Log: discard})
		if err != nil {
			t.Fatal(err)
		}
		if !holdsProof(reopened, replaced, c.node.Key, proof) {
			t.Fatal("pairing.json read back without the proof")
		}
	})
}

// TestAMismatchedWorkerGetsTheProofFromPaired pairs a Worker on another
// build version: its holder gets no intents, and its record still holds
// the proof the Pair answer carried.
func TestAMismatchedWorkerGetsTheProofFromPaired(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		c := startController(t)
		proof := bytes.Repeat([]byte{9}, 36)
		c.supersede(t, replaced, proof)
		w, p, _ := c.join(t, "0.0.1-other")
		heard := &lists{}
		holder(t, role.OverLink(w.Link, c.node.Key, linktest.Timings), func(cfg *role.HoldingConfig) { cfg.Superseded = heard.remember })
		synctest.Wait()
		if n := heard.count(); n != 0 {
			t.Fatalf("a mismatched Worker got %d node-scope intents, want none", n)
		}
		if !holdsProof(p, replaced, c.node.Key, proof) {
			rec, _ := p.Record()
			t.Fatalf("the mismatched Worker's record lists %+v, want the superseded key with its proof", rec.Superseded)
		}
	})
}
