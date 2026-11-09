package imports

import (
	"context"

	"github.com/jatassi/slipstream/internal/files"
	"github.com/jatassi/slipstream/internal/roots"
	"github.com/jatassi/slipstream/internal/scanning"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/works"
)

// Leave is library.Leaver: in a Leave files delete's Write, it lists ws's
// files on disk in an ignored entry per root and Work folder, with their
// records, so the next scan doesn't add the Works back and the Admin can
// still import them. Bytes still moving into place are left out: the disk
// queue's holding sweep removes them.
func (im *Imports) Leave(ctx context.Context, tx *storage.Tx[file], ws []works.WorkID) error {
	left, err := files.LeftIn(ctx, tx, ws)
	if err != nil {
		return err
	}
	type at struct {
		root   roots.ID
		folder string
	}
	var order []at
	by := map[at]*pend{}
	for _, f := range left {
		folder, loose := folderOf(f.Path)
		k := at{f.RootID, folder}
		p := by[k]
		if p == nil {
			root, err := roots.Get(ctx, tx, f.RootID)
			if err != nil {
				return err
			}
			hint := im.cfg.Facets[root.ContentType].Hint(scanning.Group{Folder: folder, Loose: loose}.Name())
			p = &pend{Root: f.RootID, Folder: folder, ContentType: root.ContentType, Hint: hint, Ignore: true}
			by[k] = p
			order = append(order, k)
		}
		p.Files = append(p.Files, PendingFile{
			Path: f.Path, Size: f.Size, Seen: f.Seen, Media: &f.Media, Quality: f.Quality, Cut: f.Cut,
			Reason: scanning.Reason{Code: scanning.Left},
		})
	}
	now := storage.Now()
	for _, k := range order {
		p := by[k]
		listed, err := listedAt(ctx, tx, k.root, k.folder)
		if err != nil {
			return err
		}
		p.Listed = listed
		if err := upsertPending(ctx, tx, *p, now); err != nil {
			return err
		}
	}
	return nil
}
