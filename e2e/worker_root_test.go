package e2e

import (
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/jatassi/slipstream/fakes/scenario"
	"github.com/jatassi/slipstream/fakes/world"
	"github.com/jatassi/slipstream/fakes/world/worldtest"
	"github.com/jatassi/slipstream/internal/role"
)

// workerRoot is the home Scenario with its movies root on seedbox, which
// holds roles.
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

func TestAGrabIntoAWorkerRootFromAnotherNodeFailsNeedsTransfer(t *testing.T) {
	t.Parallel()
	s := workerRoot(t, role.Searching, role.Storage)
	w := beginIn(t, s, worldtest.LaunchScenario(t, s))
	stream := worldtest.NewMessages(t, w.m, world.AdminIdentity)
	search := w.search(t)
	event(t, stream, "grabs.search-changed", `{"searchId":"`+search+`","workId":"`+w.work+`"}`)
	w.ended(t, grabChanged(t, stream, w.work))
	const words = "the Work's library root is on another Worker, and moving files into a root on another Worker isn't supported yet"
	if w.grab.Stage != "failed" || w.grab.Failure == nil || w.grab.Failure.Code != "needs_transfer" || w.grab.Failure.Message != words {
		t.Fatalf("the Grab ended %s with %+v after %v, want failed needs_transfer: %s", w.grab.Stage, w.grab.Failure, w.grab.stages(), words)
	}
	if home := w.m.Nodes[0]; w.grab.NodeID != home.NodeID {
		t.Fatalf("the Grab's Node is %s, want home %s, which downloads", w.grab.NodeID, home.NodeID)
	}
	holdsOnly(t, w.m)
}
