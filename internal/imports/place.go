package imports

import (
	"context"
	"errors"

	"github.com/jatassi/slipstream/internal/files"
	"github.com/jatassi/slipstream/internal/health"
	"github.com/jatassi/slipstream/internal/imports/internal/db"
	"github.com/jatassi/slipstream/internal/roots"
	"github.com/jatassi/slipstream/internal/rpc"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/storing"
	"github.com/jatassi/slipstream/internal/transfers"
)

// Step is a point in a file's IO where tests hold: before its probe,
// before its place and before its landing Write.
type Step struct {
	Import   ImportID
	Position int
	At       StepAt
}

type StepAt string

const (
	AtProbe StepAt = "probe"
	AtPlace StepAt = "place"
	AtLand  StepAt = "land"
)

func (im *Imports) hold(ctx context.Context, imp Import, f File, at StepAt) {
	if im.cfg.BeforeImport != nil {
		im.cfg.BeforeImport(ctx, Step{Import: imp.ID, Position: f.Position, At: at})
	}
}

// sourceOf is where f's bytes are on imp's Node.
func (f *File) sourceOf(imp Import) storing.Source {
	if imp.Source == FromGrab {
		return storing.Payload{InfoHash: *imp.InfoHash, FileIndex: *f.SourceIndex}
	}
	return storing.NodePath{Abs: f.Path}
}

// place puts f's bytes at its Dest on the root's Node and lands it. A
// root Import's file already sits in its root and lands where it is, and
// a file sent from another Node moves there from its staged copy, once
// its Transfer counts the copy verified. A link or a clone needs no room.
// A copy asks for it first, in a Write, so the grant is a row the next
// file's check reads.
func (im *Imports) place(ctx context.Context, imp Import, f File) error {
	im.hold(ctx, imp, f, AtPlace)
	if f.Sent != nil {
		var again bool
		err := im.cfg.Library.Read(ctx, func(ctx context.Context, r storage.Reader[file]) error {
			var err error
			again, err = im.sentAgain(ctx, r, imp, f.Position)
			return err
		})
		if err != nil || again {
			return err
		}
	}
	if imp.Source == FromRoot {
		var seen storing.Seen
		if f.Seen != nil {
			seen = *f.Seen
		}
		return im.land(ctx, imp, f, seen, InPlace)
	}
	var placed storing.Placed
	err := im.vacant(ctx, imp, f)
	switch {
	case err != nil:
	case f.Sent != nil:
		placed, err = im.cfg.Roots.Move(ctx, f.Dest.Root, transfers.StagedRel(f.Sent.Transfer, f.Sent.Order), f.Dest.Holding, false)
	default:
		placed, err = im.cfg.Roots.Place(ctx, f.Dest.Root, f.Dest.Holding, f.sourceOf(imp), f.PlacedBy == ByCopy)
	}
	if needs, ok := errors.AsType[*storing.NeedsCopyError](err); ok {
		_, err := im.write(ctx, imp.ID, func(ctx context.Context, tx *storage.Tx[file], cur *Import, _ *after) error {
			return im.grantCopy(ctx, tx, cur, f.Position, needs.Size)
		})
		return err
	}
	if err != nil {
		_, werr := im.write(ctx, imp.ID, func(ctx context.Context, tx *storage.Tx[file], cur *Import, _ *after) error {
			again, rerr := im.sentAgain(ctx, tx, *cur, f.Position)
			if g := fileAt(cur, f.Position); rerr == nil && !again && g.Outcome == Placing {
				g.holdOrFail(err)
			}
			return rerr
		})
		return werr
	}
	by := PlacedBy(placed.Step.String())
	if f.Sent != nil {
		by = ByTransfer
	}
	if imp.MoveFiles {
		by = ByMove
	}
	return im.land(ctx, imp, f, placed.Seen, by)
}

// vacant is nil when nothing SlipStream doesn't track sits at the name
// f's decision gives it in its root, else *files.NameTakenError, read
// before any of f's bytes move there.
func (im *Imports) vacant(ctx context.Context, imp Import, f File) error {
	var to string
	err := im.cfg.Library.Read(ctx, func(ctx context.Context, r storage.Reader[file]) error {
		cands, err := im.candidates(ctx, r, &imp, f.Dest.Root, []int{indexOf(&imp, f.Position)})
		if err != nil {
			return err
		}
		vs, err := im.cfg.Files.Judge(ctx, r, files.Batch{Work: imp.Work, Root: f.Dest.Root, Import: imp.ID.UUID, Candidates: cands})
		if err != nil {
			return err
		}
		to = vs[0].Path
		return nil
	})
	if err != nil || to == "" {
		return err
	}
	return im.cfg.Files.Vacant(ctx, f.Dest.Root, f.Dest.Holding.String(), to)
}

