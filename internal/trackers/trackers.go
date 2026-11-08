// Package trackers owns the seeding side of a Tracker: its seeding rule,
// its seeding goal, the seeding settings, and the hit and run windows and
// goal runs the Seeding goals task writes. indexers keeps the Tracker's
// identity. Package seeding decides what a window and a goal do.
package trackers

import (
	"context"
	"embed"
	"time"

	"github.com/jatassi/slipstream/internal/indexers"
	"github.com/jatassi/slipstream/internal/live"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/tasks"
	"github.com/jatassi/slipstream/internal/trackers/seeding"
)

type file = storage.Library

//go:embed schema/*.sql
var schemaFiles embed.FS

func Schema() storage.Schema[file] { return storage.NewSchema[file]("trackers", schemaFiles) }

// TaskID is the Seeding goals task, which every seeding save asks to run
// soon.
const TaskID tasks.TaskID = "seeding-goals"

// SeedingChanged names the Tracker whose seeding rule, goal or windows
// changed, nil for the default goal and the seeding settings.
type SeedingChanged struct {
	TrackerID *string `json:"trackerId" format:"uuid" doc:"The Tracker whose seeding changed; null for the default goal and the seeding settings"`
}

var SeedingChangedEvent = live.AdminEvent[SeedingChanged]("trackers.seeding-changed")

type Publisher[T any] interface {
	Publish(ctx context.Context, v T)
}

// Grabs is what trackers reads of the Grabs, as grabs.TrackerFacts
// answers it.
type Grabs interface {
	// Stated is each Tracker's most recent Grab whose Release stated a
	// seeding rule.
	Stated(ctx context.Context, r storage.Reader[file]) (map[indexers.TrackerID]Stated, error)
}

// Stated is the rule one Grab's Release stated, and when the Grab was made.
type Stated struct {
	Rule seeding.Rule
	At   time.Time
}

type Config struct {
	Library *storage.DB[file]
	Grabs   Grabs
	// Soon asks the Seeding goals task for a run soon.
	Soon    func()
	Changed Publisher[SeedingChanged]
}

type Trackers struct{ cfg Config }

func New(cfg Config) *Trackers { return &Trackers{cfg: cfg} }

// saved tells Admins that id's seeding changed, nil for the default goal
// or the settings, and asks the Seeding goals task to run on it soon.
func (t *Trackers) saved(ctx context.Context, id *indexers.TrackerID) {
	t.publish(ctx, id)
	t.cfg.Soon()
}

func (t *Trackers) publish(ctx context.Context, id *indexers.TrackerID) {
	var v SeedingChanged
	if id != nil {
		s := id.String()
		v.TrackerID = &s
	}
	t.cfg.Changed.Publish(ctx, v)
}
