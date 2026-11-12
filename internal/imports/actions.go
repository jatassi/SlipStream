package imports

import (
	"context"
	"errors"
	"slices"

	"github.com/jatassi/slipstream/internal/files"
	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/imports/internal/db"
	"github.com/jatassi/slipstream/internal/library"
	"github.com/jatassi/slipstream/internal/roots"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/storing"
	"github.com/jatassi/slipstream/internal/works"
)

// attach is the one way found files join a Work, for a scan, Add and Into:
// each of fs that ffprobe could read is placed at the Items its Items
// name, which an Admin chose, or else where its type's facet places it,
// and the placed ones are recorded in tx through files.Scan. It answers
// what Scan wrote, the files placed with their Items, and the files left,
// each waiting with why: unreadable, or the reason Place gave.
// ErrElsewhere when work lives in another root.
func (im *Imports) attach(ctx context.Context, tx *storage.Tx[file], work works.WorkID, root roots.ID, fs []File) (scanned files.Scanned, placed, rest []File, err error) {
	home, err := roots.OfWork(ctx, tx, work)
	switch {
	case errors.Is(err, roots.ErrNoRoot) || err == nil && home.ID != root:
		return files.Scanned{}, nil, nil, ErrElsewhere
	case err != nil:
		return files.Scanned{}, nil, nil, err
	}
	place, err := im.cfg.Facets[home.ContentType].Place(ctx, tx, work)
	if err != nil {
		return files.Scanned{}, nil, nil, err
	}
	var found []files.Found
	for _, f := range fs {
		if f.Probe == nil {
			rest = append(rest, f)
			continue
		}
		if len(f.Items) == 0 {
			p := place(inside(f.Path))
			item, ok := p.Item()
			if !ok {
				f.waitFor(p.Reason())
				rest = append(rest, f)
				continue
			}
			f.Items = []works.ItemID{item}
		}
		at, err := walked(f)
		if err != nil {
			return files.Scanned{}, nil, nil, err
		}
		found = append(found, files.Found{Items: f.Items, Cut: f.Cut, At: at, Media: *f.Probe, Quality: f.Quality})
		placed = append(placed, f)
	}
	if len(found) == 0 {
		return files.Scanned{}, nil, rest, nil
	}
	scanned, err = im.cfg.Files.Scan(ctx, tx, files.Scan{Work: work, Root: root, Found: found})
	return scanned, placed, rest, err
}

func walked(f File) (storing.Walked, error) {
	path, err := storing.ParseRel(f.Path)
	if err != nil {
		return storing.Walked{}, err
	}
	w := storing.Walked{Path: path, Size: f.Size}
	if f.Seen != nil {
		w.Seen = *f.Seen
	}
	return w, nil
}

// attachEntry attaches root Import id's waiting files to work in tx and
// keeps the rest waiting, with work as its suggestion.
func (im *Imports) attachEntry(ctx context.Context, tx *storage.Tx[file], id ImportID, work works.WorkID) error {
	e, err := importOf(ctx, tx, id)
	if err != nil {
		return err
	}
	_, placed, rest, err := im.attach(ctx, tx, work, e.Root, e.waiting())
	if err != nil {
		return err
	}
	_, err = im.keep(ctx, tx, e, work, placed, rest)
	return err
}

// keep writes what attaching e's files to work made of them. Each of
// placed that e lists is placed in place. Each of rest waits, appended
// when e doesn't list it yet. Every other waiting file of e is gone. While
// a file waits, work is e's suggestion.
func (im *Imports) keep(ctx context.Context, tx *storage.Tx[file], e Import, work works.WorkID, placed, rest []File) (Import, error) {
	now := map[int]File{}
	for _, f := range placed {
		if f.Position >= 0 {
			f.Outcome, f.Reason, f.Hold, f.PlacedBy = Placed, nil, nil, InPlace
			now[f.Position] = f
		}
	}
	var fresh []File
	for _, f := range rest {
		if f.Position < 0 {
			fresh = append(fresh, f)
			continue
		}
		now[f.Position] = f
	}
	var kept []File
	for _, f := range e.Files {
		if f.Outcome != Waiting {
			kept = append(kept, f)
		} else if g, ok := now[f.Position]; ok {
			kept = append(kept, g)
		}
	}
	e.Files = kept
	appendWaiting(&e, fresh...)
	if len(rest) > 0 {
		tree, err := im.cfg.Works.TreeIn(ctx, tx, work)
		if err != nil {
			return Import{}, err
		}
		s := im.suggestionOf(tree)
		s.WorkID = work
		e.Suggestion = &s
	}
	err := save(ctx, tx, &e)
	return e, err
}

