package downloadclients

import (
	"context"

	"github.com/jatassi/slipstream/internal/firstrun"
	"github.com/jatassi/slipstream/internal/live"
	"github.com/jatassi/slipstream/internal/nodes"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/storage"
)

// DownloadsStep is first setup's downloads step, shown while a Node holds
// Downloading and done only by Continue.
func (c *Clients) DownloadsStep() firstrun.Step {
	return firstrun.Step{
		ID: firstrun.DownloadsID,
		Shows: func(context.Context, storage.Reader[file]) (bool, error) {
			return len(c.cfg.Roster.Serving(role.Downloading)) > 0, nil
		},
		Events: []live.Event{nodes.ChangedEvent},
	}
}
