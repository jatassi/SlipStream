package roots

import (
	"context"

	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/transfers/source"
)

// LibraryReadiness is the readiness of each Library ref among refs, read
// in rd: WrittenOff for a file in a root whose Node's Storage the Admin
// wrote off, else Unknown, so the sender's answer decides. Other refs are
// left out.
func LibraryReadiness(ctx context.Context, rd storage.Reader[file], refs []source.Ref) (map[source.Ref]source.Readiness, error) {
	out := map[source.Ref]source.Readiness{}
	var lost map[ID]bool
	for _, ref := range refs {
		lib, ok := ref.(source.Library)
		if !ok {
			continue
		}
		if lost == nil {
			var err error
			if lost, err = LostIn(ctx, rd); err != nil {
				return nil, err
			}
		}
		root, err := storage.UUIDFromBytes(lib.Root[:])
		if err != nil {
			return nil, err
		}
		r := source.Readiness{Name: lib.Rel}
		if lost[ID{UUID: root}] {
			r.State = source.WrittenOff
		}
		out[ref] = r
	}
	return out, nil
}
