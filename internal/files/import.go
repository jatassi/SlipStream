package files

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"path"
	"slices"

	"github.com/jatassi/slipstream/internal/files/internal/db"
	"github.com/jatassi/slipstream/internal/media"
	"github.com/jatassi/slipstream/internal/release"
	"github.com/jatassi/slipstream/internal/roots"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/storing"
	"github.com/jatassi/slipstream/internal/targets"
	"github.com/jatassi/slipstream/internal/works"
)

// HoldingOf is the root-relative path where the bytes of the file or Grab
// with id wait, between their download and its name and between an old
// name and a new one: a name only it uses. ext is the file's, with its dot.
func HoldingOf(id storage.UUID, ext string) string {
	return storing.HoldingDir + "/" + id.String() + ext
}

var ErrRootChanged = errors.New("the Work moved to another library root after the file was planned")

// NameTakenError is a path in its Work's folder a file would take that
// another file is already at: a file of the Work, or with Foreign one
// SlipStream doesn't track.
type NameTakenError struct {
	Path    string
	Foreign bool
}

func (e *NameTakenError) Error() string {
	if e.Foreign {
		return fmt.Sprintf("a file SlipStream doesn't track is already at %q", e.Path)
	}
	return fmt.Sprintf("another file of the Work is already at %q", e.Path)
}

// Vacant is nil when every file root's Node lists at to, compared without
// case, is one the library tracks, which the disk queue moves or removes
// before a file whose bytes sit at from takes to. A file SlipStream
// doesn't track there is a *NameTakenError. A root its Node can't read now
// is *storing.NotReady.
func (f *Files) Vacant(ctx context.Context, root roots.ID, from, to string) error {
	var planned storing.Planned
	var err error
	if planned.From, err = storing.ParseRel(from); err != nil {
		return err
	}
	if planned.To, err = storing.ParseRel(to); err != nil {
		return err
	}
	surveyed, err := f.cfg.Disk.Survey(ctx, root, []storing.Planned{planned}, false)
	if err != nil {
		return err
	}
	return f.cfg.Library.Read(ctx, func(ctx context.Context, r storage.Reader[file]) error {
		for _, o := range surveyed[0].Occupants {
			tracked, err := db.New(r).PathTaken(ctx, db.PathTakenParams{RootID: root, Path: o.Path.String()})
			if err != nil {
				return err
			}
			if !tracked {
				return &NameTakenError{Path: o.Path.String(), Foreign: true}
			}
		}
		return nil
	})
}

// Candidate is one probed file a Batch judges. Key is the caller's, such
// as an Import file's position. Items are the Items of the Batch's Work it
// holds, in any order. Holding is root-relative and '/' separated: where
// its bytes sit, which names it until it has a name. SourceName is its name
// in its download, extension included, the name it keeps with renaming
// off. Force is the cells it takes whatever it ranks, as an Admin forced
// them, when it meets their Targets.
type Candidate struct {
	Key         int
	Items       []works.ItemID
	Cut         release.CutKey
	Size        int64
	Media       media.Record
	Quality     release.Quality
	ReleaseName string
	Title       release.Read
	SourceName  string
	Holding     string
	Force       []CellKey
}

// Batch is a Work's candidates, of one Import, zero for none, judged
// together against the Work's live files and one another, as if each
// landed now in Root.
type Batch struct {
	Work       works.WorkID
	Root       roots.ID
	Import     storage.UUID
	Candidates []Candidate
}

// Holder is what holds a cell a candidate meets and doesn't fill.
type Holder uint8

const (
	HeldByLive Holder = iota + 1
	HeldBySibling
)

// Verdict is one candidate's answer. Fills are the cells it fills, in tree
// order then Profile order, and Meets the cells of its Items, in wanted
// Editions, whose Target it meets; Lost holds each cell it meets and
// doesn't fill. EditionGone is a cut no wanted Edition takes. Refusal is a
// *NameTakenError or a *naming.TooLongError of its name, which keeps it
// from filling anything. Path is its name under its root, and Replaces the
// files its landing deletes, while it fills a cell. Reasons is each
// Target's verdict on it.
type Verdict struct {
	Fills       []CellKey
	Meets       []CellKey
	Lost        map[CellKey]Holder
	EditionGone bool
	Refusal     error
	Path        string
	Replaces    []File
	Reasons     map[targets.TargetID]targets.Verdict
}

