package node

import (
	"net/http"
	"slices"
	"testing"
	"testing/synctest"

	"github.com/jatassi/slipstream/internal/backups"
	"github.com/jatassi/slipstream/internal/housekeeping"
	"github.com/jatassi/slipstream/internal/imports"
	"github.com/jatassi/slipstream/internal/metadata"
	"github.com/jatassi/slipstream/internal/tasks"
	"github.com/jatassi/slipstream/internal/transfers"
)

func TestTheCatalogListsEachTaskOnceInItsWindowOrder(t *testing.T) {
	seen := map[tasks.TaskID]bool{}
	var window []tasks.TaskID
	for _, task := range catalog(housekeeping.Config{}, metadata.RefreshConfig{}, imports.New(imports.Config{Transfers: transfers.New(transfers.ControllerConfig{})}), &backups.Backups{}) {
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
	if want := []tasks.TaskID{"backup", "metadata-refresh", "library-scan", "housekeeping"}; !slices.Equal(window, want) {
		t.Fatalf("window order %q, want %q", window, want)
	}
}

func TestHousekeepingPrunesInItsUnitsOrder(t *testing.T) {
	var labels []string
	for _, s := range housekeepingSteps(nil, nil, nil, nil, nil, nil, nil) {
		labels = append(labels, s.Label)
	}
	want := []string{"Indexer queries and sessions", "Media Server history", "History", "Finished Grabs", "Finished Imports", "Import records", "Subtitle tries"}
	if !slices.Equal(labels, want) {
		t.Fatalf("Housekeeping's units are %q, want %q", labels, want)
	}
}

func TestEveryTaskDeclaresWhileReconciling(t *testing.T) {
	env, _ := testEnv(t)
	synctest.Test(t, func(t *testing.T) {
		n, c := startInMemory(t, env)
		if claimed := c.claim(n.Ready().SetupCode, "1234"); claimed.status != http.StatusCreated {
			t.Fatalf("claimController = %d %s", claimed.status, claimed.body)
		}
		synctest.Wait()
		want := map[tasks.TaskID]tasks.WhileReconciling{
			"metadata-refresh": tasks.Runs, "library-scan": tasks.Runs, "housekeeping": tasks.Runs,
			"indexer-catalog-refresh": tasks.Runs, "indexer-health-check": tasks.Runs, "seeding-goals": tasks.Runs, "plex-sign-in": tasks.Runs,
			"subtitle-search": tasks.Pauses, "backlog-search": tasks.Pauses, "rss-sync": tasks.Pauses,
		}
		s := n.half.Load().tasks
		for id, w := range want {
			if got := s.WhileReconcilingOf(id); got != w {
				t.Errorf("%s declares %d while Reconciling, want %d", id, got, w)
			}
		}
	})
}

func TestHousekeepingPausesItsPrunesWhileReconciling(t *testing.T) {
	want := map[string]tasks.WhileReconciling{
		"Indexer queries and sessions": tasks.Runs, "Media Server history": tasks.Runs, "History": tasks.Pauses,
		"Finished Grabs": tasks.Pauses, "Finished Imports": tasks.Pauses, "Import records": tasks.Pauses, "Subtitle tries": tasks.Runs,
	}
	for _, s := range housekeepingSteps(nil, nil, nil, nil, nil, nil, nil) {
		if s.WhileReconciling != want[s.Label] {
			t.Errorf("%s declares %d while Reconciling, want %d", s.Label, s.WhileReconciling, want[s.Label])
		}
	}
}
