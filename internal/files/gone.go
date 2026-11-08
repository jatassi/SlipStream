package files

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/jatassi/slipstream/internal/files/internal/db"
	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/media"
	"github.com/jatassi/slipstream/internal/roots"
	"github.com/jatassi/slipstream/internal/scanning"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/storing"
	"github.com/jatassi/slipstream/internal/works"
)

// OnRootFiles is a root's files as a scan compares them, gone ones too,
// and the paths whose bytes the disk queue still moves or removes.
type OnRootFiles struct {
	Files []scanning.Recorded
	Held  []storing.Rel
}

// OnRoot reads root's files and held paths in r.
func OnRoot(ctx context.Context, r storage.Reader[file], root roots.ID) (OnRootFiles, error) {
	q := db.New(r)
	rows, err := q.ScannedOfRoot(ctx, root)
	if err != nil {
		return OnRootFiles{}, err
	}
	out := OnRootFiles{Files: make([]scanning.Recorded, len(rows))}
	for i, row := range rows {
		rec := scanning.Recorded{
			ID: row.ID.UUID, Work: row.WorkID, Size: row.Size, Seen: seenOf(row.ModifiedAt, row.NodeFileID, row.NodeVolumeID),
			Gone: !row.GoneAt.IsZero(), Stale: row.ProbeVersion < media.Version,
		}
		if rec.Path, rec.Sits, rec.Next, err = placesOf(row.ID, row.Path, row.Sits); err != nil {
			return OnRootFiles{}, fmt.Errorf("file %s of root %s: %w", row.ID, root, err)
		}
		out.Files[i] = rec
	}
	held, err := q.HeldOfRoot(ctx, root)
	if err != nil {
		return OnRootFiles{}, err
	}
	out.Held = make([]storing.Rel, len(held))
	for i, p := range held {
		if out.Held[i], err = storing.ParseRel(p); err != nil {
			return OnRootFiles{}, fmt.Errorf("a path the disk queue holds in root %s: %w", root, err)
		}
	}
	return out, nil
}

// Tally is how many files of a root a Health item counts, with the first
// five.
type Tally struct {
	Count int
	First []Path
}

// Path is one file a Tally lists. Line is ffprobe's, for an unreadable
// file.
type Path struct {
	Path string
	Line string
}

const tallied = 5

// GoneOf counts each root's gone files, with the first five to go, in r.
func GoneOf(ctx context.Context, r storage.Reader[file]) (map[roots.ID]Tally, error) {
	rows, err := db.New(r).ListGone(ctx)
	if err != nil {
		return nil, err
	}
	out := map[roots.ID]Tally{}
	for _, row := range rows {
		count(out, row.RootID, Path{Path: row.Path})
	}
	return out, nil
}

// UnreadableOf counts each root's live files whose last probe failed,
// oldest first, in r.
func UnreadableOf(ctx context.Context, r storage.Reader[file]) (map[roots.ID]Tally, error) {
	rows, err := db.New(r).ListUnreadable(ctx)
	if err != nil {
		return nil, err
	}
	out := map[roots.ID]Tally{}
	for _, row := range rows {
		count(out, row.RootID, Path{Path: row.Path, Line: row.Line})
	}
	return out, nil
}

func count(out map[roots.ID]Tally, root roots.ID, p Path) {
	t := out[root]
	t.Count++
	if len(t.First) < tallied {
		t.First = append(t.First, p)
	}
	out[root] = t
}

// GoneOfWorks counts the gone files of each of ws that has one, in r.
func GoneOfWorks(ctx context.Context, r storage.Reader[file], ws works.Among[works.WorkID]) (map[works.WorkID]int, error) {
	q := db.New(r)
	rows, err := works.Read(ws, func() ([]db.CountEveryGoneRow, error) { return q.CountEveryGone(ctx) }, func(ids []works.WorkID) ([]db.CountEveryGoneRow, error) {
		rows, err := q.CountGoneOfWorks(ctx, ids)
		out := make([]db.CountEveryGoneRow, len(rows))
		for i, row := range rows {
			out[i] = db.CountEveryGoneRow(row)
		}
		return out, err
	})
	if err != nil {
		return nil, err
	}
	out := make(map[works.WorkID]int, len(rows))
	for _, row := range rows {
		out[row.WorkID] = int(row.Files)
	}
	return out, nil
}

// LeftIn is each live file of ws whose bytes sit at its path, in r: what
// a Leave files delete leaves on disk, read before the rows go. A file
// still moving is left out.
func LeftIn(ctx context.Context, r storage.Reader[file], ws []works.WorkID) ([]File, error) {
	rows, err := db.New(r).ListLeftIn(ctx, ws)
	if err != nil {
		return nil, err
	}
	out := make([]File, len(rows))
	for i, row := range rows {
		out[i] = fileOf(row)
	}
	return out, nil
}

// ErrNoFile is a Forget naming an id no file has.
var ErrNoFile = errors.New("no file has that id")

// NotGoneError is a Forget naming files a scan hasn't found gone: the
// paths of those files.
type NotGoneError struct{ Paths []string }

func (e *NotGoneError) Error() string {
	if len(e.Paths) == 1 {
		return e.Paths[0] + " isn't gone"
	}
	return strings.Join(e.Paths, ", ") + " aren't gone"
}

// Forget deletes the gone files ids name in one Write and tells each of
// their Works' pages. An id no file has refuses them all with ErrNoFile,
// and a live file among them with a *NotGoneError.
func (f *Files) Forget(ctx context.Context, _ identity.Admin, ids []ID) error {
	var touched []works.WorkID
	err := f.cfg.Library.Write(ctx, func(ctx context.Context, tx *storage.Tx[file]) error {
		var err error
		touched, err = forgetGone(ctx, db.New(tx), ids)
		return err
	})
	if err != nil {
		return err
	}
	for _, w := range touched {
		f.cfg.Changed.Publish(ctx, Changed{WorkID: w.String()})
	}
	return nil
}

func forgetGone(ctx context.Context, q *db.Queries, ids []ID) ([]works.WorkID, error) {
	rows, err := q.ListFilesByID(ctx, ids)
	if err != nil {
		return nil, err
	}
	named := map[ID]bool{}
	for _, id := range ids {
		named[id] = true
	}
	if len(rows) < len(named) {
		return nil, ErrNoFile
	}
	var live []string
	for _, row := range rows {
		if row.GoneAt.IsZero() {
			live = append(live, row.Path)
		}
	}
	if len(live) > 0 {
		return nil, &NotGoneError{Paths: live}
	}
	var touched []works.WorkID
	for _, row := range rows {
		if err := q.DeleteFile(ctx, row.ID); err != nil {
			return nil, err
		}
		if !slices.Contains(touched, row.WorkID) {
			touched = append(touched, row.WorkID)
		}
	}
	return touched, nil
}
