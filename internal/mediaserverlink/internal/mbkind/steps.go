package mbkind

import (
	"context"
	"fmt"
	"slices"
	"strconv"
	"time"

	"github.com/jatassi/slipstream/internal/mediaserverlink/link"
	"github.com/jatassi/slipstream/internal/providers/jellyfin"
)

// step is one entry of a Work's step list: the step it reaches and the
// call that reaches it. A run that must wait answers the Job at its old
// step with a Wait.
type step[W link.Work] struct {
	at  link.Step
	run func(ctx context.Context, c *conn, j link.Job, w W) (link.Advanced, error)
}

type steps[W link.Work] []step[W]

// next runs the entry after j's step, the first from StepNone.
func (s steps[W]) next(ctx context.Context, c *conn, j link.Job) (link.Advanced, error) {
	i := 0
	if j.Step != link.StepNone {
		i = slices.IndexFunc(s, func(e step[W]) bool { return e.at == j.Step }) + 1
	}
	if i == 0 && j.Step != link.StepNone || i == len(s) {
		return link.Advanced{}, fmt.Errorf("%s has no step after %s", c.f.about.Label, j.Step)
	}
	w, _ := j.Work.(W)
	return s[i].run(ctx, c, j, w)
}

// reached is j at step at now, holding w.
func reached(j link.Job, at link.Step, w link.Work) link.Job {
	j.Step, j.At, j.Work = at, time.Now(), w
	return j
}

func waitFor(j link.Job, d time.Duration) link.Advanced {
	return link.Advanced{Job: j, Wait: time.Now().Add(d)}
}

func accepted(ids ...link.ChangeID) []link.Outcome {
	out := make([]link.Outcome, len(ids))
	for i, id := range ids {
		out[i] = link.Outcome{Change: id, Result: link.ResultAccepted}
	}
	return out
}

func changeIDs(cs []link.JobChange) []link.ChangeID {
	out := make([]link.ChangeID, len(cs))
	for i, ch := range cs {
		out[i] = ch.ID
	}
	return out
}

func (c *conn) Advance(ctx context.Context, j link.Job) (link.Advanced, error) {
	a, err := c.advance(ctx, j)
	return a, problem(ctx, err)
}

func (c *conn) advance(ctx context.Context, j link.Job) (link.Advanced, error) {
	if _, err := c.same(ctx); err != nil {
		return link.Advanced{}, err
	}
	switch w := j.Work.(type) {
	case link.Refresh:
		if c.f.scanDropsReports {
			return settledRefresh.next(ctx, c, j)
		}
		return plainRefresh.next(ctx, c, j)
	case link.Carry:
		return carry.next(ctx, c, j)
	case link.Fresh:
		return c.fresh(ctx, j, w)
	case link.Rescan:
		return c.rescan(ctx, j, w)
	case link.Lookup:
		return c.lookup(ctx, j, w)
	case link.Index:
		return c.index(ctx, j, w)
	case link.Libraries:
		libs, err := c.libraries(ctx)
		return link.Advanced{Job: j, Done: true, Libraries: libs}, err
	}
	return link.Advanced{}, fmt.Errorf("%s has no steps for %T", c.f.about.Label, j.Work)
}

// settledRefresh is Jellyfin's: a running scan would drop the reports, so
// they wait it out, and a scan that starts inside their monitor delay
// sends them again (design D6).
var settledRefresh = steps[link.Refresh]{
	{link.StepIdle, waitIdle},
	{link.StepReported, report},
	{link.StepSettled, settle},
}

var plainRefresh = steps[link.Refresh]{
	{link.StepReported, report},
}

func (c *conn) task(ctx context.Context) (jellyfin.Task, error) {
	t, ok, err := c.c.Task(ctx, jellyfin.RefreshLibraryTask)
	if err != nil {
		return jellyfin.Task{}, err
	}
	if !ok {
		return jellyfin.Task{}, failed("the server has no " + jellyfin.RefreshLibraryTask + " task")
	}
	return t, nil
}

func lastRun(t jellyfin.Task) jellyfin.TaskResult {
	if t.LastExecutionResult == nil {
		return jellyfin.TaskResult{}
	}
	return *t.LastExecutionResult
}

