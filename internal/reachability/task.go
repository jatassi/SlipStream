package reachability

import (
	"context"
	"time"

	"github.com/jatassi/slipstream/internal/health"
	"github.com/jatassi/slipstream/internal/live"
	"github.com/jatassi/slipstream/internal/mediaserverlink/link"
	"github.com/jatassi/slipstream/internal/mediaservers"
	"github.com/jatassi/slipstream/internal/nodes"
	"github.com/jatassi/slipstream/internal/reachability/internal/db"
	"github.com/jatassi/slipstream/internal/reachability/judge"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/setting"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/tasks"
)

const TaskID tasks.TaskID = "reachability"

// CheckEvery is the task's cadence, which tasks stores and the Remote
// access section and the Tasks page both show.
var CheckEvery = setting.Duration{
	Key: "tasks." + string(TaskID) + ".cadence", Label: "Check every route again every",
	Default: 24 * time.Hour, Min: time.Hour, Max: 7 * 24 * time.Hour, Step: time.Hour,
}

const uiSubject = "SlipStream UI and portal"

func (r *Reachability) Task() tasks.Task {
	return tasks.Task{
		ID:      TaskID,
		Name:    "Reachability check",
		About:   "Checks that everything you chose to use away from home still works from outside your home.",
		Group:   tasks.ControllerGroup,
		Cadence: tasks.Every{Interval: CheckEvery},
		Noun:    setting.Noun{One: "route", Many: "routes"},
		Units:   r.units,
		Run:     r.runUnit,
		Follows: []live.Event{ChangedEvent},
		Health:  string(health.RemoteAccessID),
		// A restored Controller's facts and results describe another
		// moment, so the check waits until Reconciling ends.
		WhileReconciling: tasks.Pauses,
	}
}

// origins is where each wanted Exposure's unit runs: a Media Server's
// runs-on Node unless it was removed, else its reaching Node; the UI and
// portal's last placed origin unless it was removed, else the Controller.
type origins struct {
	self    role.NodeID
	live    map[role.NodeID]string
	servers map[link.ServerID]mediaservers.Record
	last    map[judge.ExposureID]role.NodeID
}

func originsIn(ctx context.Context, rd storage.Reader[storage.Library], self role.NodeID) (origins, error) {
	o := origins{self: self, live: map[role.NodeID]string{}, servers: map[link.ServerID]mediaservers.Record{}, last: map[judge.ExposureID]role.NodeID{}}
	named, err := nodes.Names(ctx, rd)
	if err != nil {
		return o, err
	}
	for _, n := range named {
		o.live[n.ID] = n.Name
	}
	recs, err := mediaservers.ServersIn(ctx, rd)
	if err != nil {
		return o, err
	}
	for _, rec := range recs {
		o.servers[rec.ID] = rec
	}
	results, err := db.New(rd).ListResults(ctx)
	if err != nil {
		return o, err
	}
	for _, res := range results {
		o.last[res.ExposureID] = res.OriginNodeID
	}
	return o, nil
}

// of is e's origin and how it was chosen.
func (o origins) of(e Exposure) (role.NodeID, judge.OriginLabel) {
	if e.Kind == judge.UIPortal {
		if n, ok := o.last[e.ID]; ok && n != o.self {
			if _, ok := o.live[n]; ok {
				return n, judge.Serving
			}
		}
		return o.self, judge.Controller
	}
	rec := o.servers[e.Server]
	if _, ok := o.live[rec.RunsOn]; ok {
		return rec.RunsOn, judge.RunsOn
	}
	return rec.Reaching, judge.Reaching
}

func (o origins) subject(e Exposure) string {
	if e.Kind == judge.UIPortal {
		return uiSubject
	}
	return o.servers[e.Server].Name
}

// units is one unit per wanted Exposure, tied to its origin.
func (r *Reachability) units(ctx context.Context, rd storage.Reader[storage.Library]) ([]tasks.Unit, error) {
	wanted, err := wantedIn(ctx, rd)
	if err != nil {
		return nil, err
	}
	o, err := originsIn(ctx, rd, r.cfg.Roster.Self())
	if err != nil {
		return nil, err
	}
	out := make([]tasks.Unit, len(wanted))
	for i, e := range wanted {
		place := tasks.OnController()
		if n, _ := o.of(e); n != o.self {
			place = tasks.TiedTo(n)
		}
		out[i] = tasks.Unit{Key: e.Unit(), Label: o.subject(e), Place: place}
	}
	return out, nil
}
