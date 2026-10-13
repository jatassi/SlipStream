package sim_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
	"testing"
	"testing/synctest"
	"time"

	"github.com/jatassi/slipstream/fakes/world/sim"
	"github.com/jatassi/slipstream/internal/link"
	"github.com/jatassi/slipstream/internal/pairing"
)

// TestAMovedControllerIsFollowed restarts home, which rebinds its link on
// a new port. Once home reaches seedbox, its Hello advertises the new
// address: seedbox rewrites its record and, after home drops that
// session, re-dials home there. home records seedbox's address from the
// Welcome.
func TestAMovedControllerIsFollowed(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		w := sim.Launch(t, home(t))
		h, box := w.Node("home"), w.Node("seedbox")
		homeKey, boxKey := key(t, w, "home"), key(t, w, "seedbox")
		before := h.Ready().Listeners.Link
		h.Stop()
		h.Start()
		moved := h.Ready().Listeners.Link
		if moved == before {
			t.Fatalf("home's link stayed on %s after a restart", before)
		}

		h.Half().Link.Keep(link.Target{Key: boxKey, Addresses: addresses(t, box.Ready().Listeners.Link)})
		synctest.Wait()
		if got := controllerAddresses(t, box); !slices.Equal(got, []string{moved.String()}) {
			t.Fatalf("seedbox's record names home at %v, want [%s]", got, moved)
		}
		h.Half().Link.Drop(boxKey)
		time.Sleep(2 * time.Minute)
		if live, ok := box.Half().Link.State(homeKey).(link.Live); !ok || live.Remote != moved {
			t.Fatalf("seedbox's link to home is %+v, want live to %s", box.Half().Link.State(homeKey), moved)
		}
		r := seedbox(t, w.API())
		if want := []string{box.Ready().Listeners.Link.String()}; r.Link.State != "up" || !slices.Equal(r.Addresses, want) {
			t.Fatalf("home lists seedbox %s at %v, want up at %v", r.Link.State, r.Addresses, want)
		}
	})
}

func controllerAddresses(t *testing.T, n *sim.Node) []string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(n.DataDir().Path(), pairing.RecordFile))
	if err != nil {
		t.Fatal(err)
	}
	var r struct {
		Controller struct {
			Addresses []string `json:"addresses"`
		} `json:"controller"`
	}
	if err := json.Unmarshal(data, &r); err != nil {
		t.Fatal(err)
	}
	return r.Controller.Addresses
}
