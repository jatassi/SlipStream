package e2e

import (
	"net/http"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/jatassi/slipstream/fakes/scenario"
	"github.com/jatassi/slipstream/fakes/world/worldtest"
	"github.com/jatassi/slipstream/internal/role"
)

type storageRoot struct {
	ID     string `json:"id"`
	NodeID string `json:"nodeId"`
	Path   string `json:"path"`
	State  string `json:"state"`
}

func listRoots(t *testing.T, a *worldtest.API) []storageRoot {
	t.Helper()
	var out struct {
		Roots []storageRoot `json:"roots"`
	}
	decode(t, "listRoots", a.Admin("home", http.MethodGet, "/api/v1/roots", ""), &out)
	return out.Roots
}

func storageState(a *worldtest.API, id string) string {
	for _, row := range a.Nodes("home") {
		if row.ID != id {
			continue
		}
		for _, r := range row.Roles {
			if r.Role == string(role.Storage) {
				return r.State
			}
		}
	}
	return "none"
}

func rootOn(t *testing.T, a *worldtest.API, id, state string) storageRoot {
	t.Helper()
	var root storageRoot
	worldtest.Until(t, "the root reads "+state, func() (bool, error) {
		roots := listRoots(t, a)
		if len(roots) != 1 {
			return false, nil
		}
		root = roots[0]
		return root.NodeID == id && root.State == state, nil
	})
	return root
}

func untickStorage(t *testing.T, w walked) string {
	t.Helper()
	seedbox := w.m.Nodes[1].NodeID
	out := w.a.SetRoles("home", seedbox, string(role.Searching), string(role.Downloading))
	if !slices.ContainsFunc(out.Words(), func(s string) bool {
		return strings.HasPrefix(s, "seedbox takes no new Storage work. It keeps its 1 root (") && strings.HasSuffix(s, ") until you move or write it off.")
	}) {
		t.Fatalf("unticking seedbox's Storage says %q, want it keeping its 1 root", out.Words())
	}
	worldtest.Until(t, "seedbox drains Storage", func() (bool, error) { return storageState(w.a, seedbox) == "draining", nil })
	return seedbox
}

func rehomeOntoHome(t *testing.T, w walked, root storageRoot) {
	t.Helper()
	home := w.m.Nodes[0].NodeID
	reply := w.a.Admin("home", http.MethodPatch, "/api/v1/roots/"+root.ID, `{"nodeId":"`+home+`","path":`+strconv.Quote(root.Path)+`}`)
	if reply.Status != http.StatusOK {
		t.Fatalf("rehomeRoot onto home = %d %s, want 200", reply.Status, reply.Body)
	}
	rootOn(t, w.a, home, "ready")
}

func TestADrainingStorageNodeKeepsServingItsRoot(t *testing.T) {
	t.Parallel()
	s := workerRoot(t, role.Searching, role.Downloading, role.Storage)
	w := walkFrom(t, beginIn(t, s, worldtest.LaunchScenario(t, s)))
	seedbox := untickStorage(t, w)
	root := rootOn(t, w.a, seedbox, "ready")
	if c := cells(t, w.a, w.work).Cells[0]; c.File == nil || c.State != "upgradable" {
		t.Fatalf("the cell while seedbox drains is %+v, want filled by the imported file", c)
	}
	rehomeOntoHome(t, w, root)
	worldtest.Until(t, "seedbox lets Storage go", func() (bool, error) { return storageState(w.a, seedbox) == "none", nil })
	checkPlaced(t, w.m, 2)
}

