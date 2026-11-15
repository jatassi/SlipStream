package grabs

import (
	"context"
	"time"

	"github.com/jatassi/slipstream/internal/setting"
	"github.com/jatassi/slipstream/internal/tasks"
)

const day = 24 * time.Hour

const (
	BacklogTaskID tasks.TaskID = "backlog-search"
	RSSTaskID     tasks.TaskID = "rss-sync"
)

// BacklogEvery and RSSEvery are the two tasks' cadences, which the
// Automatic search section shows beside its own settings. tasks stores
// them.
var (
	BacklogEvery = setting.Duration{
		Key: "tasks." + string(BacklogTaskID) + ".cadence", Label: "Backlog search runs every",
		Default: 8 * time.Hour, Min: time.Hour, Max: 7 * day, Step: time.Hour,
		Floor: "Each run can send a search to every Indexer's site.",
	}
	RSSEvery = setting.Duration{
		Key: "tasks." + string(RSSTaskID) + ".cadence", Label: "RSS sync runs every",
		Default: 15 * time.Minute, Min: 10 * time.Minute, Max: 2 * time.Hour, Step: time.Minute,
		Floor: "Many trackers ban clients that read their RSS feed more often than every 10 minutes.",
	}
)

// Cadences is the two tasks' cadences as tasks stores them, by task.
type Cadences interface {
	Every(ctx context.Context, id tasks.TaskID) (*int64, error)
	SetCadence(ctx context.Context, id tasks.TaskID, v setting.Stored[int64]) error
}

// Tasks is grabs' Scheduled tasks.
func (*Grabs) Tasks() []tasks.Task { return nil }
