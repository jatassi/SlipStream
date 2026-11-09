package files

import (
	"cmp"
	"context"
	"maps"
	"slices"
	"time"

	"github.com/jatassi/slipstream/internal/files/internal/db"
	"github.com/jatassi/slipstream/internal/formats"
	"github.com/jatassi/slipstream/internal/naming"
	"github.com/jatassi/slipstream/internal/release"
	"github.com/jatassi/slipstream/internal/roots"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/targets"
	"github.com/jatassi/slipstream/internal/works"
)

// world is everything cells, refill and names read about one Work, loaded
// in one Read or Write. The pure functions take it, so nothing in it is
// read twice.
type world struct {
	tree  works.Bare
	goals targets.Goals
	// root is the root the Work lives in, zero while it has none.
	root  roots.ID
	files []File // oldest first: ids are UUIDv7
	fills map[CellKey]Fill
	// scoring is what the files match of the formats goals score.
	scoring *scoring
	grabs   map[CellKey]GrabCell
	lost    map[roots.ID]bool
	timing  map[works.ItemID]targets.Timing
	// seen is each Item's sighted cuts and its files' cuts; cuts is every
	// cut, which names files.
	seen  map[works.ItemID][]release.CutKey
	cuts  targets.Cuts
	today time.Time
	// fresh is the file being placed, which takes its folder and its name
	// from the templates. Until names gives it one, its Path is where its
	// bytes are held.
	fresh ID
	// beside are the Work's files from before the fresh file came, the
	// ones it dooms included. The fresh file lands beside them.
	beside []File
	// meant is the name of the first Target the fresh file's Grab was
	// meant to fill, which labels it should it fill and meet none.
	meant string
	// found are a scan's new files, inserted where the scan found them.
	found map[ID]bool
	// keepPaths is a scan's: every file keeps its path.
	keepPaths bool
	// retagged is each cut whose name changes, so its files take new names.
	retagged map[release.CutKey]bool
	// templates and tokens are loaded by named, for names alone.
	templates naming.Templates
	tokens    map[works.ItemID]map[string]string
}

func (f *Files) load(ctx context.Context, r storage.Reader[file], work works.WorkID) (world, error) {
	tree, err := works.BareIn(ctx, r, work)
	if err != nil {
		return world{}, err
	}
	ws, err := f.worlds(ctx, r, []works.Bare{tree}, works.Only([]works.WorkID{work}), wholeFiles)
	if err != nil {
		return world{}, err
	}
	return ws[0], nil
}

// worlds loads the world of each Work of trees in r, in trees' order,
// reading each kind of row once for them all, and their files with read.
// among names trees' Works.
func (f *Files) worlds(ctx context.Context, r storage.Reader[file], trees []works.Bare, among works.Among[works.WorkID], read fileRead) ([]world, error) {
	byType := map[works.TypeID][]works.WorkID{}
	for _, t := range trees {
		byType[t.ContentType] = append(byType[t.ContentType], t.ID)
	}
	goals, err := targets.GoalsOf(ctx, r, among)
	if err != nil {
		return nil, err
	}
	known, err := f.cfg.Targets.CutsIn(ctx, r)
	if err != nil {
		return nil, err
	}
	timing := map[works.ItemID]targets.Timing{}
	originals := map[works.WorkID]release.Fact[release.Language]{}
	for _, typ := range slices.Sorted(maps.Keys(byType)) {
		of, err := f.cfg.Targets.TimingsIn(ctx, r, typ, works.Like(among, byType[typ]))
		if err != nil {
			return nil, err
		}
		maps.Copy(timing, of)
		langs, err := f.cfg.Targets.OriginalsIn(ctx, r, typ, works.Like(among, byType[typ]))
		if err != nil {
			return nil, err
		}
		maps.Copy(originals, langs)
	}
	catalog, err := f.cfg.Targets.FormatsIn(ctx, r)
	if err != nil {
		return nil, err
	}
	var items []works.ItemID
	for _, t := range trees {
		for _, it := range t.Items {
			items = append(items, it.ID)
		}
	}
	seen, err := targets.SeenIn(ctx, r, works.Like(among, items))
	if err != nil {
		return nil, err
	}
	grabs, err := f.cfg.Grabs.GrabCells(ctx, r, among)
	if err != nil {
		return nil, err
	}
	lost, err := roots.LostIn(ctx, r)
	if err != nil {
		return nil, err
	}
	homes, err := roots.HomesOf(ctx, r, among)
	if err != nil {
		return nil, err
	}
	out := make([]world, len(trees))
	today := f.cfg.Today()
	for i, t := range trees {
		out[i] = world{
			tree: t, goals: goals[t.ID], root: homes[t.ID], scoring: newScoring(catalog, goals[t.ID], originals[t.ID]),
			grabs: grabs, lost: lost, timing: timing, seen: seen, cuts: known, today: today,
		}
	}
	if err := shelve(ctx, db.New(r), out, among, read, seen); err != nil {
		return nil, err
	}
	return out, nil
}

