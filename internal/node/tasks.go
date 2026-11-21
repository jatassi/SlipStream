package node

import (
	"github.com/jatassi/slipstream/internal/backups"
	"github.com/jatassi/slipstream/internal/housekeeping"
	"github.com/jatassi/slipstream/internal/imports"
	"github.com/jatassi/slipstream/internal/metadata"
	"github.com/jatassi/slipstream/internal/tasks"
)

// catalog is the built-in tasks in window order. The Backup goes first, so
// the night's other work runs on a library already backed up.
func catalog(h housekeeping.Config, r metadata.RefreshConfig, im *imports.Imports, b *backups.Backups) []tasks.Task {
	return []tasks.Task{
		b.Task(),
		metadata.RefreshTask(r),
		im.Task(),
		housekeeping.Task(h),
	}
}
