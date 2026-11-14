package link

import (
	"time"

	"github.com/jatassi/slipstream/internal/roots"
	"github.com/jatassi/slipstream/internal/storing"
)

// Due is a server's open rows as Plan sees them.
type Due struct {
	Server ServerRef
	Now    time.Time
	// Rows are the server's due rows in created order. A row at a started
	// step is due again between its steps.
	Rows     []Row
	Mappings []PlanMapping
	Timings  Timings
}

// Row is one due change row.
type Row struct {
	JobChange
	Root roots.ID
	// Rel is the root-relative path, zero for a library change. FromRel is
	// zero unless the row is a carry.
	Rel, FromRel storing.Rel
	Created      time.Time
	// NextAt is the row's next_attempt_at: a retry's backoff, or a step's
	// timed wait.
	NextAt   time.Time
	Step     Step
	Attempts int
	// Pairing is a half of an upgrade waiting for the other: a removed that
	// names its replacer, or an added that a removed will follow.
	Pairing bool
}

// PlanMapping is one root's mapping on the server.
type PlanMapping struct {
	Root      roots.ID
	Library   string
	Folder    ServerPath
	Shown     bool
	Confirmed bool
	Scanned   bool
	Requested time.Time
	Started   time.Time
}

// Pick is what one pass may send.
type Pick struct {
	// Fresh is one Job per mapping whose ready rows wait on the fresh folder
	// rule. Those rows are not in Rows.
	Fresh []Fresh
	// Rows are the ready rows, in created order.
	Rows []Row
	// Next is when a row held by time could become ready, zero when none
	// is.
	Next time.Time
}

func (r Row) carry() bool { return r.FromRel != (storing.Rel{}) }

// fresh is whether the row has never been sent: a new row waits out Wait
// before sending, a retry or a started step doesn't.
func (r Row) fresh() bool { return r.Attempts == 0 && !r.Step.Started() }

type pathKey struct {
	root roots.ID
	rel  storing.Rel
}

func (r Row) paths() []pathKey {
	if r.Kind == ChangeLibrary {
		return nil
	}
	ps := []pathKey{{r.Root, r.Rel}}
	if r.carry() {
		ps = append(ps, pathKey{r.Root, r.FromRel})
	}
	return ps
}

func (r Row) touches(o Row) bool {
	for _, p := range r.paths() {
		for _, q := range o.paths() {
			if p == q {
				return true
			}
		}
	}
	return false
}

func (r Row) until(t Timings) time.Time {
	if r.Pairing {
		return later(r.NextAt, r.Created.Add(t.UpgradePairing))
	}
	return r.NextAt
}

func later(a, b time.Time) time.Time {
	if a.After(b) {
		return a
	}
	return b
}

func (p *Pick) wake(at time.Time) {
	if p.Next.IsZero() || at.Before(p.Next) {
		p.Next = at
	}
}

// Ready applies the rules every kind shares, in this order:
//
//   - A row under a mapping that isn't shown and confirmed waits until the
//     Admin confirms it.
//   - A row waits for its next_attempt_at, and either half of an upgrade
//     waits until its partner folds in or Timings.UpgradePairing passes.
//   - New rows wait until the oldest of them has waited wait.
//   - A row into a mapping with no scanned_at waits behind that mapping's
//     Fresh Job.
//   - A row waits for every earlier row that touches one of its paths,
//     except that a carry may go in the same pass as an earlier ready
//     refresh, which a kind sends first.
func (d Due) Ready(wait time.Duration) Pick {
	maps := make(map[roots.ID]PlanMapping, len(d.Mappings))
	for _, m := range d.Mappings {
		maps[m.Root] = m
	}
	var p Pick
	held := make([]bool, len(d.Rows))
	d.holdUntilDue(maps, held, wait, &p)
	d.holdFresh(maps, held, &p)
	for i, r := range d.Rows {
		if !held[i] && !d.behind(i, held) {
			p.Rows = append(p.Rows, r)
			continue
		}
		held[i] = true
	}
	return p
}

// holdUntilDue holds rows under a mapping that isn't shown and confirmed,
// rows waiting on time, and new rows until the oldest has waited wait.
func (d Due) holdUntilDue(maps map[roots.ID]PlanMapping, held []bool, wait time.Duration, p *Pick) {
	var oldest time.Time
	for i, r := range d.Rows {
		m, ok := maps[r.Root]
		if !ok || !m.Shown || !m.Confirmed {
			held[i] = true
			continue
		}
		if until := r.until(d.Timings); until.After(d.Now) {
			held[i] = true
			p.wake(until)
			continue
		}
		if r.fresh() && (oldest.IsZero() || r.Created.Before(oldest)) {
			oldest = r.Created
		}
	}
	if oldest.IsZero() || !d.Now.Before(oldest.Add(wait)) {
		return
	}
	for i, r := range d.Rows {
		held[i] = held[i] || r.fresh()
	}
	p.wake(oldest.Add(wait))
}

// holdFresh holds the ready rows into a mapping with no scanned_at, and
// adds one Fresh Job for each such mapping.
func (d Due) holdFresh(maps map[roots.ID]PlanMapping, held []bool, p *Pick) {
	fresh := map[roots.ID]bool{}
	for i, r := range d.Rows {
		m := maps[r.Root]
		if held[i] || r.Kind == ChangeLibrary || m.Scanned {
			continue
		}
		held[i] = true
		if !fresh[r.Root] {
			fresh[r.Root] = true
			p.Fresh = append(p.Fresh, Fresh{Library: m.Library, Folder: m.Folder, Requested: m.Requested, Started: m.Started})
		}
	}
}

// behind is whether row i waits for an earlier row on one of its paths. A
// carry goes beside an earlier ready refresh, which a kind sends first.
func (d Due) behind(i int, held []bool) bool {
	r := d.Rows[i]
	for j, e := range d.Rows[:i] {
		if r.touches(e) && (held[j] || e.carry() || !r.carry()) {
			return true
		}
	}
	return false
}