// lands reports whether v's candidate lands: it fills a cell.
func (v Verdict) lands() bool { return len(v.Fills) > 0 }

// Judge answers, in r, what each of b's candidates would fill if all
// landed now, in candidates' order: refill and names over the Work with
// every candidate a fresh file. A candidate whose name is refused is
// judged out, and the rest again without it. It writes nothing.
func (f *Files) Judge(ctx context.Context, r storage.Reader[file], b Batch) ([]Verdict, error) {
	j, err := f.judge(ctx, r, b, importing, nil)
	return j.verdicts, err
}

// Landing is a Batch of one candidate to land: Seen is what placing its
// bytes at Holding read of them, Grab the Grab that brought it, zero for
// none, and PlacedBy how they got there: link, clone, copy, transfer, move
// or in_place. A candidate whose Holding is already its name lands with no
// move.
type Landing struct {
	Batch    Batch
	Seen     storing.Seen
	Grab     storage.UUID
	PlacedBy string
}

// Landed is a landed file, named, and the cells it fills.
type Landed struct {
	File  File
	Fills []CellKey
}

// Refused is a Landing its judge refuses: it fills nothing, its Edition is
// gone or its name is refused. Import wrote nothing.
type Refused struct{ Verdict Verdict }

func (e *Refused) Error() string {
	switch {
	case e.Verdict.Refusal != nil:
		return e.Verdict.Refusal.Error()
	case e.Verdict.EditionGone:
		return "the file's cut is in no Edition the Work wants"
	}
	return "the file fills no Target the Work still wants"
}

// Import judges l again in tx and lands it: the file and its Items, its
// fills, with forced_at on each cell it takes by Force, the files it
// dooms (the one delete that never asks, ADR 0002), the renames it causes,
// the move from Holding to its name and a files_imported row with a
// snapshot of each file it replaced. The disk queue does the IO once tx
// commits; the caller pokes it. A file whose bytes already sit at its name
// is added for the Media Servers here. A Work no longer in the Batch's
// Root is ErrRootChanged, and a file its judge refuses *Refused.
func (f *Files) Import(ctx context.Context, tx *storage.Tx[file], l Landing) (Landed, error) {
	root, err := roots.OfWork(ctx, tx, l.Batch.Work)
	switch {
	case errors.Is(err, roots.ErrNoRoot) || err == nil && root.ID != l.Batch.Root:
		return Landed{}, ErrRootChanged
	case err != nil:
		return Landed{}, err
	}
	j, err := f.judge(ctx, tx, l.Batch, importing, func(g *File) { g.Seen, g.GrabID = &l.Seen, l.Grab })
	if err != nil {
		return Landed{}, err
	}
	v := j.verdicts[0]
	if !v.lands() {
		return Landed{}, &Refused{Verdict: v}
	}
	q := db.New(tx)
	if err := apply(ctx, q, j.placed); err != nil {
		return Landed{}, err
	}
	g := j.fresh[0]
	nm := j.names[g.ID]
	inPlace := g.Path == nm.Path
	g.Path, g.Label, g.Extra = nm.Path, nm.Label, nm.Extra
	if err := imported(ctx, q, l, g, v); err != nil {
		return Landed{}, err
	}
	if inPlace {
		if err := f.tell(ctx, tx, []fact{{file: g.ID, root: g.RootID, to: g.Path}}); err != nil {
			return Landed{}, err
		}
	}
	return Landed{File: g, Fills: v.Fills}, nil
}

// importedCell and replacedFile are a files_imported row's JSON.
type importedCell struct {
	Item    works.ItemID     `json:"item"`
	Edition targets.Edition  `json:"edition"`
	Target  targets.TargetID `json:"target"`
}

type replacedFile struct {
	Quality release.Quality `json:"quality"`
	Label   string          `json:"label"`
	Items   []works.ItemID  `json:"items"`
	Path    string          `json:"path"`
}

