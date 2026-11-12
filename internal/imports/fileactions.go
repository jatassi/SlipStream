package imports

import (
	"context"
	"errors"
	"path"
	"slices"

	"github.com/jatassi/slipstream/internal/files"
	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/imports/internal/db"
	"github.com/jatassi/slipstream/internal/release"
	"github.com/jatassi/slipstream/internal/scanning"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/works"
)

// FileRef names one file of an Import.
type FileRef struct {
	Import   ImportID
	Position int
}

// Action is what an Admin does with one file: ConfirmFile, RetryFile,
// DeclineFile, ForceFile or MatchFile.
type Action interface{ Kind() ActionKind }

type (
	ConfirmFile struct{}
	RetryFile   struct{}
	DeclineFile struct{}
	// ForceFile puts the file into Cell even ranked lower, once it meets the
	// cell's Target.
	ForceFile struct{ Cell files.CellKey }
	// MatchFile is what the file is: Items of Work in any order, the cut
	// Edition names or, when nil, the one its release name carries.
	MatchFile struct {
		Work        works.WorkID
		Items       []works.ItemID
		Edition     *release.CutKey
		ReleaseName string
	}
)

func (ConfirmFile) Kind() ActionKind { return ActConfirm }
func (RetryFile) Kind() ActionKind   { return ActRetry }
func (DeclineFile) Kind() ActionKind { return ActDecline }
func (ForceFile) Kind() ActionKind   { return ActForce }
func (MatchFile) Kind() ActionKind   { return ActMatch }

// Act does a on the file ref names, in one Write, and answers the Import
// the file is in after it: a MatchFile to another Work or Edition moves the
// file into the folder's Import of that Work and Edition. ErrNoImport when
// no Import has the file, ErrNotOffered when its row doesn't offer a, and
// ErrNotOfWork when a Match names an Item its Work doesn't hold.
func (im *Imports) Act(ctx context.Context, _ identity.Admin, ref FileRef, a Action) (Import, error) {
	var (
		out  Import
		done after
	)
	err := im.cfg.Library.Write(ctx, func(ctx context.Context, tx *storage.Tx[file]) error {
		done = after{}
		var moved *moving
		cur, err := im.writeIn(ctx, tx, ref.Import, &done, func(ctx context.Context, tx *storage.Tx[file], cur *Import, _ *after) error {
			i := slices.IndexFunc(cur.Files, func(f File) bool { return f.Position == ref.Position })
			if i < 0 {
				return ErrNoImport
			}
			f := &cur.Files[i]
			recommended, also := f.Actions(*cur)
			if a.Kind() != recommended && !slices.Contains(also, a.Kind()) {
				return ErrNotOffered
			}
			switch a := a.(type) {
			case ConfirmFile:
				f.Confirmed = true
				f.reopen()
			case RetryFile:
				if f.Outcome == FailedOut && f.Reason.Code == Unreadable {
					f.Probe = nil
				}
				f.reopen()
			case DeclineFile:
				f.Outcome, f.Reason, f.Hold, f.Fills = Declined, &Reason{Code: ByAdmin}, nil, nil
			case ForceFile:
				f.reopen()
				f.Fills = append(f.Fills, Fill{Cell: a.Cell, Forced: true})
			case MatchFile:
				var err error
				moved, err = im.match(ctx, tx, cur, i, a)
				return err
			}
			return nil
		})
		if err != nil || moved == nil {
			out = cur
			return err
		}
		out, err = im.moveInto(ctx, tx, cur, *moved, &done)
		return err
	})
	if err != nil {
		return Import{}, err
	}
	im.publish(ctx, done)
	im.cfg.Changed.Publish(ctx, changedOf())
	im.Touch()
	return out, nil
}

// reopen makes f pending again for converge to decide, past its confirm
// once confirmed, keeping its probe and the cells it is forced into.
func (f *File) reopen() {
	f.Outcome, f.Reason, f.Hold, f.Dest, f.Sent, f.Fills = Pending, nil, nil, nil, nil, fillsOf(f.forced(), f.forced())
}

// moving is a file a Match takes out of its Import, into the folder's
// Import of Work and Edition.
type moving struct {
	file        File
	work        works.WorkID
	edition     *release.CutKey
	contentType works.TypeID
}

