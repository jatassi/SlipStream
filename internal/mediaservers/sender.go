package mediaservers

import (
	"cmp"
	"context"
	"errors"
	"slices"
	"sync"
	"sync/atomic"
	"time"

	"github.com/jatassi/slipstream/internal/mediaserverlink"
	"github.com/jatassi/slipstream/internal/mediaserverlink/link"
	"github.com/jatassi/slipstream/internal/mediaservers/internal/db"
	"github.com/jatassi/slipstream/internal/mediaservers/queue"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/roots"
	"github.com/jatassi/slipstream/internal/storage"
)

// senders are the Controller's one sender per server, so a wait on one
// server never delays another.
type senders struct {
	mu    sync.Mutex
	wakes map[link.ServerID]*wake
	// follow asks Run to start a sender for each server without one.
	follow chan struct{}
}

// wake is one sender's doorbell. told is set when new rows came, so its
// next pass tells the pages.
type wake struct {
	ring chan struct{}
	told atomic.Bool
}

func newSenders() *senders {
	return &senders{wakes: map[link.ServerID]*wake{}, follow: make(chan struct{}, 1)}
}

func ring(c chan struct{}) {
	select {
	case c <- struct{}{}:
	default:
	}
}

// wake rings each server's sender, and asks for a sender for a server
// that has none yet.
func (s *senders) wake(ids ...link.ServerID) { s.ring(false, ids) }

// told is wake for servers that got new rows.
func (s *senders) told(ids ...link.ServerID) { s.ring(true, ids) }

func (s *senders) ring(told bool, ids []link.ServerID) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, id := range ids {
		w, ok := s.wakes[id]
		if !ok {
			ring(s.follow)
			continue
		}
		if told {
			w.told.Store(true)
		}
		ring(w.ring)
	}
}

func (s *senders) wakeAll() {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, w := range s.wakes {
		ring(w.ring)
	}
}

// start is the sender of each of ids that has none yet.
func (s *senders) start(ids []link.ServerID) map[link.ServerID]*wake {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := map[link.ServerID]*wake{}
	for _, id := range ids {
		if _, ok := s.wakes[id]; !ok {
			w := &wake{ring: make(chan struct{}, 1)}
			s.wakes[id], out[id] = w, w
		}
	}
	return out
}

func (s *senders) forget(id link.ServerID) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.wakes, id)
}

// follow starts a sender for each server without one.
func (m *MediaServers) follow(ctx context.Context, wg *sync.WaitGroup) {
	var ids []link.ServerID
	err := m.cfg.Library.Read(ctx, func(ctx context.Context, r storage.Reader[storage.Library]) error {
		var err error
		ids, err = db.New(r).ListServerIDs(ctx)
		return err
	})
	if err != nil {
		if ctx.Err() == nil {
			m.cfg.Log.Error("list the Media Servers to start their senders", "err", err.Error())
		}
		return
	}
	for id, w := range m.senders.start(ids) {
		wg.Go(func() { m.send(ctx, id, w) })
	}
}

// revert returns every row a sender was sending when the Controller
// stopped to due, before any sender starts.
func (m *MediaServers) revert(ctx context.Context) {
	err := m.cfg.Library.Write(ctx, func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
		q := db.New(tx)
		rows, err := q.ListSending(ctx)
		if err != nil {
			return err
		}
		return reopenRows(ctx, q, rows)
	})
	if err != nil && ctx.Err() == nil {
		m.cfg.Log.Error("return the changes cut off at the last stop to the Media Server queues", "err", err.Error())
	}
}

// send is one server's sender: it runs a pass whenever it is woken or a
// held row may be ready, until the server is removed.
func (m *MediaServers) send(ctx context.Context, id link.ServerID, w *wake) {
	defer m.senders.forget(id)
	for {
		next, gone := m.pass(ctx, id, w)
		if gone || ctx.Err() != nil {
			return
		}
		if !m.sleep(ctx, w, next) {
			return
		}
	}
}

