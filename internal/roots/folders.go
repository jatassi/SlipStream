package roots

import (
	"context"

	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/storing"
)

// Browse lists a folder on node, or node's places for an empty path, for
// purpose, zero for any. Every Node serves its folders, Storage or not. A
// Node the roster doesn't know is ErrNoNode, one out of reach ErrNodeAway,
// a folder it can't list a *storing.FolderError, and for
// storing.PurposeImport a folder outside its library roots and download
// folders *storing.OutsideError.
func (r *Roots) Browse(ctx context.Context, _ identity.Admin, node role.NodeID, path string, showHidden bool, purpose storing.Purpose) (storing.Listing, error) {
	l, err := r.cfg.Folders.Browse(ctx, node, path, showHidden, purpose)
	return l, reached(err)
}

// Suggest lists up to 25 folders on node completing typed, in the
// separators typed.
func (r *Roots) Suggest(ctx context.Context, _ identity.Admin, node role.NodeID, typed string) ([]string, error) {
	paths, err := r.cfg.Folders.Suggest(ctx, node, typed)
	return paths, reached(err)
}

// MakeFolder makes the folder name in parent on node and answers its path,
// joined by the Node.
func (r *Roots) MakeFolder(ctx context.Context, _ identity.Admin, node role.NodeID, parent, name string) (string, error) {
	p, err := r.cfg.Folders.MakeFolder(ctx, node, parent, name)
	return p, reached(err)
}

// ListVideos lists the video files under path on node by the Admin's video
// extensions. A path outside node's library roots and download folders is
// *storing.OutsideError.
func (r *Roots) ListVideos(ctx context.Context, node role.NodeID, path string) (storing.Listed, error) {
	videos, err := r.cfg.Videos(ctx, r.cfg.Library)
	if err != nil {
		return storing.Listed{}, err
	}
	l, err := r.cfg.Folders.ListVideos(ctx, node, path, videos)
	return l, reached(err)
}

// RemoveSource removes the file at path on node once a Move placed it.
func (r *Roots) RemoveSource(ctx context.Context, node role.NodeID, path string) error {
	return reached(r.cfg.Folders.RemoveSource(ctx, node, path))
}
