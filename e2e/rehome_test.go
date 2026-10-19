package e2e

import (
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/jatassi/slipstream/fakes/world/worldtest"
	"github.com/jatassi/slipstream/internal/storing"
)

type rootView struct {
	ID        string `json:"id"`
	Path      string `json:"path"`
	State     string `json:"state"`
	FileBytes int64  `json:"fileBytes"`
}

func TestARootWithAnImportedFileIsRehomedThenDeleted(t *testing.T) {
	t.Parallel()
	w := walk(t, "single-machine")
	listRoots := func() []rootView {
		var out struct {
			Roots []rootView `json:"roots"`
		}
		decode(t, "listRoots", w.a.Admin("home", http.MethodGet, "/api/v1/roots", ""), &out)
		return out.Roots
	}
	roots := listRoots()
	if len(roots) != 1 || roots[0].Path != filepath.Join(w.m.MediaDir, "movies") {
		t.Fatalf("listRoots = %+v, want the movies root alone", roots)
	}
	root := roots[0]
	rel := strings.TrimPrefix(placed, "movies/")
	bytes, err := os.ReadFile(filepath.Join(w.m.MediaDir, filepath.FromSlash(placed)))
	if err != nil {
		t.Fatal(err)
	}

	moved := filepath.Join(w.m.MediaDir, "moved")
	if err := os.MkdirAll(filepath.Join(moved, filepath.Dir(filepath.FromSlash(rel))), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(moved, filepath.FromSlash(rel)), bytes, 0o600); err != nil {
		t.Fatal(err)
	}
	patch := func(body string) worldtest.Reply {
		return w.a.Admin("home", http.MethodPatch, "/api/v1/roots/"+root.ID, body)
	}
	if r := patch(`{"path":` + strconv.Quote(moved) + `}`); r.Status != http.StatusOK {
		t.Fatalf("rehomeRoot onto a copy = %d %s, want 200", r.Status, r.Body)
	}
	worldtest.Until(t, "the re-homed root is ready at its new folder", func() (bool, error) {
		r := listRoots()[0]
		return r.Path == moved && r.State == "ready", nil
	})
	if marker, err := os.ReadFile(filepath.Join(moved, storing.MarkerName)); err != nil || strings.TrimSpace(string(marker)) != root.ID {
		t.Fatalf("the new folder's marker reads %q, %v; want the root's id", marker, err)
	}
	c := cells(t, w.a, w.work).Cells[0]
	if c.File == nil || c.File.Size != int64(len(bytes)) {
		t.Fatalf("the cell after the re-home holds %+v, want the imported file", c.File)
	}

	empty := filepath.Join(w.m.MediaDir, "empty")
	if err := os.Mkdir(empty, 0o700); err != nil {
		t.Fatal(err)
	}
	worldtest.Expect(t, "rehomeRoot onto an empty folder", patch(`{"path":`+strconv.Quote(empty)+`}`), worldtest.Reply{Status: http.StatusConflict,
		Body: `{"status":409,"code":"rehome_missing","title":"Recorded files aren't at the new place","missing":{"count":1,"first":[{"path":` + strconv.Quote(rel) + `,"reason":"missing"}]}}` + "\n"})
	if r := patch(`{"path":` + strconv.Quote(empty) + `,"forget":1}`); r.Status != http.StatusOK {
		t.Fatalf("rehomeRoot forgetting the file = %d %s, want 200", r.Status, r.Body)
	}
	worldtest.Expect(t, "getWorkCells after forgetting", w.a.Admin("home", http.MethodGet, "/api/v1/works/"+w.work+"/cells", ""), worldtest.Reply{Status: 200,
		Body: `{"cells":[{"itemId":"` + c.ItemID + `","edition":"any","targetId":"` + c.TargetID + `","targetName":"1080p","wanted":true,"state":"missing"}],` +
			`"items":[{"itemId":"` + c.ItemID + `","status":"missing"}]}` + "\n"})

	if r := w.a.Admin("home", http.MethodDelete, "/api/v1/roots/"+root.ID, ""); r.Status != http.StatusNoContent {
		t.Fatalf("deleteRoot = %d %s, want 204", r.Status, r.Body)
	}
	if roots := listRoots(); len(roots) != 0 {
		t.Fatalf("listRoots after the delete = %+v, want none", roots)
	}
	if g := decode(t, "getGrab", w.a.Admin("home", http.MethodGet, "/api/v1/grabs/"+w.grab.ID, ""), &grabView{}); g.Stage != "imported" {
		t.Fatalf("the Grab after its root went is %s, want its history kept as imported", g.Stage)
	}
	for _, f := range []string{placed, "moved/" + rel} {
		if _, err := os.Stat(filepath.Join(w.m.MediaDir, filepath.FromSlash(f))); err != nil {
			t.Fatalf("a re-home or delete removed %s: %v", f, err)
		}
	}
}
