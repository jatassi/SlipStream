package downloading

import (
	"context"
	"os"
	"path/filepath"
	"strings"

	dlv1 "github.com/jatassi/slipstream/internal/pb/slipstream/downloading/v1"
	"github.com/jatassi/slipstream/internal/rpc"
)

// Look is whether each entry's file lies under the folder at its size. A
// name is a path under the folder with either separator, since the
// Controller may have recorded it from a Node on another system.
func (n *Node) Look(ctx context.Context, req *dlv1.LookRequest) (*dlv1.LookResponse, error) {
	folder := req.GetFolder()
	if !filepath.IsAbs(folder) {
		return nil, rpc.Errorf(rpc.CodeInvalidArgument, "the folder %q isn't an absolute path on this Node", folder)
	}
	found := make([]bool, len(req.GetEntries()))
	for i, e := range req.GetEntries() {
		if err := ctx.Err(); err != nil {
			return nil, err //nolint:wrapcheck // the call's own end
		}
		name := filepath.FromSlash(strings.ReplaceAll(e.GetName(), `\`, "/"))
		if !filepath.IsLocal(name) {
			return nil, rpc.Errorf(rpc.CodeInvalidArgument, "the name %q isn't a path under the folder", e.GetName())
		}
		info, err := os.Stat(filepath.Join(folder, name))
		found[i] = err == nil && info.Mode().IsRegular() && info.Size() == e.GetSize()
	}
	return &dlv1.LookResponse{Found: found}, nil
}