// match reads cur's file i as m says, pending for the decision. When m
// names another Work or Edition the file leaves cur, and match answers it.
func (im *Imports) match(ctx context.Context, tx *storage.Tx[file], cur *Import, i int, m MatchFile) (*moving, error) {
	tree, err := im.cfg.Works.TreeIn(ctx, tx, m.Work)
	if errors.Is(err, works.ErrNoWork) {
		return nil, ErrNotOfWork
	}
	if err != nil {
		return nil, err
	}
	order := map[works.ItemID]int{}
	for k, it := range tree.Items {
		order[it.ID] = k
	}
	for _, it := range m.Items {
		if _, ok := order[it]; !ok {
			return nil, ErrNotOfWork
		}
	}
	f := &cur.Files[i]
	f.Items = slices.SortedFunc(slices.Values(m.Items), func(a, b works.ItemID) int { return order[a] - order[b] })
	f.ReleaseName, f.Confirmed, f.Fills = m.ReleaseName, false, nil
	f.reopen()
	if f.Probe != nil {
		if err := im.readName(ctx, tx, tree.ContentType, m.Edition, f); err != nil {
			return nil, err
		}
	}
	if m.Work == cur.Work && editionsEqual(m.Edition, cur.Edition) {
		return nil, nil
	}
	moved := &moving{file: *f, work: m.Work, edition: m.Edition, contentType: tree.ContentType}
	cur.Files = slices.Delete(cur.Files, i, i+1)
	return moved, nil
}

func editionsEqual(a, b *release.CutKey) bool {
	return a == nil && b == nil || a != nil && b != nil && *a == *b
}

// readName reads f's quality and cut from its release name and probe, as
// t's names read them, the cut being edition when the Admin named one.
func (im *Imports) readName(ctx context.Context, r storage.Reader[file], t works.TypeID, edition *release.CutKey, f *File) error {
	cuts, err := im.cfg.Targets.CutsIn(ctx, r)
	if err != nil {
		return err
	}
	read, quality := scanning.QualityOf(f.ReleaseName+path.Ext(f.Path), im.cfg.Facets[t], cuts.For(t), f.Probe.Resolution())
	f.Quality, f.Cut = quality, read.Facts.Cut
	if edition != nil {
		f.Cut = *edition
	}
	return nil
}

// moveInto adds m's file to the unfinished folder Import of from's folder
// for its Work and Edition, created when none is open, and answers that
// Import.
func (im *Imports) moveInto(ctx context.Context, tx *storage.Tx[file], from Import, m moving, a *after) (Import, error) {
	row, err := db.New(tx).GetOpenFolderImport(ctx, db.GetOpenFolderImportParams{NodeID: from.Node, Folder: &from.Folder, WorkID: m.work, Edition: m.edition})
	id := row.ID
	switch {
	case errors.Is(err, storage.ErrNoRows):
		id = ImportID{UUID: storage.NewUUID()}
		err = insertImport(ctx, tx, Import{
			ID: id, Source: FromFolder, Node: from.Node, Folder: from.Folder, ContentType: m.contentType, Hint: from.Hint, Work: m.work,
			Edition: m.edition, MoveFiles: from.MoveFiles, Created: storage.Now(),
		})
		if err != nil {
			return Import{}, err
		}
	case err != nil:
		return Import{}, err
	}
	return im.writeIn(ctx, tx, id, a, func(_ context.Context, _ *storage.Tx[file], to *Import, _ *after) error {
		f := m.file
		f.Position = 0
		for _, g := range to.Files {
			f.Position = max(f.Position, g.Position+1)
		}
		to.Files = append(to.Files, f)
		return nil
	})
}

// ConfirmAll confirms every file of ids that waits confirm, in one Write:
// Import all across a folder's Imports. It answers the Imports.
// ErrNoImport when an id names none.
func (im *Imports) ConfirmAll(ctx context.Context, _ identity.Admin, ids []ImportID) ([]Import, error) {
	var (
		out  []Import
		done after
	)
	err := im.cfg.Library.Write(ctx, func(ctx context.Context, tx *storage.Tx[file]) error {
		out, done = nil, after{}
		for _, id := range ids {
			imp, err := im.writeIn(ctx, tx, id, &done, func(_ context.Context, _ *storage.Tx[file], cur *Import, _ *after) error {
				for i, f := range cur.Files {
					if f.Outcome == Waiting && f.Reason.Code == Confirm {
						cur.Files[i].Confirmed = true
						cur.Files[i].reopen()
					}
				}
				return nil
			})
			if err != nil {
				return err
			}
			out = append(out, imp)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	im.publish(ctx, done)
	im.cfg.Changed.Publish(ctx, changedOf())
	im.Touch()
	return out, nil
}
