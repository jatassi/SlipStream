package imports

import (
	"context"

	"github.com/jatassi/slipstream/internal/grabs"
	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/works"
)

// GrabImports is grabs.Imports: each Grab's Import counted as Activity
// shows it. Stateless, so the composition root builds it before grabs.New.
type GrabImports struct{}

func (GrabImports) Of(ctx context.Context, r storage.Reader[file], ids []grabs.GrabID) (map[grabs.GrabID]grabs.ImportOf, error) {
	imps, err := importsOfGrabs(ctx, r, ids)
	if err != nil {
		return nil, err
	}
	out := make(map[grabs.GrabID]grabs.ImportOf, len(imps))
	for id, imp := range imps {
		out[grabs.GrabID{UUID: id}] = imp.progress()
	}
	return out, nil
}

// progress is im's files counted by how far they got.
func (im Import) progress() grabs.ImportOf {
	out := grabs.ImportOf{ID: im.ID.UUID, OneItemEach: true, Finished: !im.Finished.IsZero()}
	held := map[works.ItemID]bool{}
	for _, f := range im.Files {
		switch f.Outcome {
		case Pending:
			out.Reading++
			continue
		case Waiting:
			out.Waiting++
		case Sending:
			out.Sending++
		case Placed:
			out.Placed++
		case Placing:
		case LeftOut, NotBetter, Declined, FailedOut:
			continue
		}
		out.Planned++
		if len(f.Items) != 1 || held[f.Items[0]] {
			out.OneItemEach = false
			continue
		}
		held[f.Items[0]] = true
	}
	return out
}

// GrabsOf reads the Grab of each of ims whose download it is.
func (im *Imports) GrabsOf(ctx context.Context, _ identity.Admin, ims []Import) (map[grabs.GrabID]grabs.Handoff, error) {
	out := map[grabs.GrabID]grabs.Handoff{}
	err := im.cfg.Library.Read(ctx, func(ctx context.Context, r storage.Reader[file]) error {
		for _, imp := range ims {
			if imp.Grab.IsZero() {
				continue
			}
			h, err := grabs.Handed(ctx, r, imp.Grab)
			if err != nil {
				return err
			}
			out[imp.Grab] = h
		}
		return nil
	})
	return out, err
}