// sleep waits until next, zero for no time, or a ring. It reports false
// once ctx ends.
func (m *MediaServers) sleep(ctx context.Context, w *wake, next time.Time) bool {
	var timer <-chan time.Time
	if !next.IsZero() {
		t := time.NewTimer(time.Until(next))
		defer t.Stop()
		timer = t.C
	}
	select {
	case <-ctx.Done():
		return false
	case <-w.ring:
	case <-timer:
	}
	return true
}

var errGone = errors.New("the Media Server was removed")

// via is how a sender reaches its server for one pass.
type via int

const (
	// viaReaching is the reaching Node, the Controller's own through Local.
	viaReaching via = iota
	// viaFallback is the Controller in process at the fallback address,
	// while the reaching Node is away and the Controller's own test passed.
	viaFallback
	// viaHold sends nothing: the changes stay due until the Node is back.
	viaHold
)

// routeOf is the route while the reaching Node's link is up or not, for
// the server's fallback state.
func routeOf(up bool, fallback string) via {
	switch {
	case up:
		return viaReaching
	case fallback == "available":
		return viaFallback
	}
	return viaHold
}

func (m *MediaServers) up(node role.NodeID) bool {
	return node == m.cfg.Roster.Self() || m.cfg.Roster.InStep(node)
}

// sending is one pass's taken Jobs and the route they go by.
type sending struct {
	id   link.ServerID
	port mediaserverlink.Port
	jobs []link.Job
	// libraries are the roots each library key's mappings lead into, for
	// the changes a whole library refresh covers.
	libraries map[string][]roots.ID
	// next is when a row held by time may be ready, zero for none.
	next time.Time
}

// pass takes what is ready and sends it. It answers when to pass again,
// zero for when woken, and whether the server is gone.
func (m *MediaServers) pass(ctx context.Context, id link.ServerID, w *wake) (time.Time, bool) {
	told := w.told.Swap(false)
	s, err := m.take(ctx, id)
	switch {
	case errors.Is(err, errGone):
		return time.Time{}, true
	case err != nil:
		if ctx.Err() == nil {
			m.cfg.Log.Error("take a Media Server's due changes", "server", id.String(), "err", err.Error())
		}
		return time.Now().Add(m.cfg.Timings.RetryMin), false
	}
	if told || len(s.jobs) > 0 {
		m.publish(ctx, id)
	}
	if len(s.jobs) == 0 {
		return s.next, false
	}
	for i, j := range s.jobs {
		until, err := m.drive(ctx, s, j)
		if err == nil {
			continue
		}
		m.putBack(ctx, s.jobs[i+1:])
		if _, server := link.ServerProblem(err); !server {
			m.cfg.Log.Warn("a step on a Media Server was cut off and resumes at that step", "server", id.String(), "err", err.Error())
		}
		return until, false
	}
	return time.Now(), false
}

// take plans the server's due rows and takes the Jobs' rows to send, in
// one Write: a change's Write ends before it starts, so it plans from
// every row recorded so far.
func (m *MediaServers) take(ctx context.Context, id link.ServerID) (sending, error) {
	s := sending{id: id}
	err := m.cfg.Library.Write(ctx, func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
		q := db.New(tx)
		row, err := q.GetServer(ctx, id)
		switch {
		case errors.Is(err, storage.ErrNoRows):
			return errGone
		case err != nil:
			return err
		}
		v := routeOf(m.up(row.ReachingNodeID), row.Fallback)
		if v == viaHold {
			return nil
		}
		k, _ := m.cfg.Kinds.Of(row.Kind)
		d, err := m.due(ctx, q, row, &s)
		if err != nil {
			return err
		}
		s.next = d.Ready(d.Wait).Next
		for _, j := range sendable(k.Plan(d)) {
			if err := q.TakeChanges(ctx, db.TakeChangesParams{ServerID: id, Ids: jobIDs(j)}); err != nil {
				return err
			}
			s.jobs = append(s.jobs, j)
		}
		if len(s.jobs) == 0 {
			return nil
		}
		s.port, err = m.port(ctx, q, row, k, v)
		return err
	})
	return s, err
}

