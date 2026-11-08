package imports

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"slices"
	"sync"

	"github.com/jatassi/slipstream/internal/files"
	"github.com/jatassi/slipstream/internal/imports/internal/db"
	"github.com/jatassi/slipstream/internal/media"
	"github.com/jatassi/slipstream/internal/nodes"
	"github.com/jatassi/slipstream/internal/release"
	"github.com/jatassi/slipstream/internal/roots"
	"github.com/jatassi/slipstream/internal/scanning"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/storing"
	"github.com/jatassi/slipstream/internal/tasks"
	"github.com/jatassi/slipstream/internal/works"
)

// walkPage is how many files one Walk page lists.
const walkPage = 1000

// scan is one root's unit in progress: what it read before comparing and
// what it found so far.
type scan struct {
	root  roots.Root
	at    storing.At
	p     *tasks.Progress
	known known
	shelf *scanning.Shelf
	// probed counts the files read so far of total.
	mu            sync.Mutex
	probed, total int
	added         int
	pending       bool
}

// known is what one Read gives a comparison of a root.
type known struct {
	files   files.OnRootFiles
	entries map[string]Entry
	guard   scanning.Guard
	probes  int
	cuts    []release.Cut
	shelved []scanning.Shelved
}

// unit is one root's Library scan: it walks the root on its Node,
// compares the walk with the records, confirms the gone files, writes
// each Work's recorded files, then settles each folder of found files,
// each in its own Write, so a cancel or a crash loses at most the folder
// in flight. A root in trouble is Skipped, and a comparison the guard
// stopped ends Skipped in the Library Health category.
func (im *Imports) unit(ctx context.Context, u tasks.Unit, p *tasks.Progress) error {
	id, err := rootOf(u.Key)
	if err != nil {
		return err
	}
	root, state, err := im.cfg.Roots.StateOf(ctx, id)
	if err != nil {
		return err
	}
	if !walks(state) {
		return im.skip(ctx, root, state)
	}
	s := &scan{root: root, at: storing.At{Node: root.Node, Root: root.ID.UUID}, p: p}
	defer func() {
		if s.pending {
			im.cfg.Changed.Publish(context.WithoutCancel(ctx), changedOf(root.ID))
		}
	}()
	im.before(ctx, root.ID, Listing)
	listing, err := im.walk(ctx, s)
	if err != nil {
		return im.ended(ctx, root, err)
	}
	im.before(ctx, root.ID, Comparing)
	if s.known, err = im.read(ctx, root); err != nil {
		return err
	}
	listed, err := s.known.listed()
	if err != nil {
		return err
	}
	p.Step(tasks.Step{Phase: string(Comparing), Total: new(listing.Len())})
	plan := scanning.Compare(listing, s.known.files.Files, scanning.Known{Held: s.known.files.Held, Listed: listed}, s.known.guard)
	gone, err := im.confirm(ctx, s.at, plan.Gone)
	if err != nil {
		return im.ended(ctx, root, err)
	}
	s.shelf = scanning.NewShelf(root.ID, s.known.shelved, s.known.files.Files)
	shares := plan.ByWork(gone)
	groups := scanning.Groups(plan, s.known.folders())
	s.total = toRead(shares, groups)
	im.before(ctx, root.ID, Probing)
	s.step()
	for _, work := range slices.SortedFunc(maps.Keys(shares), func(a, b works.WorkID) int { return a.Compare(b.UUID) }) {
		if err := im.share(ctx, s, work, shares[work]); err != nil {
			return im.ended(ctx, root, err)
		}
	}
	for _, g := range groups {
		if err := im.settle(ctx, s, g); err != nil {
			return im.ended(ctx, root, err)
		}
	}
	if err := im.prune(ctx, s, plan); err != nil {
		return err
	}
	c := Comparison{Place: root.Place, Found: len(plan.Found), Gone: len(gone), Returned: returned(plan), Added: s.added, Consumed: s.known.guard.Confirmed}
	stopped, isStopped := plan.Gone.(scanning.Stopped)
	if isStopped {
		c.Stopped = &stopped
	}
	err = im.cfg.Library.Write(ctx, func(ctx context.Context, tx *storage.Tx[file]) error {
		return record(ctx, tx, root.ID, c, storage.Now())
	})
	if err != nil {
		return err
	}
	im.cfg.Roots.Publish(ctx, []roots.ID{root.ID})
	if isStopped {
		return tasks.SkipOwned(stoppedWords(stopped))
	}
	return nil
}

// walks reports whether a root in state is walked. Checking and Offline
// wait on the Node, which judges the root again when the walk reaches it:
// an offline Node fails the walk as Lost, and the unit runs again on its
// return.
func walks(state roots.State) bool {
	return state == roots.Ready || state == roots.Checking || state == roots.Offline
}

func (im *Imports) before(ctx context.Context, root roots.ID, phase Phase) {
	if im.cfg.BeforeStep != nil {
		im.cfg.BeforeStep(ctx, root, phase)
	}
}

