package imports

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strconv"

	"github.com/jatassi/slipstream/internal/files"
	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/library"
	"github.com/jatassi/slipstream/internal/metadata/artwork"
	"github.com/jatassi/slipstream/internal/roots"
	"github.com/jatassi/slipstream/internal/scanning"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/storing"
	"github.com/jatassi/slipstream/internal/works"
)

// settle reads one folder's found files and writes them: they join the
// one library Work the folder names, become a new Work the scan is sure
// of, or wait in the folder's root Import. A folder with a root Import is
// never added as a new Work: only the library rules run for it.
func (im *Imports) settle(ctx context.Context, s *scan, g scanning.Group) error {
	found := im.found(ctx, s, g.Found)
	if err := ctx.Err(); err != nil {
		return fmt.Errorf("read the files in %s: %w", g.Folder, err)
	}
	hint := im.cfg.Facets[s.root.ContentType].Hint(g.Name())
	m := s.shelf.Match(g.Folder, hint)
	switch {
	case m.Elsewhere != nil:
		return im.pend(ctx, s, g, hint, nil, withReason(found, elsewhere(hint, *m.Elsewhere)))
	case !m.Work.IsZero():
		return im.join(ctx, s, g, hint, m.Work, found)
	case g.Entry != nil:
		e := s.known.entries[g.Folder]
		if len(found) == 0 && len(g.Listed) == len(e.waiting()) {
			return nil
		}
		return im.pend(ctx, s, g, hint, nil, withReason(found, folderReason(e)))
	}
	cs, err := im.lookup(ctx, s.root.ContentType, hint)
	if errors.Is(err, works.ErrNotOnSource) {
		return im.pend(ctx, s, g, hint, nil, withReason(found, scanning.Reason{Code: scanning.Unlisted, Title: hint.Title, Year: hint.Year}))
	}
	if err != nil {
		return err
	}
	switch i := scanning.Confident(hint, bare(cs)); {
	case i >= 0 && !cs[i].WorkID.IsZero():
		return im.join(ctx, s, g, hint, cs[i].WorkID, found)
	case i >= 0:
		return im.adopt(ctx, s, g, hint, cs[i], found)
	}
	reason, named := unsure(hint, bare(cs))
	var suggestion *Suggestion
	if named >= 0 {
		suggestion = new(cs[named].suggestion(false))
	}
	return im.pend(ctx, s, g, hint, suggestion, withReason(found, reason))
}

// found is each of ws read, as a file of no Import yet: one ffprobe can't
// read waits with its line. A file the read couldn't reach is left for the
// next scan.
func (im *Imports) found(ctx context.Context, s *scan, ws []storing.Walked) []File {
	probes := im.probe(ctx, s, ws)
	var out []File
	for _, w := range ws {
		p, ok := probes[w.Path]
		if !ok {
			continue
		}
		f := File{Position: -1, Path: w.Path.String(), Size: w.Size, Seen: &w.Seen}
		if p.Error != "" {
			f.waitFor(scanning.Reason{Code: scanning.Unreadable, Line: p.Error})
		} else {
			f.Probe = &p.Media
			read, quality := scanning.QualityOf(f.Path, im.cfg.Facets[s.root.ContentType], s.known.cuts, p.Media.Resolution())
			f.Quality, f.Cut = quality, read.Facts.Cut
		}
		out = append(out, f)
	}
	return out
}

// join attaches the folder's files, its root Import's waiting ones too, to
// work in one Write. The files work's type can't place wait in the
// folder's root Import with work as its suggestion. A Work in another root
// takes none of them.
func (im *Imports) join(ctx context.Context, s *scan, g scanning.Group, hint scanning.Hint, work works.WorkID, found []File) error {
	var scanned files.Scanned
	err := im.cfg.Library.Write(ctx, func(ctx context.Context, tx *storage.Tx[file]) error {
		var err error
		scanned, err = im.joinIn(ctx, tx, s, g, hint, work, found)
		return err
	})
	if errors.Is(err, ErrElsewhere) {
		var home *roots.Root
		err = im.cfg.Library.Read(ctx, func(ctx context.Context, r storage.Reader[file]) error {
			root, err := roots.OfWork(ctx, r, work)
			if err == nil {
				home = &root
			}
			return err
		})
		if err != nil {
			return err
		}
		return im.pend(ctx, s, g, hint, nil, withReason(found, elsewhere(hint, home.ID)))
	}
	if err != nil {
		return err
	}
	s.pending = true
	if scanned.Changed {
		im.cfg.FilesChanged.Publish(ctx, files.Changed{WorkID: work.String()})
	}
	return nil
}

