package imports

import (
	"context"
	"fmt"
	"path"
	"slices"
	"strconv"

	"github.com/jatassi/slipstream/internal/files"
	"github.com/jatassi/slipstream/internal/release"
	"github.com/jatassi/slipstream/internal/roots"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/storing"
	"github.com/jatassi/slipstream/internal/works"
)

// after is what to publish once a Write commits.
type after struct {
	roots  []roots.ID
	landed []works.WorkID
}

// publish tells the readers what a committed Write changed.
func (im *Imports) publish(ctx context.Context, a after) {
	ctx = context.WithoutCancel(ctx)
	if len(a.roots) > 0 {
		im.cfg.Changed.Publish(ctx, changedOf(slices.Compact(a.roots)...))
	}
	for _, w := range slices.Compact(a.landed) {
		im.cfg.FilesChanged.Publish(ctx, files.Changed{WorkID: w.String()})
	}
	if len(a.landed) > 0 {
		im.cfg.Files.Poke()
	}
}

// write is imports' one Write over one Import: writeIn, then it publishes
// what changed once the Write commits. ErrNoImport when the Import went.
func (im *Imports) write(ctx context.Context, id ImportID, change func(context.Context, *storage.Tx[file], *Import, *after) error) (Import, error) {
	var (
		out Import
		a   after
	)
	err := im.cfg.Library.Write(ctx, func(ctx context.Context, tx *storage.Tx[file]) error {
		var err error
		a = after{}
		out, err = im.writeIn(ctx, tx, id, &a, change)
		return err
	})
	if err != nil {
		return Import{}, err
	}
	im.publish(ctx, a)
	return out, nil
}

// writeIn loads id in tx, runs change, converges and saves, noting in a
// what to publish.
func (im *Imports) writeIn(ctx context.Context, tx *storage.Tx[file], id ImportID, a *after, change func(context.Context, *storage.Tx[file], *Import, *after) error) (Import, error) {
	imp, err := importOf(ctx, tx, id)
	if err != nil {
		return Import{}, err
	}
	if err := change(ctx, tx, &imp, a); err != nil {
		return Import{}, err
	}
	if err := im.converge(ctx, tx, &imp); err != nil {
		return Import{}, err
	}
	if !imp.Root.IsZero() {
		a.roots = append(a.roots, imp.Root)
	}
	err = save(ctx, tx, &imp)
	return imp, err
}

// converge brings everything derived from imp's rows into line in tx: a
// busy file back to pending once its Items are free, the decision once the
// barrier is met, a download's release faults and finished_at.
// Running it twice changes nothing.
func (im *Imports) converge(ctx context.Context, tx *storage.Tx[file], imp *Import) error {
	if err := unbusy(ctx, tx, imp); err != nil {
		return err
	}
	if imp.barrierMet() {
		if err := im.decide(ctx, tx, imp); err != nil {
			return err
		}
	}
	if imp.Source == FromGrab {
		rules, err := RulesIn(ctx, tx)
		if err != nil {
			return err
		}
		faults(imp, rules)
	}
	finishIfFinal(imp, storage.Now())
	return nil
}

// barrierMet reports whether the decision may run: the Import has a Work
// and no fault, no pending file lacks its probe, and one pending file has
// one. A pending file whose probe hold the Admin must answer leaves the
// barrier, so the rest decide without it.
func (im Import) barrierMet() bool {
	if im.Work.IsZero() || im.Failure != nil {
		return false
	}
	probed := false
	for _, f := range im.Files {
		switch {
		case f.Outcome != Pending || f.Hold != nil && f.Hold.offered():
		case f.Probe == nil:
			return false
		default:
			probed = true
		}
	}
	return probed
}

// toDecide is the positions of the pending files the decision takes.
func (im Import) toDecide() []int {
	var out []int
	for i, f := range im.Files {
		if f.Outcome == Pending && f.Probe != nil && (f.Hold == nil || !f.Hold.offered()) {
			out = append(out, i)
		}
	}
	return out
}

// othersFill reports whether a file of im outside batch, positions in
// Files, is sending, placing or placed: the pack rule's "other files fill
// something".
func (im Import) othersFill(batch []int) bool {
	for i, f := range im.Files {
		if !slices.Contains(batch, i) && (f.Outcome == Sending || f.Outcome == Placing || f.Outcome == Placed) {
			return true
		}
	}
	return false
}

// finishIfFinal finishes im once a fault failed it or each file is final
// and each moved file's source is gone.
// A root Import left with no files has no history to keep, and save
// deletes it.
func finishIfFinal(im *Import, now storage.Time) {
	if !im.Finished.IsZero() || len(im.Files) == 0 {
		return
	}
	if im.Failure != nil || !slices.ContainsFunc(im.Files, func(f File) bool { return !f.Outcome.Final() || f.sourceDue() }) {
		im.Finished = now
	}
}

// fileAt is the file of im at position.
func fileAt(im *Import, position int) *File {
	i := slices.IndexFunc(im.Files, func(f File) bool { return f.Position == position })
	if i < 0 {
		panic(fmt.Sprintf("imports: import %s has no file %d", im.ID, position))
	}
	return &im.Files[i]
}

// holdingOf is where f's bytes wait in its destination root until they
// land: a name only it uses.
func holdingOf(id ImportID, f File) (storing.Rel, error) {
	return storing.ParseRel(storing.HoldingDir + "/" + id.String() + "-" + strconv.Itoa(f.Position) + path.Ext(f.Path))
}

