package mbkind

import (
	"slices"
	"time"

	"github.com/jatassi/slipstream/internal/mediaserverlink/link"
	"github.com/jatassi/slipstream/internal/storing"
)

// Plan sends, in order: one Fresh Job per fresh mapping, one Rescan of
// every library row, then the refreshes and then the carries, each one
// Job per step the rows reached together, so a monitor delay is waited
// once for the whole group.
func (f family) Plan(d link.Due) []link.Job {
	p := d.Ready(d.Wait)
	jobs := make([]link.Job, 0, len(p.Fresh)+3)
	for _, fr := range p.Fresh {
		jobs = append(jobs, link.Job{Server: d.Server, Work: fr, Step: link.StepNone})
	}
	var rescan link.Rescan
	var refreshes, carries groups
	for _, r := range p.Rows {
		switch {
		case r.Kind == link.ChangeLibrary:
			if !slices.Contains(rescan.Libraries, r.Library) {
				rescan.Libraries = append(rescan.Libraries, r.Library)
			}
			rescan.Changes = append(rescan.Changes, r.ID)
		case r.FromRel != storing.Rel{}:
			carries.add(r)
		default:
			refreshes.add(r)
		}
	}
	if len(rescan.Changes) > 0 {
		jobs = append(jobs, link.Job{Server: d.Server, Work: rescan, Step: link.StepNone})
	}
	for _, g := range refreshes {
		jobs = append(jobs, link.Job{Server: d.Server, Work: link.Refresh{Changes: g.changes}, Step: g.step, At: g.at})
	}
	for _, g := range carries {
		jobs = append(jobs, link.Job{Server: d.Server, Work: link.Carry{Changes: g.changes}, Step: g.step, At: g.at})
	}
	return jobs
}

// group is the rows that reached one step together.
type group struct {
	step    link.Step
	at      time.Time
	changes []link.JobChange
}

type groups []group

func (gs *groups) add(r link.Row) {
	i := slices.IndexFunc(*gs, func(g group) bool { return g.step == r.Step && g.at.Equal(r.StepAt) })
	if i < 0 {
		*gs = append(*gs, group{step: r.Step, at: r.StepAt})
		i = len(*gs) - 1
	}
	(*gs)[i].changes = append((*gs)[i].changes, r.JobChange)
}