func (im *Imports) joinIn(ctx context.Context, tx *storage.Tx[file], s *scan, g scanning.Group, hint scanning.Hint, work works.WorkID, found []File) (files.Scanned, error) {
	if g.Entry == nil {
		scanned, _, rest, err := im.attach(ctx, tx, work, s.root.ID, found)
		if err != nil || len(rest) == 0 {
			return scanned, err
		}
		tree, err := im.cfg.Works.TreeIn(ctx, tx, work)
		if err != nil {
			return files.Scanned{}, err
		}
		suggestion := im.suggestionOf(tree)
		return scanned, upsertPending(ctx, tx, pend{Root: s.root.ID, Folder: g.Folder, ContentType: s.root.ContentType, Hint: hint, Suggestion: &suggestion, Files: rest}, storage.Now())
	}
	e, err := importOf(ctx, tx, ImportID{UUID: g.Entry.ID})
	if err != nil {
		return files.Scanned{}, err
	}
	listed := map[string]bool{}
	for _, w := range g.Listed {
		listed[w.Path.String()] = true
	}
	fs := slices.DeleteFunc(e.waiting(), func(f File) bool { return !listed[f.Path] })
	scanned, placed, rest, err := im.attach(ctx, tx, work, s.root.ID, append(fs, found...))
	if err != nil {
		return files.Scanned{}, err
	}
	err = im.keep(ctx, tx, e, work, placed, rest)
	return scanned, err
}

// adopt adds c, the Work the scan is sure the folder holds, into the root
// with its type's add defaults, and attaches the folder's files in the
// add's Write.
func (im *Imports) adopt(ctx context.Context, s *scan, g scanning.Group, hint scanning.Hint, c candidate, found []File) error {
	attach := func(ctx context.Context, tx *storage.Tx[file], w works.Summary) error {
		_, err := im.joinIn(ctx, tx, s, g, hint, w.ID, found)
		return err
	}
	added, err := im.cfg.Add(ctx, identity.Admin{}, library.Adding{Type: s.root.ContentType, Key: c.Key, Root: &s.root.ID, By: works.AddedByScan, Attach: attach})
	if err != nil {
		return err
	}
	work := added.Tree.ID
	if !added.Created {
		return im.join(ctx, s, g, hint, work, found)
	}
	s.added++
	s.pending = true
	s.shelf.Add(scanning.Shelved{Work: work, Root: s.root.ID, Label: added.Tree.Label, Claim: scanning.Claim{Year: c.Year, IDs: hint.IDs}})
	im.cfg.FilesChanged.Publish(ctx, files.Changed{WorkID: work.String()})
	return nil
}

// pend writes the folder's found files into its root Import with their
// reasons, keeping the files it lists that the walk found unchanged.
func (im *Imports) pend(ctx context.Context, s *scan, g scanning.Group, hint scanning.Hint, suggestion *Suggestion, found []File) error {
	listed := make([]string, len(g.Listed))
	for i, w := range g.Listed {
		listed[i] = w.Path.String()
	}
	p := pend{Root: s.root.ID, Folder: g.Folder, ContentType: s.root.ContentType, Hint: hint, Suggestion: suggestion, Files: found, Listed: listed}
	err := im.cfg.Library.Write(ctx, func(ctx context.Context, tx *storage.Tx[file]) error {
		return upsertPending(ctx, tx, p, storage.Now())
	})
	if err == nil {
		s.pending = true
	}
	return err
}

// candidate is one record the folder may hold at its type's source.
type candidate struct {
	scanning.Candidate
	label  string
	poster artwork.Ref
}