// imported records l's landing of g, which v judged, and that g replaced
// each doomed file of v.Replaces.
func imported(ctx context.Context, q *db.Queries, l Landing, g File, v Verdict) error {
	cells := make([]importedCell, len(v.Fills))
	for i, k := range v.Fills {
		cells[i] = importedCell(k)
	}
	replaced := make([]replacedFile, len(v.Replaces))
	for i, d := range v.Replaces {
		replaced[i] = replacedFile{Quality: d.Quality, Label: d.Label, Items: d.Items, Path: d.Path}
		if err := q.SetReplacedBy(ctx, db.SetReplacedByParams{FileID: d.ID, ReplacedBy: g.ID}); err != nil {
			return err
		}
	}
	var cols [3][]byte
	for i, v := range []any{g.Items, cells, replaced} {
		var err error
		if cols[i], err = json.Marshal(v); err != nil {
			return fmt.Errorf("file %s: encode its import record: %w", g.ID, err)
		}
	}
	return q.InsertImported(ctx, db.InsertImportedParams{
		ID: storage.NewUUID(), FileID: g.ID, ImportID: l.Batch.Import, GrabID: l.Grab, WorkID: l.Batch.Work, PlacedBy: l.PlacedBy,
		Items: string(cols[0]), Cells: string(cols[1]), Replaced: string(cols[2]), At: storage.Now(),
	})
}

// judged is a Batch judged: its Work converged with every candidate that
// lands, each candidate as a fresh file, in candidates' order, and their
// verdicts.
type judged struct {
	placed
	fresh    []File
	verdicts []Verdict
}

// judge judges b in r under how. landing, when set, completes each fresh
// file with what only a landing knows.
func (f *Files) judge(ctx context.Context, r storage.Reader[file], b Batch, how rule, landing func(*File)) (judged, error) {
	before, err := f.load(ctx, r, b.Work)
	if err != nil {
		return judged{}, err
	}
	if before.imports, err = importsOf(ctx, db.New(r), before.fills); err != nil {
		return judged{}, err
	}
	j := judged{fresh: make([]File, len(b.Candidates))}
	for i, c := range b.Candidates {
		items, err := inTreeOrder(before.tree, c.Items)
		if err != nil {
			return judged{}, err
		}
		j.fresh[i] = File{
			ID: ID{UUID: storage.NewUUID()}, WorkID: b.Work, Items: items, Cut: c.Cut, RootID: b.Root, Path: c.Holding, Size: c.Size, Media: c.Media,
			Quality: c.Quality, ReleaseName: c.ReleaseName, Title: c.Title, SourceName: c.SourceName,
		}
		if landing != nil {
			landing(&j.fresh[i])
		}
	}
	refusals := map[ID]error{}
	for {
		after := arriving(before, b, j.fresh, refusals)
		j.placed, err = f.converge(ctx, r, before, after, refill(after), how)
		var out *refusal
		if !errors.As(err, &out) {
			break
		}
		refusals[out.file] = out.err
	}
	if err != nil {
		return judged{}, err
	}
	j.verdicts = make([]Verdict, len(j.fresh))
	for i, g := range j.fresh {
		j.verdicts[i] = j.verdictOf(g, refusals[g.ID])
	}
	return j, nil
}

// refusal is a fresh file's name that names refuses: a *NameTakenError or
// a *naming.TooLongError.
type refusal struct {
	file ID
	err  error
}

func (e *refusal) Error() string { return e.err.Error() }
func (e *refusal) Unwrap() error { return e.err }

// arriving is before with b's fresh files arriving, those refusals names
// left out.
func arriving(before world, b Batch, fresh []File, refusals map[ID]error) world {
	after := before
	after.files, after.beside = slices.Clone(before.files), before.files
	after.seen = maps.Clone(before.seen)
	after.fresh, after.force, after.importing, after.forcedAt = map[ID]bool{}, map[CellKey]ID{}, b.Import, storage.Now()
	for i, g := range fresh {
		if _, out := refusals[g.ID]; out {
			continue
		}
		after.files = append(after.files, g)
		after.fresh[g.ID] = true
		after.see(g.Items, g.Cut)
		for _, key := range b.Candidates[i].Force {
			if _, taken := after.force[key]; !taken {
				after.force[key] = g.ID
			}
		}
	}
	return after
}