func TestWriteOffMakesCellsMissingAndARehomeRestoresThem(t *testing.T) {
	t.Parallel()
	s := workerRoot(t, role.Searching, role.Downloading, role.Storage)
	w := walkFrom(t, beginIn(t, s, worldtest.LaunchScenario(t, s)))
	seedbox := untickStorage(t, w)
	var wrote worldtest.Outcome
	decode(t, "writeOff", w.a.Admin("home", http.MethodPost, "/api/v1/nodes/"+seedbox+"/write-offs", `{"roles":["storage"]}`), &wrote)
	if words := wrote.Words(); len(words) != 1 || !strings.HasPrefix(words[0], "The 1 root (") || !strings.HasSuffix(words[0], ") seedbox holds for Storage is written off. No files are deleted.") {
		t.Fatalf("writing off seedbox's Storage says %q, want its 1 root written off", words)
	}
	root := rootOn(t, w.a, seedbox, "written_off")
	if c := cells(t, w.a, w.work).Cells[0]; c.File != nil || c.State != "missing" {
		t.Fatalf("the cell after the write-off is %+v, want missing", c)
	}
	worldtest.Expect(t, "searchWork", w.a.Admin("home", http.MethodPost, "/api/v1/works/"+w.work+"/search", ""), worldtest.Reply{
		Status: http.StatusConflict,
		Body: `{"status":409,"code":"root_unreachable","title":"The Work's library root is on a Node that takes no new files now",` +
			`"detail":"seedbox's Storage was written off, so the library root there takes no new files."}` + "\n",
	})

	rehomeOntoHome(t, w, root)
	if c := cells(t, w.a, w.work).Cells[0]; c.File == nil || c.State != "upgradable" {
		t.Fatalf("the cell after the re-home is %+v, want filled by the imported file", c)
	}
	worldtest.Until(t, "the write-off settles", func() (bool, error) { return storageState(w.a, seedbox) == "none", nil })
	checkPlaced(t, w.m, 2)
}

func TestTheLastStorageNodeHandsBackToTheController(t *testing.T) {
	t.Parallel()
	s := workerRoot(t, role.Searching, role.Downloading, role.Storage)
	s.Nodes[0].Roles = slices.DeleteFunc(s.Nodes[0].Roles, func(r role.Name) bool { return r == role.Storage })
	run := worldtest.LaunchScenario(t, s)
	a := api(t, run)
	seedbox := a.Node("seedbox").NodeID
	rootOn(t, a, seedbox, "ready")
	out := a.SetRoles("home", seedbox, string(role.Searching), string(role.Downloading))
	if want := "home takes over Storage, since SlipStream always needs a Node that stores the library."; !slices.Contains(out.Words(), want) {
		t.Fatalf("unticking the last Storage Node says %q, want %q", out.Words(), want)
	}
	worldtest.Until(t, "seedbox drains Storage", func() (bool, error) { return storageState(a, seedbox) == "draining", nil })
	if st := storageState(a, a.Node("home").NodeID); st != "holding" {
		t.Fatalf("home stands %s in Storage, want holding", st)
	}
}

func TestTwoRootsOnOneVolumeListedOnce(t *testing.T) {
	t.Parallel()
	s, err := scenario.Load("single-machine")
	if err != nil {
		t.Fatal(err)
	}
	s.Library.Roots = append(s.Library.Roots, scenario.LibraryRoot{ContentType: "movie", Folder: "films"})
	run := worldtest.LaunchScenario(t, s)
	a := api(t, run)
	var list struct {
		Nodes []struct {
			NodeID  string `json:"nodeId"`
			Live    bool   `json:"live"`
			Volumes []struct {
				ID      string   `json:"id"`
				Total   int64    `json:"total"`
				Free    int64    `json:"free"`
				RootIDs []string `json:"rootIds"`
			} `json:"volumes"`
		} `json:"nodes"`
	}
	worldtest.Until(t, "both roots are ready", func() (bool, error) {
		roots := listRoots(t, a)
		return len(roots) == 2 && roots[0].State == "ready" && roots[1].State == "ready", nil
	})
	decode(t, "listVolumes", a.Admin("home", http.MethodGet, "/api/v1/volumes", ""), &list)
	if len(list.Nodes) != 1 || !list.Nodes[0].Live || len(list.Nodes[0].Volumes) != 1 {
		t.Fatalf("listVolumes = %+v, want home's one volume, live", list)
	}
	v := list.Nodes[0].Volumes[0]
	if len(v.RootIDs) != 2 || v.Total <= 0 || v.Free <= 0 || v.Free > v.Total {
		t.Fatalf("the volume reads %+v, want both roots on it with its figures", v)
	}
	for _, r := range listRoots(t, a) {
		if !slices.Contains(v.RootIDs, r.ID) || filepath.Dir(r.Path) != run.Manifest().MediaDir {
			t.Fatalf("root %+v isn't on the volume %v", r, v.RootIDs)
		}
	}
}
