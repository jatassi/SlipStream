package imports

import (
	"context"
	"errors"
	"path"
	"slices"
	"strings"

	"github.com/jatassi/slipstream/internal/files"
	"github.com/jatassi/slipstream/internal/grabs"
	"github.com/jatassi/slipstream/internal/imports/internal/db"
	"github.com/jatassi/slipstream/internal/media"
	"github.com/jatassi/slipstream/internal/scanning"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/works"
)

// starter is the grabs.Importer imports registers: a Grab's download
// becomes its Import in the Write that moves the Grab to importing.
type starter struct{ im *Imports }

// Start creates h's Grab's Import, and converges it, which reports back
// to the Grab.
func (s starter) Start(ctx context.Context, tx *storage.Tx[file], h grabs.Handoff) error {
	if _, err := db.New(tx).GetOpenImportOfGrab(ctx, h.Grab); !errors.Is(err, storage.ErrNoRows) {
		return err
	}
	imp, err := s.im.grabImport(ctx, tx, h, ImportID{UUID: storage.NewUUID()})
	if err != nil {
		return err
	}
	return s.im.insertConverged(ctx, tx, imp)
}

// End ends g's unfinished Import for why: a cancel with no file placing
// at once, else as a removal. Each file not yet final or placing is
// declined, and a placing one finishes, so the Grab is imported once
// anything was placed and fails as why says otherwise.
func (s starter) End(ctx context.Context, tx *storage.Tx[file], g grabs.GrabID, why grabs.Failure, cancel bool) error {
	id, err := db.New(tx).GetOpenImportOfGrab(ctx, g)
	if errors.Is(err, storage.ErrNoRows) {
		return nil
	}
	if err != nil {
		return err
	}
	_, err = s.im.writeIn(ctx, tx, id, &after{}, func(_ context.Context, _ *storage.Tx[file], cur *Import, _ *after) error {
		if cancel && slices.ContainsFunc(cur.Files, func(f File) bool { return f.Outcome == Placing && f.Hold == nil }) {
			return &grabs.GrabBusyError{Stage: grabs.Importing}
		}
		if cur.Ended != nil {
			return nil
		}
		reason := Removed
		if cancel {
			reason = Canceled
		}
		endImport(cur, Ending{Reason: reason, Code: string(why.Code), Message: why.Message})
		return nil
	})
	return err
}

// Wake has the reconciler and the Transfers look again, once a grabs
// Write that started or ended an Import commits.
func (s starter) Wake() {
	s.im.Touch()
	s.im.cfg.Transfers.Touch()
	s.im.cfg.Changed.Publish(context.Background(), changedOf())
}

// insertConverged stores imp, a new Import, and converges it in tx.
func (im *Imports) insertConverged(ctx context.Context, tx *storage.Tx[file], imp Import) error {
	if err := insertImport(ctx, tx, imp); err != nil {
		return err
	}
	_, err := im.writeIn(ctx, tx, imp.ID, &after{}, func(context.Context, *storage.Tx[file], *Import, *after) error { return nil })
	return err
}

