package files

import (
	"context"
	"time"

	"github.com/jatassi/slipstream/internal/files/internal/db"
	"github.com/jatassi/slipstream/internal/naming"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/targets"
	"github.com/jatassi/slipstream/internal/works"
)

// world is everything cells, refill and names read about one Work, loaded
// in one Read or Write. The pure functions take it, so nothing in it is
// read twice.
type world struct {
	tree   works.Tree
	goals  targets.Goals
	files  []File // oldest first: ids are UUIDv7
	fills  map[CellKey]ID
	grabs  map[CellKey]GrabState
	timing map[works.ItemID]targets.Timing
	today  time.Time
	// fresh is the file being placed, which takes its folder and its name
	// from the templates. Until names gives it one, its Path is where its
	// bytes are held.
	fresh ID
	// templates and tokens are loaded by named, for names alone.
	templates naming.Templates
	tokens    map[works.ItemID]map[string]string
}

func (f *Files) load(ctx context.Context, r storage.Reader[file], work works.WorkID) (world, error) {
	tree, err := f.cfg.Works.TreeIn(ctx, r, work)
	if err != nil {
		return world{}, err
	}
	w := world{tree: tree, fills: map[CellKey]ID{}, today: f.cfg.Today()}
	if w.goals, err = targets.GoalsIn(ctx, r, work); err != nil {
		return world{}, err
	}
	if w.timing, err = f.cfg.Targets.TimingsIn(ctx, r, tree.ContentType, []works.WorkID{work}); err != nil {
		return world{}, err
	}
	q := db.New(r)
	rows, err := q.ListFilesOfWork(ctx, work)
	if err != nil {
		return world{}, err
	}
	for _, row := range rows {
		w.files = append(w.files, File(row))
	}
	fills, err := q.ListFillsOfWork(ctx, work)
	if err != nil {
		return world{}, err
	}
	for _, fill := range fills {
		w.fills[CellKey{Item: fill.ItemID, Edition: fill.Edition, Target: fill.TargetID}] = fill.FileID
	}
	w.grabs, err = f.cfg.Grabs.GrabCells(ctx, r, work)
	return w, err
}

// named adds what names reads to w under fills: the Content Type's
// templates and the tokens of every Item holding a file whose name
// changes, read Item by Item. The placed file's come first: labels reads
// them for its folder.
func (f *Files) named(ctx context.Context, r storage.Reader[file], w *world, fills map[CellKey]ID) error {
	facet := f.cfg.Naming[w.tree.ContentType]
	w.templates, w.tokens = facet.DefaultTemplates(), map[works.ItemID]map[string]string{}
	need := func(item works.ItemID) error {
		if _, ok := w.tokens[item]; ok {
			return nil
		}
		tokens, err := facet.NamingTokens(ctx, r, item)
		w.tokens[item] = tokens
		return err
	}
	for _, file := range w.files {
		if file.ID == w.fresh {
			if err := need(file.ItemID); err != nil {
				return err
			}
		}
	}
	next, err := labels(*w, fills)
	if err != nil {
		return err
	}
	for _, file := range w.files {
		if renamed(*w, file, next[file.ID]) {
			if err := need(file.ItemID); err != nil {
				return err
			}
		}
	}
	return nil
}

func (w world) byID() map[ID]*File {
	out := make(map[ID]*File, len(w.files))
	for i := range w.files {
		out[w.files[i].ID] = &w.files[i]
	}
	return out
}

// cells crosses each Item with the Profile's Targets in order. A filled
// cell is met at or past the Target's cutoff, else upgradable. An empty
// one is downloading or failed from its latest Grab, unreleased until the
// Item is out, else missing.
func cells(w world) []Cell {
	files := w.byID()
	var out []Cell
	for _, it := range w.tree.Items {
		timing := w.timing[it.ID]
		released := timing.ReleasedBy(w.today)
		for _, t := range w.goals.Targets {
			key := CellKey{Item: it.ID, Edition: Standard, Target: t.ID}
			c := Cell{Item: it, Edition: Standard, Target: t, State: Missing, Timing: timing, Released: released}
			if id, ok := w.fills[key]; ok {
				c.File = files[id]
			}
			switch {
			case c.File != nil && final(t, *c.File):
				c.State = Met
			case c.File != nil:
				c.State = Upgradable
			case w.grabs[key] == GrabDownloading:
				c.State = Downloading
			case w.grabs[key] == GrabFailed:
				c.State = Failed
			case !released:
				c.State = Unreleased
			}
			out = append(out, c)
		}
	}
	return out
}

// final reports whether f is at or past t's cutoff. A file whose quality t
// doesn't rank is not.
func final(t targets.Target, f File) bool {
	r, ok := t.RankOf(f.Quality)
	return ok && t.Final(r)
}

// filling is the set of files fills holds.
func filling(fills map[CellKey]ID) map[ID]bool {
	out := make(map[ID]bool, len(fills))
	for _, id := range fills {
		out[id] = true
	}
	return out
}
