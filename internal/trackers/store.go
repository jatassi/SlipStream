package trackers

import (
	"context"
	"fmt"
	"slices"
	"time"

	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/indexers"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/trackers/internal/db"
	"github.com/jatassi/slipstream/internal/trackers/seeding"
)

var ErrNoTracker = indexers.ErrNoTracker

// PublicError refuses a seeding rule on a public Tracker, named Name.
type PublicError struct{ Name string }

func (e *PublicError) Error() string { return "a public Tracker has no seeding rule" }

// InitialGoal is the default goal until an Admin sets one: whichever first
// of ratio 2.0 or 14 days, then remove the torrent and delete its download
// copy.
var InitialGoal seeding.Goal = seeding.MustFinite(seeding.First, 200, 14*24*time.Hour, seeding.RemoveDelete)

// Source is where a Tracker's rule came from.
type Source string

const (
	FromReleases Source = "releases"
	FromAdmin    Source = "admin"
)

// Rules are a Tracker's seeding rules by source, each nil for none.
// Releases is the latest rule its Releases stated, and the Admin's wins
// over it while set.
type Rules struct {
	Admin, Releases seeding.Rule
}

// InForce is the Admin's rule, else the rule from Releases, and its
// source. It is nil for none.
func (r Rules) InForce() (seeding.Rule, Source) {
	if r.Admin != nil {
		return r.Admin, FromAdmin
	}
	return r.Releases, FromReleases
}

// Summary is a Tracker with its seeding side.
type Summary struct {
	indexers.Tracker
	// Rules are empty for a public Tracker.
	Rules Rules
	// Goal is the Tracker's own goal, nil while it follows the default goal.
	Goal seeding.Goal
}

// Tracker is Tracker id with its seeding side, or ErrNoTracker.
func (t *Trackers) Tracker(ctx context.Context, _ identity.Admin, id indexers.TrackerID) (Summary, error) {
	var out Summary
	err := t.cfg.Library.Read(ctx, func(ctx context.Context, r storage.Reader[file]) error {
		var err error
		out, err = summaryIn(ctx, r, id)
		return err
	})
	return out, err
}

// SetRule stores the Admin's rule for Tracker id, which Releases never
// replace. It fails with ErrNoTracker, or *PublicError for a public
// Tracker.
func (t *Trackers) SetRule(ctx context.Context, _ identity.Admin, id indexers.TrackerID, rule seeding.Rule) (Summary, error) {
	return t.trackerWrite(ctx, id, func(ctx context.Context, tx *storage.Tx[file], s Summary) error {
		if err := ruled(s); err != nil {
			return err
		}
		ratio, seed := columnsOf(rule)
		return db.New(tx).SetAdminRule(ctx, db.SetAdminRuleParams{
			TrackerID: id, Clears: string(rule.Clears()), RatioCenti: ratio, SeedSeconds: seed, UpdatedAt: storage.Now(),
		})
	})
}

// ClearRule deletes the Admin's rule for Tracker id, so the latest rule
// its Releases stated is in force again. It fails with ErrNoTracker, or
// *PublicError for a public Tracker.
func (t *Trackers) ClearRule(ctx context.Context, _ identity.Admin, id indexers.TrackerID) (Summary, error) {
	return t.trackerWrite(ctx, id, func(ctx context.Context, tx *storage.Tx[file], s Summary) error {
		if err := ruled(s); err != nil {
			return err
		}
		if err := db.New(tx).ClearAdminRule(ctx, id); err != nil {
			return err
		}
		_, err := syncReleases(ctx, tx, t.cfg.Grabs, func(other indexers.TrackerID) bool { return other == id })
		return err
	})
}

// ruled refuses a seeding rule on a public Tracker.
func ruled(s Summary) error {
	if s.Privacy == indexers.Public {
		return &PublicError{Name: s.Name}
	}
	return nil
}

