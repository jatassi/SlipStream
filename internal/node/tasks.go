package node

import (
	"github.com/jatassi/slipstream/internal/housekeeping"
	"github.com/jatassi/slipstream/internal/tasks"
)

// catalog is every Scheduled task the binary runs, in the Tasks page's
// order within each group and the order window tasks start in.
func catalog(h housekeeping.Config) []tasks.Task {
	return []tasks.Task{
		housekeeping.Task(h),
	}
}
