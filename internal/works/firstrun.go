package works

import (
	"context"

	"github.com/jatassi/slipstream/internal/firstrun"
	"github.com/jatassi/slipstream/internal/live"
	"github.com/jatassi/slipstream/internal/storage"
)

// MediaStep is first setup's kinds of media step, shown when the binary
// registers more than one Content Type and done only by Continue.
func (w *Works) MediaStep() firstrun.Step {
	return firstrun.Step{
		ID: firstrun.MediaID,
		Shows: func(context.Context, storage.Reader[file]) (bool, error) {
			return len(w.ids) > 1, nil
		},
		Events: []live.Event{ContentTypesChangedEvent},
	}
}
