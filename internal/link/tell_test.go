package link_test

import (
	"bytes"
	"net/netip"
	"sync"
	"testing"
	"testing/synctest"
	"time"

	"github.com/jatassi/slipstream/fakes/fault"
	"github.com/jatassi/slipstream/internal/link"
	"github.com/jatassi/slipstream/internal/link/linktest"
)

// hearing records each superseded claim a Node's link hears.
type hearing struct {
	mu    sync.Mutex
	heard []claim
}

type claim struct {
	from link.Key
	s    link.Superseded
}

func (h *hearing) told(from link.Key, s link.Superseded) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.heard = append(h.heard, claim{from, s})
}

func (h *hearing) all() []claim {
	h.mu.Lock()
	defer h.mu.Unlock()
	return append([]claim(nil), h.heard...)
}

func sameClaim(c claim, from link.Key, want link.Superseded) bool {
	return c.from == from && c.s.By == want.By && c.s.Addresses == want.Addresses && bytes.Equal(c.s.Proof, want.Proof)
}

// An old Controller that dials its Worker hears the Worker's answer with
// the key that replaced it and the proof, and never goes live.
func TestTheOldControllerDialingGetsTheProof(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		n := fault.NewNetwork()
		heard := &hearing{}
		worker := linktest.Start(t, n, "worker", []netip.Addr{linktest.WorkerAddr}, linktest.Side{})
		old := linktest.Start(t, n, "controller", []netip.Addr{linktest.ControllerAddr}, linktest.Side{Told: heard.told})
		answer := link.Superseded{By: link.NewPrivateKey([32]byte{7}).Public(), Addresses: addresses(t, "127.0.0.9:8485"), Proof: []byte("the proof")}
		worker.Trust.Set(old.Key, answer)
		old.Trust.Set(worker.Key, link.Pinned{})
		old.Keep(worker.Target())
		synctest.Wait()

		got := heard.all()
		if len(got) != 1 || !sameClaim(got[0], worker.Key, answer) {
			t.Fatalf("the old Controller heard %+v, want one claim from the Worker carrying %+v", got, answer)
		}
		if got := old.State(worker.Key); got != (link.Down{}) {
			t.Fatalf("the old Controller sees %+v, want Down", got)
		}
	})
}

// A Worker's Tell reaches the Controller it names, which hears the claim
// from the pinned key and refuses the session, so neither side goes live
// and Tell returns once answered.
func TestATellIsHeardAndNeverGoesLive(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		heard := &hearing{}
		p := linktest.New(t, linktest.Side{Told: heard.told}, linktest.Side{})
		atController := p.Controller.Subscribe(t.Context())
		sup := link.Superseded{By: link.NewPrivateKey([32]byte{7}).Public(), Addresses: addresses(t, "127.0.0.9:8485"), Proof: []byte("the proof")}
		if err := p.Worker.Tell(t.Context(), p.Controller.Target(), sup); err != nil {
			t.Fatal(err)
		}
		synctest.Wait()

		got := heard.all()
		if len(got) != 1 || !sameClaim(got[0], p.Worker.Key, sup) {
			t.Fatalf("the Controller heard %+v, want one claim from the Worker carrying %+v", got, sup)
		}
		select {
		case c := <-atController:
			t.Fatalf("the Controller saw %+v after a Tell", c)
		case <-time.After(time.Minute):
		}
		if p.Controller.State(p.Worker.Key) != (link.Down{}) || p.Worker.State(p.Controller.Key) != (link.Down{}) {
			t.Fatal("a Tell left a session live")
		}
	})
}

// A Tell to a Node that doesn't pin the teller is answered as an unknown
// key, and its Node hears nothing.
func TestATellFromAnUnpinnedKeyIsNotHeard(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		heard := &hearing{}
		n := fault.NewNetwork()
		controller := linktest.Start(t, n, "controller", []netip.Addr{linktest.ControllerAddr}, linktest.Side{Told: heard.told})
		stranger := linktest.Start(t, n, "stranger", []netip.Addr{linktest.WorkerAddr}, linktest.Side{})
		controller.Trust.Set(stranger.Key, link.Pairing{})
		if err := stranger.Tell(t.Context(), controller.Target(), link.Superseded{By: stranger.Key, Proof: []byte("forged")}); err != nil {
			t.Fatal(err)
		}
		synctest.Wait()
		if got := heard.all(); len(got) != 0 {
			t.Fatalf("the Controller heard %+v from a key it doesn't pin", got)
		}
	})
}
