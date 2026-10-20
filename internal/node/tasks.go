package node

import (
	"github.com/jatassi/slipstream/internal/housekeeping"
	"github.com/jatassi/slipstream/internal/metadata"
	"github.com/jatassi/slipstream/internal/tasks"
)

func catalog(h housekeeping.Config, r metadata.RefreshConfig) []tasks.Task {
	return []tasks.Task{
		metadata.RefreshTask(r),
		housekeeping.Task(h),
	}
}
