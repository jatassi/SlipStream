package files

import (
	"context"
	"errors"
	"time"

	"github.com/jatassi/slipstream/internal/media"
	"github.com/jatassi/slipstream/internal/release"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/roots"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/storing"
	"github.com/jatassi/slipstream/internal/targets"
	"github.com/jatassi/slipstream/internal/works"
)

// Sourced is one made-from cell: its source Target and grace, the file
// filling it, and the file behind its source cell.
type Sourced struct {
	Work    works.WorkID
	Items   []works.ItemID
	Edition targets.Edition
	Target  targets.TargetID
	From    targets.TargetID
	// Grace is a "Downloaded, or made from" cell's wait for a download.
	Grace             time.Duration
	Counted, Released bool
	File              *File
	SourceFile        *File
}

// Source is the file it is made from, through derived_from when its
// source cell holds a made file.
func (s Sourced) Source() (File, bool) { return File{}, false }

// FilledByDownload reports whether a download fills the cell.
func (s Sourced) FilledByDownload() bool { return false }

// Located is a file with where it lies: its root, the Node holding that
// root, and its path in it.
type Located struct {
	ID    ID
	Root  roots.ID
	Rel   storing.Rel
	Node  role.NodeID
	Name  string
	Size  int64
	Media media.Record
	Read  release.Read
}

var errNotBuilt = errors.New("files' Conversion reads aren't built yet")

// Sourced is the made-from cells of ws, read in r. One read per batch.
func (f *Files) Sourced(context.Context, storage.Reader[file], works.Among[works.WorkID]) (map[works.WorkID][]Sourced, error) {
	return nil, errNotBuilt
}

func (f *Files) Located(context.Context, storage.Reader[file], ID) (Located, error) {
	return Located{}, errNotBuilt
}
