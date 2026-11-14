package plexkind

import (
	"context"

	"github.com/jatassi/slipstream/internal/mediaserverlink/link"
	"github.com/jatassi/slipstream/internal/providers/plex"
)

// refresh is one partial refresh per folder of the changes, and an item
// refresh for each item beside a changed Sidecar (recorded run
// 2026-11-14, cases A and F). Plex takes the folder with backslashes or
// forward slashes and stores it as its OS names it.
var refresh = []step[link.Refresh]{{link.StepScanned, scanFolders}}

// rescan is one section refresh per library: library changes, and Plan's
// refresh of a section past Timings.SectionFolders.
var rescan = []step[link.Rescan]{{link.StepScanned, scanSections}}

func sidecar(k link.ChangeKind) bool {
	return k == link.ChangeSidecarAdded || k == link.ChangeSidecarReplaced || k == link.ChangeSidecarRemoved
}

func scanFolders(ctx context.Context, c *conn, _ link.Job, w *link.Refresh) (did, error) {
	var videos, sidecars []link.JobChange
	for _, ch := range w.Changes {
		if sidecar(ch.Kind) {
			sidecars = append(sidecars, ch)
		} else {
			videos = append(videos, ch)
		}
	}
	if err := c.refreshFolders(ctx, places(videos, newPath)); err != nil {
		return did{}, err
	}
	if err := c.refreshSidecars(ctx, sidecars); err != nil {
		return did{}, err
	}
	return did{outcomes: accepted(ids(w.Changes))}, nil
}

// refreshSidecars refreshes each item with a file in a changed Sidecar's
// folder, and the folder itself when it holds none.
func (c *conn) refreshSidecars(ctx context.Context, cs []link.JobChange) error {
	libraries, _ := groups(cs, func(link.JobChange) bool { return true })
	ps := places(cs, newPath)
	for _, lib := range libraries {
		var folders []link.ServerPath
		for _, p := range ps {
			if p.library == lib {
				folders = append(folders, p.folder)
			}
		}
		if err := c.refreshSidecarsIn(ctx, lib, folders); err != nil {
			return err
		}
	}
	return nil
}

func (c *conn) refreshSidecarsIn(ctx context.Context, lib string, folders []link.ServerPath) error {
	items, err := c.under(ctx, lib, folders)
	if err != nil {
		return err
	}
	for _, f := range folders {
		if len(items[f.Key()]) == 0 {
			if err := c.c.RefreshPath(ctx, plex.ID(lib), f.String()); err != nil {
				return err
			}
		}
		for _, m := range items[f.Key()] {
			if err := c.c.RefreshItem(ctx, m.RatingKey); err != nil {
				return err
			}
		}
	}
	return nil
}

func scanSections(ctx context.Context, c *conn, _ link.Job, w *link.Rescan) (did, error) {
	for _, lib := range w.Libraries {
		if err := c.c.RefreshSection(ctx, plex.ID(lib)); err != nil {
			return did{}, err
		}
	}
	return did{outcomes: accepted(w.Changes)}, nil
}

func ids(cs []link.JobChange) []link.ChangeID {
	out := make([]link.ChangeID, len(cs))
	for i, ch := range cs {
		out[i] = ch.ID
	}
	return out
}