// destOf is where f's bytes go in root: for a root Import the path they
// already sit at.
func (im Import) destOf(root roots.ID, f File) (*Dest, error) {
	if im.Source == FromRoot {
		rel, err := storing.ParseRel(f.Path)
		return &Dest{Root: root, Holding: rel}, err
	}
	rel, err := holdingOf(im.ID, f)
	return &Dest{Root: root, Holding: rel}, err
}

// take moves f to o, keeping its Dest from the decision on.
func (f *File) take(o outcome, dest *Dest) {
	f.Outcome, f.Reason, f.Fills, f.Hold = o.To, o.Reason, o.Fills, nil
	if o.To == Sending || o.To == Placing {
		f.Dest = dest
	}
}

// decide is the one decision: every probed pending file of imp judged at
// once against the Work's live files and one another, in tx.
func (im *Imports) decide(ctx context.Context, tx *storage.Tx[file], imp *Import) error {
	batch := imp.toDecide()
	busy, err := busyOf(ctx, tx, *imp, batch)
	if err != nil {
		return err
	}
	var judged []int
	for _, i := range batch {
		f := &imp.Files[i]
		if held := busy(f.Items); len(held) > 0 {
			f.take(outcome{To: Waiting, Reason: &Reason{Code: Busy, Reading: held}}, nil)
			continue
		}
		judged = append(judged, i)
	}
	if len(judged) == 0 {
		return nil
	}
	root, err := imp.rootOf(ctx, tx)
	if err != nil {
		return err
	}
	cands, err := im.candidates(ctx, tx, imp, root.ID, judged)
	if err != nil {
		return err
	}
	vs, err := im.cfg.Files.Judge(ctx, tx, files.Batch{Work: imp.Work, Root: root.ID, Import: imp.ID.UUID, Candidates: cands})
	if err != nil {
		return err
	}
	c := imp.outcomeContext(judged, imp.Source != FromRoot && imp.Node != root.Node)
	for k, o := range outcomes(vs, c) {
		f := &imp.Files[judged[k]]
		dest, err := imp.destOf(root.ID, *f)
		if err != nil {
			return err
		}
		f.take(o, dest)
	}
	return nil
}

// rootOf is the root imp's files go into: a root Import's own, else its
// Work's.
func (im Import) rootOf(ctx context.Context, r storage.Reader[file]) (roots.Root, error) {
	if im.Source == FromRoot {
		return roots.Get(ctx, r, im.Root)
	}
	return roots.OfWork(ctx, r, im.Work)
}

func (im Import) outcomeContext(batch []int, sends bool) outcomeContext {
	return outcomeContext{
		othersFill: im.othersFill(batch),
		source:     im.Source,
		sends:      sends,
		cut:        func(k int) release.CutKey { return im.Files[batch[k]].Cut },
		confirmed:  func(k int) bool { return im.Files[batch[k]].Confirmed },
		forced:     func(k int) []files.CellKey { return im.Files[batch[k]].forced() },
	}
}

// forced is the cells f takes by Force.
func (f *File) forced() []files.CellKey {
	var out []files.CellKey
	for _, fl := range f.Fills {
		if fl.Forced {
			out = append(out, fl.Cell)
		}
	}
	return out
}

// candidates is the files of imp at positions batch as files.Judge takes
// them, read by the Work's Content Type.
func (im *Imports) candidates(ctx context.Context, r storage.Reader[file], imp *Import, root roots.ID, batch []int) ([]files.Candidate, error) {
	cuts, err := im.cfg.Targets.CutsIn(ctx, r)
	if err != nil {
		return nil, err
	}
	facet := im.cfg.Facets[imp.ContentType]
	out := make([]files.Candidate, len(batch))
	for k, i := range batch {
		f := imp.Files[i]
		dest, err := imp.destOf(root, f)
		if err != nil {
			return nil, err
		}
		out[k] = files.Candidate{
			Key: f.Position, Items: f.Items, Cut: f.Cut, Size: f.Size, Media: *f.Probe, Quality: f.Quality,
			ReleaseName: f.ReleaseName, Title: release.File(f.Path, facet, cuts.For(imp.ContentType)), SourceName: f.SourceName,
			Holding: dest.Holding.String(), Force: f.forced(),
		}
	}
	return out, nil
}

// busyOf answers, for a file's Items, the ones another Import holds: a
// file of it sending or placing, or pending in an Import older than imp.
// A file waiting for the Admin holds nothing, and a Work-less Import
// never does.
func busyOf(ctx context.Context, r storage.Reader[file], imp Import, batch []int) (func([]works.ItemID) []works.ItemID, error) {
	var items []works.ItemID
	for _, i := range batch {
		items = append(items, imp.Files[i].Items...)
	}
	held, err := holders(ctx, r, items, imp)
	if err != nil {
		return nil, err
	}
	return func(its []works.ItemID) []works.ItemID {
		var out []works.ItemID
		for _, it := range its {
			if _, ok := held[it]; ok {
				out = append(out, it)
			}
		}
		return out
	}, nil
}

// unbusy makes each of imp's busy files whose Items no other Import holds
// pending again, for the decision to take.
func unbusy(ctx context.Context, r storage.Reader[file], imp *Import) error {
	var items []works.ItemID
	for _, f := range imp.Files {
		if f.busy() {
			items = append(items, f.Reason.Reading...)
		}
	}
	if len(items) == 0 {
		return nil
	}
	held, err := holders(ctx, r, items, *imp)
	if err != nil {
		return err
	}
	for i, f := range imp.Files {
		if f.busy() && !slices.ContainsFunc(f.Reason.Reading, func(it works.ItemID) bool { _, ok := held[it]; return ok }) {
			imp.Files[i].Outcome, imp.Files[i].Reason = Pending, nil
		}
	}
	return nil
}
