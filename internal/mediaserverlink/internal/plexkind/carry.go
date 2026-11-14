package plexkind

import (
	"context"
	"slices"
	"time"

	"github.com/jatassi/slipstream/internal/mediaserverlink/link"
	"github.com/jatassi/slipstream/internal/providers/plex"
)

// carry is the Plex moved change (recorded run 2026-11-14, cases B to E).
// The destination folders are refreshed first, and the source folders
// that differ only once the item is confirmed at its new path: with
// autoEmptyTrash on, a refresh of the vacated folder first deletes the
// item and its watch data (case E). Plex keeps no key across folders, so
// the item is looked up again by its new path and compared with the
// snapshot. A same-folder rename is one refresh and one comparison.
var carry = []step[link.Carry]{
	{link.StepSnapshotted, snapshot},
	{link.StepScanned, scanDestinations},
	{link.StepConfirmed, confirmAtNewPath},
	{link.StepSourceScanned, scanSources},
}

// snapshot reads each change's item at its old path. A change with none
// keeps a nil Snapshot and ends converted.
func snapshot(ctx context.Context, c *conn, _ link.Job, w *link.Carry) (did, error) {
	libraries, members := groups(w.Changes, func(ch link.JobChange) bool { return ch.Snapshot == nil })
	for _, lib := range libraries {
		paths := make([]link.ServerPath, len(members[lib]))
		for n, i := range members[lib] {
			paths[n] = w.Changes[i].From
		}
		found, err := c.at(ctx, lib, paths)
		if err != nil {
			return did{}, err
		}
		for _, i := range members[lib] {
			if m, ok := found[w.Changes[i].From.Key()]; ok {
				s := snapshotOf(m)
				w.Changes[i].Snapshot = &s
			}
		}
	}
	return did{}, nil
}

func scanDestinations(ctx context.Context, c *conn, _ link.Job, w *link.Carry) (did, error) {
	return did{}, c.refreshFolders(ctx, places(w.Changes, newPath))
}

// confirmAtNewPath waits, up to Timings.MovedConfirm after the
// destinations were refreshed, until an item holds each snapshotted
// change's new path.
func confirmAtNewPath(ctx context.Context, c *conn, j link.Job, w *link.Carry) (did, error) {
	found, err := c.atNew(ctx, w.Changes)
	if err != nil {
		return did{}, err
	}
	want := 0
	for _, ch := range w.Changes {
		if ch.Snapshot != nil {
			want++
		}
	}
	if len(found) < want && time.Now().Before(j.At.Add(c.t.MovedConfirm)) {
		return did{waiting: true, wait: later(c.t.StepPoll)}, nil
	}
	return did{}, nil
}

// scanSources refreshes each source folder no destination shares, then
// compares each item at its new path with its snapshot.
func scanSources(ctx context.Context, c *conn, _ link.Job, w *link.Carry) (did, error) {
	dest := places(w.Changes, newPath)
	src := slices.DeleteFunc(places(w.Changes, oldPath), func(p place) bool { return slices.ContainsFunc(dest, p.same) })
	if err := c.refreshFolders(ctx, src); err != nil {
		return did{}, err
	}
	found, err := c.atNew(ctx, w.Changes)
	if err != nil {
		return did{}, err
	}
	var d did
	for _, ch := range w.Changes {
		m, ok := found[ch.Path.Key()]
		if ok {
			d.learned = append(d.learned, entry(ch.Library, ch.Path, m))
		}
		if ch.Snapshot == nil {
			d.outcomes = append(d.outcomes, link.Outcome{Change: ch.ID, Result: link.ResultConverted})
			continue
		}
		var now link.Snapshot
		if ok {
			now = snapshotOf(m)
		}
		d.outcomes = append(d.outcomes, link.Outcome{Change: ch.ID, Result: link.ResultAccepted, Lost: lost(*ch.Snapshot, now)})
	}
	return d, nil
}

// atNew is the item at each snapshotted change's new path, by path key.
// The snapshot's item is read first, since a rename in its folder keeps
// it (case B), and the section is listed only for the rest.
func (c *conn) atNew(ctx context.Context, cs []link.JobChange) (map[string]plex.Metadata, error) {
	found := map[string]plex.Metadata{}
	var rest []link.JobChange
	for _, ch := range cs {
		if ch.Snapshot == nil {
			continue
		}
		m, ok, err := c.c.Item(ctx, plex.ID(ch.Snapshot.Item.Item))
		if err != nil {
			return nil, err
		}
		if ok && slices.ContainsFunc(files(m), func(p link.ServerPath) bool { return p.Key() == ch.Path.Key() }) {
			found[ch.Path.Key()] = m
			continue
		}
		rest = append(rest, ch)
	}
	libraries, members := groups(rest, func(link.JobChange) bool { return true })
	for _, lib := range libraries {
		paths := make([]link.ServerPath, len(members[lib]))
		for n, i := range members[lib] {
			paths[n] = rest[i].Path
		}
		more, err := c.at(ctx, lib, paths)
		if err != nil {
			return nil, err
		}
		for k, m := range more {
			found[k] = m
		}
	}
	return found, nil
}