func (c candidate) suggestion(sure bool) Suggestion {
	return Suggestion{Key: c.Key, Label: c.label, Year: c.Year, Poster: c.poster, Confident: sure}
}

func bare(cs []candidate) []scanning.Candidate {
	out := make([]scanning.Candidate, len(cs))
	for i, c := range cs {
		out[i] = c.Candidate
	}
	return out
}

// lookup is the records the hint may name at t's source: the one its id
// tag keys, else a search for its title and year.
func (im *Imports) lookup(ctx context.Context, t works.TypeID, h scanning.Hint) ([]candidate, error) {
	facet := im.cfg.Facets[t]
	if h.Key != "" {
		tree, stored, err := im.cfg.Works.Preview(ctx, identity.Admin{}, t, h.Key, func(context.Context, storage.Reader[file], works.Tree) error { return nil })
		if err != nil {
			return nil, err
		}
		s := im.suggestionOf(tree)
		c := candidate{Candidate: scanning.Candidate{Key: tree.Key, Titles: []string{tree.Label}, Year: s.Year}, label: tree.Label, poster: s.Poster}
		if stored {
			c.WorkID = tree.ID
		}
		return []candidate{c}, nil
	}
	query := h.Title
	if h.Year != 0 {
		query += " " + strconv.Itoa(h.Year)
	}
	found, err := im.cfg.Works.Search(ctx, identity.Admin{}, t, query)
	if err != nil {
		return nil, err
	}
	out := make([]candidate, len(found))
	for i, w := range found {
		titles := []string{w.Label}
		if w.Original != "" {
			titles = append(titles, w.Original)
		}
		out[i] = candidate{Candidate: scanning.Candidate{Key: w.Key, Titles: titles, Year: facet.YearOf(w), WorkID: w.WorkID}, label: w.Label, poster: w.Poster}
	}
	return out, nil
}

// withReason makes each of fs ffprobe could read wait for r.
func withReason(fs []File, r scanning.Reason) []File {
	for i := range fs {
		if fs[i].Probe != nil {
			fs[i].waitFor(r)
		}
	}
	return fs
}

func elsewhere(h scanning.Hint, home roots.ID) scanning.Reason {
	return scanning.Reason{Code: scanning.Elsewhere, Title: h.Title, Year: h.Year, Root: home.String()}
}

// folderReason is why the files of e's folder wait: the reason its files
// already wait for that is about the folder, not one file.
func folderReason(e Import) scanning.Reason {
	for _, f := range e.waiting() {
		switch r := f.Scan(); r.Code {
		case scanning.Several, scanning.NoMatch, scanning.NoYear, scanning.Elsewhere, scanning.Unlisted, scanning.Left:
			return r
		case scanning.NoNumber, scanning.NoSuchItem, scanning.Unreadable, scanning.NoRule:
		}
	}
	return scanning.Reason{Code: scanning.NoMatch, Title: e.Hint.Title, Year: e.Hint.Year}
}

// unsure is why a folder whose candidates cs name no Work with confidence
// waits, and the first of cs that reason names, its suggestion: several
// candidates share its title, one does but the name has no year, or none
// matches and named is -1.
func unsure(h scanning.Hint, cs []scanning.Candidate) (r scanning.Reason, named int) {
	r = scanning.Reason{Title: h.Title, Year: h.Year}
	title := scanning.Fold(h.Title)
	var titled, dated []int
	for i, c := range cs {
		if slices.ContainsFunc(c.Titles, func(t string) bool { return scanning.Fold(t) == title }) {
			titled = append(titled, i)
			if c.Year == h.Year {
				dated = append(dated, i)
			}
		}
	}
	switch {
	case h.Year == 0 && len(titled) == 1:
		r.Code = scanning.NoYear
		return r, titled[0]
	case h.Year == 0 && len(titled) > 1:
		r.Code, r.Count = scanning.Several, len(titled)
		return r, titled[0]
	case len(dated) > 1:
		r.Code, r.Count = scanning.Several, len(dated)
		return r, dated[0]
	}
	r.Code = scanning.NoMatch
	return r, -1
}