// shelve puts each file of ws's Works on its Work's world, oldest first,
// with its fills, and adds the file's cut to seen. among names ws's Works.
func shelve(ctx context.Context, q *db.Queries, ws []world, among works.Among[works.WorkID], read fileRead, seen map[works.ItemID][]release.CutKey) error {
	at := make(map[works.WorkID]*world, len(ws))
	for i := range ws {
		at[ws[i].tree.ID] = &ws[i]
	}
	rows, err := read(ctx, q, among)
	if err != nil {
		return err
	}
	of := make(map[ID]*world, len(rows))
	for _, row := range rows {
		w := at[row.WorkID]
		w.files = append(w.files, row)
		of[row.ID] = w
		if !slices.Contains(seen[row.ItemID], row.Cut) {
			seen[row.ItemID] = append(seen[row.ItemID], row.Cut)
		}
	}
	for i := range ws {
		slices.SortFunc(ws[i].files, func(a, b File) int { return a.ID.Compare(b.ID.UUID) })
	}
	fills, err := works.Read(among, func() ([]db.FilesFill, error) { return q.ListEveryFill(ctx) }, func(ids []works.WorkID) ([]db.FilesFill, error) {
		return q.ListFillsOfWorks(ctx, ids)
	})
	if err != nil {
		return err
	}
	for _, fill := range fills {
		w := of[fill.FileID]
		if w.fills == nil {
			w.fills = map[CellKey]Fill{}
		}
		w.fills[CellKey{Item: fill.ItemID, Edition: fill.Edition, Target: fill.TargetID}] = Fill{File: fill.FileID, Score: int(fill.Score)}
	}
	return nil
}

// fileRead reads the files of among's Works.
type fileRead func(context.Context, *db.Queries, works.Among[works.WorkID]) ([]File, error)

func wholeFiles(ctx context.Context, q *db.Queries, among works.Among[works.WorkID]) ([]File, error) {
	rows, err := works.Read(among, func() ([]db.FilesFile, error) { return q.ListEveryFile(ctx) }, func(ids []works.WorkID) ([]db.FilesFile, error) {
		return q.ListFilesOfWorks(ctx, ids)
	})
	out := make([]File, len(rows))
	for i, row := range rows {
		out[i] = fileOf(row)
	}
	return out, err
}

// briefFiles reads only what cells and a Rollup read of each file: not its
// path, media record, stamps, Grab or label.
func briefFiles(ctx context.Context, q *db.Queries, among works.Among[works.WorkID]) ([]File, error) {
	rows, err := works.Read(among, func() ([]db.ListEveryBriefFileRow, error) { return q.ListEveryBriefFile(ctx) }, func(ids []works.WorkID) ([]db.ListEveryBriefFileRow, error) {
		rows, err := q.ListBriefFilesOfWorks(ctx, ids)
		out := make([]db.ListEveryBriefFileRow, len(rows))
		for i, row := range rows {
			out[i] = db.ListEveryBriefFileRow(row)
		}
		return out, err
	})
	out := make([]File, len(rows))
	for i, row := range rows {
		out[i] = File{ID: row.ID, WorkID: row.WorkID, ItemID: row.ItemID, Cut: row.Cut, RootID: row.RootID, Size: row.Size, Quality: row.Quality, Extra: row.Extra, GoneAt: row.GoneAt}
	}
	return out, err
}

