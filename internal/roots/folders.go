package roots

import (
	"context"

	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/storing"
)

// Browse lists a folder on node, or node's places for an empty path. Every
// Node serves its folders, Storage or not. A Node the roster doesn't know
// is ErrNoNode, one out of reach ErrNodeAway, and a folder it can't list a
// *storing.FolderError.
func (r *Roots) Browse(ctx context.Context, _ identity.Admin, node role.NodeID, path string, showHidden bool) (storing.Listing, error) {
	l, err := r.cfg.Folders.Browse(ctx, node, path, showHidden)
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
