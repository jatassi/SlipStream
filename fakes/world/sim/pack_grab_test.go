package sim_test

import (
	"net/http"
	"strings"
	"testing"
	"testing/synctest"
	"time"

	"github.com/jatassi/slipstream/fakes/world/sim"
	"github.com/jatassi/slipstream/fakes/world/sim/simtest"
	"github.com/jatassi/slipstream/internal/node"
)

// searchItem searches the Item of the Work whose key is key, alone.
func (c crossNode) searchItem(t *testing.T, key string) {
	t.Helper()
	work := simtest.Decode[struct {
		Items []struct {
			ID  string `json:"id"`
			Key string `json:"key"`
		} `json:"items"`
	}](t, "getWork", c.w.API().Admin("home", http.MethodGet, "/api/v1/works/"+c.work, ""))
	for _, it := range work.Items {
		if it.Key == key {
			body := `{"levelIds":[],"itemIds":["` + it.ID + `"]}`
			if r := c.w.API().Admin("home", http.MethodPost, "/api/v1/works/"+c.work+"/search", body); r.Status != http.StatusAccepted {
				t.Fatalf("searchWork %s = %d %s, want 202", key, r.Status, r.Body)
			}
			return
		}
	}
	t.Fatalf("The Copper Coast has no Item %s", key)
}

// grabbed waits for the one Grab's Import and answers its id.
func (c crossNode) grabbed(t *testing.T) string {
	t.Helper()
	for range simtest.Settled / time.Second {
		if ims := grabImports(t, c.w); len(ims) == 1 {
			return ims[0].ID
		}
		time.Sleep(time.Second)
	}
	t.Fatalf("no Grab's Import opened: %+v", simtest.GrabsListed(t, c.w))
	return ""
}

// TestAGrabbedPackLandsInAnyOrder searches episode 1 of The Copper Coast,
// with episode 4 already in home's TV root. The search grabs the season
// pack, which holds every open cell of its episodes. Its Import sends only
// the nine files with cells, episode 4 is not better, and each lands on
// its own, in any order. The Grab reads imported once the last lands.
func TestAGrabbedPackLandsInAnyOrder(t *testing.T) {
	t.Parallel()
	synctest.Test(t, func(t *testing.T) {
		p, l := parkedAt(3), newLands()
		c := launchCrossNode(t, func(e *node.Env) { e.Test.Transfers.Sent = p.sent }, sim.Edit(func(name string, e *node.Env) {
			if name == "home" {
				e.Test.BeforeImport = l.before
			}
		}))
		t.Cleanup(func() {
			l.open()
			select {
			case <-p.release:
			default:
				close(p.release)
			}
		})
		c.importHome(t)
		c.searchItem(t, "110111")
		id := c.grabbed(t)
		<-p.reached
		v := c.await(t, id, "the first file to arrive lands", func(v importView) bool { return v.count("placed") == 1 })
		if v.count("sending") != 8 || !strings.HasSuffix(v.outcomes()[3], episodeName(4)+": not_better kept_better") {
			t.Fatalf("once the first file landed the files read %q, want episode 4 not better and the other eight still sending", v.outcomes())
		}
		l.arm(id)
		close(p.release)
		c.landInTurn(t, id, l, []int{6, 0, 9, 2, 4, 8, 1, 7, 5})
		if got := c.library(t); len(got) != 10 {
			t.Fatalf("home's TV root holds %q, want the ten episodes", got)
		}
		g := simtest.Await(t, c.w)
		simtest.ExpectImported(t, g)
	})
}
