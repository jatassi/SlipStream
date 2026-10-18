package pairing_test

import (
	"context"
	"encoding/json"
	"net/netip"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"testing/synctest"
	"time"

	"github.com/jatassi/slipstream/fakes/fault"
	"github.com/jatassi/slipstream/internal/link"
	"github.com/jatassi/slipstream/internal/link/linktest"
	"github.com/jatassi/slipstream/internal/nodes"
	"github.com/jatassi/slipstream/internal/pairing"
	"github.com/jatassi/slipstream/internal/system"
)

// TestARemovedWorkerRecordsItAndJoinsAgainAsANewNode removes box on home:
// home refuses box's next Hello as removed, so box writes the removal into
// pairing.json, stops dialing home and reads as Unclaimed. A new Join Code
// pairs it as a new row, and its record no longer says it was removed.
func TestARemovedWorkerRecordsItAndJoinsAgainAsANewNode(t *testing.T) {
	dir := t.TempDir()
	synctest.Test(t, func(t *testing.T) {
		net := fault.NewNetwork()
		home := startController(t, net, "home", "127.0.0.2", nil)
		l := &lazy{}
		box := linktest.Start(t, net, "box", []netip.Addr{linktest.WorkerAddr}, linktest.Side{Trust: l, Pair: l.Pair})
		var removals atomic.Int32
		p, err := pairing.Open(pairing.Config{
			DataDir: dir, Key: box.Key, Link: box.Link, Name: "box", Claims: claims(system.Unclaimed), AwaitDial: time.Minute,
			Removed: func(context.Context) { removals.Add(1) },
			Log:     discard,
		})
		if err != nil {
			t.Fatal(err)
		}
		l.trust, l.pair = p, p.Pair
		go func() { _ = p.Run(t.Context()) }()
		first, err := p.Claim(t.Context(), home.mint(t), ignore)
		if err != nil {
			t.Fatal(err)
		}

		if _, err := home.svc.Apply(t.Context(), nodes.Remove{Node: first.NodeID}); err != nil {
			t.Fatal(err)
		}
		synctest.Wait()
		time.Sleep(time.Minute)
		synctest.Wait()

		if home.svc.Admit(box.Key) != (link.Removed{}) {
			t.Fatalf("home admits box as %#v, want Removed", home.svc.Admit(box.Key))
		}
		if removals.Load() != 1 {
			t.Fatalf("Removed was called %d times, want once", removals.Load())
		}
		expectRemoved(t, p, dir, home.node.Key)
		if got := box.State(home.node.Key); got != (link.Down{}) {
			t.Fatalf("box's link to home is %#v, want down and no longer kept", got)
		}

		again, err := p.Claim(t.Context(), home.mint(t), ignore)
		if err != nil {
			t.Fatal(err)
		}
		if again.NodeID == first.NodeID {
			t.Fatalf("box joined again as node %s, want a new row", again.NodeID)
		}
		if _, removed := p.Removal(); removed || !p.Paired() || record(t, dir)["removed"] != nil {
			t.Fatalf("after joining again the record is %v, want paired and not removed", record(t, dir))
		}
		if home.svc.Admit(box.Key) != (link.Pinned{}) {
			t.Fatalf("home admits the re-paired box as %#v, want Pinned", home.svc.Admit(box.Key))
		}
		if _, err := home.svc.Get(t.Context(), first.NodeID); err == nil {
			t.Fatal("home still lists the removed row")
		}
	})
}

// expectRemoved checks p and its record in dir say the Controller holding
// controller removed the Node.
func expectRemoved(t *testing.T, p *pairing.Pairing, dir string, controller link.Key) {
	t.Helper()
	if p.Paired() || p.Admit(controller) != (link.Refused{}) {
	t.Fatalf("after the removal box reads paired %v and admits home as %#v, want unpaired and refused", p.Paired(), p.Admit(controller))
	}
	if worker, err := pairing.IsWorker(dir); err != nil || worker {
		t.Fatalf("IsWorker = %v, %v; want false", worker, err)
	}
	removal, ok := p.Removal()
	if !ok || removal.ControllerName != "home" {
		t.Fatalf("Removal() = %+v, %v; want removed by home", removal, ok)
	}
	if want := "This Node was removed from home on " + removal.At.Local().Format("2 Jan") + ". Join with a new Join Code to use it again."; removal.Sentence() != want {
		t.Fatalf("the sentence reads %q, want %q", removal.Sentence(), want)
	}
	if record(t, dir)["schema"] != float64(2) {
		t.Fatalf("pairing.json is %v, want schema 2", record(t, dir))
	}
}

func record(t *testing.T, dir string) map[string]any {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(dir, pairing.RecordFile))
	if err != nil {
		t.Fatal(err)
	}
	var out map[string]any
	if err := json.Unmarshal(data, &out); err != nil {
		t.Fatal(err)
	}
	return out
}
