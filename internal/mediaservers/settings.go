package mediaservers

import (
	"context"
	"fmt"
	"slices"
	"time"

	"github.com/jatassi/slipstream/internal/mediaservers/internal/db"
	"github.com/jatassi/slipstream/internal/setting"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/tasks"
)

const day = 24 * time.Hour

// RenewalTaskID is the task that renews the Plex sign-in.
const RenewalTaskID tasks.TaskID = "plex-sign-in"

// The Media Servers section: one stored value each. Every other bound is
// in link.Timings.
var (
	WaitBeforeSending = setting.Duration{
		Key: "mediaServers.waitBeforeSending", Label: "Wait before sending",
		Default: 10 * time.Second, Min: 0, Max: 10 * time.Minute, Step: time.Second,
	}
	SessionsEvery = setting.Duration{
		Key: "mediaServers.sessionsEvery", Label: "Check sessions every",
		Default: 15 * time.Second, Min: 5 * time.Second, Max: 10 * time.Minute, Step: time.Second,
	}
	UnhealthyAfter = setting.Duration{
		Key: "mediaServers.unhealthyAfter", Label: "Mark unhealthy after",
		Default: 5 * time.Minute, Min: time.Minute, Max: day, Step: time.Minute,
	}
	KeepPlays = setting.Duration{
		Key: "mediaServers.keepPlays", Label: "Keep play history for",
		Default: 30 * day, Min: day, Max: 365 * day, Step: day,
	}
	// RenewEvery is the renewal task's cadence, which the Tasks page shows
	// too. tasks stores it.
	RenewEvery = setting.Duration{
		Key: "tasks." + string(RenewalTaskID) + ".cadence", Label: "Renew Plex sign-in every",
		Default: day, Min: time.Hour, Max: 6 * day, Step: time.Hour,
	}
	// RenewMargin must be longer than RenewEvery, or the sign-in could
	// expire between two runs.
	RenewMargin = setting.Duration{
		Key: "mediaServers.renewMargin", Label: "Renew when less than this is left",
		Default: 3 * day, Min: time.Hour, Max: 6 * day, Step: time.Hour,
	}
)

// Cadences is the renewal task's cadence as tasks stores it.
type Cadences interface {
	Every(ctx context.Context, id tasks.TaskID) (*int64, error)
	SetCadence(ctx context.Context, id tasks.TaskID, v setting.Stored[int64]) error
}

// MarginTooShortError is a renewal margin no longer than the renewal
// task's cadence.
type MarginTooShortError struct{ Margin, Cadence time.Duration }

func (e *MarginTooShortError) Error() string {
	return fmt.Sprintf("the renewal margin %s isn't longer than the renewal cadence %s", e.Margin, e.Cadence)
}

// RenewalTask renews the Plex sign-in. Its one unit is the signed-in
// account, so it is listed only while SlipStream is signed in to Plex.
func (m *MediaServers) RenewalTask() tasks.Task {
	return tasks.Task{
		ID:      RenewalTaskID,
		Name:    "Plex sign-in",
		About:   "Renews the Plex sign-in before it expires and reads each Plex server's token again.",
		Group:   tasks.ControllerGroup,
		Cadence: tasks.Every{Interval: RenewEvery},
		Noun:    setting.Noun{One: "Plex account", Many: "Plex accounts"},
		Units:   m.renewalUnits,
		Run:     m.renew,
	}
}

// Settings is the Media Servers section.
func (m *MediaServers) Settings() setting.Panel { return panel{m} }

type panel struct{ m *MediaServers }

// stored is a setting of the section's row and the query that stores it.
type stored struct {
	setting.Duration
	get func(db.MediaserversSetting) *int64
	set func(*db.Queries, context.Context, *int64) error
}

