package library

import (
	"context"
	"errors"
	"slices"

	"github.com/jatassi/slipstream/internal/files"
	"github.com/jatassi/slipstream/internal/grabs"
	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/roots"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/works"
)

// FilesChoice is what a delete does with the Works' library files.
type FilesChoice string

const (
	DeleteFiles FilesChoice = "delete"
	LeaveFiles  FilesChoice = "leave"
)

type Deleting struct {
	Files   FilesChoice
	Preview bool
}

// DeletePlan is what a delete does to the Works it removes. Downloading
// counts their unfinished Grabs, whose torrents keep downloading and
// seeding in the client.
type DeletePlan struct {
	Works       int
	Files, Kept int
	Bytes       int64
	Downloading int
}

// Delete removes each of ws, with everything SlipStream records about it,
// in one Write. Under DeleteFiles that Write also queues the Works' files
// for the disk queue, which removes the bytes on each Node once it
// commits. A Work with a file in a root whose Node is offline or written
// off then fails with the *roots.UnreachableError and stays whole: the
// Write checks each root itself, so a Node that dropped since a preview
// still refuses. An id no Work has is skipped, so a repeat converges. A
// Preview answers the same plan and outcomes from a Read and changes
// nothing.
func (l *Library) Delete(ctx context.Context, by identity.Admin, ws []works.WorkID, d Deleting) (DeletePlan, []Outcome, error) {
	ids := distinct(ws)
	if d.Preview {
		return l.previewDelete(ctx, ids, d.Files)
	}
	var c culling
	_, err := l.cfg.Works.Delete(ctx, by, ids, func(ctx context.Context, tx *storage.Tx[storage.Library], present []works.WorkID) ([]works.WorkID, error) {
		var err error
		if c, err = l.cull(ctx, tx, present, d.Files); err != nil {
			return nil, err
		}
		if d.Files == DeleteFiles {
			if err := files.DoomIn(ctx, tx, c.removed); err != nil {
				return nil, err
			}
		}
		return c.removed, nil
	})
	if err != nil {
		return DeletePlan{}, nil, err
	}
	l.cfg.Roots.Publish(ctx, c.roots)
	if d.Files == DeleteFiles {
		l.cfg.Files.Poke()
	}
	return c.plan, c.outcomes(ids), nil
}

func (l *Library) previewDelete(ctx context.Context, ids []works.WorkID, choice FilesChoice) (DeletePlan, []Outcome, error) {
	var c culling
	err := l.cfg.Library.Read(ctx, func(ctx context.Context, r storage.Reader[storage.Library]) error {
		summaries, err := works.SummariesOf(ctx, r, ids)
		if err != nil {
			return err
		}
		c, err = l.cull(ctx, r, presentOf(ids, summaries), choice)
		return err
	})
	if err != nil {
		return DeletePlan{}, nil, err
	}
	return c.plan, c.outcomes(ids), nil
}

// culling is what a delete of the present Works decides in one snapshot:
// the Works it removes, those it refuses and why, and the roots their
// files sit in.
type culling struct {
	removed []works.WorkID
	refused map[works.WorkID]*roots.UnreachableError
	roots   []roots.ID
	plan    DeletePlan
}

func (l *Library) cull(ctx context.Context, r storage.Reader[storage.Library], present []works.WorkID, choice FilesChoice) (culling, error) {
	held, err := files.Holding(ctx, r, present)
	if err != nil {
		return culling{}, err
	}
	unfinished, err := grabs.UnfinishedOf(ctx, r, present)
	if err != nil {
		return culling{}, err
	}
	c := culling{refused: map[works.WorkID]*roots.UnreachableError{}}
	reached := map[roots.ID]*roots.UnreachableError{}
	for _, id := range present {
		h := held[id]
		if choice == DeleteFiles {
			away, err := l.unreachable(ctx, r, h.Roots, reached)
			if err != nil {
				return culling{}, err
			}
			if away != nil {
				c.refused[id] = away
				continue
			}
		}
		c.removed = append(c.removed, id)
		c.plan.Works++
		c.plan.Files += h.Files
		c.plan.Kept += h.Kept
		c.plan.Bytes += h.Bytes
		c.plan.Downloading += unfinished[id]
		for _, root := range h.Roots {
			if !slices.Contains(c.roots, root) {
				c.roots = append(c.roots, root)
			}
		}
	}
	return c, nil
}

// unreachable is the first of ids whose Node can't remove files now, nil
// when the Node of every one can. reached remembers each root's answer
// across Works.
func (l *Library) unreachable(ctx context.Context, r storage.Reader[storage.Library], ids []roots.ID, reached map[roots.ID]*roots.UnreachableError) (*roots.UnreachableError, error) {
	for _, id := range ids {
		away, asked := reached[id]
		if !asked {
			root, err := roots.Get(ctx, r, id)
			if err != nil {
				return nil, err
			}
			err = l.cfg.Roots.Reachable(ctx, r, root)
			if away, _ = errors.AsType[*roots.UnreachableError](err); away == nil && err != nil {
				return nil, err
			}
			reached[id] = away
		}
		if away != nil {
			return away, nil
		}
	}
	return nil, nil
}

func (c culling) outcomes(ids []works.WorkID) []Outcome {
	removed := make(map[works.WorkID]bool, len(c.removed))
	for _, id := range c.removed {
		removed[id] = true
	}
	out := make([]Outcome, len(ids))
	for i, id := range ids {
		switch why, refused := c.refused[id]; {
		case refused:
			out[i] = failed(id, why)
		case removed[id]:
			out[i] = done(id)
		default:
			out[i] = skipped(id, works.ErrNoWork)
		}
	}
	return out
}
