package metadata

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"slices"
	"strings"
	"time"

	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/live"
	"github.com/jatassi/slipstream/internal/metadata/internal/db"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/setting"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/targets"
	"github.com/jatassi/slipstream/internal/tasks"
	"github.com/jatassi/slipstream/internal/works"
)

// RefreshTaskID is the Metadata refresh task.
const RefreshTaskID tasks.TaskID = "metadata-refresh"

var refreshCadence = setting.Duration{Key: "tasks.metadata-refresh.cadence", Label: "Runs every", Default: day, Min: day, Max: 7 * day, Step: day, Floor: sharedKeys}

// refreshSpread is how far into the window a night's run may start, so
// every install doesn't call TMDB at the same minute.
const refreshSpread = 30 * time.Minute

type RefreshConfig struct {
	Library  *storage.DB[storage.Library]
	Metadata *Metadata
	Works    *works.Works
	Targets  *targets.Targets
	// Types are the library halves, which say which Works can still gain
	// Items.
	Types map[works.TypeID]Refreshing
	// Today is the time now in the Controller's zone, whose calendar days
	// the tiers count.
	Today func() time.Time
	Log   *slog.Logger
}

// RefreshTask is Metadata refresh: each night in the Maintenance window it
// refreshes every Work due, one unit each, from the Node holding the
// Metadata Role.
func RefreshTask(cfg RefreshConfig) tasks.Task {
	r := refresh{cfg}
	return tasks.Task{
		ID:      RefreshTaskID,
		Name:    "Metadata refresh",
		About:   "Refreshes the Works their metadata source reports changed. Then it refreshes every Work that is due by the intervals below.",
		Group:   tasks.GroupOf(role.Metadata.Group()),
		Cadence: tasks.InWindow{Days: refreshCadence, Spread: refreshSpread},
		Overdue: tasks.Wait,
		Noun:    setting.Noun{One: "Work", Many: "Works"},
		Shown:   shown,
		Prepare: r.prepare,
		Units:   r.units,
		Run:     r.run,
		Follows: []live.Event{works.ChangedEvent, works.TypeChangedEvent},
		Tuning:  cfg.Metadata.Tuning(),
	}
}

// RefreshAll asks the next Metadata refresh to refresh every Work of type
// t, however recently each was refreshed.
func (m *Metadata) RefreshAll(ctx context.Context, _ identity.Admin, t works.TypeID) error {
	if !slices.Contains(m.cfg.Types, t) {
		return works.ErrNoType
	}
	return m.cfg.Library.Write(ctx, func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
		return db.New(tx).RefreshAll(ctx, db.RefreshAllParams{ContentType: string(t), RefreshAllAt: storage.Now()})
	})
}

// RefreshAllAt is when every Work of t was last asked to refresh, by
// Refresh all or a new language or region, zero if never. The first run
// of Metadata refresh started at or after it refreshes them all.
func (m *Metadata) RefreshAllAt(ctx context.Context, _ identity.Admin, t works.TypeID) (time.Time, error) {
	if !slices.Contains(m.cfg.Types, t) {
		return time.Time{}, works.ErrNoType
	}
	at, err := db.New(m.cfg.Library).GetRefreshAll(ctx, string(t))
	if errors.Is(err, storage.ErrNoRows) {
		return time.Time{}, nil
	}
	return at.Time(), err
}

// shown lists the task once the library holds a Work.
func shown(ctx context.Context, r storage.Reader[storage.Library]) (bool, error) {
	held, err := works.Held(ctx, r)
	return len(held) > 0, err
}

type refresh struct{ cfg RefreshConfig }

// prepare reads each type's changes list since the last list read, when
// the changes switch is on, and marks the Works it names due. It drops
// every mark a refresh has since satisfied.
func (r refresh) prepare(ctx context.Context, _ tasks.Trigger) {
	started := storage.Now()
	var held []works.TypeID
	err := r.cfg.Library.Read(ctx, func(ctx context.Context, rd storage.Reader[storage.Library]) error {
		var err error
		held, err = works.Held(ctx, rd)
		return err
	})
	if err != nil {
		r.cfg.Log.Error("read the Content Types holding Works", "err", err)
		return
	}
	if FollowChanges.Value(r.cfg.Metadata.stored().FollowChanges) {
		for _, t := range held {
			r.follow(ctx, t, started)
		}
	}
	if err := r.cfg.Library.Write(ctx, purge); err != nil {
		r.cfg.Log.Error("drop the satisfied due marks", "err", err)
	}
}

