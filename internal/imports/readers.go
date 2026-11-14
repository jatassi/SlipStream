package imports

import (
	"context"
	"slices"

	"github.com/jatassi/slipstream/internal/downloadclients"
	"github.com/jatassi/slipstream/internal/grabs"
	"github.com/jatassi/slipstream/internal/storage"
)

// Uses is downloadclients.Uses: grabs' Uses of each Torrent, each with the
// files its Grab's Import holds, and an importing one raised to waiting
// while a file of its Import waits for the Admin and none reads, sends or
// places. Stateless.
type Uses struct{ grabs.TorrentUses }

func (u Uses) Of(ctx context.Context, r storage.Reader[file], ids []downloadclients.ID) (map[downloadclients.ID][]downloadclients.Use, error) {
	uses, err := u.TorrentUses.Of(ctx, r, ids)
	if err != nil {
		return nil, err
	}
	var grabbed []grabs.GrabID
	for _, us := range uses {
		for _, use := range us {
			grabbed = append(grabbed, grabs.GrabID{UUID: use.Grab})
		}
	}
	imps, err := importsOfGrabs(ctx, r, grabbed)
	if err != nil {
		return nil, err
	}
	for id, us := range uses {
		for i, use := range us {
			imp, ok := imps[use.Grab]
			if !ok {
				continue
			}
			uses[id][i].Files = imp.useFiles()
			if use.Stage == downloadclients.UseImporting && imp.waits() {
				uses[id][i].Stage = downloadclients.UseWaiting
			}
		}
	}
	return uses, nil
}

// useFiles is each file of im that is or may be in the library, by its
// index in the download.
func (im Import) useFiles() []downloadclients.UseFile {
	var out []downloadclients.UseFile
	for _, f := range im.Files {
		switch f.Outcome {
		case Pending, Waiting, Sending, Placing, Placed:
			out = append(out, downloadclients.UseFile{Index: *f.SourceIndex, Items: f.Items})
		case LeftOut, NotBetter, Declined, FailedOut:
		}
	}
	return out
}

// waits reports whether every file of im still at work waits for the
// Admin: one waits, and none reads, sends or places.
func (im Import) waits() bool {
	asked := func(f File) bool {
		return f.Outcome == Waiting || f.Outcome == Pending && f.Hold != nil && f.Hold.offered()
	}
	busy := func(f File) bool { return !f.Outcome.Final() && !asked(f) }
	return slices.ContainsFunc(im.Files, asked) && !slices.ContainsFunc(im.Files, busy)
}