// sendable is the Jobs of plan one pass sends. A whole library refresh
// goes alone: the changes it covers are sent by it, so the next pass
// plans again without them.
func sendable(plan []link.Job) []link.Job {
	for _, j := range plan {
		if _, ok := j.Work.(link.Rescan); ok {
			return []link.Job{j}
		}
	}
	return plan
}

func (m *MediaServers) port(ctx context.Context, q *db.Queries, row db.MediaserversServer, k link.Kind, v via) (mediaserverlink.Port, error) {
	if v == viaReaching {
		return mediaserverlink.Remote(m.cfg.Link(row.ReachingNodeID)), nil
	}
	cred, err := unsealCredential(m.cfg.Sealer, row.ID, row.SecretsSealed)
	if err != nil {
		return nil, err
	}
	device, err := deviceOf(ctx, q, row)
	if err != nil {
		return nil, err
	}
	var delay time.Duration
	if row.MonitorDelayMs != nil {
		delay = time.Duration(*row.MonitorDelayMs) * time.Millisecond
	}
	return mediaserverlink.Direct(k, link.Dialed{
		Server:  link.ServerRef{ID: row.ID, Kind: row.Kind, Identity: row.Identity, SettingsVersion: row.SettingsVersion},
		Address: cmp.Or(deref(row.FallbackAddress), row.Address), Credential: cred, DeviceID: device,
		MonitorDelay: delay, HTTP: m.cfg.HTTP, Timings: m.cfg.Timings,
	}), nil
}

// due is the server's due rows as Plan reads them, their server paths
// joined from their mappings.
func (m *MediaServers) due(ctx context.Context, q *db.Queries, row db.MediaserversServer, s *sending) (link.Due, error) {
	settings, err := q.GetSettings(ctx)
	if err != nil {
		return link.Due{}, err
	}
	d := link.Due{
		Server: link.ServerRef{ID: row.ID, Kind: row.Kind, Identity: row.Identity, SettingsVersion: row.SettingsVersion},
		Now:    time.Now(), Wait: WaitBeforeSending.Value(settings.WaitSeconds), Timings: m.cfg.Timings,
	}
	mappings, err := q.ListMappings(ctx, row.ID)
	if err != nil {
		return d, err
	}
	folders := map[roots.ID]link.PlanMapping{}
	s.libraries = map[string][]roots.ID{}
	for _, mp := range mappings {
		pm := link.PlanMapping{
			Root: mp.RootID, Shown: mp.Shown, Confirmed: !mp.ConfirmedAt.IsZero(), Scanned: !mp.ScannedAt.IsZero(),
			Requested: mp.ScanRequestedAt.Time(), Started: mp.ScanStartedAt.Time(),
		}
		if mp.Shown {
			if pm.Folder, err = link.ParsePath(*mp.ServerFolder); err != nil {
				return d, err
			}
			pm.Library = *mp.LibraryKey
			s.libraries[pm.Library] = append(s.libraries[pm.Library], mp.RootID)
		}
		folders[mp.RootID] = pm
		d.Mappings = append(d.Mappings, pm)
	}
	rows, err := q.ListDue(ctx, row.ID)
	if err != nil {
		return d, err
	}
	cs, err := changesOf(rows)
	if err != nil {
		return d, err
	}
	for _, c := range cs {
		d.Rows = append(d.Rows, rowOf(c, folders[c.Root]))
	}
	return d, nil
}

// rowOf is c as Plan reads it, at its server path under mp.
func rowOf(c queue.Change, mp link.PlanMapping) link.Row {
	r := link.Row{
		JobChange: link.JobChange{ID: c.ID, Kind: c.Kind, Library: mp.Library, Snapshot: c.Snapshot},
		Root:      c.Root, Rel: c.Path, FromRel: c.From, Created: c.Created, NextAt: c.NextAt,
		Step: c.Step, StepAt: c.StepAt, Attempts: c.Attempts, Pairing: c.Pairing(),
	}
	if mp.Folder.IsZero() {
		return r
	}
	if c.Kind != link.ChangeLibrary {
		r.Path = mp.Folder.Join(c.Path)
	}
	if c.Carry() {
		r.From = mp.Folder.Join(c.From)
	}
	return r
}