// walk pages through the root's files to the end.
func (im *Imports) walk(ctx context.Context, s *scan) (scanning.Listing, error) {
	var (
		pages []storing.Page
		after storing.Cursor
		n     int
	)
	for {
		page, err := im.cfg.Remote.Walk(ctx, s.at, after, walkPage)
		if err != nil {
			return scanning.Listing{}, err
		}
		pages = append(pages, page)
		n += len(page.Entries)
		s.p.Step(tasks.Step{Phase: string(Listing), Done: n})
		if page.Done {
			return scanning.Finish(pages)
		}
		after = page.Next
	}
}

// read is the root's records, its pending entries, the guard and what
// matching a folder reads, in one Read.
func (im *Imports) read(ctx context.Context, root roots.Root) (known, error) {
	var k known
	err := im.cfg.Library.Read(ctx, func(ctx context.Context, r storage.Reader[file]) error {
		var err error
		if k.files, err = files.OnRoot(ctx, r, root.ID); err != nil {
			return err
		}
		all, err := entries(ctx, r)
		if err != nil {
			return err
		}
		k.entries = map[string]Entry{}
		for _, e := range all {
			if e.Root == root.ID {
				k.entries[e.Folder] = e
			}
		}
		if k.guard, err = guardOf(ctx, r, root.ID); err != nil {
			return err
		}
		set, err := db.New(r).GetSettings(ctx)
		if err != nil {
			return err
		}
		k.probes = int(Probes.Value(set.Probes))
		cuts, err := im.cfg.Targets.CutsIn(ctx, r)
		if err != nil {
			return err
		}
		k.cuts = cuts.For(root.ContentType)
		k.shelved, err = im.shelved(ctx, r, root.ContentType)
		return err
	})
	return k, err
}

// listed is every file the root's pending entries list, as the walk read
// it when they listed it.
func (k known) listed() ([]storing.Walked, error) {
	var out []storing.Walked
	for _, e := range k.entries {
		for _, f := range e.Files {
			path, err := storing.ParseRel(f.Path)
			if err != nil {
				return nil, err
			}
			w := storing.Walked{Path: path, Size: f.Size}
			if f.Seen != nil {
				w.Seen = *f.Seen
			}
			out = append(out, w)
		}
	}
	return out, nil
}

func (k known) folders() map[string]scanning.Entry {
	out := make(map[string]scanning.Entry, len(k.entries))
	for folder, e := range k.entries {
		out[folder] = scanning.Entry{ID: e.ID.UUID, Ignored: e.Ignored(), Hint: e.Hint}
	}
	return out
}

// shelved is every library Work of t, with its home, titles and claims.
func (im *Imports) shelved(ctx context.Context, r storage.Reader[file], t works.TypeID) ([]scanning.Shelved, error) {
	all, err := works.Summaries(ctx, r)
	if err != nil {
		return nil, err
	}
	all = slices.DeleteFunc(all, func(w works.Summary) bool { return w.ContentType != t })
	ids := make([]works.WorkID, len(all))
	for i, w := range all {
		ids[i] = w.ID
	}
	homes, err := roots.HomesOf(ctx, r, works.Only(ids))
	if err != nil {
		return nil, err
	}
	claims, err := im.cfg.Facets[t].Claims(ctx, r, works.Only(ids))
	if err != nil {
		return nil, err
	}
	titles, err := works.TitlesIn(ctx, r, t)
	if err != nil {
		return nil, err
	}
	out := make([]scanning.Shelved, len(all))
	for i, w := range all {
		out[i] = scanning.Shelved{Work: w.ID, Root: homes[w.ID], Label: w.Label, Titles: titles[w.ID], Claim: claims[w.ID]}
	}
	return out, nil
}

// confirm reads each gone candidate again on its Node and keeps the ones
// still missing.
func (im *Imports) confirm(ctx context.Context, at storing.At, g scanning.Gone) ([]scanning.Recorded, error) {
	m, ok := g.(scanning.Marked)
	if !ok || len(m.Files) == 0 {
		return nil, nil
	}
	recs := make([]storing.Recorded, len(m.Files))
	for i, r := range m.Files {
		recs[i] = storing.Recorded{Path: r.Path, Sits: r.Sits, Next: r.Next, Size: r.Size}
	}
	missing, err := im.cfg.Remote.VerifyIn(ctx, at, recs, func(int) {})
	if err != nil {
		return nil, err
	}
	var out []scanning.Recorded
	for _, f := range missing {
		if f.Reason == storing.FileMissing {
			out = append(out, m.Files[f.Index])
		}
	}
	return out, nil
}