var rowSettings = []stored{
	{WaitBeforeSending, func(s db.MediaserversSetting) *int64 { return s.WaitSeconds }, (*db.Queries).SetWait},
	{SessionsEvery, func(s db.MediaserversSetting) *int64 { return s.SessionsEverySeconds }, (*db.Queries).SetSessionsEvery},
	{UnhealthyAfter, func(s db.MediaserversSetting) *int64 { return s.UnhealthyAfterSeconds }, (*db.Queries).SetUnhealthyAfter},
	{KeepPlays, func(s db.MediaserversSetting) *int64 { return s.KeepPlaysSeconds }, (*db.Queries).SetKeepPlays},
	{RenewMargin, func(s db.MediaserversSetting) *int64 { return s.RenewMarginSeconds }, (*db.Queries).SetRenewMargin},
}

func (p panel) read(ctx context.Context) (db.MediaserversSetting, *int64, error) {
	var row db.MediaserversSetting
	err := p.m.cfg.Library.Read(ctx, func(ctx context.Context, r storage.Reader[storage.Library]) error {
		var err error
		row, err = db.New(r).GetSettings(ctx)
		return err
	})
	if err != nil {
		return row, nil, err
	}
	every, err := p.m.cfg.Tasks.Every(ctx, RenewalTaskID)
	return row, every, err
}

func (p panel) Fields(ctx context.Context) ([]setting.Field, error) {
	row, every, err := p.read(ctx)
	if err != nil {
		return nil, err
	}
	var out []setting.Field
	for _, s := range rowSettings[:4] {
		out = append(out, s.Field(s.get(row)))
	}
	return append(out, RenewEvery.Field(every), RenewMargin.Field(row.RenewMarginSeconds)), nil
}

// Apply stores the patch, all or none, refusing a renewal margin that
// isn't longer than the renewal cadence either keeps or gets.
func (p panel) Apply(ctx context.Context, patch setting.Patch) error {
	row, every, err := p.read(ctx)
	if err != nil {
		return err
	}
	c, err := parse(patch, row.RenewMarginSeconds, every)
	if err != nil {
		return err
	}
	if m, e := RenewMargin.Value(c.margin), RenewEvery.Value(c.every); m <= e {
		return &MarginTooShortError{Margin: m, Cadence: e}
	}
	if len(c.sets) > 0 {
		err := p.m.cfg.Library.Write(ctx, func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
			q := db.New(tx)
			for _, set := range c.sets {
				if err := set(ctx, q); err != nil {
					return err
				}
			}
			return nil
		})
		if err != nil {
			return err
		}
	}
	if c.cadence != nil {
		if err := p.m.cfg.Tasks.SetCadence(ctx, RenewalTaskID, *c.cadence); err != nil {
			return err
		}
	}
	if c.sessions {
		p.m.touchAll(ctx)
	}
	return nil
}

// change is a parsed patch: the row's writes, the renewal cadence when it
// changes, and the margin and cadence the section would hold after.
type change struct {
	sets          []func(context.Context, *db.Queries) error
	cadence       *setting.Stored[int64]
	margin, every *int64
	sessions      bool
}

func parse(patch setting.Patch, margin, every *int64) (change, error) {
	c := change{margin: margin, every: every}
	for key, raw := range patch {
		if key == RenewEvery.Key {
			v, err := RenewEvery.ParseJSON(raw)
			if err != nil {
				return c, err
			}
			c.cadence, c.every = &v, v.Override()
			continue
		}
		i := slices.IndexFunc(rowSettings, func(s stored) bool { return s.Key == key })
		if i < 0 {
			return c, &setting.UnknownKeyError{Key: key}
		}
		v, err := rowSettings[i].ParseJSON(raw)
		if err != nil {
			return c, err
		}
		switch key {
		case RenewMargin.Key:
			c.margin = v.Override()
		case SessionsEvery.Key:
			c.sessions = true
		}
		set := rowSettings[i].set
		c.sets = append(c.sets, func(ctx context.Context, q *db.Queries) error { return set(q, ctx, v.Override()) })
	}
	return c, nil
}

// sessionsEvery is the stored Check sessions every, which every intent
// carries.
func sessionsEvery(ctx context.Context, q *db.Queries) (time.Duration, error) {
	row, err := q.GetSettings(ctx)
	if err != nil {
		return 0, err
	}
	return SessionsEvery.Value(row.SessionsEverySeconds), nil
}
