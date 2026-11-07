package e2e

import (
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/jatassi/slipstream/fakes/scenario"
	"github.com/jatassi/slipstream/fakes/world/worldtest"
	"github.com/jatassi/slipstream/internal/role"
)

func workerRoot(t *testing.T, roles ...role.Name) scenario.Scenario {
	t.Helper()
	s, err := scenario.Load("home")
	if err != nil {
		t.Fatal(err)
	}
	s.Nodes[1].Roles = roles
	s.Library.Roots = []scenario.LibraryRoot{{ContentType: "movie", Folder: "movies", Node: "seedbox"}}
	return s
}

func TestAWorkerRootThatDownloadsImportsByHardLinkThere(t *testing.T) {
	t.Parallel()
	s := workerRoot(t, role.Searching, role.Downloading, role.Storage)
	w := walkFrom(t, beginIn(t, s, worldtest.LaunchScenario(t, s)))
	seedbox := w.m.Nodes[1]
	if want := []string{"fetching", "downloading", "probing", "importing", "imported"}; !slices.Equal(w.grab.stages(), want) {
		t.Fatalf("the Grab's history is %v, want %v", w.grab.stages(), want)
	}
	if w.grab.NodeID != seedbox.NodeID {
		t.Fatalf("the Grab's Node is %s, want seedbox %s", w.grab.NodeID, seedbox.NodeID)
	}
	checkPlaced(t, w.m, 2)
	if _, err := os.Stat(filepath.Join(w.m.MediaDir, "movies", ".slipstream-staging")); !os.IsNotExist(err) {
		t.Fatalf("the root has a staging folder (%v), want none: no Transfer ran", err)
	}
}

func TestAGrabIntoAWorkerRootFromAnotherNodeTransfersThere(t *testing.T) {
	t.Parallel()
	s := workerRoot(t, role.Searching, role.Storage)
	w := walkFrom(t, beginIn(t, s, worldtest.LaunchScenario(t, s)))
	if want := []string{"fetching", "downloading", "probing", "transferring", "importing", "imported"}; !slices.Equal(w.grab.stages(), want) {
		t.Fatalf("the Grab's history is %v, want %v", w.grab.stages(), want)
	}
	if home := w.m.Nodes[0]; w.grab.NodeID != home.NodeID {
		t.Fatalf("the Grab's Node is %s, want home %s, which downloads", w.grab.NodeID, home.NodeID)
	}
	checkPlaced(t, w.m, 1)
	if entries, err := os.ReadDir(filepath.Join(w.m.MediaDir, "movies", ".slipstream-staging")); err != nil || len(entries) != 0 {
		t.Fatalf("the staging folder holds %v, %v; want nothing", entries, err)
	}
}