// trackerWrite runs write on Tracker id as it stands, in one Write, and
// answers the Tracker as write left it. It fails with ErrNoTracker.
func (t *Trackers) trackerWrite(ctx context.Context, id indexers.TrackerID, write func(context.Context, *storage.Tx[file], Summary) error) (Summary, error) {
	var out Summary
	err := t.cfg.Library.Write(ctx, func(ctx context.Context, tx *storage.Tx[file]) error {
		s, err := summaryIn(ctx, tx, id)
		if err != nil {
			return err
		}
		if err := write(ctx, tx, s); err != nil {
			return err
		}
		out, err = summaryIn(ctx, tx, id)
		return err
	})
	if err != nil {
		return Summary{}, err
	}
	t.saved(ctx, &id)
	return out, nil
}

// SetGoal stores Tracker id's own goal. A goal equal to the default goal
// deletes it instead, so the Tracker follows a later change to the
// default. It fails with ErrNoTracker.
func (t *Trackers) SetGoal(ctx context.Context, _ identity.Admin, id indexers.TrackerID, g seeding.Goal) (Summary, error) {
	return t.trackerWrite(ctx, id, func(ctx context.Context, tx *storage.Tx[file], _ Summary) error {
		q := db.New(tx)
		def, err := defaultGoalIn(ctx, tx)
		if err != nil {
			return err
		}
		if g == def {
			return q.ClearGoal(ctx, id)
		}
		c := goalColumnsOf(g)
		return q.SetGoal(ctx, db.SetGoalParams{TrackerID: id, Combine: *c.combine, RatioCenti: c.ratio, SeedSeconds: c.seed, Action: c.action})
	})
}

// ClearGoal deletes Tracker id's own goal, so it follows the default goal.
// It fails with ErrNoTracker.
func (t *Trackers) ClearGoal(ctx context.Context, _ identity.Admin, id indexers.TrackerID) (Summary, error) {
	return t.trackerWrite(ctx, id, func(ctx context.Context, tx *storage.Tx[file], _ Summary) error {
		return db.New(tx).ClearGoal(ctx, id)
	})
}

// DefaultGoal is the goal of every Tracker with none of its own.
func (t *Trackers) DefaultGoal(ctx context.Context, _ identity.Admin) (seeding.Goal, error) {
	var out seeding.Goal
	err := t.cfg.Library.Read(ctx, func(ctx context.Context, r storage.Reader[file]) error {
		var err error
		out, err = defaultGoalIn(ctx, r)
		return err
	})
	return out, err
}

// SetDefaultGoal stores g as the default goal.
func (t *Trackers) SetDefaultGoal(ctx context.Context, _ identity.Admin, g seeding.Goal) error {
	c := goalColumns{}
	if g != InitialGoal {
		c = goalColumnsOf(g)
	}
	err := t.cfg.Library.Write(ctx, func(ctx context.Context, tx *storage.Tx[file]) error {
		return db.New(tx).SetDefaultGoal(ctx, db.SetDefaultGoalParams{Combine: c.combine, RatioCenti: c.ratio, SeedSeconds: c.seed, Action: c.action})
	})
	if err != nil {
		return err
	}
	t.saved(ctx, nil)
	return nil
}

// SyncReleases stores each Tracker's latest rule from Releases where a
// Grab stated one after the stored rule. The Admin's rule is never
// touched. It is idempotent.
func (t *Trackers) SyncReleases(ctx context.Context) error {
	var moved []indexers.TrackerID
	err := t.cfg.Library.Write(ctx, func(ctx context.Context, tx *storage.Tx[file]) error {
		var err error
		moved, err = syncReleases(ctx, tx, t.cfg.Grabs, func(indexers.TrackerID) bool { return true })
		return err
	})
	if err != nil {
		return err
	}
	for _, id := range moved {
		t.publish(ctx, &id)
	}
	return nil
}

// syncReleases is SyncReleases for the Trackers keep lets through, in tx.
// It answers the Trackers whose rule moved.
func syncReleases(ctx context.Context, tx *storage.Tx[file], grabs Grabs, keep func(indexers.TrackerID) bool) ([]indexers.TrackerID, error) {
	stated, err := grabs.Stated(ctx, tx)
	if err != nil {
		return nil, err
	}
	q, now := db.New(tx), storage.Now()
	var moved []indexers.TrackerID
	for id, s := range stated {
		if !keep(id) {
			continue
		}
		ratio, seed := columnsOf(s.Rule)
		n, err := q.SetStatedRule(ctx, db.SetStatedRuleParams{
			TrackerID: id, Clears: string(s.Rule.Clears()), RatioCenti: ratio, SeedSeconds: seed,
			StatedAt: storage.FromTime(s.At), UpdatedAt: now,
		})
		if err != nil {
			return nil, err
		}
		if n > 0 {
			moved = append(moved, id)
		}
	}
	return moved, nil
}

