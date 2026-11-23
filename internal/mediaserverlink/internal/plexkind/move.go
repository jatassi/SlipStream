package plexkind

import (
	"context"
	"slices"
	"time"

	"github.com/jatassi/slipstream/internal/mediaserverlink/link"
	"github.com/jatassi/slipstream/internal/providers/plex"
)

// moveSteps is the Plex Move's carry: the file was copied to another root
// and is deleted from its source only once Plex holds it at its new path.
// Its shape is fixed; plexMove, filled from the recorded run, decides what
// each step expects. The planner holds a row after StepRestored until the
// Move removed its source files.
var moveSteps = []step[link.Carry]{
	{link.StepSnapshotted, snapshot},
	{link.StepScanned, scanDestinations},
	{link.StepConfirmed, confirmMoved},
	{link.StepRestored, restoreCollections},
	{link.StepSourceScanned, scanSourcesAfterGone},
	{link.StepSettled, settleSources},
}

// copyAppears is how Plex shows a copy of an item's file in another folder
// while the source still exists.
type copyAppears int

const (
	newItem copyAppears = iota
	secondPart
)

type moveFacts struct {
	// Measured is false until the recorded run fills this table. The
	// conformance cases that depend on an unmeasured field skip.
	Measured bool
	// SameSection is what a copy in another folder of the item's section
	// makes.
	SameSection copyAppears
	// CollectionsCarry is whether collections come back without a write.
	CollectionsCarry bool
	// PartialTrashes is whether a partial refresh of a vacated folder
	// destroys the whole item, its other Parts with it, while a section
	// refresh drops only the lost Part.
	PartialTrashes bool
}

// plexMove is what Plex 1.43.5.11029 did in the recorded run of
// 2026-11-23 (conformance/plex/recorded/2026-11-23-plex-moves.md).
var plexMove = moveFacts{Measured: true, SameSection: secondPart, CollectionsCarry: false, PartialTrashes: true}

// MoveMeasured is whether the recorded run measured the Plex Move.
func MoveMeasured() bool { return plexMove.Measured }

// SameSectionPart is whether a copy in the item's section is a second
// Part of the item.
func SameSectionPart() bool { return plexMove.SameSection == secondPart }

// confirmMoved waits until an item holds each snapshotted change's new
// path, as a new item or a second Part of the snapshot's. Past
// Timings.MovedConfirm it refreshes the destinations again and waits
// anew: the source is only removed once Plex holds the copy, so a timeout
// never passes.
func confirmMoved(ctx context.Context, c *conn, j link.Job, w *link.Carry) (did, error) {
	found, err := c.atNew(ctx, w.Changes)
	if err != nil {
		return did{}, err
	}
	if len(unconfirmed(w.Changes, found)) == 0 {
		return did{}, nil
	}
	if time.Now().Before(j.At.Add(c.t.MovedConfirm)) {
		return did{waiting: true, wait: later(c.t.StepPoll)}, nil
	}
	if err := c.refreshFolders(ctx, places(w.Changes, newPlace)); err != nil {
		return did{}, err
	}
	return did{waiting: true, restart: true, wait: later(c.t.StepPoll)}, nil
}

// restoreCollections writes back the snapshot's collections on each item
// at its new path before the Move removes its source.
func restoreCollections(ctx context.Context, c *conn, _ link.Job, w *link.Carry) (did, error) {
	found, err := c.atNew(ctx, w.Changes)
	if err != nil {
		return did{}, err
	}
	_, err = c.restore(ctx, w.Changes, found)
	return did{}, err
}

// restore adds to each item found at a change's new path the snapshot's
// collections it lacks, when Plex doesn't carry them, and answers whether
// it wrote any. A collection write only adds (recorded run 2026-11-23,
// case 2), so the item's own titles go with the snapshot's and a repeat
// writes nothing.
func (c *conn) restore(ctx context.Context, cs []link.JobChange, found map[string]plex.Metadata) (bool, error) {
	if plexMove.CollectionsCarry {
		return false, nil
	}
	wrote := false
	for _, ch := range cs {
		m, ok := found[ch.Path.Key()]
		if !ok || len(ch.Snapshot.Collections) == 0 {
			continue
		}
		var titles []string
		for _, t := range m.Collections {
			titles = append(titles, t.Tag)
		}
		all := titles
		for _, t := range ch.Snapshot.Collections {
			if !slices.Contains(all, t) {
				all = append(all, t)
			}
		}
		if len(all) == len(titles) {
			continue
		}
		if err := c.c.SetCollections(ctx, m.RatingKey, plex.TypeOf(m.Type), all); err != nil {
			return false, err
		}
		wrote = true
	}
	return wrote, nil
}

