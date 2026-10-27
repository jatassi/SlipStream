package indexers

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jatassi/slipstream/internal/health"
	"github.com/jatassi/slipstream/internal/live"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/setting"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/tasks"
)

const (
	// CatalogRefreshTaskID is the Indexer catalog refresh.
	CatalogRefreshTaskID tasks.TaskID = "indexer-catalog-refresh"
	// HealthCheckTaskID is the Indexer health check.
	HealthCheckTaskID tasks.TaskID = "indexer-health-check"
)

const day = 24 * time.Hour

var catalogCadence = setting.Duration{
	Key: "tasks." + string(CatalogRefreshTaskID) + ".cadence", Label: "Runs every",
	Default: day, Min: time.Hour, Max: 7 * day, Step: time.Hour,
	Floor: "Every Prowlarr and SlipStream install downloads from the same free catalog.",
}

var healthCheckCadence = setting.Duration{
	Key: "tasks." + string(HealthCheckTaskID) + ".cadence", Label: "Runs every",
	Default: 6 * time.Hour, Min: time.Hour, Max: 7 * day, Step: time.Hour,
	Floor: "Each run sends a request to the site of every Indexer that is on.",
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
	}, {
		ID:      HealthCheckTaskID,
		Name:    "Indexer health check",
		About:   "Tests each Indexer that is on from the Node it searches from. An Indexer stopped because its site rejected the sign-in or the account isn't tested again until you test it or save a change.",
		Group:   tasks.GroupOf(role.Searching.Group()),
		Cadence: tasks.Every{Interval: healthCheckCadence},
		Noun:    setting.Noun{One: "Indexer", Many: "Indexers"},
		Units:   x.healthUnits,
		Shown:   anyIndexer,
		Run:     func(ctx context.Context, u tasks.Unit, _ *tasks.Progress) error { return x.healthCheck(ctx, u.Key) },
		Follows: []live.Event{ChangedEvent, StatusChangedEvent},
		Health:  string(health.IndexersID),
	}}
}

// healthUnits is one unit per Indexer that is on, but for one stopped for
// a cause the health check doesn't retry. A unit goes where the Indexer
// searches from: tied to its home while it waits for home, else portable.
func (x *Indexers) healthUnits(ctx context.Context, r storage.Reader[storage.Library]) ([]tasks.Unit, error) {
	all, err := All(ctx, r)
	if err != nil {
		return nil, err
	}
	statuses, err := statusesIn(ctx, r)
	if err != nil {
		return nil, err
	}
	var out []tasks.Unit
	for _, idx := range all {
		st := statuses[idx.ID]
		if !idx.On || !checked(st) {
			continue
		}
		place := tasks.TiedTo(idx.Home)
		if portable(idx, st) {
			place = tasks.PortableOn(role.Searching, idx.Home)
		}
		out = append(out, tasks.Unit{Key: tasks.UnitKey(idx.ID.String()), Label: idx.Name, Place: place})
	}
	return out, nil
}

// anyIndexer shows the health check while an Indexer exists, though
// every one is off or stopped.
func anyIndexer(ctx context.Context, r storage.Reader[storage.Library]) (bool, error) {
	all, err := All(ctx, r)
	return len(all) > 0, err
}

// checked reports whether the health check tests an Indexer at st.
func checked(st Status) bool { return st.State != Stopped || Retried(st.Cause) }

// healthCheck tests Indexer id. It succeeds once the Test ran, whatever
// its outcome, since the outcome is the Indexer's status. An Indexer
// stopped since its unit was listed is left alone.
func (x *Indexers) healthCheck(ctx context.Context, key tasks.UnitKey) error {
	u, err := storage.ParseUUID(string(key))
	if err != nil {
		return fmt.Errorf("read the unit's Indexer id: %w", err)
	}
	id := ID{UUID: u}
	st, err := status(ctx, x.cfg.Library, id)
	if err != nil {
		return err
	}
	if !checked(st) {
		return nil
	}
	_, err = x.testSaved(ctx, id)
	var failed *TestFailedError
	switch {
	case err == nil, errors.As(err, &failed), errors.Is(err, ErrNoIndexer), errors.Is(err, ErrLate):
		return nil
	}
	return fmt.Errorf("test the Indexer: %w", err)
}