func waitIdle(ctx context.Context, c *conn, j link.Job, w link.Refresh) (link.Advanced, error) {
	t, err := c.task(ctx)
	if err != nil {
		return link.Advanced{}, err
	}
	if t.State != jellyfin.TaskIdle {
		return waitFor(j, c.d.Timings.StepPoll), nil
	}
	return link.Advanced{Job: reached(j, link.StepIdle, w)}, nil
}

// updates are the path reports for cs: the file's path Created, Modified
// or Deleted, and a Sidecar's own path, which each product resolves to
// the item beside it.
func updates(cs []link.JobChange) []jellyfin.Update {
	out := make([]jellyfin.Update, len(cs))
	for i, ch := range cs {
		out[i] = jellyfin.Update{Path: ch.Path.String(), UpdateType: updateTypes[ch.Kind]}
	}
	return out
}

var updateTypes = map[link.ChangeKind]jellyfin.UpdateType{
	link.ChangeAdded:           jellyfin.Created,
	link.ChangeReplaced:        jellyfin.Modified,
	link.ChangeRemoved:         jellyfin.Deleted,
	link.ChangeMoved:           jellyfin.Created,
	link.ChangeSidecarAdded:    jellyfin.Modified,
	link.ChangeSidecarReplaced: jellyfin.Modified,
	link.ChangeSidecarRemoved:  jellyfin.Deleted,
}

func report(ctx context.Context, c *conn, j link.Job, w link.Refresh) (link.Advanced, error) {
	if err := c.c.ReportUpdated(ctx, updates(w.Changes)); err != nil {
		return link.Advanced{}, err
	}
	if !c.f.scanDropsReports {
		return link.Advanced{Job: reached(j, link.StepReported, w), Done: true, Outcomes: accepted(changeIDs(w.Changes)...)}, nil
	}
	return waitFor(reached(j, link.StepReported, w), c.d.MonitorDelay), nil
}

// settle ends the reports once their monitor delay passed with no scan
// started since they were sent. A scan that started drops them, so the
// Job goes back to StepNone and reports again once the scan ends.
func settle(ctx context.Context, c *conn, j link.Job, w link.Refresh) (link.Advanced, error) {
	t, err := c.task(ctx)
	if err != nil {
		return link.Advanced{}, err
	}
	if t.State != jellyfin.TaskIdle || lastRun(t).StartTimeUtc.After(j.At) {
		j.Step, j.At = link.StepNone, time.Time{}
		return link.Advanced{Job: j}, nil
	}
	return link.Advanced{Job: reached(j, link.StepSettled, w), Done: true, Outcomes: accepted(changeIDs(w.Changes)...)}, nil
}

// rescan refreshes whole libraries: Jellyfin's refresh of every library,
// which has no one-library form, or Emby's recursive refresh of each.
func (c *conn) rescan(ctx context.Context, j link.Job, w link.Rescan) (link.Advanced, error) {
	if c.f.flavor == jellyfin.Emby {
		for _, lib := range w.Libraries {
			if err := c.c.RefreshLibrary(ctx, lib); err != nil {
				return link.Advanced{}, err
			}
		}
	} else if err := c.c.RefreshAll(ctx); err != nil {
		return link.Advanced{}, err
	}
	return link.Advanced{Job: reached(j, link.StepScanned, w), Done: true, Outcomes: accepted(w.Changes...)}, nil
}

// fresh is the fresh folder rule for one mapping, its step derived from
// the mapping's columns: an item under the folder means scanned. Else a
// full scan, asked for once Requested is written: the Job comes back with
// Requested set and no scan sent, and the next call sends it. Emby's full
// scan is the same RefreshLibrary task.
//
// Started is the task's last end before the scan was sent, on the
// server's own clock, or Requested when the task never ran. The scan has
// ended when the task is Idle with a later last end (graft 3), which no
// skew between the two clocks can fool. A task already running when the
// scan would be sent is waited on, not started again.
func (c *conn) fresh(ctx context.Context, j link.Job, w link.Fresh) (link.Advanced, error) {
	if w.Requested.IsZero() {
		has, err := c.anyUnder(ctx, w.Library, w.Folder)
		if err != nil || has {
			return link.Advanced{Job: j, Done: true, Mapping: &link.MappingFact{Scanned: true}}, err
		}
		w.Requested = time.Now()
		return link.Advanced{Job: reached(j, link.StepNone, w)}, nil
	}
	t, err := c.task(ctx)
	if err != nil {
		return link.Advanced{}, err
	}
	end, now := lastRun(t).EndTimeUtc, time.Now()
	switch {
	case !w.Started.IsZero() && t.State == jellyfin.TaskIdle && end.After(w.Started):
		return link.Advanced{Job: j, Done: true, Mapping: &link.MappingFact{Started: w.Started, Scanned: true}}, nil
	case now.After(w.Requested.Add(c.d.Timings.ScanEndBound)):
		return link.Advanced{}, failed(fmt.Sprintf("the library scan SlipStream asked for didn't end within %s", c.d.Timings.ScanEndBound))
	case w.Started.IsZero():
		if t.State == jellyfin.TaskIdle {
			if err := c.c.RefreshAll(ctx); err != nil {
				return link.Advanced{}, err
			}
		}
		w.Started = end
		if end.IsZero() {
			w.Started = w.Requested
		}
	}
	poll := min(max(c.d.Timings.ScanEndPoll, now.Sub(w.Requested)), c.d.Timings.ScanEndPollMax)
	return link.Advanced{Job: reached(j, link.StepNone, w), Wait: now.Add(poll), Mapping: &link.MappingFact{Started: w.Started}}, nil
}