// named adds what names reads to w under fills: the Content Type's
// templates and the tokens of every Item holding a file whose name
// changes, read Item by Item. The placed file's come first: labels reads
// them for its folder.
func (f *Files) named(ctx context.Context, r storage.Reader[file], w *world, fills map[CellKey]Fill) error {
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

// cells crosses each Item with each Edition it wants, in list order, and
// the Profile's Targets in order. A filled cell is met at or past the
// Target's cutoff, else upgradable. An empty one is downloading or failed
// from its latest Grab, unreleased until the Item is out, else missing; a
// file in a lost root leaves its cell empty. A cut not seen for the Item
// makes no cell, so it never counts.
func cells(w world) []Cell { return appendCells(nil, w) }

func appendCells(out []Cell, w world) []Cell {
	files := w.byID()
	for _, it := range w.tree.Items {
		timing := w.timing[it.ID]
		released := timing.ReleasedBy(w.today)
		for _, e := range targets.Wanted(w.goals.Editions, w.seen[it.ID]) {
			for _, t := range w.goals.Targets {
				key := CellKey{Item: it.ID, Edition: e, Target: t.ID}
				c := Cell{Item: it, Edition: e, Target: t, State: Missing, Timing: timing, Released: released}
				if fill, ok := w.fills[key]; ok && !w.lost[files[fill.File].RootID] {
					c.File, c.Score = files[fill.File], fill.Score
				}
				switch {
				case c.File != nil && final(t, *c.File, c.Score):
					c.State = Met
				case c.File != nil:
					c.State = Upgradable
				case w.grabs[key].State == GrabDownloading:
					c.State = Downloading
				case w.grabs[key].State == GrabFailed:
					c.State = Failed
				case !released:
					c.State = Unreleased
				}
				out = append(out, c)
			}
		}
	}
	return out
}

// final reports whether f, scoring score, is final at t. A file whose
// quality t doesn't rank is not.
func final(t targets.Target, f File, score int) bool {
	r, ok := t.RankOf(f.Quality)
	return ok && t.Final(r, score)
}

// filling is the set of files fills holds.
func filling(fills map[CellKey]Fill) map[ID]bool {
	out := make(map[ID]bool, len(fills))
	for _, fill := range fills {
		out[fill.File] = true
	}
	return out
}

// scoring is how a world's files score at its Targets. A file is matched
// against the formats the Targets score only when refill first asks, and
// once for every Target.
type scoring struct {
	all      *formats.Catalog
	goals    targets.Goals
	original release.Fact[release.Language]
	scored   *formats.Catalog
	matched  map[ID]formats.Set
	// known is each file's score at each Target that a Change narrowed by
	// its format edits already knows: its stored fill score, moved.
	known map[ID]map[targets.TargetID]int
}

func newScoring(all *formats.Catalog, goals targets.Goals, original release.Fact[release.Language]) *scoring {
	return &scoring{all: all, goals: goals, original: original, matched: map[ID]formats.Set{}}
}

// facts is f's Facts, under the Work's original language.
func (s *scoring) facts(f File) release.Facts {
	facts := f.Facts()
	facts.Coverage.Original = s.original
	return facts
}

// of is f with the formats it matches of those the Targets score.
func (s *scoring) of(f File) targets.Scored {
	set, ok := s.matched[f.ID]
	if !ok {
		if s.scored == nil {
			s.scored = s.all.Only(targets.ScoredIn(s.goals))
		}
		if s.scored.Len() > 0 {
			set = s.scored.Match(asNamed(f.Title, f.ReleaseName, s.original)).Matched
		}
		s.matched[f.ID] = set
	}
	return targets.Scored{Facts: s.facts(f), Formats: set}
}

// score is f's score at t.
func (s *scoring) score(t targets.Target, f File) int {
	if n, ok := s.known[f.ID][t.ID]; ok {
		return n
	}
	return t.Score(s.of(f))
}

// compare orders a and b at t by its steps, +1 when a is better. They are
// scored only once the two tie before the score.
func (s *scoring) compare(t targets.Target, a, b File) int {
	sa, sb := targets.Scored{Facts: s.facts(a)}, targets.Scored{Facts: s.facts(b)}
	for _, step := range t.Steps() {
		if step == targets.ByScore {
			return cmp.Compare(s.score(t, a), s.score(t, b))
		}
		if c := t.CompareBy(step, sa, sb); c != 0 {
			return c
		}
	}
	return 0
}
