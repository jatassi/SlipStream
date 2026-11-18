package mediaservers

import (
	"context"

	"github.com/jatassi/slipstream/internal/files"
	"github.com/jatassi/slipstream/internal/roots"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/storing"
)

// SidecarKind is what happened to a Sidecar beside a library file.
type SidecarKind uint8

const (
	SidecarAdded SidecarKind = iota + 1
	SidecarReplaced
	SidecarRemoved
)

var sidecarKinds = map[SidecarKind]files.PathChangeKind{
	SidecarAdded:    files.PathAdded,
	SidecarReplaced: files.PathReplaced,
	SidecarRemoved:  files.PathRemoved,
}

// SidecarChange is one Sidecar change at the Sidecar's own root-relative
// path. Sidecar is its own id, never its video's.
type SidecarChange struct {
	Kind    SidecarKind
	Root    roots.ID
	Path    storing.Rel
	Sidecar storage.UUID
}

// SidecarsChanged folds each change into the queue of every server whose
// mapping shows its root, in tx, as Changed does. Each row's file is the
// Sidecar's id, so no open carry of its video absorbs it.
func (m *MediaServers) SidecarsChanged(ctx context.Context, tx *storage.Tx[storage.Library], cs []SidecarChange) error {
	out := make([]files.PathChange, len(cs))
	for i, c := range cs {
		out[i] = files.PathChange{Kind: sidecarKinds[c.Kind], File: files.ID{UUID: c.Sidecar}, Root: c.Root, Path: c.Path}
	}
	return m.Changed(ctx, tx, out)
}
