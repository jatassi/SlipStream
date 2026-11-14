package mediaservers

import (
	"context"
	"slices"
	"time"

	"github.com/jatassi/slipstream/internal/files"
	"github.com/jatassi/slipstream/internal/mediaserverlink/link"
	"github.com/jatassi/slipstream/internal/mediaservers/internal/db"
	"github.com/jatassi/slipstream/internal/roots"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/storing"
)

// indexJob is the one Job a pass with no change to send spends on the
// path index: a Lookup of the entries a push marked stale, else the next
// page of a listing due. A library is listed once it is mapped, and again
// Timings.IndexAge after its last listing ended. next is when a listing
// is next due, zero for none.
func (m *MediaServers) indexJob(ctx context.Context, q *db.Queries, d link.Due) (job *link.Job, next time.Time, err error) {
	stale, err := q.ListStaleItems(ctx, db.ListStaleItemsParams{ServerID: d.Server.ID, N: int64(m.cfg.Timings.IndexPage)})
	if err != nil {
		return nil, time.Time{}, err
	}
	if len(stale) > 0 {
		ps := make([]link.ServerPath, len(stale))
		for i, s := range stale {
			if ps[i], err = link.ParsePath(s); err != nil {
				return nil, time.Time{}, err
			}
		}
		return &link.Job{Server: d.Server, Work: link.Lookup{Paths: ps}, Step: link.StepNone}, time.Time{}, nil
	}
	rows, err := q.ListListings(ctx, d.Server.ID)
	if err != nil {
		return nil, time.Time{}, err
	}
	listings := map[string]db.MediaserversListing{}
	for _, l := range rows {
		listings[l.LibraryKey] = l
	}
	for _, lib := range mapped(d.Mappings) {
		l := listings[lib]
		if !l.StartedAt.IsZero() {
			return &link.Job{Server: d.Server, Work: link.Index{Library: lib, Cursor: l.Cursor}, Step: link.StepNone}, time.Time{}, nil
		}
		due := l.ListedAt.Time().Add(m.cfg.Timings.IndexAge)
		if l.ListedAt.IsZero() || !due.After(d.Now) {
			err := q.StartListing(ctx, db.StartListingParams{ServerID: d.Server.ID, LibraryKey: lib, StartedAt: storage.FromTime(d.Now)})
			return &link.Job{Server: d.Server, Work: link.Index{Library: lib}, Step: link.StepNone}, time.Time{}, err
		}
		next = earliest(next, due)
	}
	return nil, next, nil
}

// mapped are the libraries of the server's shown, confirmed mappings.
func mapped(ms []link.PlanMapping) []string {
	var out []string
	for _, mp := range ms {
		if mp.Shown && mp.Confirmed && !slices.Contains(out, mp.Library) {
			out = append(out, mp.Library)
		}
	}
	slices.Sort(out)
	return out
}

func earliest(a, b time.Time) time.Time {
	if a.IsZero() || (!b.IsZero() && b.Before(a)) {
		return b
	}
	return a
}

// indexed writes an index Job's answer: what it learned, and the
// listing's next page, or its end, which forgets every entry of the
// library the listing didn't see.
func (m *MediaServers) indexed(ctx context.Context, tx *storage.Tx[storage.Library], s sending, out link.Advanced) error {
	now := time.Now()
	if err := m.learn(ctx, tx, s, out.Learned, now); err != nil {
		return err
	}
	w, ok := out.Job.Work.(link.Index)
	if !ok {
		return nil
	}
	q := db.New(tx)
	if !out.Done {
		return q.SetListingCursor(ctx, db.SetListingCursorParams{ServerID: s.id, LibraryKey: w.Library, Cursor: w.Cursor})
	}
	if err := q.PruneLibrary(ctx, db.PruneLibraryParams{ServerID: s.id, LibraryKey: w.Library}); err != nil {
		return err
	}
	return q.EndListing(ctx, db.EndListingParams{ServerID: s.id, LibraryKey: w.Library, ListedAt: storage.FromTime(now)})
}

func isIndex(j link.Job) bool {
	switch j.Work.(type) {
	case link.Index, link.Lookup:
		return true
	}
	return false
}

// place is a server path as a root-relative path under one of the
// server's shown mappings.
type place struct {
	root roots.ID
	rel  storing.Rel
	ok   bool
}

// learn writes es in the server's path index, each entry naming the
// library file whose bytes sit at its path. A gone entry is forgotten.
func (m *MediaServers) learn(ctx context.Context, tx *storage.Tx[storage.Library], s sending, es []link.IndexEntry, now time.Time) error {
	places := make([]place, len(es))
	rels := map[roots.ID][]storing.Rel{}
	for i, e := range es {
		if e.Gone {
			continue
		}
		for _, mp := range s.mappings {
			if rel, ok := e.Path.Rel(mp.Folder); mp.Shown && ok {
				places[i] = place{mp.Root, rel, true}
				rels[mp.Root] = append(rels[mp.Root], rel)
				break
			}
		}
	}
	ids := map[roots.ID]map[storing.Rel]files.ID{}
	for root, rs := range rels {
		var err error
		if ids[root], err = m.cfg.FileIDs(ctx, tx, root, rs); err != nil {
			return err
		}
	}
	q := db.New(tx)
	for i, e := range es {
		if e.Gone {
			if err := q.DeleteItem(ctx, db.DeleteItemParams{ServerID: s.id, PathKey: e.Path.Key()}); err != nil {
				return err
			}
			continue
		}
		var file files.ID
		if p := places[i]; p.ok {
			file = ids[p.root][p.rel]
		}
		err := q.UpsertItem(ctx, db.UpsertItemParams{
			ServerID: s.id, PathKey: e.Path.Key(), ServerPath: e.Path.String(), LibraryKey: e.Library, FileID: file,
			ItemKey: e.Item.Item, TopKey: e.Item.Top, SeenAt: storage.FromTime(now),
		})
		if err != nil {
			return err
		}
	}
	return nil
}