// inSection is whether ch moved inside its section, where a partial refresh
// of the vacated folder destroys the item (recorded run 2026-11-23, case
// 4) and only a section refresh keeps it (case 4b).
func inSection(ch link.JobChange) bool {
	return plexMove.PartialTrashes && ch.Snapshot != nil && ch.OldLibrary() == ch.Library
}

// scanSourcesAfterGone runs once the Move removed its source files. Each
// section a change moved within is refreshed whole. Each other vacated
// folder no destination shares is refreshed, but not that of a change
// Plex never confirmed at its new path, since with autoEmptyTrash on that
// deletes the item.
func scanSourcesAfterGone(ctx context.Context, c *conn, _ link.Job, w *link.Carry) (did, error) {
	found, err := c.atNew(ctx, w.Changes)
	if err != nil {
		return did{}, err
	}
	whole, _ := groups(w.Changes, library, inSection)
	kept := slices.Concat(places(w.Changes, newPlace), places(unconfirmed(w.Changes, found), oldPlace))
	src := slices.DeleteFunc(places(w.Changes, oldPlace), func(p place) bool {
		return slices.Contains(whole, p.library) || slices.ContainsFunc(kept, p.same)
	})
	if err := c.refreshFolders(ctx, src); err != nil {
		return did{}, err
	}
	for _, lib := range whole {
		if err := c.c.RefreshSection(ctx, plex.ID(lib)); err != nil {
			return did{}, err
		}
	}
	return did{wait: later(c.t.ScanEndPoll)}, nil
}

// settleSources waits, up to Timings.ScanEndBound, until each change moved
// within its section has an item at its new path alone. Plex's own
// partial scan of the vacated folder can still destroy the item after the
// section refresh and make it again later under a new ratingKey, so a
// section that isn't scanning is refreshed again. Then the snapshot's
// collections are written back on each item, which a new ratingKey never
// carries, and each change is compared with its snapshot. A change Plex
// never confirmed at its new path ends with its watch state and
// collections lost.
func settleSources(ctx context.Context, c *conn, j link.Job, w *link.Carry) (did, error) {
	found, err := c.atNew(ctx, w.Changes)
	if err != nil {
		return did{}, err
	}
	open, _ := groups(w.Changes, library, func(ch link.JobChange) bool { return inSection(ch) && !alone(ch, found) })
	if now := time.Now(); len(open) > 0 && now.Before(j.At.Add(c.t.ScanEndBound)) {
		for _, lib := range open {
			s, err := c.section(ctx, lib)
			if err != nil {
				return did{}, err
			}
			if !s.Refreshing {
				if err := c.c.RefreshSection(ctx, s.Key); err != nil {
					return did{}, err
				}
			}
		}
		return did{waiting: true, wait: later(c.t.ScanWait(j.At, now))}, nil
	}
	if wrote, err := c.restore(ctx, w.Changes, found); err != nil || !wrote {
		return compareMoved(w.Changes, found), err
	}
	if found, err = c.atNew(ctx, w.Changes); err != nil {
		return did{}, err
	}
	return compareMoved(w.Changes, found), nil
}

// alone is whether an item holds ch's new path and no longer its old one.
func alone(ch link.JobChange, found map[string]plex.Metadata) bool {
	m, ok := found[ch.Path.Key()]
	return ok && !slices.ContainsFunc(files(m), func(p link.ServerPath) bool { return p.Key() == ch.From.Key() })
}

// compareMoved is compared for the changes found at their new paths, and
// watch state and collections lost for the rest.
func compareMoved(cs []link.JobChange, found map[string]plex.Metadata) did {
	missing := unconfirmed(cs, found)
	rest := slices.DeleteFunc(slices.Clone(cs), func(ch link.JobChange) bool {
		return slices.ContainsFunc(missing, func(m link.JobChange) bool { return m.ID == ch.ID })
	})
	d := compared(rest, found)
	for _, ch := range missing {
		d.outcomes = append(d.outcomes, link.Outcome{Change: ch.ID, Result: link.ResultAccepted,
			Lost: []link.Loss{link.LostWatchState, link.LostCollections}})
	}
	return d
}

// unconfirmed are the snapshotted changes of cs with no item found at
// their new path.
func unconfirmed(cs []link.JobChange, found map[string]plex.Metadata) []link.JobChange {
	var out []link.JobChange
	for _, ch := range cs {
		if _, ok := found[ch.Path.Key()]; ch.Snapshot != nil && !ok {
			out = append(out, ch)
		}
	}
	return out
}

// moved is whether w is a Move's carry.
func moved(w link.Carry) bool { return slices.ContainsFunc(w.Changes, link.JobChange.Move) }