// grabImport is the Import id of h's download, read in r: a row for each
// file with a video extension, in the download's order, each through the
// fixed check and its Content Type's reading of its name.
func (im *Imports) grabImport(ctx context.Context, r storage.Reader[file], h grabs.Handoff, id ImportID) (Import, error) {
	rules, err := RulesIn(ctx, r)
	if err != nil {
		return Import{}, err
	}
	tree, err := im.cfg.Works.TreeIn(ctx, r, h.Work)
	if err != nil {
		return Import{}, err
	}
	place, err := im.cfg.Facets[tree.ContentType].Place(ctx, r, h.Work)
	if err != nil {
		return Import{}, err
	}
	imp := Import{
		ID: id, Source: FromGrab, Grab: h.Grab, Node: h.Node, InfoHash: &h.InfoHash, ContentType: tree.ContentType,
		Work: h.Work, Edition: h.Edition, Created: storage.Now(),
	}
	for _, tf := range h.Files {
		rel := strings.ReplaceAll(tf.Path, `\`, "/")
		if !media.DefaultVideos.Video(rel) && !rules.Videos.Video(rel) {
			continue
		}
		f := File{
			Position: len(imp.Files), Path: tf.Path, SourceIndex: new(tf.Index), Size: tf.Size, ReleaseName: h.Release,
			SourceName: path.Base(rel), Outcome: Pending,
		}
		f.check(rel, rules, h.Items, place)
		f.force(h.Cells)
		imp.Files = append(imp.Files, f)
	}
	return imp, nil
}

// check applies the fixed check to f, a download's file at rel: left out
// by its extension, folder, name or size, else waiting when its name
// reads as no Item or Items the Grab doesn't cover, else pending with its
// Items.
func (f *File) check(rel string, rules Rules, grabbed []works.ItemID, place scanning.Placer) {
	if why, out := leftOutOf(rel, f.Size, rules); out {
		f.Outcome, f.Reason = LeftOut, &why
		return
	}
	read := place(f.SourceName)
	items, ok := read.Items()
	switch {
	case !ok:
		f.waitFor(read.Reason())
	case slices.ContainsFunc(items, func(it works.ItemID) bool { return !slices.Contains(grabbed, it) }):
		f.Outcome, f.Reason = Waiting, &Reason{Code: OtherItems, Reading: items, Of: grabbed}
	default:
		f.Items = items
	}
}

// force gives f, when it is pending, each of cells of its Items that a
// Manual search grabbed as forced: the Admin chose the Release over the
// file it replaces, so the file takes the cell by Force.
func (f *File) force(cells []grabs.Cell) {
	for _, c := range cells {
		if c.Forced && f.Outcome == Pending && slices.Contains(f.Items, c.Key.Item) {
			f.Fills = append(f.Fills, Fill{Cell: c.Key, Forced: true})
		}
	}
}

// leftOutOf is why the video at rel, of size bytes, is left out of a
// download's Import, if it is.
func leftOutOf(rel string, size int64, rules Rules) (Reason, bool) {
	folders := strings.Split(rel, "/")
	name := strings.ToLower(folders[len(folders)-1])
	stem := strings.TrimSuffix(name, path.Ext(name))
	switch {
	case !rules.Videos.Video(name):
		return Reason{Code: NotAVideoType}, true
	case media.Counts(rel, rules.Videos):
	case stem == "sample" || strings.HasSuffix(stem, "-sample") || slices.ContainsFunc(folders[:len(folders)-1], sampleFolder):
		return Reason{Code: Sample}, true
	case strings.HasPrefix(name, ".") || slices.ContainsFunc(folders[:len(folders)-1], hiddenFolder):
		return Reason{Code: HiddenFolder}, true
	default:
		return Reason{Code: Bonus}, true
	}
	if size < rules.Smallest {
		return Reason{Code: TooSmall, Value: rules.Smallest}, true
	}
	return Reason{}, false
}

func sampleFolder(name string) bool {
	return strings.EqualFold(name, "sample") || strings.EqualFold(name, "samples")
}

// hiddenFolder is a folder a walk skips for being hidden, NAS or OS
// clutter or a disc structure, rather than for holding bonus features.
func hiddenFolder(name string) bool {
	return strings.HasPrefix(name, ".") || slices.Contains(clutter, strings.ToLower(name))
}

var clutter = []string{"@eadir", "#recycle", "#snapshot", "$recycle.bin", "system volume information", "lost+found", "bdmv", "video_ts"}

// report is what imp tells its Grab, whose Handoff is h: how each of the
// Grab's open cells settled, whether anything was placed and how imp
// ended.
func (im Import) report(h grabs.Handoff) grabs.Report {
	r := grabs.Report{Settled: map[files.CellKey]grabs.Settled{}, Ended: im.ended()}
	r.Placed = slices.ContainsFunc(im.Files, func(f File) bool { return f.Outcome == Placed })
	for _, c := range h.Cells {
		if c.Settled != nil {
			continue
		}
		if s, ok := im.settles(c.Key); ok {
			r.Settled[c.Key] = s
		}
	}
	return r
}

// settles is how cell k settled, false while a file of im could still
// fill it: placed when a placed file fills it, declined when a declined
// file was for it or im was ended, else none.
func (im Import) settles(k files.CellKey) (grabs.Settled, bool) {
	if slices.ContainsFunc(im.Files, func(f File) bool { return f.Outcome == Placed && f.fills(k) }) {
		return grabs.SettledPlaced, true
	}
	if slices.ContainsFunc(im.Files, func(f File) bool { return f.holds(k) }) {
		return "", false
	}
	declined := im.Ended != nil || slices.ContainsFunc(im.Files, func(f File) bool {
		return f.Outcome == Declined && (f.fills(k) || slices.Contains(f.Items, k.Item))
	})
	if declined {
		return grabs.SettledDeclined, true
	}
	return grabs.SettledNone, true
}

// verdictWaits are the waits on the decision's verdict for a file's own
// Items, which hold only those Items' cells. Any other wait is on the
// file's reading or the Import as a whole, and holds every cell.
var verdictWaits = []ReasonCode{NotAnUpgrade, NameTaken, NameTooLong, EditionNotWanted}

// holds reports whether f could still fill cell k: a pending file or a
// whole wait any cell, a verdict wait its Items' cells, and a file
// sending or placing the cells it fills.
func (f *File) holds(k files.CellKey) bool {
	switch f.Outcome {
	case Pending:
		return true
	case Waiting:
		return !slices.Contains(verdictWaits, f.Reason.Code) || slices.Contains(f.Items, k.Item)
	case Sending, Placing:
		return f.fills(k)
	case LeftOut, NotBetter, Placed, Declined, FailedOut:
	}
	return false
}

func (f *File) fills(k files.CellKey) bool {
	return slices.ContainsFunc(f.Fills, func(fl Fill) bool { return fl.Cell == k })
}

// ended is how im ended for its Grab, nil while a file of it may still
// be placed: a removal or cancel as stored, a release fault, or with
// every file final, a failed file's error or the Admin's declines.
func (im Import) ended() *grabs.Failure {
	switch {
	case im.Ended != nil:
		return &grabs.Failure{Stage: grabs.Importing, Code: grabs.Code(im.Ended.Code), Message: im.Ended.Message}
	case im.Failure != nil && *im.Failure == NoVideoFile:
		return &grabs.Failure{Stage: grabs.Importing, Code: grabs.NoVideoFile, Message: "The download holds no video file."}
	case im.Failure != nil:
		return &grabs.Failure{Stage: grabs.Importing, Code: grabs.Unreadable, Message: "SlipStream can't read any video file in the download."}
	case slices.ContainsFunc(im.Files, func(f File) bool { return !f.Outcome.Final() }):
		return nil
	}
	if i := slices.IndexFunc(im.Files, func(f File) bool { return f.Outcome == FailedOut }); i >= 0 {
		return &grabs.Failure{Stage: grabs.Importing, Code: grabs.ImportFailed, Message: im.Files[i].Reason.Line}
	}
	return &grabs.Failure{Stage: grabs.Importing, Code: grabs.Declined, Message: "The Admin declined every file of the download."}
}

// follow reports imp to its Grab in tx, which settles its cells and moves
// it.
func (im *Imports) follow(ctx context.Context, tx *storage.Tx[file], imp *Import, a *after) error {
	h, err := grabs.Handed(ctx, tx, imp.Grab)
	if errors.Is(err, grabs.ErrNoGrab) {
		return nil
	}
	if err != nil {
		return err
	}
	f, err := im.cfg.Grabs.Follow(ctx, tx, imp.Grab, imp.report(h))
	a.followed = append(a.followed, f)
	return err
}