// holdOrFail makes a place's error an import wait: a name a file
// SlipStream doesn't track holds waits for the Admin, a storage wait
// retries, and any other storage error is a failure the Admin retries.
func (f *File) holdOrFail(err error) {
	var (
		waiting *roots.WaitingError
		taken   *files.NameTakenError
	)
	switch {
	case errors.As(err, &taken):
		f.Outcome, f.Reason, f.Hold = Waiting, &Reason{Code: NameTaken, Name: taken.Path}, nil
	case errors.As(err, &waiting), errors.As(err, new(*storing.NotReady)):
		f.wait(RootNotReady)
	case away(err):
		f.wait(NodeAway)
	case errors.As(err, new(*storing.NeedsSpaceError)):
		f.wait(NoSpace)
	default:
		f.Outcome, f.Reason, f.Hold = FailedOut, &Reason{Code: ImportFail, Line: err.Error()}, nil
	}
}

// away reports a call that never reached its Node.
func away(err error) bool {
	_, ok := rpc.NotSent(err)
	return ok
}

// wait holds f for code, counting the tries that ran into it.
func (f *File) wait(code HoldCode) {
	attempts := 1
	if f.Hold != nil && f.Hold.Code == code {
		attempts = f.Hold.Attempts + 1
	}
	f.Hold = &Hold{Code: code, Attempts: attempts, Since: storage.Now()}
}

// grantCopy grants file position of imp a copy when its size fits its
// volume: the latest free space less the Storage error level, what live
// Transfers still send there and every other copy granted on it. Else the
// file holds no_space.
func (im *Imports) grantCopy(ctx context.Context, tx *storage.Tx[file], imp *Import, position int, size int64) error {
	f := fileAt(imp, position)
	if f.Outcome != Placing {
		return nil
	}
	d, err := im.cfg.Roots.DestinationIn(ctx, tx, f.Dest.Root)
	if err != nil {
		return err
	}
	if d.Volume == nil {
		f.PlacedBy, f.Hold = ByCopy, nil
		return nil
	}
	levels, err := health.LevelsIn(ctx, tx)
	if err != nil {
		return err
	}
	reserved, err := im.cfg.Transfers.ReservedIn(ctx, tx, d.Node, d.Volume.ID)
	if err != nil {
		return err
	}
	copies, err := im.copiesOn(ctx, tx, d)
	if err != nil {
		return err
	}
	room := d.Volume.Free - levels.For(d.Node, d.Volume.Mount, d.Volume.Total).Error.Bytes - reserved - copies
	if size > room {
		f.wait(NoSpace)
		f.Hold.Free, f.Hold.Needs = max(room, 0), size
		return nil
	}
	f.PlacedBy, f.Hold = ByCopy, nil
	return nil
}

// copiesOn is the bytes of every copy granted into a root on d's volume.
func (im *Imports) copiesOn(ctx context.Context, tx *storage.Tx[file], d roots.Destination) (int64, error) {
	rows, err := db.New(tx).ListCopies(ctx)
	if err != nil {
		return 0, err
	}
	var n int64
	for _, row := range rows {
		other, err := im.cfg.Roots.DestinationIn(ctx, tx, row.RootID)
		if err != nil {
			return 0, err
		}
		if other.Node == d.Node && other.Volume != nil && other.Volume.ID == d.Volume.ID {
			n += row.Size
		}
	}
	return n, nil
}

// land is the landing Write: f is judged again against the live library
// and lands through files.Import, or waits with the reason its judge
// gives, its bytes and cells kept. A file that left placing meanwhile is
// nothing to do, and its holding bytes lose their claim.
func (im *Imports) land(ctx context.Context, imp Import, f File, seen storing.Seen, by PlacedBy) error {
	im.hold(ctx, imp, f, AtLand)
	_, err := im.write(ctx, imp.ID, func(ctx context.Context, tx *storage.Tx[file], cur *Import, a *after) error {
		g := fileAt(cur, f.Position)
		if g.Outcome != Placing {
			return nil
		}
		cands, err := im.candidates(ctx, tx, cur, g.Dest.Root, []int{indexOf(cur, g.Position)})
		if err != nil {
			return err
		}
		landed, err := im.cfg.Files.Import(ctx, tx, files.Landing{
			Batch: files.Batch{Work: cur.Work, Root: g.Dest.Root, Import: cur.ID.UUID, Candidates: cands},
			Seen:  seen, Grab: cur.Grab.UUID, PlacedBy: string(by),
		})
		var refused *files.Refused
		switch {
		case errors.As(err, &refused):
			batch := []int{indexOf(cur, g.Position)}
			g.take(outcomes([]files.Verdict{refused.Verdict}, cur.outcomeContext(batch, false))[0], g.Dest)
			return nil
		case errors.Is(err, files.ErrRootChanged):
			g.Outcome, g.Reason, g.Hold = FailedOut, &Reason{Code: RootChanged}, nil
			return nil
		case err != nil:
			return err
		}
		g.Outcome, g.Reason, g.Hold, g.PlacedBy, g.FileID = Placed, nil, nil, by, landed.File.ID
		g.Fills = fillsOf(landed.Fills, g.forced())
		a.landed = append(a.landed, cur.Work)
		if g.Sent != nil {
			return im.cfg.Transfers.Placed(ctx, tx, g.Sent.Transfer, g.Sent.Order)
		}
		return nil
	})
	return err
}

func indexOf(imp *Import, position int) int {
	for i, f := range imp.Files {
		if f.Position == position {
			return i
		}
	}
	return -1
}
