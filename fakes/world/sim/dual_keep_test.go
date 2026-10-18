package sim_test

import (
	"context"
	"net"
	"net/netip"
	"sync"
	"testing"
	"testing/synctest"
	"time"

	"github.com/jatassi/slipstream/fakes/fault"
	"github.com/jatassi/slipstream/fakes/world/sim"
	"github.com/jatassi/slipstream/internal/link"
	"github.com/jatassi/slipstream/internal/node"
)

// dials counts each Node's dials by address.
type dials struct {
	mu   sync.Mutex
	seen map[string]map[string]int
}

type counting struct {
	link.Network
	name string
	d    *dials
}

func (c counting) DialContext(ctx context.Context, network, address string) (net.Conn, error) {
	c.d.mu.Lock()
	c.d.seen[c.name][address]++
	c.d.mu.Unlock()
	return c.Network.DialContext(ctx, network, address)
}

func (d *dials) to(name string, addr netip.AddrPort) int {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.seen[name][addr.String()]
}

// TestDualKeepHoldsOneSession has home keep seedbox while seedbox keeps
// home, as a Controller that dials its Workers will. Both reset the
// session at once over a path whose 300 ms each way outlasts the gap
// between their redials, so both sessions handshake together. Each side
// dials once: they settle on one session, the same on both sides, and
// keep it for a fake minute.
func TestDualKeepHoldsOneSession(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		d := &dials{seen: map[string]map[string]int{}}
		w := sim.Launch(t, home(t), sim.Edit(func(name string, e *node.Env) {
			d.mu.Lock()
			d.seen[name] = map[string]int{}
			d.mu.Unlock()
			e.Net = counting{Network: e.Net, name: name, d: d}
		}))
		homeKey, boxKey := key(t, w, "home"), key(t, w, "seedbox")
		homeAddr, boxAddr := w.Node("home").Ready().Listeners.Link, w.Node("seedbox").Ready().Listeners.Link
		homeLink, boxLink := w.Node("home").Half().Link, w.Node("seedbox").Half().Link
		for _, path := range []*fault.Plan{w.Path("home", "seedbox"), w.Path("seedbox", "home")} {
			if _, err := path.Set(fault.Rule{Fault: fault.Latency(300 * time.Millisecond)}); err != nil {
				t.Fatal(err)
			}
		}
		atHome, atBox := homeLink.Subscribe(t.Context()), boxLink.Subscribe(t.Context())
		homeLink.Keep(link.Target{Key: boxKey, Addresses: addresses(t, boxAddr)})
		synctest.Wait()
		before := [2]int{d.to("home", boxAddr), d.to("seedbox", homeAddr)}
		homeLink.Reset(boxKey)
		boxLink.Reset(homeKey)
		time.Sleep(time.Minute)
		drain(atHome)
		drain(atBox)
		if got := [2]int{d.to("home", boxAddr) - before[0], d.to("seedbox", homeAddr) - before[1]}; got != [2]int{1, 1} {
			t.Fatalf("after the reset home dialed seedbox %d times and seedbox dialed home %d, want once each", got[0], got[1])
		}

		h, hok := homeLink.State(boxKey).(link.Live)
		b, bok := boxLink.State(homeKey).(link.Live)
		if !hok || !bok || h.Local != b.Remote || h.Remote != b.Local || h.Dialed == b.Dialed {
			t.Fatalf("home sees %+v and seedbox %+v, want one live session", homeLink.State(boxKey), boxLink.State(homeKey))
		}
		time.Sleep(time.Minute)
		if got := [2][]link.Change{drain(atHome), drain(atBox)}; len(got[0])+len(got[1]) > 0 {
			t.Fatalf("after settling home saw %+v and seedbox %+v", got[0], got[1])
		}
	})
}

// drain is every Change queued on changes, once the bubble is idle.
func drain(changes <-chan link.Change) []link.Change {
	var got []link.Change
	for {
		synctest.Wait()
		select {
		case c := <-changes:
			got = append(got, c)
		default:
			return got
		}
	}
}