func summaryIn(ctx context.Context, r storage.Reader[file], id indexers.TrackerID) (Summary, error) {
	all, err := summariesIn(ctx, r)
	if err != nil {
		return Summary{}, err
	}
	i := slices.IndexFunc(all, func(s Summary) bool { return s.ID == id })
	if i < 0 {
		return Summary{}, ErrNoTracker
	}
	return all[i], nil
}

func summariesIn(ctx context.Context, r storage.Reader[file]) ([]Summary, error) {
	all, err := indexers.TrackersIn(ctx, r)
	if err != nil {
		return nil, err
	}
	q := db.New(r)
	rules, err := q.ListRules(ctx)
	if err != nil {
		return nil, err
	}
	goals, err := q.ListGoals(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]Summary, len(all))
	at := map[indexers.TrackerID]*Summary{}
	for i, tr := range all {
		out[i] = Summary{Tracker: tr}
		at[tr.ID] = &out[i]
	}
	for _, row := range rules {
		s := at[row.TrackerID]
		if s.Privacy == indexers.Public {
			continue
		}
		rule, err := ruleOf(row.Clears, row.RatioCenti, row.SeedSeconds)
		if err != nil {
			return nil, err
		}
		if Source(row.Source) == FromAdmin {
			s.Rules.Admin = rule
		} else {
			s.Rules.Releases = rule
		}
	}
	for _, row := range goals {
		if at[row.TrackerID].Goal, err = goalOf(&row.Combine, row.RatioCenti, row.SeedSeconds, row.Action); err != nil {
			return nil, err
		}
	}
	return out, nil
}

func defaultGoalIn(ctx context.Context, r storage.Reader[file]) (seeding.Goal, error) {
	s, err := db.New(r).GetSettings(ctx)
	if err != nil {
		return nil, err
	}
	if s.GoalCombine == nil {
		return InitialGoal, nil
	}
	return goalOf(s.GoalCombine, s.GoalRatioCenti, s.GoalSeedSeconds, s.GoalAction)
}

func ruleOf(clears string, ratio, seed *int64) (seeding.Rule, error) {
	r, err := seeding.RuleOf(seeding.Clears(clears), seeding.Ratio(deref(ratio)), time.Duration(deref(seed))*time.Second)
	if err != nil {
		return nil, fmt.Errorf("read a stored rule: %w", err)
	}
	return r, nil
}

// columnsOf is r's ratio and seed columns, nil for a part it doesn't count.
func columnsOf(r seeding.Rule) (ratio, seed *int64) {
	return partsOf(r.Parts())
}

func partsOf(r seeding.Ratio, d time.Duration) (ratio, seed *int64) {
	if r > 0 {
		v := int64(r)
		ratio = &v
	}
	if d > 0 {
		v := int64(d / time.Second)
		seed = &v
	}
	return ratio, seed
}

const forever = "forever"

func goalOf(combine *string, ratio, seed *int64, action *string) (seeding.Goal, error) {
	if *combine == forever {
		return seeding.Forever{}, nil
	}
	g, err := seeding.NewFinite(seeding.Combine(*combine), seeding.Ratio(deref(ratio)), time.Duration(deref(seed))*time.Second, seeding.Action(*action))
	if err != nil {
		return nil, fmt.Errorf("read a stored goal: %w", err)
	}
	return g, nil
}

// goalColumns is a goal as trackers_goal and trackers_settings store it.
type goalColumns struct {
	combine, action *string
	ratio, seed     *int64
}

func goalColumnsOf(g seeding.Goal) goalColumns {
	f, ok := g.(seeding.Finite)
	if !ok {
		c := forever
		return goalColumns{combine: &c}
	}
	combine, action := string(f.Combine()), string(f.Then())
	ratio, seed := partsOf(f.Parts())
	return goalColumns{combine: &combine, action: &action, ratio: ratio, seed: seed}
}

func deref(v *int64) int64 {
	if v == nil {
		return 0
	}
	return *v
}
