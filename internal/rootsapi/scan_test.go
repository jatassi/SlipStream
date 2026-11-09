package rootsapi_test

import (
	"context"
	"fmt"
	"net/http"
	"regexp"
	"testing"

	"github.com/jatassi/slipstream/internal/api/ownertest"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/tasks"
)

const notScanned = `{"lastAt":null,"summary":"","stopped":null,"running":null}`

// compared records a finished comparison of root at its place: found new
// files and gone ones, or a stop at gone of all.
func (s *server) compared(t *testing.T, root string, found, gone int, stopped *[2]int) {
	t.Helper()
	id, err := storage.ParseUUID(root)
	if err != nil {
		t.Fatal(err)
	}
	var stoppedGone, stoppedOf *int
	if stopped != nil {
		stoppedGone, stoppedOf = &stopped[0], &stopped[1]
	}
	err = s.lib.Write(t.Context(), func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
		_, err := tx.ExecContext(ctx, `INSERT INTO imports_root (root_id, place, compared_at, found, gone, stopped_gone, stopped_of)
			SELECT id, place, 1700000000000000, ?, ?, ?, ? FROM roots_root WHERE id = ?
			ON CONFLICT (root_id) DO UPDATE SET compared_at = excluded.compared_at, found = excluded.found, gone = excluded.gone,
				stopped_gone = excluded.stopped_gone, stopped_of = excluded.stopped_of`, found, gone, stoppedGone, stoppedOf, id)
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
}

// scanOf is the scan member listRoots answers for the one root, its times
// masked.
func (s *server) scanOf(t *testing.T) string {
	t.Helper()
	code, reply := s.Call(t, http.MethodGet, "/api/v1/roots", "")
	if code != http.StatusOK {
		t.Fatalf("listRoots = %d %s", code, reply)
	}
	return scanMember.FindString(ownertest.Masked(reply))
}

var scanMember = regexp.MustCompile(`"scan":\{.*"running":(null|\{[^}]*\})\}`)

func TestScanNowQueuesTheRootsUnitOfLibraryScan(t *testing.T) {
	s := serve(t)
	id := s.create(t, body("fixture", t.TempDir()))
	s.Expect(t, http.MethodPost, "/api/v1/roots/"+id+"/scan", "", http.StatusAccepted, `{"runId":1}`)
	if got := fmt.Sprint(s.started); got != fmt.Sprint([]tasks.Start{{Task: "library-scan", Units: []tasks.UnitKey{tasks.UnitKey("root:" + id)}, Trigger: tasks.ByRunNow}}) {
		t.Fatalf("Scan now started %s, want the root's unit of Library scan", got)
	}
	s.Expect(t, http.MethodPost, "/api/v1/roots/"+storage.NewUUID().String()+"/scan", "", http.StatusNotFound,
		`{"status":404,"code":"root_not_found","title":"No root has that id"}`)
	if len(s.started) != 1 {
		t.Fatalf("Scan now on no root started %v, want nothing more", s.started[1:])
	}
}

func TestTheRootViewCarriesItsLastScanAndTheRunningStep(t *testing.T) {
	s := serve(t)
	id := s.create(t, body("fixture", t.TempDir()))
	if got := s.scanOf(t); got != `"scan":`+notScanned {
		t.Fatalf("a root never scanned reads %s, want no scan", got)
	}
	s.compared(t, id, 3, 1, nil)
	if got, want := s.scanOf(t), `"scan":{"lastAt":"T","summary":"It found 3 new files and 1 gone file.","stopped":null,"running":null}`; got != want {
		t.Fatalf("after a scan the root reads %s, want %s", got, want)
	}
	s.steps = map[tasks.UnitKey]tasks.Step{tasks.UnitKey("root:" + id): {Phase: "listing", Done: 1204}}
	if got, want := s.scanOf(t), `"scan":{"lastAt":"T","summary":"It found 3 new files and 1 gone file.","stopped":null,"running":{"runId":7,"phase":"listing","done":1204,"total":null}}`; got != want {
		t.Fatalf("while its unit lists the root reads %s, want %s", got, want)
	}
	total := 40
	s.steps = map[tasks.UnitKey]tasks.Step{tasks.UnitKey("root:" + id): {Phase: "probing", Done: 12, Total: &total}}
	if got, want := s.scanOf(t), `"scan":{"lastAt":"T","summary":"It found 3 new files and 1 gone file.","stopped":null,"running":{"runId":7,"phase":"probing","done":12,"total":40}}`; got != want {
		t.Fatalf("while its unit probes the root reads %s, want %s", got, want)
	}
}

func TestMarkThemGoneConfirmsOnlyAStoppedScan(t *testing.T) {
	s := serve(t)
	id := s.create(t, body("fixture", t.TempDir()))
	notStopped := `{"status":409,"code":"scan_not_stopped","title":"The root's last library scan didn't stop"}`
	s.Expect(t, http.MethodPost, "/api/v1/roots/"+id+"/gone", "", http.StatusConflict, notStopped)
	s.compared(t, id, 0, 0, nil)
	s.Expect(t, http.MethodPost, "/api/v1/roots/"+id+"/gone", "", http.StatusConflict, notStopped)
	s.Expect(t, http.MethodPost, "/api/v1/roots/"+storage.NewUUID().String()+"/gone", "", http.StatusNotFound,
		`{"status":404,"code":"root_not_found","title":"No root has that id"}`)
	if len(s.started) != 0 {
		t.Fatalf("a refused Mark them gone started %v, want nothing", s.started)
	}

	s.compared(t, id, 0, 0, &[2]int{1912, 2000})
	want := `"scan":{"lastAt":"T","summary":"Library scan stopped because 1,912 of 2,000 files are gone. Check the disk or share it is on, then scan again.","stopped":{"gone":1912,"of":2000},"running":null}`
	if got := s.scanOf(t); got != want {
		t.Fatalf("a stopped scan reads %s, want %s", got, want)
	}
	s.Expect(t, http.MethodPost, "/api/v1/roots/"+id+"/gone", "", http.StatusAccepted, `{"runId":1}`)
	if len(s.started) != 1 || fmt.Sprint(s.started[0].Units) != "[root:"+id+"]" {
		t.Fatalf("Mark them gone started %v, want the root's unit", s.started)
	}
	s.Expect(t, http.MethodPost, "/api/v1/roots/"+id+"/gone", "", http.StatusAccepted, `{"runId":2}`)
}