func (im *Imports) suggestionOf(t works.Tree) Suggestion {
	s := Suggestion{Key: t.Key, Label: t.Label, Year: im.cfg.Facets[t.ContentType].YearOf(works.Candidate{Fields: t.Fields})}
	if d, ok := im.cfg.Works.Describe(t.ContentType); ok {
		s.Poster = t.Artwork[d.Art.Poster]
	}
	return s
}

// Added is what Add did with one root Import: the Work its files joined,
// or why it refused.
type Added struct {
	ID   ImportID
	Work works.WorkID
	Err  error
}

// Add adds each root Import's suggested Work into its root with its type's
// add defaults, and attaches the files Place places, in the add's Write. A
// suggestion already in the library takes the files when it lives in the
// Import's root. An Import with no suggestion refuses with
// ErrNoSuggestion, and one whose Work lives in another root with
// ErrElsewhere, on its own: the rest go on.
func (im *Imports) Add(ctx context.Context, by identity.Admin, ids []ImportID) ([]Added, error) {
	out := make([]Added, len(ids))
	for i, id := range ids {
		out[i] = Added{ID: id}
		out[i].Work, out[i].Err = im.add(ctx, by, id)
	}
	return out, nil
}

func (im *Imports) add(ctx context.Context, by identity.Admin, id ImportID) (works.WorkID, error) {
	e, err := importOf(ctx, im.cfg.Library, id)
	if err != nil {
		return works.WorkID{}, err
	}
	if e.Suggestion == nil {
		return works.WorkID{}, ErrNoSuggestion
	}
	added, err := im.cfg.Add(ctx, by, library.Adding{
		Type: e.ContentType, Key: e.Suggestion.Key, Root: &e.Root,
		Attach: func(ctx context.Context, tx *storage.Tx[file], w works.Summary) error {
			return im.attachEntry(ctx, tx, id, w.ID)
		},
	})
	if err != nil {
		return works.WorkID{}, err
	}
	work := added.Tree.ID
	if !added.Created {
		err = im.cfg.Library.Write(ctx, func(ctx context.Context, tx *storage.Tx[file]) error {
			return im.attachEntry(ctx, tx, id, work)
		})
		if err != nil {
			return works.WorkID{}, err
		}
	}
	im.cfg.Changed.Publish(ctx, changedOf(e.Root))
	im.cfg.FilesChanged.Publish(ctx, files.Changed{WorkID: work.String()})
	return work, nil
}

// Chosen is a waiting file of a root Import Into attaches, and the Items
// of the Work it holds, in any order.
type Chosen struct {
	Position int
	Items    []works.ItemID
}

// Into attaches the chosen files to work, a Work in the root Import's
// root, at the Items chosen, in one Write. It answers the Import, finished
// once no file waits. ErrNoImport, ErrChanged when the Import no longer
// has a chosen file waiting, ErrElsewhere and ErrNotOfWork refuse it
// whole. A chosen file ffprobe can't read stays.
func (im *Imports) Into(ctx context.Context, _ identity.Admin, id ImportID, work works.WorkID, chosen []Chosen) (Import, error) {
	var (
		left  Import
		moved bool
	)
	err := im.cfg.Library.Write(ctx, func(ctx context.Context, tx *storage.Tx[file]) error {
		e, err := importOf(ctx, tx, id)
		if err != nil {
			return err
		}
		items, err := works.ItemsOf(ctx, tx, works.Only([]works.WorkID{work}))
		if err != nil {
			return err
		}
		picked, rest, err := pick(e, items[work], chosen)
		if err != nil {
			return err
		}
		scanned, placed, unread, err := im.attach(ctx, tx, work, e.Root, picked)
		if err != nil {
			return err
		}
		moved = scanned.Changed
		left, err = im.keep(ctx, tx, e, work, placed, append(rest, unread...))
		return err
	})
	if err != nil {
		return Import{}, err
	}
	im.cfg.Changed.Publish(ctx, changedOf(left.Root))
	if moved {
		im.cfg.FilesChanged.Publish(ctx, files.Changed{WorkID: work.String()})
	}
	return left, nil
}

