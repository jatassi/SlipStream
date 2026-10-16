package node

import (
	"github.com/jatassi/slipstream/internal/housekeeping"
	"github.com/jatassi/slipstream/internal/tasks"
)

func catalog(h housekeeping.Config) []tasks.Task {
	return []tasks.Task{
		housekeeping.Task(h),
	}
}