// anyUnder is whether the library holds an item with a file under folder.
func (c *conn) anyUnder(ctx context.Context, library string, folder link.ServerPath) (bool, error) {
	us, err := c.users(ctx)
	if err != nil {
		return false, err
	}
	size := c.d.Timings.IndexPage
	for start := 0; ; start += size {
		page, err := c.c.Items(ctx, jellyfin.Query{User: us[0].ID, Parent: library, Recursive: true, Types: videoTypes, Start: start, Limit: size})
		if err != nil {
			return false, err
		}
		for _, it := range page.Items {
			if slices.ContainsFunc(files(it), func(p link.ServerPath) bool { return p.Within(folder) }) {
				return true, nil
			}
		}
		if start+size >= page.Total {
			return false, nil
		}
	}
}

// lookup reads the item at each path in whichever library holds it.
func (c *conn) lookup(ctx context.Context, j link.Job, w link.Lookup) (link.Advanced, error) {
	libs, err := c.libraries(ctx)
	if err != nil {
		return link.Advanced{}, err
	}
	us, err := c.users(ctx)
	if err != nil {
		return link.Advanced{}, err
	}
	asked, libOf := want{}, map[string]string{}
	for _, p := range w.Paths {
		for _, l := range libs {
			if slices.ContainsFunc(l.Folders, p.Within) {
				asked.add(l.Key, p)
				libOf[p.Key()] = l.Key
				break
			}
		}
	}
	got, err := c.find(ctx, us[0].ID, asked)
	if err != nil {
		return link.Advanced{}, err
	}
	a := link.Advanced{Job: j, Done: true}
	for _, p := range w.Paths {
		e := link.IndexEntry{Path: p, Library: libOf[p.Key()], Gone: true}
		if it, ok := got.at(p); ok {
			e.Item, e.Gone = key(it), false
		}
		a.Learned = append(a.Learned, e)
	}
	return a, nil
}

// index reads one page of a library for the path index: an entry per
// file. The Job answered carries the next page's cursor, and is done
// after the last.
func (c *conn) index(ctx context.Context, j link.Job, w link.Index) (link.Advanced, error) {
	start := 0
	if w.Cursor != "" {
		n, err := strconv.Atoi(w.Cursor)
		if err != nil {
			return link.Advanced{}, fmt.Errorf("read the index cursor %q: %w", w.Cursor, err)
		}
		start = n
	}
	us, err := c.users(ctx)
	if err != nil {
		return link.Advanced{}, err
	}
	size := c.d.Timings.IndexPage
	page, err := c.c.Items(ctx, jellyfin.Query{User: us[0].ID, Parent: w.Library, Recursive: true, Types: videoTypes, Start: start, Limit: size})
	if err != nil {
		return link.Advanced{}, err
	}
	var a link.Advanced
	for _, it := range page.Items {
		for _, f := range files(it) {
			a.Learned = append(a.Learned, link.IndexEntry{Path: f, Library: w.Library, Item: key(it)})
		}
	}
	w.Cursor = ""
	if next := start + len(page.Items); len(page.Items) > 0 && next < page.Total {
		w.Cursor = strconv.Itoa(next)
	}
	j.Work = w
	a.Job, a.Done = j, w.Cursor == ""
	return a, nil
}