// jobIDs are the change rows j sends.
func jobIDs(j link.Job) []link.ChangeID {
	switch w := j.Work.(type) {
	case link.Refresh:
		return changeIDs(w.Changes)
	case link.Carry:
		return changeIDs(w.Changes)
	case link.Rescan:
		return w.Changes
	case link.Fresh:
		return w.Changes
	}
	return nil
}

func changeIDs(cs []link.JobChange) []link.ChangeID {
	out := make([]link.ChangeID, len(cs))
	for i, c := range cs {
		out[i] = c.ID
	}
	return out
}

// drive runs j one step per call until it is done, waits or fails,
// recording each step on its rows. A failed call answers when the next
// step may start: never before the call's deadline, by when the reaching
// Node's handler has ended too.
func (m *MediaServers) drive(ctx context.Context, s sending, j link.Job) (time.Time, error) {
	for {
		start := time.Now()
		call, cancel := context.WithTimeout(ctx, m.cfg.Timings.Call)
		out, err := s.port.Advance(call, j)
		cancel()
		more := err == nil && !out.Done && out.Wait.IsZero()
		if rerr := m.record(ctx, s, j, out, err, start, more); rerr != nil {
			if ctx.Err() == nil {
				m.cfg.Log.Error("record a step on a Media Server", "server", s.id.String(), "err", rerr.Error())
			}
			return start.Add(m.cfg.Timings.Call), rerr
		}
		m.publish(ctx, s.id)
		if err != nil {
			return start.Add(m.cfg.Timings.Call), err
		}
		if !more {
			return time.Time{}, nil
		}
		j = out.Job
	}
}

// record writes one step's answer on j's rows. A server's failure counts
// an attempt and backs the rows off. A link's failure leaves them at their
// step with no attempt counted, not before the call's deadline. more is a
// Job going on at once, whose rows stay taken.
func (m *MediaServers) record(ctx context.Context, s sending, j link.Job, out link.Advanced, callErr error, start time.Time, more bool) error {
	return m.cfg.Library.Write(ctx, func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
		q := db.New(tx)
		rows, err := q.GetChanges(ctx, jobIDs(j))
		if err != nil {
			return err
		}
		cs, err := changesOf(rows)
		if err != nil {
			return err
		}
		if callErr != nil {
			return m.failed(ctx, q, s.id, cs, callErr, later(time.Now(), start.Add(m.cfg.Timings.Call)))
		}
		if f, ok := out.Job.Work.(link.Fresh); ok && len(cs) > 0 {
			if err := scan(ctx, q, s.id, cs[0].Root, f, out.Done); err != nil {
				return err
			}
		}
		st := stepped{s: s, out: out, start: start, now: time.Now(), more: more, ended: map[link.ChangeID]link.Outcome{}, snaps: snapshotsOf(out.Job)}
		for _, o := range out.Outcomes {
			st.ended[o.Change] = o
		}
		for _, c := range cs {
			if err := st.land(ctx, q, c); err != nil {
				return err
			}
		}
		return q.SetSendsOK(ctx, db.SetSendsOKParams{ID: s.id, At: storage.FromTime(st.now)})
	})
}

// stepped is one step's answer as record writes it on each row.
type stepped struct {
	s          sending
	out        link.Advanced
	start, now time.Time
	more       bool
	ended      map[link.ChangeID]link.Outcome
	snaps      map[link.ChangeID]*link.Snapshot
}

