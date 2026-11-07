package library

import (
	"context"
	"errors"

	"github.com/jatassi/slipstream/internal/files"
	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/roots"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/targets"
	"github.com/jatassi/slipstream/internal/works"
)

// refreshing is how many Works a Refresh fetches at once.
const refreshing = 4

// Monitor is SetMonitoring with All when on, each Work keeping its own
// asides, and with None and asides off when not, so bulk edit means the
// same for every Content Type.
func (l *Library) Monitor(ctx context.Context, by identity.Admin, ws []works.WorkID, on bool) ([]Outcome, error) {
	c := works.Choice{Preset: works.PresetAll}
	if !on {
		c = works.Choice{Preset: works.PresetNone, Asides: new(false)}
	}
	os, err := l.SetMonitoring(ctx, by, ws, c, false)
	out := make([]Outcome, len(os))
	for i, o := range os {
		out[i] = o.Outcome
	}
	return out, err
}

// SetProfile gives each of ws Profile p through one Impact across all of
// them, as m says. A Work already on p is skipped, and one of another
// Content Type fails; neither is in the Impact. A save deleting more
// files than m confirmed is a *files.ImpactChangedError for the whole
// request, because the Admin confirmed one count for all of them.
func (l *Library) SetProfile(ctx context.Context, by identity.Admin, ws []works.WorkID, p targets.ProfileID, m files.Mode) (files.Impact, []Outcome, error) {
	ids := distinct(ws)
	profile, err := l.cfg.Targets.GetProfile(ctx, by, p)
	if err != nil {
		return files.Impact{}, nil, err
	}
	var (
		summaries map[works.WorkID]works.Summary
		goals     map[works.WorkID]targets.Goals
	)
	err = l.cfg.Library.Read(ctx, func(ctx context.Context, r storage.Reader[storage.Library]) error {
		var err error
		if summaries, err = works.SummariesOf(ctx, r, ids); err != nil {
			return err
		}
		goals, err = targets.GoalsOf(ctx, r, works.Only(presentOf(ids, summaries)))
		return err
	})
	if err != nil {
		return files.Impact{}, nil, err
	}
	out := make([]Outcome, len(ids))
	var change []works.WorkID
	for i, id := range ids {
		w, ok := summaries[id]
		switch {
		case !ok:
			out[i] = failed(id, works.ErrNoWork)
		case w.ContentType != profile.ContentType:
			out[i] = failed(id, targets.ErrProfileWrongType)
		case goals[id].Profile.ID == p:
			out[i] = skipped(id, ErrAlready)
		default:
			out[i] = done(id)
			change = append(change, id)
		}
	}
	if len(change) == 0 {
		return files.Impact{}, out, nil
	}
	impact, err := l.cfg.Files.Change(ctx, by, targets.SetProfiles{Works: change, Profile: p}, m)
	if err != nil {
		return files.Impact{}, nil, err
	}
	return impact, out, nil
}

// SetRoot moves each of ws without files into root in one Write. A Work
// with files stays where it is, skipped, because changing its root is a
// Move.
func (l *Library) SetRoot(ctx context.Context, by identity.Admin, ws []works.WorkID, root roots.ID) ([]Outcome, error) {
	ids := distinct(ws)
	moved, refused, err := l.cfg.Roots.HomeAll(ctx, by, ids, root)
	if err != nil {
		return nil, err
	}
	went := make(map[works.WorkID]bool, len(moved))
	for _, id := range moved {
		went[id] = true
	}
	out := make([]Outcome, len(ids))
	for i, id := range ids {
		why, refusedIt := refused[id]
		switch {
		case refusedIt && errors.Is(why, roots.ErrHasFiles):
			out[i] = skipped(id, why)
		case refusedIt:
			out[i] = failed(id, why)
		case went[id]:
			out[i] = done(id)
		default:
			out[i] = skipped(id, ErrAlready)
		}
	}
	l.cfg.Works.Touched(ctx, moved)
	return out, nil
}

// Refresh refreshes each of ws from its metadata source now, a few at
// once, skipping each Work its source no longer lists.
func (l *Library) Refresh(ctx context.Context, by identity.Admin, ws []works.WorkID) ([]Outcome, error) {
	ids := distinct(ws)
	errs := l.cfg.Works.RefreshEach(ctx, by, ids, refreshing)
	out := make([]Outcome, len(ids))
	for i, id := range ids {
		switch err := errs[id]; {
		case err == nil:
			out[i] = done(id)
		case errors.Is(err, works.ErrGoneUpstream):
			out[i] = skipped(id, err)
		default:
			out[i] = failed(id, err)
		}
	}
	return out, nil
}
