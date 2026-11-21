package pairing_test

import (
	"sync"
	"testing"
	"testing/synctest"
	"time"

	"github.com/jatassi/slipstream/fakes/fault"
	"github.com/jatassi/slipstream/internal/link/linktest"
	"github.com/jatassi/slipstream/internal/system"
)

// localPage records each Open and Shut, in order.
type localPage struct {
	mu    sync.Mutex
	calls []string
}

func (l *localPage) Open() { l.note("open") }
func (l *localPage) Shut() { l.note("shut") }

func (l *localPage) note(s string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.calls = append(l.calls, s)
}

func (l *localPage) seen() []string {
	l.mu.Lock()
	defer l.mu.Unlock()
	return append([]string(nil), l.calls...)
}

// TestTheLocalPageOpensOnlyAfterUnreachableAfter pairs box with home,
// then cuts the network: box's local page stays shut before
// unreachable_after and opens once it passes. Healed, box reaches home
// again and the page shuts.
func TestTheLocalPageOpensOnlyAfterUnreachableAfter(t *testing.T) {
	dir := t.TempDir()
	synctest.Test(t, func(t *testing.T) {
		net := fault.NewNetwork()
		home := startController(t, net, "home", "127.0.0.2", nil)
		page := &localPage{}
		_, p := workerWith(t, net, dir, system.Unclaimed, page)
		go func() { _ = p.Run(t.Context()) }()
		if _, err := p.Claim(t.Context(), home.mint(t), ignore); err != nil {
			t.Fatal(err)
		}
		synctest.Wait()
		if got := page.seen(); len(got) != 0 {
			t.Fatalf("a reached Controller left the page %v, want untouched", got)
		}

		path := net.Path(linktest.ControllerAddr, linktest.WorkerAddr)
		if _, err := path.Set(fault.Rule{Fault: fault.Stall{}}); err != nil {
			t.Fatal(err)
		}
		time.Sleep(14 * time.Minute)
		synctest.Wait()
		if got := page.seen(); len(got) != 0 {
			t.Fatalf("14 minutes out of reach the page saw %v, want still shut", got)
		}
		time.Sleep(5 * time.Minute)
		synctest.Wait()
		if got := page.seen(); len(got) != 1 || got[0] != "open" {
			t.Fatalf("19 minutes out of reach the page saw %v, want it opened", got)
		}

		path.Clear("")
		time.Sleep(10 * time.Minute)
		synctest.Wait()
		if got := page.seen(); len(got) != 2 || got[1] != "shut" {
			t.Fatalf("healed, the page saw %v, want it shut", got)
		}
	})
}
