package grabs

import (
	"context"
	"time"

	"github.com/jatassi/slipstream/internal/setting"
	"github.com/jatassi/slipstream/internal/storage"
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
	// Next is when task id next opens on schedule, zero while it runs.
	Next(ctx context.Context, id tasks.TaskID) (time.Time, error)
}

// Tasks is grabs' Scheduled tasks: Backlog search, on the Controller,
// whose one unit reconciles the queue and queues what is due.
func (gr *Grabs) Tasks() []tasks.Task {
	return []tasks.Task{{
		ID:   BacklogTaskID,
		Name: "Backlog search",
		About: "Searches the Wants that are due. A Want that keeps coming up empty is searched less often, up to the longest gap. " +
			"Run now searches only the Wants already due. To search every Want now, use Search all on Wanted.",
		Group:   tasks.ControllerGroup,
		Cadence: tasks.Every{Interval: BacklogEvery},
		Noun:    setting.Noun{One: "step", Many: "steps"},
		Units: func(context.Context, storage.Reader[storage.Library]) ([]tasks.Unit, error) {
			return []tasks.Unit{{Key: "due", Label: "Due searches", Place: tasks.OnController()}}, nil
		},
		Run: gr.backlog,
	}}
}

// interval is Backlog search's cadence as tasks stores it.
func (gr *Grabs) interval(ctx context.Context) (time.Duration, error) {
	every, err := gr.cfg.Tasks.Every(ctx, BacklogTaskID)
	if err != nil {
		return 0, err
	}
	return BacklogEvery.Value(every), nil
}
