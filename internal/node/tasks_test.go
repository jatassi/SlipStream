package node

import (
	"slices"
	"testing"

	"github.com/jatassi/slipstream/internal/housekeeping"
	"github.com/jatassi/slipstream/internal/tasks"
)

func TestTheCatalogListsEachTaskOnceInItsWindowOrder(t *testing.T) {
	seen := map[tasks.TaskID]bool{}
	var window []tasks.TaskID
	for _, task := range catalog(housekeeping.Config{}) {
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
	if want := []tasks.TaskID{"housekeeping"}; !slices.Equal(window, want) {
		t.Fatalf("window order %q, want %q", window, want)
	}
}
