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
	"github.com/jatassi/slipstream/internal/scanning"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/storing"
	"github.com/jatassi/slipstream/internal/works"
)

// attach is the one way found files join a Work, for a scan, Add and Into:
// each of fs that ffprobe could read is placed at the one Item its Items
// names, which an Admin chose, or else where its type's facet places it,
// and the placed ones are recorded in tx through files.Scan. It answers
// what Scan wrote and the files left, each with why: unreadable, or the
// reason Place gave. ErrElsewhere when work lives in another root.
func (im *Imports) attach(ctx context.Context, tx *storage.Tx[file], work works.WorkID, root roots.ID, fs []PendingFile) (files.Scanned, []PendingFile, error) {
	home, err := roots.OfWork(ctx, tx, work)
	switch {
	case errors.Is(err, roots.ErrNoRoot) || err == nil && home.ID != root:
		return files.Scanned{}, nil, ErrElsewhere
	case err != nil:
		return files.Scanned{}, nil, err
	}
	place, err := im.cfg.Facets[home.ContentType].Place(ctx, tx, work)
	if err != nil {
		return files.Scanned{}, nil, err
	}
	var (
		found []files.Found
		rest  []PendingFile
	)
	for _, f := range fs {
		if f.Media == nil {
			rest = append(rest, f)
			continue
		}
		var placed scanning.Placement
		if len(f.Items) == 1 {
			placed = scanning.Placed(f.Items[0])
		} else {
			placed = place(inside(f.Path))
		}
		item, ok := placed.Item()
		if !ok {
			f.Items, f.Reason = nil, placed.Reason()
			rest = append(rest, f)
			continue
		}
		at, err := walked(f)
		if err != nil {
			return files.Scanned{}, nil, err
		}
		found = append(found, files.Found{Item: item, Cut: f.Cut, At: at, Media: *f.Media, Quality: f.Quality})
	}
	if len(found) == 0 {
		return files.Scanned{}, rest, nil
	}
	scanned, err := im.cfg.Files.Scan(ctx, tx, files.Scan{Work: work, Root: root, Found: found})
	return scanned, rest, err
}

