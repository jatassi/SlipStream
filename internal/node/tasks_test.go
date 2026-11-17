package node

import (
	"slices"
	"testing"

	"github.com/jatassi/slipstream/internal/housekeeping"
	"github.com/jatassi/slipstream/internal/imports"
	"github.com/jatassi/slipstream/internal/metadata"
	"github.com/jatassi/slipstream/internal/tasks"
	"github.com/jatassi/slipstream/internal/transfers"
)

func TestTheCatalogListsEachTaskOnceInItsWindowOrder(t *testing.T) {
	seen := map[tasks.TaskID]bool{}
	var window []tasks.TaskID
	for _, task := range catalog(housekeeping.Config{}, metadata.RefreshConfig{}, imports.New(imports.Config{Transfers: transfers.New(transfers.ControllerConfig{})})) {
		if seen[task.ID] {
			t.Fatalf("the catalog lists %q twice", task.ID)
		}
		seen[task.ID] = true
		if err := task.Setting().Check(); err != nil {
			t.Fatalf("%s: %v", task.ID, err)
		}
		if _, ok := task.Cadence.(tasks.InWindow); ok {
			window = append(window, task.ID)
		}
	}
	if want := []tasks.TaskID{"metadata-refresh", "library-scan", "housekeeping"}; !slices.Equal(window, want) {
		t.Fatalf("window order %q, want %q", window, want)
	}
}

func TestHousekeepingPrunesInItsUnitsOrder(t *testing.T) {
	var labels []string
	for _, s := range housekeepingSteps(nil, nil, nil, nil, nil, nil) {
		labels = append(labels, s.Label)
	}
	want := []string{"Indexer queries and sessions", "Media Server history", "History", "Finished Grabs", "Finished Imports", "Import records"}
	if !slices.Equal(labels, want) {
		t.Fatalf("Housekeeping's units are %q, want %q", labels, want)
	}
}