// follow reads t's changes list from where the last read ended. A type
// read for the first time starts from now. A source that lists none from
// that far back moves the start to now, and the tiers cover the gap; any
// other failure leaves it, so the next run asks from the same point.
func (r refresh) follow(ctx context.Context, t works.TypeID, started storage.Time) {
	var through storage.Time
	err := r.cfg.Library.Read(ctx, func(ctx context.Context, rd storage.Reader[storage.Library]) error {
		types, err := db.New(rd).ListTypes(ctx)
		for _, row := range types {
			if row.ContentType == string(t) {
				through = row.ChangesThrough
			}
		}
		return err
	})
	if err != nil {
		r.cfg.Log.Error("read where the changes list ends", "contentType", string(t), "err", err)
		return
	}
	if through.IsZero() {
		r.mark(ctx, t, started, nil)
		return
	}
	keys, err := r.cfg.Metadata.Changes(ctx, t, through.Time())
	switch {
	case errors.Is(err, ErrNoChanges):
		r.cfg.Log.Info("skipped the changes list", "contentType", string(t), "since", through.String(), "why", err.Error())
		r.mark(ctx, t, started, nil)
	case err != nil:
		r.cfg.Log.Warn("read the changes list", "contentType", string(t), "since", through.String(), "err", err)
	default:
		r.mark(ctx, t, started, keys)
	}
}

// mark makes each Work of t a key in keys names due from started, and has
// t's next changes list start there.
func (r refresh) mark(ctx context.Context, t works.TypeID, started storage.Time, keys []works.Key) {
	changed := map[works.Key]bool{}
	for _, k := range keys {
		changed[k] = true
	}
	marked := 0
	err := r.cfg.Library.Write(ctx, func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
		q := db.New(tx)
		if len(changed) > 0 {
			all, err := works.Summaries(ctx, tx)
			if err != nil {
				return err
			}
			for _, w := range all {
				if w.ContentType != t || !changed[w.Key] {
					continue
				}
				if err := q.MarkDue(ctx, db.MarkDueParams{WorkID: w.ID, Reason: db.Changed, Since: started}); err != nil {
					return err
				}
				marked++
			}
		}
		return q.SetChangesThrough(ctx, db.SetChangesThroughParams{ContentType: string(t), ChangesThrough: started})
	})
	if err != nil {
		r.cfg.Log.Error("mark the changed Works due", "contentType", string(t), "err", err)
		return
	}
	if keys != nil {
		r.cfg.Log.Info("read the changes list", "contentType", string(t), "changed", len(keys), "due", marked)
	}
}

func purge(ctx context.Context, tx *storage.Tx[storage.Library]) error {
	q := db.New(tx)
	marks, err := q.ListDue(ctx)
	if err != nil {
		return err
	}
	all, err := works.Summaries(ctx, tx)
	if err != nil {
		return err
	}
	last := make(map[works.WorkID]storage.Time, len(all))
	for _, w := range all {
		last[w.ID] = answered(w)
	}
	for _, d := range marks {
		if d.Since.Compare(last[d.WorkID]) <= 0 {
			if err := q.DeleteDue(ctx, db.DeleteDueParams{WorkID: d.WorkID, Since: d.Since}); err != nil {
				return err
			}
		}
	}
	return nil
}

// answered is when w's source last answered for it: its refresh, or its
// gone-upstream mark when that is later.
func answered(w works.Summary) storage.Time {
	if w.GoneUpstreamAt.Compare(w.RefreshedAt) > 0 {
		return w.GoneUpstreamAt
	}
	return w.RefreshedAt
}

func (r refresh) units(ctx context.Context, rd storage.Reader[storage.Library]) ([]tasks.Unit, error) {
	all, err := works.Summaries(ctx, rd)
	if err != nil {
		return nil, err
	}
	q := db.New(rd)
	marks, err := q.ListDue(ctx)
	if err != nil {
		return nil, err
	}
	types, err := q.ListTypes(ctx)
	if err != nil {
		return nil, err
	}
	marked := map[works.WorkID]time.Time{}
	for _, d := range marks {
		marked[d.WorkID] = d.Since.Time()
	}
	asked := map[works.TypeID]time.Time{}
	for _, t := range types {
		asked[works.TypeID(t.ContentType)] = t.RefreshAllAt.Time()
	}
	now := r.cfg.Today()
	byType := map[works.TypeID][]works.WorkID{}
	for _, w := range all {
		byType[w.ContentType] = append(byType[w.ContentType], w.ID)
	}
	unreleased, growing := map[works.WorkID]bool{}, map[works.WorkID]bool{}
	for t, ids := range byType {
		if err := r.tiersOf(ctx, rd, t, ids, now, unreleased, growing); err != nil {
			return nil, err
		}
	}
	standings := make([]standing, len(all))
	for i, w := range all {
		standings[i] = standing{
			work: w.ID, label: w.Label, answered: answered(w).Time(), gone: !w.GoneUpstreamAt.IsZero(),
			unreleased: unreleased[w.ID], growing: growing[w.ID], marked: marked[w.ID], all: asked[w.ContentType],
		}
	}
	self := r.cfg.Metadata.cfg.Roster.Self()
	var out []tasks.Unit
	for _, s := range due(now, standings, r.cfg.Metadata.tiers()) {
		out = append(out, tasks.Unit{Key: unitKey(s.work), Label: s.label, Place: tasks.PortableOn(role.Metadata, self)})
	}
	return out, nil
}

