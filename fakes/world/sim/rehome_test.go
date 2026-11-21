package sim_test

import (
	"net/http"
	"path/filepath"
	"slices"
	"testing"
	"testing/synctest"
	"time"

	"github.com/jatassi/slipstream/fakes/world/sim"
	"github.com/jatassi/slipstream/fakes/world/sim/simtest"
)

type reconcilingPage struct {
	Restore *struct {
		Backup     string `json:"backup"`
		NewMachine bool   `json:"newMachine"`
		Rehome     bool   `json:"rehome"`
	} `json:"restore"`
	Nodes []reconcilingNode `json:"nodes"`
	Roots []struct {
		ID      string `json:"id"`
		Label   string `json:"label"`
		Saved   bool   `json:"saved"`
		Scanned bool   `json:"scanned"`
	} `json:"roots"`
}

type reconcilingNode struct {
	Name    string  `json:"name"`
	Adopted bool    `json:"adopted"`
	Granted *string `json:"granted"`
	Link    string  `json:"link"`
}

func readReconciling(t *testing.T, w *sim.World) reconcilingPage {
	t.Helper()
	return simtest.Decode[reconcilingPage](t, "getReconciling", w.API().Admin("home", http.MethodGet, "/api/v1/reconciling", ""))
}

// restoredHome launches home with its Movies root scanned, then begins
// Reconciling for a restore onto a new machine, and returns the root.
func restoredHome(t *testing.T) (*sim.World, simtest.RootRow) {
	t.Helper()
	w := sim.Launch(t, simtest.Home(t), sim.WithWalk())
	root := simtest.RootAt(t, w.API(), filepath.Join(w.Manifest().MediaDir, "movies"))
	simtest.Eventually(t, "the Movies root's first scan finishes", func() bool {
		return simtest.RootAt(t, w.API(), root.Path).Scan.Summary != ""
	})
	beginRestore(t, w)
	p := readReconciling(t, w)
	if p.Restore == nil || p.Restore.Backup != "nightly" || !p.Restore.NewMachine || !p.Restore.Rehome {
		t.Fatalf("getReconciling's restore = %+v, want the nightly Backup on a new machine, its folders not found", p.Restore)
	}
	if len(p.Roots) != 1 || p.Roots[0].ID != root.ID || p.Roots[0].Saved || p.Roots[0].Label == "" {
		t.Fatalf("getReconciling's roots = %+v, want the Movies root, labeled and not saved", p.Roots)
	}
	if !slices.ContainsFunc(p.Nodes, func(n reconcilingNode) bool {
		return n.Name == "seedbox" && n.Link != ""
	}) {
		t.Fatalf("getReconciling's nodes = %+v, want seedbox with its link", p.Nodes)
	}
	return w, root
}

// TestLaterLeavesARootUnscanned leaves home's Movies root for later in
// the re-home step: a scan of it is skipped and Reconciling still waits
// on it.
func TestLaterLeavesARootUnscanned(t *testing.T) {
	t.Parallel()
	synctest.Test(t, func(t *testing.T) {
		w, root := restoredHome(t)
		a := w.API()
		if r := a.Admin("home", http.MethodPost, "/api/v1/reconciling/rehome", `{"roots":[],"done":true}`); r.Status != http.StatusOK {
			t.Fatalf("rehomeReconciling with no root = %d %s, want 200", r.Status, r.Body)
		}
		if r := a.Admin("home", http.MethodPost, "/api/v1/roots/"+root.ID+"/scan", ""); r.Status != http.StatusAccepted {
			t.Fatalf("scanning the Movies root = %d %s, want 202", r.Status, r.Body)
		}
		time.Sleep(simtest.Settled)
		synctest.Wait()
		var cause string
		if err := w.Node("home").Half().Library.QueryRowContext(t.Context(), "SELECT coalesce(last_cause, '') FROM tasks_unit WHERE task_id = 'library-scan'").Scan(&cause); err != nil {
			t.Fatal(err)
		}
		if want := "Skipped because this library root's folder isn't found yet after the restore. Give its path on the Reconciling page, and SlipStream scans it."; cause != want {
			t.Errorf("the Movies root's scan ended %q, want %q", cause, want)
		}
		p := readReconciling(t, w)
		if p.Restore == nil || p.Restore.Rehome || p.Roots[0].Saved || p.Roots[0].Scanned {
			t.Errorf("after Later and a scan, getReconciling = %+v %+v, want the step done and the root neither saved nor scanned", p.Restore, p.Roots)
		}
	})
}

// TestSaveAllGoesThroughRehome saves home's Movies root where it is: the
// root edit's re-home checks it, its scan starts and finishes, and
// Reconciling stops waiting on it.
func TestSaveAllGoesThroughRehome(t *testing.T) {
	t.Parallel()
	synctest.Test(t, func(t *testing.T) {
		w, root := restoredHome(t)
		r := w.API().Admin("home", http.MethodPost, "/api/v1/reconciling/rehome", `{"roots":[{"rootId":"`+root.ID+`"}],"done":true}`)
		saved := simtest.Decode[struct {
			Roots []struct {
				RootID  string  `json:"rootId"`
				Refused *string `json:"refused"`
			} `json:"roots"`
		}](t, "rehomeReconciling", r).Roots
		if len(saved) != 1 || saved[0].RootID != root.ID || saved[0].Refused != nil {
			t.Fatalf("rehomeReconciling = %s, want the Movies root saved", r.Body)
		}
		simtest.Eventually(t, "the Movies root's scan finishes for Reconciling", func() bool {
			p := readReconciling(t, w)
			return len(p.Roots) == 0 || p.Roots[0].Saved && p.Roots[0].Scanned
		})
		if p := readReconciling(t, w); p.Restore != nil && p.Restore.Rehome {
			t.Errorf("after Save all, getReconciling's restore = %+v, want its folders found", p.Restore)
		}
	})
}
