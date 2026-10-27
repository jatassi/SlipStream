package housekeeping

import (
	"context"
	"time"

	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/nodes"
	housekeepingv1 "github.com/jatassi/slipstream/internal/pb/slipstream/housekeeping/v1"
	"github.com/jatassi/slipstream/internal/pb/slipstream/housekeeping/v1/housekeepingv1connect"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/rpc"
	"github.com/jatassi/slipstream/internal/setting"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/tasks"
)

const day = 24 * time.Hour

var cadence = setting.Duration{Key: "tasks.housekeeping.cadence", Label: "Runs every", Default: day, Min: day, Max: 7 * day, Step: day}

const (
	runHistory tasks.UnitKey = "run-history"
	joinCodes  tasks.UnitKey = "join-codes"
	sessions   tasks.UnitKey = "sessions"
	library    tasks.UnitKey = "library"
)

type Config struct {
	Library  *storage.DB[storage.Library]
	Nodes    *nodes.Service
	Identity *identity.Service
	Tidy     func(node role.NodeID) housekeepingv1connect.HousekeepingServiceHandler
	// Steps are the features' own prunes, each run on the Controller
	// after the sessions and before the databases.
	Steps []Step
}

// Step is one feature's prune.
type Step struct {
	Key   tasks.UnitKey
	Label string
	Run   func(ctx context.Context) error
}

func Task(cfg Config) tasks.Task {
	return tasks.Task{
		ID:      "housekeeping",
		Name:    "Housekeeping",
		About:   "Prunes old runs, expired Join Codes, idle sessions and old Indexer queries, then tidies every database.",
		Group:   tasks.ControllerGroup,
		Cadence: tasks.InWindow{Days: cadence},
		Overdue: tasks.Wait,
		Noun:    setting.Noun{One: "step", Many: "steps"},
		Units:   cfg.units,
		Run:     cfg.run,
	}
}

func (c Config) units(ctx context.Context, r storage.Reader[storage.Library]) ([]tasks.Unit, error) {
	named, err := nodes.Names(ctx, r)
	if err != nil {
		return nil, err
	}
	out := []tasks.Unit{
		{Key: runHistory, Label: "Run history", Place: tasks.OnController()},
		{Key: joinCodes, Label: "Join Codes", Place: tasks.OnController()},
		{Key: sessions, Label: "Sessions", Place: tasks.OnController()},
	}
	for _, s := range c.Steps {
		out = append(out, tasks.Unit{Key: s.Key, Label: s.Label, Place: tasks.OnController()})
	}
	for _, n := range named {
		out = append(out, tasks.Unit{Key: tasks.UnitKey("node:" + n.ID.String()), Label: "Local database on " + n.Name, Place: tasks.TiedTo(n.ID)})
	}
	return append(out, tasks.Unit{Key: library, Label: "Library database", Place: tasks.OnController()}), nil
}

func (c Config) run(ctx context.Context, u tasks.Unit, _ *tasks.Progress) error {
	switch u.Key {
	case runHistory:
		return tasks.PruneRuns(ctx, c.Library)
	case joinCodes:
		return c.Nodes.PruneJoinCodes(ctx)
	case sessions:
		return c.Identity.PruneSessions(ctx)
	case library:
		return c.Library.Optimize(ctx)
	}
	for _, s := range c.Steps {
		if s.Key == u.Key {
			return s.Run(ctx)
		}
	}
	node, _ := u.Place.Node()
	_, err := c.Tidy(node).Tidy(ctx, &housekeepingv1.TidyRequest{})
	return err
}

type Service struct {
	Local *storage.DB[storage.Local]
}

func (s Service) Tidy(ctx context.Context, _ *housekeepingv1.TidyRequest) (*housekeepingv1.TidyResponse, error) {
	if err := s.Local.Optimize(ctx); err != nil {
		return nil, rpc.Errorf(rpc.CodeInternal, "%v", err)
	}
	return &housekeepingv1.TidyResponse{}, nil
}
