package plexkind

import (
	"slices"
	"time"

	"github.com/jatassi/slipstream/internal/mediaserverlink/link"
)

// Plan runs the shared rules, then answers in this order: one Fresh Job
// per fresh mapping, one Rescan of the library changes and of each section
// past Timings.SectionFolders folders, one Refresh of the rest, and one
// Carry per step the ready carries are at. A carry not yet confirmed holds
// back every refresh of its source folder and every refresh of its
// section as a whole, since Plex deletes an item whose folder is refreshed
// before its new path is (recorded run 2026-11-14, case E).
func (kind) Plan(d link.Due) []link.Job {
	pick := d.Ready(d.Wait)
	open := openCarries(d.Rows)
	var jobs []link.Job
	for _, f := range pick.Fresh {
		jobs = append(jobs, link.Job{Server: d.Server, Work: f, Step: link.StepNone})
	}
	var library, refreshes, carries []link.Row
	for _, r := range pick.Rows {
		switch {
		case r.Kind == link.ChangeLibrary:
			if !open.section(r.Library) {
				library = append(library, r)
			}
		case !r.From.IsZero():
			carries = append(carries, r)
		case !open.source(r):
			refreshes = append(refreshes, r)
		}
	}
	whole, folders := split(refreshes, open, d.Timings.SectionFolders)
	if rs := slices.Concat(library, whole); len(rs) > 0 {
		jobs = append(jobs, link.Job{Server: d.Server, Work: rescanOf(rs), Step: link.StepNone})
	}
	if len(folders) > 0 {
		jobs = append(jobs, link.Job{Server: d.Server, Work: link.Refresh{Changes: changes(folders)}, Step: link.StepNone})
	}
	return append(jobs, carryJobs(d.Server, carries)...)
}

// opened is the carries not yet confirmed at their new paths.
type opened struct{ rows []link.Row }

func openCarries(rows []link.Row) opened {
	var o opened
	for _, r := range rows {
		if !r.From.IsZero() && r.Step != link.StepConfirmed && r.Step != link.StepSourceScanned {
			o.rows = append(o.rows, r)
		}
	}
	return o
}

func (o opened) section(library string) bool {
	return slices.ContainsFunc(o.rows, func(c link.Row) bool { return c.Library == library })
}

// source is whether r's folder is the old folder of an open carry.
func (o opened) source(r link.Row) bool {
	return slices.ContainsFunc(o.rows, func(c link.Row) bool {
		return c.Library == r.Library && c.From.Dir().Key() == r.Path.Dir().Key()
	})
}

// split is the refreshes of each section whose distinct video folders
// pass limit and that no open carry is in, and the rest. Sidecar changes
// always go as item refreshes.
func split(rows []link.Row, open opened, limit int) (whole, folders []link.Row) {
	count := map[string]map[string]bool{}
	for _, r := range rows {
		if sidecar(r.Kind) {
			continue
		}
		if count[r.Library] == nil {
			count[r.Library] = map[string]bool{}
		}
		count[r.Library][r.Path.Dir().Key()] = true
	}
	for _, r := range rows {
		if !sidecar(r.Kind) && len(count[r.Library]) > limit && !open.section(r.Library) {
			whole = append(whole, r)
		} else {
			folders = append(folders, r)
		}
	}
	return whole, folders
}

func rescanOf(rows []link.Row) link.Rescan {
	var w link.Rescan
	for _, r := range rows {
		if !slices.Contains(w.Libraries, r.Library) {
			w.Libraries = append(w.Libraries, r.Library)
		}
		w.Changes = append(w.Changes, r.ID)
	}
	return w
}

func changes(rows []link.Row) []link.JobChange {
	out := make([]link.JobChange, len(rows))
	for i, r := range rows {
		out[i] = r.JobChange
	}
	return out
}

// carryJobs is one Carry per step the rows are at, in the order the rows
// reach it, at the earliest time a row reached that step.
func carryJobs(server link.ServerRef, rows []link.Row) []link.Job {
	var jobs []link.Job
	for _, r := range rows {
		i := slices.IndexFunc(jobs, func(j link.Job) bool { return j.Step == r.Step })
		if i < 0 {
			jobs = append(jobs, link.Job{Server: server, Work: link.Carry{}, Step: r.Step, At: r.StepAt})
			i = len(jobs) - 1
		}
		w, _ := jobs[i].Work.(link.Carry)
		w.Changes = append(w.Changes, r.JobChange)
		jobs[i].Work = w
		jobs[i].At = earliest(jobs[i].At, r.StepAt)
	}
	return jobs
}

func earliest(a, b time.Time) time.Time {
	if b.Before(a) {
		return b
	}
	return a
}
