package node

import (
	"context"

	"github.com/jatassi/slipstream/internal/health"
	"github.com/jatassi/slipstream/internal/nodes"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/tasks"
)

// healthCategories is every Health category after Storage, one line each,
// in the order health.IDs lists them.
func healthCategories(h *half) []health.Category {
	return []health.Category{
		{About: nodes.HealthAbout, Source: h.nodes.HealthSource()},
		{About: tasks.HealthAbout, Source: h.tasks.HealthSource(health.ScheduledTasksID)},
	}
}

// storageParts is what the Storage category judges: a line per holder
// that samples volumes, and the library roots' own items.
func storageParts(*half) health.StorageParts {
	return health.StorageParts{}
}

func (n *Node) openHealth(ctx context.Context, lib *storage.DB[storage.Library], h *half) (*health.Health, error) {
	return health.Open(ctx, health.Config{
		Library:    lib,
		Hub:        n.hub,
		Changed:    health.ChangedEvent.Bind(n.hub),
		Storage:    storageParts(h),
		Categories: healthCategories(h),
		Names:      liveNames,
		Zone:       n.zone(),
		Timings:    n.cfg.Timings.Health,
		Log:        n.rootLog.With("subsystem", "health"),
	})
}

func liveNames(ctx context.Context, rd storage.Reader[storage.Library]) (map[role.NodeID]string, error) {
	named, err := nodes.Names(ctx, rd)
	if err != nil {
		return nil, err
	}
	out := make(map[role.NodeID]string, len(named))
	for _, n := range named {
		out[n.ID] = n.Name
	}
	return out, nil
}
