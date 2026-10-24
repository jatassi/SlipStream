package node

import (
	"slices"
	"testing"

	"github.com/jatassi/slipstream/internal/health"
	"github.com/jatassi/slipstream/internal/housekeeping"
	"github.com/jatassi/slipstream/internal/metadata"
	"github.com/jatassi/slipstream/internal/node/surface"
)

// The registry's sources are built from an empty half: only the
// categories' declarations are read.
func TestTheHealthRegistryMatchesItsIDsEventsAndTasks(t *testing.T) {
	cats := healthCategories(&half{})
	ids := []health.ID{health.StorageID}
	events := storageParts(&half{}).RootEvents
	for _, v := range storageParts(&half{}).Volumes {
		events = append(events, v.Events...)
	}
	for _, c := range cats {
		ids = append(ids, c.ID)
		events = append(events, c.Events...)
	}
	if !slices.Equal(ids, health.IDs()) {
		t.Errorf("the registry binds %q, want health.IDs %q in order", ids, health.IDs())
	}
	for _, e := range events {
		if !slices.Contains(surface.Events, e) {
			t.Errorf("a Health category follows %s, which surface.Events doesn't declare", e.Name())
		}
	}
	for _, task := range catalog(housekeeping.Config{}, metadata.RefreshConfig{}) {
		if task.Health != "" && !slices.Contains(health.IDs(), health.ID(task.Health)) {
			t.Errorf("task %s warns in Health category %q, which nobody registers", task.ID, task.Health)
		}
	}
}