// land writes c at the Job's new step: sent when its change ended, still
// taken when the Job goes on, else due again at the step's wait.
func (st stepped) land(ctx context.Context, q *db.Queries, c queue.Change) error {
	c.Step, c.StepAt, c.Problem = st.out.Job.Step, st.out.Job.At, nil
	if c.Step == link.StepNone {
		c.StepAt = time.Time{}
	}
	if c.Snapshot == nil {
		c.Snapshot = st.snaps[c.ID]
	}
	o, done := st.ended[c.ID]
	switch {
	case done:
		c.State, c.Sent, c.Result, c.Lost = queue.Sent, st.now, o.Result, o.Lost
		if err := apply(ctx, q, []queue.Op{{Do: queue.Update, Change: c}}); err != nil || c.Kind != link.ChangeLibrary {
			return err
		}
		return q.CoverLibrary(ctx, db.CoverLibraryParams{
			At: storage.FromTime(st.now), ServerID: st.s.id, Before: storage.FromTime(st.start), Roots: st.s.libraries[libraryOf(st.s, c.Root)],
		})
	case st.more:
		return apply(ctx, q, []queue.Op{{Do: queue.Update, Change: c}})
	}
	c.NextAt = later(st.now, st.out.Wait)
	return reopen(ctx, q, c)
}

// scan writes a Fresh Job's answer on root's mapping: the decision and
// the server's last scan end before it, and scanned_at once it is done.
// Its rows wait until then and go again once it is done.
func scan(ctx context.Context, q *db.Queries, server link.ServerID, root roots.ID, f link.Fresh, done bool) error {
	p := db.SetMappingScanParams{
		ServerID: server, RootID: root, RequestedAt: storage.FromTime(f.Requested), StartedAt: storage.FromTime(f.Started),
	}
	if done {
		p.ScannedAt = storage.Now()
	}
	return q.SetMappingScan(ctx, p)
}

// failed backs cs off after callErr, not before notBefore.
func (m *MediaServers) failed(ctx context.Context, q *db.Queries, id link.ServerID, cs []queue.Change, callErr error, notBefore time.Time) error {
	p, server := link.ServerProblem(callErr)
	for _, c := range cs {
		c.NextAt = notBefore
		if server {
			c.Attempts++
			c.Problem = &p
			c.NextAt = later(notBefore, time.Now().Add(m.backoff(c.Attempts)))
		}
		if err := reopen(ctx, q, c); err != nil {
			return err
		}
	}
	if !server {
		return nil
	}
	return q.SetSendsFailing(ctx, db.SetSendsFailingParams{ID: id, At: storage.Now()})
}

// backoff is the wait after a change's attempts-th failure: RetryMin,
// doubling up to RetryMax.
func (m *MediaServers) backoff(attempts int) time.Duration {
	d := m.cfg.Timings.RetryMin
	for range attempts - 1 {
		if d >= m.cfg.Timings.RetryMax {
			break
		}
		d *= 2
	}
	return min(d, m.cfg.Timings.RetryMax)
}

func libraryOf(s sending, root roots.ID) string {
	for key, rs := range s.libraries {
		if slices.Contains(rs, root) {
			return key
		}
	}
	return ""
}

// snapshotsOf is each snapshot a carry's step took.
func snapshotsOf(j link.Job) map[link.ChangeID]*link.Snapshot {
	out := map[link.ChangeID]*link.Snapshot{}
	if w, ok := j.Work.(link.Carry); ok {
		for _, c := range w.Changes {
			if c.Snapshot != nil {
				out[c.ID] = c.Snapshot
			}
		}
	}
	return out
}

// putBack returns the rows of Jobs a pass took but didn't send to due.
func (m *MediaServers) putBack(ctx context.Context, jobs []link.Job) {
	var ids []link.ChangeID
	for _, j := range jobs {
		ids = append(ids, jobIDs(j)...)
	}
	if len(ids) == 0 {
		return
	}
	err := m.cfg.Library.Write(ctx, func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
		q := db.New(tx)
		rows, err := q.GetChanges(ctx, ids)
		if err != nil {
			return err
		}
		return reopenRows(ctx, q, rows)
	})
	if err != nil && ctx.Err() == nil {
		m.cfg.Log.Error("return a Media Server's unsent changes to due", "err", err.Error())
	}
}

func later(a, b time.Time) time.Time {
	if b.After(a) {
		return b
	}
	return a
}

// publish tells the pages a server's changes moved.
func (m *MediaServers) publish(ctx context.Context, id link.ServerID) {
	m.cfg.Events.Changed.Publish(ctx, Changed{ServerIDs: []string{id.String()}})
}