// verdictOf is g's verdict in j, with refused the refusal of its name.
func (j judged) verdictOf(g File, refused error) Verdict {
	w := j.after
	v := Verdict{Lost: map[CellKey]Holder{}, EditionGone: true, Refusal: refused, Reasons: make(map[targets.TargetID]targets.Verdict, len(w.goals.Targets))}
	scored := w.scoring.of(g)
	var met []targets.TargetID
	for _, t := range w.goals.Targets {
		v.Reasons[t.ID] = targets.Judge(t, scored, nil)
		if targets.Judge(t, targets.Scored{Facts: w.scoring.facts(g)}, nil).Meets() {
			met = append(met, t.ID)
		}
	}
	for _, it := range g.Items {
		for _, e := range targets.Wanted(w.goals.Editions, seenWith(w.seen[it], g.Cut)) {
			if !e.Takes(g.Cut) {
				continue
			}
			v.EditionGone = false
			for _, t := range met {
				j.judgeCell(&v, g, CellKey{Item: it, Edition: e, Target: t})
			}
		}
	}
	if v.lands() {
		v.Path = j.names[g.ID].Path
		v.Replaces = j.replaced(v.Fills)
	}
	return v
}

// judgeCell adds key, a cell g meets, to v: to its Fills when g fills it,
// else to Lost with what holds it.
func (j judged) judgeCell(v *Verdict, g File, key CellKey) {
	v.Meets = append(v.Meets, key)
	switch by := j.fills[key].File; {
	case by == g.ID:
		v.Fills = append(v.Fills, key)
	case j.after.fresh[by]:
		v.Lost[key] = HeldBySibling
	case !by.IsZero():
		v.Lost[key] = HeldByLive
	}
}

// replaced is the doomed files that filled one of cells before.
func (j judged) replaced(cells []CellKey) []File {
	var out []File
	for _, d := range j.doomed {
		if slices.ContainsFunc(cells, func(k CellKey) bool { return j.before.fills[k].File == d.ID }) {
			out = append(out, d)
		}
	}
	return out
}

// inTreeOrder is items in tree's order, an error when tree doesn't hold
// one or there is none.
func inTreeOrder(tree works.Bare, items []works.ItemID) ([]works.ItemID, error) {
	if len(items) == 0 {
		return nil, fmt.Errorf("a file of work %s holds no item", tree.ID)
	}
	var out []works.ItemID
	for _, it := range tree.Items {
		if slices.Contains(items, it.ID) {
			out = append(out, it.ID)
		}
	}
	for _, it := range items {
		if !slices.Contains(out, it) {
			return nil, fmt.Errorf("a file names item %s, which work %s doesn't hold", it, tree.ID)
		}
	}
	return out, nil
}

// importsOf is the Import that landed each file holding a forced fill.
func importsOf(ctx context.Context, q *db.Queries, fills map[CellKey]Fill) (map[ID]storage.UUID, error) {
	var ids []ID
	for _, fill := range fills {
		if !fill.Forced.IsZero() && !slices.Contains(ids, fill.File) {
			ids = append(ids, fill.File)
		}
	}
	if len(ids) == 0 {
		return nil, nil
	}
	rows, err := q.ListImportsOfFiles(ctx, ids)
	if err != nil {
		return nil, err
	}
	out := make(map[ID]storage.UUID, len(rows))
	for _, row := range rows {
		out[row.FileID] = row.ImportID
	}
	return out, nil
}

// placed is a Work converged: the world with its new files and without
// the files it dooms or forgets, and the fills and names refill and names
// give that world.
type placed struct {
	before world
	after  world
	fills  map[CellKey]Fill
	doomed []File
	// forgot are gone files whose rows go: no bytes are left to remove.
	forgot []File
	names  map[ID]name
}

// rule is how a converge treats the files a refill leaves behind.
type rule struct {
	// doom says whether a file that lost its last fill is deleted; false
	// keeps it as a Kept file.
	doom func(File) bool
	// keepFresh lands a fresh file filling nothing as a Kept file; else it
	// lands nowhere.
	keepFresh bool
	// mode is which files take a path from the templates: a library scan
	// never renames.
	mode renaming
	// forgetGone forgets each gone file that would win back no cell if it
	// returned.
	forgetGone bool
	// draft names its type's files in place of the stored settings.
	draft *drafted
}

// importing is an import's rule: it dooms the files the fresh files
// displace.
var importing = rule{doom: func(File) bool { return true }, forgetGone: true}

