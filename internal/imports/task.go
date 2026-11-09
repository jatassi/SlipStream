package imports

import (
	"context"
	"strings"
	"time"

	"github.com/jatassi/slipstream/internal/live"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/roots"
	"github.com/jatassi/slipstream/internal/setting"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/tasks"
)

const day = 24 * time.Hour

var cadence = setting.Duration{Key: "tasks.library-scan.cadence", Label: "Runs every", Default: day, Min: day, Max: 7 * day, Step: day}

// Task is Library scan: in the Maintenance window every cadence, one unit
// per root on the root's Node, listed while any root exists.
func (im *Imports) Task() tasks.Task {
	return tasks.Task{
		ID:      TaskID,
		Name:    "Library scan",
		About:   "Lists the files in every library root and records what changed. Files in folders no Work has join a Work the scan is sure of, or wait in Pending imports.",
		Group:   tasks.GroupOf(role.Storage.Group()),
		Cadence: tasks.InWindow{Days: cadence},
		Overdue: tasks.Wait,
		Noun:    setting.Noun{One: "root", Many: "roots"},
		Units:   im.units,
		Run:     im.unit,
		Follows: []live.Event{roots.ChangedEvent},
	}
}

func (im *Imports) units(ctx context.Context, r storage.Reader[file]) ([]tasks.Unit, error) {
	all, err := roots.All(ctx, r)
	if err != nil {
		return nil, err
	}
	words, err := im.cfg.Roots.WordsIn(ctx, r)
	if err != nil {
		return nil, err
	}
	out := make([]tasks.Unit, len(all))
	for i, root := range all {
		out[i] = tasks.Unit{Key: unitKey(root.ID), Label: words[root.ID], Place: tasks.TiedTo(root.Node)}
	}
	return out, nil
}

func rootOf(key tasks.UnitKey) (roots.ID, error) {
	id, err := storage.ParseUUID(strings.TrimPrefix(string(key), "root:"))
	return roots.ID{UUID: id}, err
}