func walked(f PendingFile) (storing.Walked, error) {
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

// attachEntry attaches entry id's files to work in tx and keeps the rest
// under the entry, with work as its suggestion.
func (im *Imports) attachEntry(ctx context.Context, tx *storage.Tx[file], id EntryID, work works.WorkID) error {
	e, err := entry(ctx, tx, id)
	if err != nil {
		return err
	}
	_, rest, err := im.attach(ctx, tx, work, e.Root, e.Files)
	if err != nil {
		return err
	}
	_, err = im.keep(ctx, tx, e, work, rest)
	return err
}

// keep leaves rest under e, which work's files joined, with work as its
// suggestion. e is deleted when rest is empty.
func (im *Imports) keep(ctx context.Context, tx *storage.Tx[file], e Entry, work works.WorkID, rest []PendingFile) (Entry, error) {
	e.Files = rest
	if len(rest) > 0 {
		tree, err := im.cfg.Works.TreeIn(ctx, tx, work)
		if err != nil {
			return Entry{}, err
		}
		s := im.suggestionOf(tree)
		s.WorkID = work
		e.Suggestion = &s
	}
	return e, write(ctx, tx, e)
}

func (im *Imports) suggestionOf(t works.Tree) Suggestion {
	s := Suggestion{Key: t.Key, Label: t.Label, Year: im.cfg.Facets[t.ContentType].YearOf(works.Candidate{Fields: t.Fields})}
	if d, ok := im.cfg.Works.Describe(t.ContentType); ok {
		s.Poster = t.Artwork[d.Art.Poster]
	}
	return s
}

// Outcome is what Add did with one entry: the Work its files joined, or
// why it refused.
type Outcome struct {
	ID   EntryID
	Work works.WorkID
	Err  error
}

// Add adds each entry's suggested Work into the entry's root with its
// type's add defaults, and attaches the files Place places, in the add's
// Write. A suggestion already in the library takes the files when it lives
// in the entry's root. An entry with no suggestion refuses with
// ErrNoSuggestion, and one whose Work lives in another root with
// ErrElsewhere, on its own: the rest go on.
func (im *Imports) Add(ctx context.Context, by identity.Admin, ids []EntryID) ([]Outcome, error) {
	out := make([]Outcome, len(ids))
	for i, id := range ids {
		out[i] = Outcome{ID: id}
		out[i].Work, out[i].Err = im.add(ctx, by, id)
	}
	return out, nil
}

func (im *Imports) add(ctx context.Context, by identity.Admin, id EntryID) (works.WorkID, error) {
	e, err := entry(ctx, im.cfg.Library, id)
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

// Chosen is a file Into attaches, and the Item it holds.
type Chosen struct {
	Path string
	Item works.ItemID
}

// Into attaches the chosen files to work, a Work in the entry's root, at
// the Items chosen, in one Write. It answers the entry with the files
// left, nil once none is. ErrNoEntry, ErrChanged when the entry no longer
// lists a chosen path, ErrElsewhere and ErrNotOfWork refuse it whole. A
// chosen file ffprobe can't read stays.
func (im *Imports) Into(ctx context.Context, _ identity.Admin, id EntryID, work works.WorkID, chosen []Chosen) (*Entry, error) {
	var (
		left  Entry
		moved bool
	)
	err := im.cfg.Library.Write(ctx, func(ctx context.Context, tx *storage.Tx[file]) error {
		e, err := entry(ctx, tx, id)
		if err != nil {
			return err
		}
		items, err := works.ItemsOf(ctx, tx, works.Only([]works.WorkID{work}))
		if err != nil {
			return err
		}
		var picked, rest []PendingFile
		for _, f := range e.Files {
			i := slices.IndexFunc(chosen, func(c Chosen) bool { return c.Path == f.Path })
			if i < 0 {
				rest = append(rest, f)
				continue
			}
			if !slices.ContainsFunc(items[work], func(it works.Item) bool { return it.ID == chosen[i].Item }) {
				return ErrNotOfWork
			}
			f.Items = []works.ItemID{chosen[i].Item}
			picked = append(picked, f)
		}
		if len(picked) < len(chosen) {
			return ErrChanged
		}
		scanned, unread, err := im.attach(ctx, tx, work, e.Root, picked)
		if err != nil {
			return err
		}
		moved = scanned.Changed
		left, err = im.keep(ctx, tx, e, work, append(rest, unread...))
		return err
	})
	if err != nil {
		return nil, err
	}
	im.cfg.Changed.Publish(ctx, changedOf(left.Root))
	if moved {
		im.cfg.FilesChanged.Publish(ctx, files.Changed{WorkID: work.String()})
	}
	if len(left.Files) == 0 {
		return nil, nil
	}
	return &left, nil
}

// Match makes the record t's source knows by key the entry's suggestion.
// Later scans keep it while the folder's name reads the same.
func (im *Imports) Match(ctx context.Context, by identity.Admin, id EntryID, key works.Key) (Entry, error) {
	e, err := entry(ctx, im.cfg.Library, id)
	if err != nil {
		return Entry{}, err
	}
	tree, stored, err := im.cfg.Works.Preview(ctx, by, e.ContentType, key, func(context.Context, storage.Reader[file], works.Tree) error { return nil })
	if err != nil {
		return Entry{}, err
	}
	s := im.suggestionOf(tree)
	err = im.cfg.Library.Write(ctx, func(ctx context.Context, tx *storage.Tx[file]) error {
		if e, err = entry(ctx, tx, id); err != nil {
			return err
		}
		e.Suggestion = &s
		return write(ctx, tx, e)
	})
	if err != nil {
		return Entry{}, err
	}
	if stored {
		e.Suggestion.WorkID = tree.ID
	}
	im.cfg.Changed.Publish(ctx, changedOf(e.Root))
	return e, nil
}

// Ignore ignores each of ids, so it leaves the list that needs the Admin
// until its files change. An id no entry has is skipped.
func (im *Imports) Ignore(ctx context.Context, _ identity.Admin, ids []EntryID) error {
	return im.mark(ctx, ids, func(ctx context.Context, q *db.Queries) error {
		return q.Ignore(ctx, db.IgnoreParams{IgnoredAt: storage.Now(), Ids: ids})
	})
}

// Unignore brings each of ids back to the list that needs the Admin.
func (im *Imports) Unignore(ctx context.Context, _ identity.Admin, ids []EntryID) error {
	return im.mark(ctx, ids, func(ctx context.Context, q *db.Queries) error { return q.Unignore(ctx, ids) })
}

func (im *Imports) mark(ctx context.Context, ids []EntryID, set func(context.Context, *db.Queries) error) error {
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
