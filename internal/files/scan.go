package files

import (
	"context"
	"errors"
	"maps"
	"slices"

	"github.com/jatassi/slipstream/internal/files/internal/db"
	"github.com/jatassi/slipstream/internal/media"
	"github.com/jatassi/slipstream/internal/release"
	"github.com/jatassi/slipstream/internal/roots"
	"github.com/jatassi/slipstream/internal/scanning"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/storing"
	"github.com/jatassi/slipstream/internal/works"
)

// Scan is one Work's part of a library scan of Root: what
// scanning.Compare decided for its recorded files, the media records read
// again for them, and the found files it takes.
type Scan struct {
	Work  works.WorkID
	Root  roots.ID
	Share scanning.Share
	// Probes is what reading each of Share.Reprobes again gave, by its
	// walked path. A file to read again with no Probe keeps its row as it
	// is, so the next scan reads it again.
	Probes map[storing.Rel]Probe
	Found  []Found
}

// Probe is what reading a file's media record gave: the record, or the
// first line ffprobe printed when it couldn't read the file.
type Probe struct {
	Media media.Record
	Error string
}

// Found is a file the walk found that no record names, matched to an Item
// of the Work, with what reading it gave.
type Found struct {
	Item    works.ItemID
	Cut     release.CutKey
	At      storing.Walked
	Media   media.Record
	Quality release.Quality
}

// Scanned is what Scan wrote. Dropped counts the transitions and found
// files it left out because their rows or paths changed since the walk:
// the next scan judges them again.
type Scanned struct {
	Changed bool
	Dropped int
}

// Scan records s in tx and converges its Work under the scan's rule: no
// file is doomed, every path is kept, gone files hold no cell. Each found
// file is inserted at its path with its label and N, and no move.
//
// The walk and VerifyIn ran outside the Write, so Scan re-reads here what
// they checked: each recorded file must still read as the compare read
// it, at the same path, sitting at the same place, at the same size and
// as gone or live as it was, else its transition is dropped. A found
// file, or a move's new path, that a row, a move or a doomed file names
// now is dropped too. A Work no longer in Root is ErrRootChanged.
func (f *Files) Scan(ctx context.Context, tx *storage.Tx[file], s Scan) (Scanned, error) {
	root, err := roots.OfWork(ctx, tx, s.Work)
	switch {
	case errors.Is(err, roots.ErrNoRoot) || err == nil && root.ID != s.Root:
		return Scanned{}, ErrRootChanged
	case err != nil:
		return Scanned{}, err
	}
	before, err := f.load(ctx, tx, s.Work)
	if err != nil {
		return Scanned{}, err
	}
	sc := scan{Scan: s, q: db.New(tx), now: storage.Now(), cuts: before.cuts.For(before.tree.ContentType), left: map[ID]bool{}}
	after := before
	after.files = slices.Clone(before.files)
	sc.at = make(map[ID]*File, len(after.files))
	for i := range after.files {
		sc.at[after.files[i].ID] = &after.files[i]
	}
	if err := sc.record(ctx); err != nil {
		return Scanned{}, err
	}
	after.seen = maps.Clone(before.seen)
	after.found = map[ID]bool{}
	for _, fd := range s.Found {
		switch taken, err := sc.taken(ctx, fd.At.Path); {
		case err != nil:
			return Scanned{}, err
		case taken:
			sc.dropped++
			continue
		}
		g := File{
			ID: ID{UUID: storage.NewUUID()}, WorkID: s.Work, ItemID: fd.Item, Cut: fd.Cut, RootID: s.Root, Path: fd.At.Path.String(),
			Size: fd.At.Size, Media: fd.Media, ProbedAt: sc.now, Seen: &fd.At.Seen, Quality: fd.Quality,
		}
		after.files = append(after.files, g)
		after.found[g.ID] = true
		if !slices.Contains(after.seen[fd.Item], fd.Cut) {
			after.seen[fd.Item] = append(slices.Clone(after.seen[fd.Item]), fd.Cut)
		}
	}
	if len(sc.changed) == 0 && len(after.found) == 0 {
		return Scanned{Dropped: sc.dropped}, nil
	}
	for _, id := range sc.changed {
		if err := sc.write(ctx, *sc.at[id]); err != nil {
			return Scanned{}, err
		}
	}
	p, err := f.converge(ctx, tx, before, after, refill(after), scanned)
	if err != nil {
		return Scanned{}, err
	}
	return Scanned{Changed: true, Dropped: sc.dropped}, apply(ctx, sc.q, p)
}

// scan is one Scan's Write in progress: the Work's files by id, as
// changed so far, and the ones changed, in the order they were. left are
// the files that left the place a pending move would take their bytes
// from, by a move or by going.
type scan struct {
	Scan
	q       *db.Queries
	now     storage.Time
	cuts    []release.Cut
	at      map[ID]*File
	changed []ID
	left    map[ID]bool
	dropped int
}