// changing is a Targets change's rule: the files it leaves filling
// nothing go as m chose.
func changing(m Mode) rule { return rule{doom: func(File) bool { return m.Unwanted == Delete }} }

// scanned is a library scan's rule: it never dooms a file and never
// renames one.
var scanned = rule{doom: func(File) bool { return false }, mode: keepEvery}

// converge is a Work going from before to after, whose files are before's
// and any fresh or found ones, refilled as fills: a file that filled a
// cell before and fills none after goes when the rule dooms it and is
// named Kept otherwise, and a fresh file filling none lands nowhere unless
// the rule keeps it. A gone file is forgotten when the rule forgets those
// that would win nothing back, and when a live file takes its path, where
// its bytes could no longer return. Every file is named under fills and
// after's cut names.
func (f *Files) converge(ctx context.Context, r storage.Reader[file], before, after world, fills map[CellKey]Fill, how rule) (placed, error) {
	p := placed{before: before, after: after, fills: fills}
	p.after.retagged = retagged(before.cuts, after.cuts)
	p.after.mode = how.mode
	was, is := filling(before.fills), filling(fills)
	p.after.files = slices.DeleteFunc(slices.Clone(after.files), func(g File) bool {
		switch {
		case is[g.ID]:
			return false
		case after.fresh[g.ID]:
			return !how.keepFresh
		case !was[g.ID] || !how.doom(g):
			return false
		}
		p.doomed = append(p.doomed, g)
		return true
	})
	if how.forgetGone {
		forget(&p, func(g File) bool { return !winsBack(p.after, fills, g) })
	}
	if err := f.named(ctx, r, &p.after, p.fills, how.draft); err != nil {
		return placed{}, err
	}
	var err error
	if p.names, _, err = names(p.after, p.fills); err != nil {
		return placed{}, err
	}
	taken := map[place]bool{}
	for _, g := range p.after.files {
		if !g.Gone() {
			taken[place{g.RootID, p.names[g.ID].Path}] = true
		}
	}
	forget(&p, func(g File) bool { return taken[place{g.RootID, g.Path}] })
	return p, nil
}

// forget takes the gone files which picks out of p's world: their rows
// go when p is applied.
func forget(p *placed, which func(File) bool) {
	p.after.files = slices.DeleteFunc(p.after.files, func(g File) bool {
		if !g.Gone() || !which(g) {
			return false
		}
		p.forgot = append(p.forgot, g)
		return true
	})
}

// winsBack reports whether g, a gone file of w, would fill a cell were it
// back, with fills' files holding their cells on a tie.
func winsBack(w world, fills map[CellKey]Fill, g File) bool {
	w.files = slices.Clone(w.files)
	for i := range w.files {
		if w.files[i].ID == g.ID {
			w.files[i].GoneAt = storage.Time{}
		}
	}
	w.fills = fills
	return filling(refill(w))[g.ID]
}

// retagged is each cut of is whose name differs in was.
func retagged(was, is targets.Cuts) map[release.CutKey]bool {
	out := map[release.CutKey]bool{}
	for _, c := range is.All() {
		if old, ok := was.Get(c.Key); ok && old.Name != c.Name {
			out[c.Key] = true
		}
	}
	return out
}

// apply writes p: its fills and their scores, its dooms, the gone files
// it forgets, its found files, every name that changed, and a move for
// each file whose bytes must reach a new path. A row's path changes in two
// steps, through the file's holding path, so files may trade names in one
// Write; the disk queue moves the bytes straight unless they trade paths.
func apply(ctx context.Context, q *db.Queries, p placed) error {
	if err := unfill(ctx, q, p); err != nil {
		return err
	}
	if err := drop(ctx, q, p); err != nil {
		return err
	}
	var moved []File
	for _, g := range p.after.files {
		moves, err := rename(ctx, q, p, g)
		if err != nil {
			return err
		}
		if moves {
			moved = append(moved, g)
		}
	}
	for _, g := range moved {
		nm := p.names[g.ID]
		if err := q.RenameFile(ctx, db.RenameFileParams{ID: g.ID, Path: nm.Path, Label: nm.Label, Extra: nm.Extra}); err != nil {
			return err
		}
	}
	for key, fill := range p.fills {
		if p.before.fills[key] != fill {
			err := q.InsertFill(ctx, db.InsertFillParams{
				FileID: fill.File, TargetID: key.Target, ItemID: key.Item, Edition: key.Edition, Score: int64(fill.Score), ForcedAt: fill.Forced,
			})
			if err != nil {
				return err
			}
		}
	}
	return nil
}

