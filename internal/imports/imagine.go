package imports

import (
	"context"
	"errors"
	"path"
	"strings"

	"github.com/jatassi/slipstream/internal/files"
	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/release"
	"github.com/jatassi/slipstream/internal/roots"
	"github.com/jatassi/slipstream/internal/scanning"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/works"
)

// ErrNoQuality is a tested name that gives no quality, neither in its
// words nor by its extension.
var ErrNoQuality = errors.New("the name gives no quality")

// UnplacedError is a tested name its Work's Content Type places at no
// Item, and why.
type UnplacedError struct{ Reason scanning.Reason }

func (e *UnplacedError) Error() string {
	return "the name matches no single Item of the Work"
}

// Imagine is where a file named name would go in work, read and placed
// as a scan reads and places a found file there, under the stored naming
// settings or draft over its type's: the path the file name tester shows.
// name is a path inside the Work's folder, with either separator. A file
// that fills nothing is named as a Kept file. It writes nothing.
//
// A name placed at no Item is an *UnplacedError, one with no quality
// ErrNoQuality, and a Work in no root roots.ErrNoRoot; files' refusals
// are Imagine's, and a draft the save check refuses is its
// *naming.RefusedError.
func (im *Imports) Imagine(ctx context.Context, _ identity.Admin, work works.WorkID, name string, draft *files.Draft) (string, error) {
	rel := strings.ReplaceAll(name, `\`, "/")
	var out string
	err := im.cfg.Library.Read(ctx, func(ctx context.Context, r storage.Reader[file]) error {
		if _, err := works.SummaryOf(ctx, r, work); err != nil {
			return err
		}
		root, err := roots.OfWork(ctx, r, work)
		if err != nil {
			return err
		}
		facet := im.cfg.Facets[root.ContentType]
		place, err := facet.Place(ctx, r, work)
		if err != nil {
			return err
		}
		placed := place(rel)
		items, ok := placed.Items()
		if !ok {
			return &UnplacedError{Reason: placed.Reason()}
		}
		cuts, err := im.cfg.Targets.CutsIn(ctx, r)
		if err != nil {
			return err
		}
		read := release.File(rel, facet, cuts.For(root.ContentType))
		quality, ok := read.Facts.Quality.Get()
		if !ok {
			return ErrNoQuality
		}
		out, err = im.cfg.Files.Imagine(ctx, r, files.Batch{Work: work, Root: root.ID, Candidates: []files.Candidate{{
			Items: items, Cut: read.Facts.Cut, Holding: files.HoldingOf(storage.NewUUID(), path.Ext(rel)),
			Quality: quality, ReleaseName: path.Base(rel), Title: read, SourceName: path.Base(rel),
		}}}, draft)
		return err
	})
	return out, err
}
