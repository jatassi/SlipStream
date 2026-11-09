package scan_test

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"
	"testing/synctest"

	"github.com/jatassi/slipstream/fakes/world/sim"
	"github.com/jatassi/slipstream/fakes/world/sim/simtest"
	"github.com/jatassi/slipstream/internal/imports"
	"github.com/jatassi/slipstream/internal/node"
	"github.com/jatassi/slipstream/internal/roots"
)

// TestAScanDuringATargetRenameChangesNothing walks the movies root, then
// renames every Target, so the disk queue renames every film the walk
// listed before the scan compares: nothing reads gone, found or pending.
func TestAScanDuringATargetRenameChangesNothing(t *testing.T) {
	t.Parallel()
	synctest.Test(t, func(t *testing.T) {
		scan := newGate()
		s := shelve(t, nil, sim.Edit(func(_ string, e *node.Env) {
			e.Test.BeforeScanStep = func(_ context.Context, _ roots.ID, p imports.Phase) {
				if p == imports.Comparing {
					scan.pass()
				}
			}
		}))
		s.stock(films...)
		s.scan()

		scan.arm()
		simtest.RunNow(t, s.w.API(), "library-scan")
		<-scan.reached
		s.renameTargets(t)
		simtest.Eventually(t, "the disk queue renames the films", func() bool {
			return s.count("SELECT count(*) FROM files_move") == 0 && s.count("SELECT count(*) FROM files_renamed") > 0
		})
		close(scan.open)
		s.idle()

		var found, gone int
		if err := s.w.Node("home").Half().Library.QueryRowContext(t.Context(), "SELECT found, gone FROM imports_root").Scan(&found, &gone); err != nil {
			t.Fatal(err)
		}
		if found != 0 || gone != 0 {
			t.Fatalf("the scan during the rename found %d files and %d gone, want none", found, gone)
		}
		if n := s.count("SELECT count(*) FROM imports_pending"); n != 0 {
			t.Fatalf("the scan during the rename left %d pending imports, want none", n)
		}
		s.expectHealth()
	})
}

// renameTargets adds " Renamed" to every Target's name.
func (s *shelf) renameTargets(t *testing.T) {
	t.Helper()
	var list struct {
		Targets []struct {
			ID string `json:"id"`
		} `json:"targets"`
	}
	reply := s.w.API().Admin("home", http.MethodGet, "/api/v1/targets", "")
	if err := json.Unmarshal([]byte(reply.Body), &list); err != nil {
		t.Fatalf("listTargets = %d %s: %v", reply.Status, reply.Body, err)
	}
	for _, target := range list.Targets {
		var held map[string]json.RawMessage
		reply := s.w.API().Admin("home", http.MethodGet, "/api/v1/targets/"+target.ID, "")
		if err := json.Unmarshal([]byte(reply.Body), &held); err != nil {
			t.Fatalf("getTarget = %d %s: %v", reply.Status, reply.Body, err)
		}
		var name string
		if err := json.Unmarshal(held["name"], &name); err != nil {
			t.Fatal(err)
		}
		body := map[string]json.RawMessage{"name": json.RawMessage(`"` + name + ` Renamed"`)}
		for _, k := range []string{"tiers", "cutoff", "filters", "sizes", "propers", "scores"} {
			if v, ok := held[k]; ok {
				body[k] = v
			}
		}
		b, err := json.Marshal(body)
		if err != nil {
			t.Fatal(err)
		}
		if put := s.w.API().Admin("home", http.MethodPut, "/api/v1/targets/"+target.ID, string(b)); put.Status != http.StatusOK {
			t.Fatalf("updateTarget = %d %s", put.Status, put.Body)
		}
	}
}