// tiersOf notes which of type t's Works ids have an Item not out by now,
// and which can still gain Items.
func (r refresh) tiersOf(ctx context.Context, rd storage.Reader[storage.Library], t works.TypeID, ids []works.WorkID, now time.Time, unreleased, growing map[works.WorkID]bool) error {
	items, err := works.ItemsOf(ctx, rd, ids)
	if err != nil {
		return err
	}
	timings, err := r.cfg.Targets.TimingsIn(ctx, rd, t, ids)
	if err != nil {
		return fmt.Errorf("read the %s timings: %w", t, err)
	}
	for w, its := range items {
		unreleased[w] = slices.ContainsFunc(its, func(it works.Item) bool {
			return it.WithdrawnAt.IsZero() && !timings[it.ID].ReleasedBy(now)
		})
	}
	grows, err := r.cfg.Types[t].Growing(ctx, rd, ids)
	if err != nil {
		return fmt.Errorf("read which %s Works can still gain Items: %w", t, err)
	}
	for w, on := range grows {
		growing[w] = on
	}
	return nil
}

func (m *Metadata) tiers() tiers {
	s := m.stored()
	return tiers{
		unreleased: UnreleasedEvery.Value(s.UnreleasedEverySeconds),
		growing:    GrowingEvery.Value(s.GrowingEverySeconds),
		others:     OthersEvery.Value(s.OthersEverySeconds),
	}
}

const unitPrefix = "work:"

func unitKey(w works.WorkID) tasks.UnitKey { return tasks.UnitKey(unitPrefix + w.String()) }

func (r refresh) run(ctx context.Context, u tasks.Unit, _ *tasks.Progress) error {
	id, err := storage.ParseUUID(strings.TrimPrefix(string(u.Key), unitPrefix))
	if err != nil {
		return fmt.Errorf("unit %s names no Work: %w", u.Key, err)
	}
	via, _ := u.Place.Node()
	return r.cfg.Works.RefreshDue(ctx, works.WorkID{UUID: id}, via)
}

// standing is one Work as the refresh tiers read it.
type standing struct {
	work  works.WorkID
	label string
	// answered is when its source last answered for it.
	answered time.Time
	// gone: its source last answered that it has no such record.
	gone       bool
	unreleased bool
	growing    bool
	// marked is its latest due mark, zero when none.
	marked time.Time
	// all is its type's latest refresh-all request, zero when none.
	all time.Time
}

// tiers is how long a Work of each tier goes between refreshes, in whole
// days.
type tiers struct {
	unreleased, growing, others time.Duration
}

// of is s's tier. A Work gone upstream rides the every-other tier, so it
// isn't asked for again every night.
func (t tiers) of(s standing) time.Duration {
	switch {
	case s.gone:
		return t.others
	case s.unreleased:
		return t.unreleased
	case s.growing:
		return t.growing
	}
	return t.others
}

// due is the Works due at now, longest due first, then by label. A Work is
// due from the earliest of its due mark and its type's refresh-all request
// made after its source last answered, and the start of the calendar day,
// in now's zone, its tier's days after that answer.
func due(now time.Time, ss []standing, t tiers) []standing {
	type pending struct {
		standing
		since time.Time
	}
	var out []pending
	for _, s := range ss {
		a := s.answered.In(now.Location())
		since := time.Date(a.Year(), a.Month(), a.Day()+int(t.of(s)/day), 0, 0, 0, 0, now.Location())
		if since.After(now) {
			since = time.Time{}
		}
		for _, asked := range []time.Time{s.marked, s.all} {
			if asked.After(s.answered) && (since.IsZero() || asked.Before(since)) {
				since = asked
			}
		}
		if !since.IsZero() {
			out = append(out, pending{s, since})
		}
	}
	slices.SortFunc(out, func(a, b pending) int {
		return cmp.Or(a.since.Compare(b.since), strings.Compare(a.label, b.label), strings.Compare(a.work.String(), b.work.String()))
	})
	ss = make([]standing, len(out))
	for i, p := range out {
		ss[i] = p.standing
	}
	return ss
}