// pick splits e's waiting files into the chosen ones, each holding its
// chosen Items in tree order, and the rest. ErrNotOfWork when a chosen
// Item isn't one of items, ErrChanged when a chosen file doesn't wait.
func pick(e Import, items []works.Item, chosen []Chosen) (picked, rest []File, err error) {
	order := map[works.ItemID]int{}
	for i, it := range items {
		order[it.ID] = i
	}
	for _, f := range e.waiting() {
		i := slices.IndexFunc(chosen, func(c Chosen) bool { return c.Position == f.Position })
		if i < 0 {
			rest = append(rest, f)
			continue
		}
		for _, it := range chosen[i].Items {
			if _, ok := order[it]; !ok {
				return nil, nil, ErrNotOfWork
			}
		}
		f.Items = slices.SortedFunc(slices.Values(chosen[i].Items), func(a, b works.ItemID) int { return order[a] - order[b] })
		picked = append(picked, f)
	}
	if len(picked) < len(chosen) {
		return nil, nil, ErrChanged
	}
	return picked, rest, nil
}

// Match makes the record t's source knows by key the root Import's
// suggestion. Later scans keep it while the folder's name reads the same.
func (im *Imports) Match(ctx context.Context, by identity.Admin, id ImportID, key works.Key) (Import, error) {
	e, err := importOf(ctx, im.cfg.Library, id)
	if err != nil {
		return Import{}, err
	}
	tree, stored, err := im.cfg.Works.Preview(ctx, by, e.ContentType, key, func(context.Context, storage.Reader[file], works.Tree) error { return nil })
	if err != nil {
		return Import{}, err
	}
	s := im.suggestionOf(tree)
	err = im.cfg.Library.Write(ctx, func(ctx context.Context, tx *storage.Tx[file]) error {
		if e, err = importOf(ctx, tx, id); err != nil {
			return err
		}
		e.Suggestion = &s
		return save(ctx, tx, &e)
	})
	if err != nil {
		return Import{}, err
	}
	if stored {
		e.Suggestion.WorkID = tree.ID
	}
	im.cfg.Changed.Publish(ctx, changedOf(e.Root))
	return e, nil
}

// Ignore ignores each unfinished Import of ids, so it leaves the list that
// needs the Admin until its files change. An id no Import has is skipped.
func (im *Imports) Ignore(ctx context.Context, _ identity.Admin, ids []ImportID) error {
	return im.mark(ctx, ids, func(ctx context.Context, q *db.Queries) error {
		return q.Ignore(ctx, db.IgnoreParams{IgnoredAt: storage.Now(), Ids: ids})
	})
}

// Unignore brings each of ids back to the list that needs the Admin.
func (im *Imports) Unignore(ctx context.Context, _ identity.Admin, ids []ImportID) error {
	return im.mark(ctx, ids, func(ctx context.Context, q *db.Queries) error { return q.Unignore(ctx, ids) })
}

func (im *Imports) mark(ctx context.Context, ids []ImportID, set func(context.Context, *db.Queries) error) error {
	var touched []roots.ID
	err := im.cfg.Library.Write(ctx, func(ctx context.Context, tx *storage.Tx[file]) error {
		q := db.New(tx)
		var err error
		if touched, err = q.ListRootsOf(ctx, ids); err != nil {
			return err
		}
		return set(ctx, q)
	})
	if err != nil {
		return err
	}
	if len(touched) > 0 {
		im.cfg.Changed.Publish(ctx, changedOf(touched...))
	}
	return nil
}
