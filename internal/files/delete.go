package files

import (
	"context"
	"slices"

	"github.com/jatassi/slipstream/internal/files/internal/db"
	"github.com/jatassi/slipstream/internal/roots"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/works"
)

// Held is what one Work's live library files add up to.
type Held struct {
	// Files counts them, Kept files included, and Bytes is their size.
	Files, Kept int
	Bytes       int64
	// Roots are the roots they sit in, in id order.
	Roots []roots.ID
}

// Holding reads what the live library files of each of ws add up to in
// r. A Work with none has no entry.
func Holding(ctx context.Context, r storage.Reader[file], ws []works.WorkID) (map[works.WorkID]Held, error) {
	rows, err := db.New(r).ListFilesOfWorks(ctx, ws)
	if err != nil {
		return nil, err
	}
	out := map[works.WorkID]Held{}
	for _, row := range rows {
		if f := fileOf(row); !f.Gone() {
			h := out[row.WorkID]
			h.add(f)
			out[row.WorkID] = h
		}
	}
	return out, nil
}

// add counts f, a live file.
func (h *Held) add(f File) {
	h.Files++
	h.Bytes += f.Size
	if f.Kept() {
		h.Kept++
	}
	if i, found := slices.BinarySearchFunc(h.Roots, f.RootID, func(a, b roots.ID) int { return a.Compare(b.UUID) }); !found {
		h.Roots = slices.Insert(h.Roots, i, f.RootID)
	}
}

// DoomIn queues every live library file of ws for the disk queue in tx,
// from wherever its bytes sit, and deletes its row, and deletes the rows
// of their gone files, whose bytes are nowhere to remove. The queue
// removes the bytes on each root's Node once tx commits and it is poked,
// retrying until they are gone.
func DoomIn(ctx context.Context, tx *storage.Tx[file], ws []works.WorkID) error {
	q := db.New(tx)
	rows, err := q.ListFilesOfWorks(ctx, ws)
	if err != nil {
		return err
	}
	for _, row := range rows {
		f := fileOf(row)
		if f.Gone() {
			err = q.DeleteFile(ctx, f.ID)
		} else {
			err = doom(ctx, q, f)
		}
		if err != nil {
			return err
		}
	}
	return nil
}
