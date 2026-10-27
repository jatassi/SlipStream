package indexers

import (
	"context"
	"time"

	"github.com/jatassi/slipstream/internal/health"
	"github.com/jatassi/slipstream/internal/setting"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/tasks"
)

// CatalogRefreshTaskID is the Indexer catalog refresh.
const CatalogRefreshTaskID tasks.TaskID = "indexer-catalog-refresh"

const day = 24 * time.Hour

var catalogCadence = setting.Duration{
	Key: "tasks." + string(CatalogRefreshTaskID) + ".cadence", Label: "Runs every",
	Default: day, Min: time.Hour, Max: 7 * day, Step: time.Hour,
	Floor: "Every Prowlarr and SlipStream install downloads from the same free catalog.",
}

// Tasks are the Indexers' scheduled tasks.
func (x *Indexers) Tasks() []tasks.Task {
	return []tasks.Task{{
		ID:      CatalogRefreshTaskID,
		Name:    "Indexer catalog refresh",
		About:   "Downloads the Indexer catalog's new and changed definitions. Each Indexer moves to its definition's new version once that version loads.",
		Group:   tasks.ControllerGroup,
		Cadence: tasks.Every{Interval: catalogCadence, AtStart: true},
		Noun:    setting.Noun{One: "catalog", Many: "catalogs"},
		Units: func(context.Context, storage.Reader[storage.Library]) ([]tasks.Unit, error) {
			return []tasks.Unit{{Key: "catalog", Label: "the Indexer catalog", Place: tasks.OnController()}}, nil
		},
		Run:    func(ctx context.Context, _ tasks.Unit, _ *tasks.Progress) error { return x.refresh(ctx) },
		Health: string(health.IndexersID),
	}}
}