// drop dooms p's doomed files and deletes the rows of the gone files it
// forgets.
func drop(ctx context.Context, q *db.Queries, p placed) error {
	for _, d := range p.doomed {
		if err := doom(ctx, q, d); err != nil {
			return err
		}
	}
	for _, g := range p.forgot {
		if err := q.DeleteFile(ctx, g.ID); err != nil {
			return err
		}
	}
	return nil
}

// unfill deletes every fill p changes or rescores, before any file it
// held goes.
func unfill(ctx context.Context, q *db.Queries, p placed) error {
	for key, fill := range p.before.fills {
		if p.fills[key] != fill {
			if err := q.DeleteFill(ctx, db.DeleteFillParams{ItemID: key.Item, Edition: key.Edition, TargetID: key.Target}); err != nil {
				return err
			}
		}
	}
	return nil
}

// rename writes g's new label and N, and reports whether its path changes:
// then g's row holds its holding path until apply's last pass names it,
// and a move takes its bytes there.
func rename(ctx context.Context, q *db.Queries, p placed, g File) (bool, error) {
	nm := p.names[g.ID]
	switch {
	case p.after.fresh[g.ID] && nm.Path == g.Path:
		g.Label, g.Extra = nm.Label, nm.Extra
		return false, insertFile(ctx, q, g)
	case p.after.fresh[g.ID]:
		sits := g.Path
		g.Path, g.Label, g.Extra = holding(g), nm.Label, nm.Extra
		if err := insertFile(ctx, q, g); err != nil {
			return false, err
		}
		return true, q.InsertMove(ctx, db.InsertMoveParams{FileID: g.ID, Sits: sits})
	case p.after.found[g.ID]:
		g.Label, g.Extra = nm.Label, nm.Extra
		return false, insertFile(ctx, q, g)
	case nm.Path != g.Path:
		if err := q.RenameFile(ctx, db.RenameFileParams{ID: g.ID, Path: holding(g), Label: nm.Label, Extra: nm.Extra}); err != nil {
			return false, err
		}
		return true, queueMove(ctx, q, g, nm.Path)
	case nm.Label != g.Label || nm.Extra != g.Extra:
		return false, q.RenameFile(ctx, db.RenameFileParams{ID: g.ID, Path: g.Path, Label: nm.Label, Extra: nm.Extra})
	}
	return false, nil
}

// holding is where g's bytes wait between two names.
func holding(g File) string { return HoldingOf(g.ID.UUID, path.Ext(g.Path)) }

// queueMove records that g's bytes, at its old path unless a move already
// took them elsewhere, must reach to. Bytes already at to need no move.
func queueMove(ctx context.Context, q *db.Queries, g File, to string) error {
	at, err := q.GetMove(ctx, g.ID)
	switch {
	case errors.Is(err, storage.ErrNoRows):
		return q.InsertMove(ctx, db.InsertMoveParams{FileID: g.ID, Sits: g.Path})
	case err != nil:
		return err
	case at.Sits == to:
		_, err := q.DeleteMove(ctx, db.DeleteMoveParams{FileID: g.ID, Sits: at.Sits})
		return err
	}
	return nil
}

// doom removes d from the library and queues its bytes' removal from
// wherever they sit: its path, or where its move had taken them. A
// doomed move keeps its origin, where a Media Server still lists d.
func doom(ctx context.Context, q *db.Queries, d File) error {
	var moving, origin *string
	at, err := q.GetMove(ctx, d.ID)
	switch {
	case err == nil:
		moving, origin = &at.Sits, &at.Origin
	case !errors.Is(err, storage.ErrNoRows):
		return err
	}
	if err := q.InsertDoomed(ctx, db.InsertDoomedParams{FileID: d.ID, RootID: d.RootID, Path: d.Path, Sits: moving, Origin: origin}); err != nil {
		return err
	}
	return q.DeleteFile(ctx, d.ID)
}