// share writes one Work's recorded files in its own Write, after reading
// again the ones that need it. A Work that left the root since the read
// waits for the next scan.
func (im *Imports) share(ctx context.Context, s *scan, work works.WorkID, sh scanning.Share) error {
	probes := im.probe(ctx, s, sh.Reprobes())
	if err := ctx.Err(); err != nil {
		return fmt.Errorf("read the files of %s: %w", work, err)
	}
	var scanned files.Scanned
	err := im.cfg.Library.Write(ctx, func(ctx context.Context, tx *storage.Tx[file]) error {
		var err error
		scanned, err = im.cfg.Files.Scan(ctx, tx, files.Scan{Work: work, Root: s.root.ID, Share: sh, Probes: probes})
		return err
	})
	switch {
	case errors.Is(err, files.ErrRootChanged):
		return nil
	case err != nil:
		return err
	}
	if scanned.Changed {
		im.cfg.FilesChanged.Publish(ctx, files.Changed{WorkID: work.String()})
	}
	return nil
}

// probe reads each of ws's media records on the root's Node,
// library.scan.probes at a time. A file the read couldn't reach, such as
// one gone since the walk, has no answer: the next scan reads it again.
func (im *Imports) probe(ctx context.Context, s *scan, ws []storing.Walked) map[storing.Rel]files.Probe {
	out := make(map[storing.Rel]files.Probe, len(ws))
	var (
		mu sync.Mutex
		wg sync.WaitGroup
	)
	slots := make(chan struct{}, s.known.probes)
	for _, w := range ws {
		select {
		case slots <- struct{}{}:
		case <-ctx.Done():
			wg.Wait()
			return out
		}
		wg.Go(func() {
			defer func() { <-slots }()
			rec, err := im.cfg.Remote.Probe(ctx, s.at, w.Path)
			p := files.Probe{Media: rec}
			if u, ok := errors.AsType[*media.UnreadableError](err); ok {
				p.Error = u.Line
			} else if err != nil {
				return
			}
			mu.Lock()
			out[w.Path] = p
			mu.Unlock()
			s.read()
		})
	}
	wg.Wait()
	return out
}

func (s *scan) read() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.probed++
	s.step()
}

func (s *scan) step() {
	s.p.Step(tasks.Step{Phase: string(Probing), Done: s.probed, Total: new(s.total)})
}

// prune drops from the root's pending entries no folder of this scan
// touched each listed file the walk no longer found.
func (im *Imports) prune(ctx context.Context, s *scan, plan scanning.Plan) error {
	vanished := map[string]bool{}
	for _, p := range plan.Vanished {
		vanished[p.String()] = true
	}
	for _, folder := range slices.Sorted(maps.Keys(s.known.entries)) {
		e := s.known.entries[folder]
		var listed []string
		for _, f := range e.Files {
			if !vanished[f.Path] {
				listed = append(listed, f.Path)
			}
		}
		if len(listed) == len(e.Files) {
			continue
		}
		err := im.cfg.Library.Write(ctx, func(ctx context.Context, tx *storage.Tx[file]) error {
			return upsertPending(ctx, tx, pend{Root: s.root.ID, Folder: folder, ContentType: s.root.ContentType, Hint: e.Hint, Listed: listed}, storage.Now())
		})
		if err != nil {
			return err
		}
		s.pending = true
	}
	return nil
}

// toRead is how many files a scan reads: the recorded ones read again and
// the found ones.
func toRead(shares map[works.WorkID]scanning.Share, groups []scanning.Group) int {
	n := 0
	for _, sh := range shares {
		n += len(sh.Reprobes())
	}
	for _, g := range groups {
		n += len(g.Found)
	}
	return n
}

func returned(p scanning.Plan) int {
	n := 0
	for _, s := range slices.Concat(p.Same, p.Changed) {
		if s.Returned {
			n++
		}
	}
	for _, m := range p.Moved {
		if m.Returned {
			n++
		}
	}
	return n
}

// ended is what err ends root's unit with: Skipped when the root stopped
// reading Ready, Failed with the root's trouble when its folder stopped
// responding, else err.
func (im *Imports) ended(ctx context.Context, root roots.Root, err error) error {
	if nr, ok := errors.AsType[*storing.NotReady](err); ok {
		state := roots.Checking
		if nr.State != 0 {
			state = roots.State(nr.State.String())
		}
		return im.skip(ctx, root, state)
	}
	if fe, ok := errors.AsType[*storing.FolderError](err); ok && fe.Kind == storing.FolderUnresponsive {
		return troubleError{roots.Unresponsive}
	}
	return err
}

// skip ends root's unit Skipped with why a root in state isn't scanned.
func (im *Imports) skip(ctx context.Context, root roots.Root, state roots.State) error {
	var node string
	err := im.cfg.Library.Read(ctx, func(ctx context.Context, r storage.Reader[file]) error {
		named, err := nodes.Names(ctx, r)
		if i := slices.IndexFunc(named, func(n nodes.Named) bool { return n.ID == root.Node }); i >= 0 {
			node = named[i].Name
		}
		return err
	})
	if err != nil {
		return err
	}
	return tasks.Skip(skipCause(state, node))
}

// troubleError is a root's trouble, as its Storage item says it.
type troubleError struct{ state roots.State }

func (e troubleError) Error() string {
	sentence, fix, _ := roots.Trouble(e.state)
	return sentence + " " + fix
}
