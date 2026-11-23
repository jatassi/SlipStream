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
}

// copyAppears is how Plex shows a copy of an item's file in another folder
// while the source still exists.
type copyAppears int

const (
	newItem copyAppears = iota
	secondPart
)

// afterDelete is what a refresh of a Move's vacated folder leaves.
type afterDelete int

const (
	keepsOtherPart afterDelete = iota
	trash
)

type moveFacts struct {
	// Measured is false until the recorded run fills this table. The
	// conformance cases that depend on an unmeasured field skip.
	Measured bool
	// SameSection is what a copy in another folder of the item's section
	// makes.
	SameSection copyAppears
	// CrossSection is what a copy in another section makes.
	CrossSection copyAppears
	// WatchCarries is whether watch state and rating come back by guid.
	WatchCarries bool
	// CollectionsCarry is whether collections come back without a write.
	CollectionsCarry bool
	// CollectionsReplace is whether a collection write replaces the item's
	// list or adds to it.
	CollectionsReplace bool
	// AfterDelete is what a refresh of the vacated folder leaves.
	AfterDelete afterDelete
}

// plexMove is what Plex 1.43.5.11029 did in the recorded run of
// 2026-11-23 (conformance/plex/recorded/2026-11-23-plex-moves.md).
var plexMove = moveFacts{Measured: true, SameSection: secondPart, CrossSection: newItem,
	WatchCarries: true, CollectionsCarry: false, CollectionsReplace: false, AfterDelete: trash}

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
// that moved to another section, when Plex doesn't carry them. It sends
// the item's own titles too, which is right whether the write replaces the
// list or adds to it.
func restoreCollections(ctx context.Context, c *conn, _ link.Job, w *link.Carry) (did, error) {
	if plexMove.CollectionsCarry {
		return did{}, nil
	}
	found, err := c.atNew(ctx, w.Changes)
	if err != nil {
		return did{}, err
	}
	for _, ch := range w.Changes {
		m, ok := found[ch.Path.Key()]
		if !ok || ch.Library == ch.FromLibrary || len(ch.Snapshot.Collections) == 0 {
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
			return did{}, err
		}
	}
	return did{}, nil
}

// scanSourcesAfterGone runs once the Move removed its source files. A
// change Plex never confirmed at its new path ends with its watch state
// and collections lost, and its source folder is not refreshed, since with
// autoEmptyTrash on that deletes the item and its watch data. The other
// source folders no destination shares are refreshed and their items
// compared with their snapshots.
func scanSourcesAfterGone(ctx context.Context, c *conn, _ link.Job, w *link.Carry) (did, error) {
	found, err := c.atNew(ctx, w.Changes)
	if err != nil {
		return did{}, err
	}
	missing := unconfirmed(w.Changes, found)
	kept := slices.Concat(places(w.Changes, newPlace), places(missing, oldPlace))
	src := slices.DeleteFunc(places(w.Changes, oldPlace), func(p place) bool { return slices.ContainsFunc(kept, p.same) })
	if err := c.refreshFolders(ctx, src); err != nil {
		return did{}, err
	}
	rest := slices.DeleteFunc(slices.Clone(w.Changes), func(ch link.JobChange) bool {
		return slices.ContainsFunc(missing, func(m link.JobChange) bool { return m.ID == ch.ID })
	})
	if found, err = c.atNew(ctx, rest); err != nil {
		return did{}, err
	}
	d := compared(rest, found)
	for _, ch := range missing {
		d.outcomes = append(d.outcomes, link.Outcome{Change: ch.ID, Result: link.ResultAccepted,
			Lost: []link.Loss{link.LostWatchState, link.LostCollections}})
	}
	return d, nil
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