// record applies each of the share's transitions whose row still reads as
// the compare read it.
func (sc *scan) record(ctx context.Context) error {
	for _, same := range sc.Share.Same {
		if err := sc.transition(ctx, same.Rec, func(g *File) bool { return sc.same(g, same) }); err != nil {
			return err
		}
	}
	for _, c := range sc.Share.Changed {
		if err := sc.transition(ctx, c.Rec, func(g *File) bool { return sc.resized(g, c) }); err != nil {
			return err
		}
	}
	for _, m := range sc.Share.Moved {
		switch taken, err := sc.taken(ctx, m.To.Path); {
		case err != nil:
			return err
		case taken:
			sc.dropped++
			continue
		}
		err := sc.transition(ctx, m.Rec, func(g *File) bool {
			g.Path, g.Seen, g.GoneAt = m.To.Path.String(), &m.To.Seen, storage.Time{}
			sc.left[g.ID] = true
			return true
		})
		if err != nil {
			return err
		}
	}
	for _, r := range sc.Share.Gone {
		err := sc.transition(ctx, r, func(g *File) bool {
			g.GoneAt, sc.left[g.ID] = sc.now, true
			return true
		})
		if err != nil {
			return err
		}
	}
	return nil
}

// transition changes rec's file with change when its row still reads as
// rec does, and counts it dropped otherwise. change reports whether it
// changed anything to write.
func (sc *scan) transition(ctx context.Context, rec scanning.Recorded, change func(*File) bool) error {
	g, ok := sc.at[ID{UUID: rec.ID}]
	if !ok {
		sc.dropped++
		return nil
	}
	sits, err := sc.q.GetMove(ctx, g.ID)
	switch {
	case errors.Is(err, storage.ErrNoRows):
		sits = g.Path
	case err != nil:
		return err
	}
	if g.Path != rec.Path.String() || sits != rec.Sits.String() || g.Size != rec.Size || g.Gone() != rec.Gone {
		sc.dropped++
		return nil
	}
	if change(g) {
		sc.changed = append(sc.changed, g.ID)
	}
	return nil
}

// same records what the walk read of a file at its place: its stamps, a
// return, and the record read again when it was to be.
func (sc *scan) same(g *File, s scanning.Same) bool {
	if s.Reprobe != scanning.Keep {
		p, ok := sc.Probes[s.At.Path]
		if !ok {
			return false
		}
		sc.reprobe(g, p, func(m media.Record) release.Quality {
			return scanning.Requalified(g.Quality, g.Path, sc.cuts, m.Resolution())
		})
	}
	g.Seen = &s.At.Seen
	if s.Returned {
		g.GoneAt = storage.Time{}
	}
	return true
}

// resized records a file at its path at another size: read again, its
// quality read again from its name and record, and its Grab link dropped.
func (sc *scan) resized(g *File, s scanning.Same) bool {
	p, ok := sc.Probes[s.At.Path]
	if !ok {
		return false
	}
	sc.reprobe(g, p, func(m media.Record) release.Quality {
		q, _ := scanning.QualityOf(g.Path, sc.cuts, m.Resolution())
		return q
	})
	g.Size, g.Seen, g.GrabID = s.At.Size, &s.At.Seen, storage.UUID{}
	if s.Returned {
		g.GoneAt = storage.Time{}
	}
	return true
}

// reprobe records p, what reading g again gave: its record and the quality
// quality reads from it, or the line that says why there is none, which
// keeps the record before.
func (sc *scan) reprobe(g *File, p Probe, quality func(media.Record) release.Quality) {
	g.ProbedAt = sc.now
	if p.Error != "" {
		g.ProbeError = p.Error
		return
	}
	g.Media, g.ProbeError, g.Quality = p.Media, "", quality(p.Media)
}

// taken reports whether a row, a move or a doomed file of the root names
// path now.
func (sc *scan) taken(ctx context.Context, path storing.Rel) (bool, error) {
	return sc.q.PathTaken(ctx, db.PathTakenParams{RootID: sc.Root, Path: path.String()})
}

// write stores g's scanned columns, and drops the pending move of a file
// that left its place.
func (sc *scan) write(ctx context.Context, g File) error {
	if sc.left[g.ID] {
		if err := sc.q.DeleteFileMove(ctx, g.ID); err != nil {
			return err
		}
	}
	modified, file, volume := seenColumns(g.Seen)
	var line *string
	if g.ProbeError != "" {
		line = &g.ProbeError
	}
	return sc.q.UpdateScanned(ctx, db.UpdateScannedParams{
		ID: g.ID, Path: g.Path, Size: g.Size, Probe: g.Media, GrabID: g.GrabID, Quality: g.Quality,
		ModifiedAt: modified, NodeFileID: file, NodeVolumeID: volume, GoneAt: g.GoneAt, ProbedAt: g.ProbedAt, ProbeError: line,
	})
}
