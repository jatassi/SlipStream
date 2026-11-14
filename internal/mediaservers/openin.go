package mediaservers

import (
	"context"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/jatassi/slipstream/internal/files"
	"github.com/jatassi/slipstream/internal/mediaserverlink/link"
	"github.com/jatassi/slipstream/internal/mediaservers/internal/db"
	"github.com/jatassi/slipstream/internal/roots"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/storing"
	"github.com/jatassi/slipstream/internal/works"
)

// OpenIn is a link to a Work in one Media Server's web app.
type OpenIn struct {
	Server link.ServerID
	Kind   link.KindName
	// Label is the kind's, such as Plex.
	Label string
	Name  string
	URL   string
}

// missedWait is how long a path a lookup found no item at goes unasked,
// so reloading a Work page never pages through a Plex section each time.
const missedWait = 10 * time.Minute

// OpenIn links to work on each server whose shown mappings lead to one of
// its files, at the top item of the first file by path the path index
// holds. Jellyfin and Emby need a browser address. A server whose index
// holds none of the files is asked to look them up, and
// media-servers.items-found names the Work once one is found.
func (s service) OpenIn(ctx context.Context, work works.WorkID) ([]OpenIn, error) {
	var out []OpenIn
	asked := map[link.ServerID][]link.ServerPath{}
	err := s.m.cfg.Library.Read(ctx, func(ctx context.Context, r storage.Reader[storage.Library]) error {
		fs, err := s.m.cfg.WorkFiles(ctx, r, work)
		if err != nil || len(fs) == 0 {
			return err
		}
		slices.SortFunc(fs, func(a, b files.File) int { return strings.Compare(a.Path, b.Path) })
		q := db.New(r)
		folders, err := shownFolders(ctx, q)
		if err != nil {
			return err
		}
		rows, err := q.ListServers(ctx)
		if err != nil {
			return err
		}
		for _, row := range rows {
			l, ps, err := s.m.linkOf(ctx, q, row, fs, folders[row.ID])
			switch {
			case err != nil:
				return err
			case l.URL != "":
				out = append(out, l)
			case len(ps) > 0:
				asked[row.ID] = ps
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	for id, ps := range asked {
		if s.m.lookups.ask(id, work, ps, time.Now()) {
			s.m.senders.wake(id)
		}
	}
	return out, nil
}

// linkOf is row's link to the Work of fs, or the paths to look up when
// its index holds none of them, or neither when it shows no button or
// lists none of fs.
func (m *MediaServers) linkOf(ctx context.Context, q *db.Queries, row db.MediaserversServer, fs []files.File, folders map[roots.ID]link.ServerPath) (OpenIn, []link.ServerPath, error) {
	k, _ := m.cfg.Kinds.Of(row.Kind)
	app := m.cfg.PlexTV.Desktop.String()
	if k.About().Browser {
		if row.BrowserAddress == nil {
			return OpenIn{}, nil, nil
		}
		app = *row.BrowserAddress
	}
	paths, err := pathsOf(fs, folders)
	if err != nil || len(paths) == 0 {
		return OpenIn{}, nil, err
	}
	top, err := topOf(ctx, q, row.ID, paths)
	if err != nil || top == "" {
		return OpenIn{}, paths, err
	}
	return OpenIn{Server: row.ID, Kind: row.Kind, Label: k.About().Label, Name: row.Name, URL: k.OpenIn(app, row.Identity, top)}, nil, nil
}

// shownFolders is each server's folder for each root it shows.
func shownFolders(ctx context.Context, q *db.Queries) (map[link.ServerID]map[roots.ID]link.ServerPath, error) {
	ms, err := q.ListAllMappings(ctx)
	if err != nil {
		return nil, err
	}
	out := map[link.ServerID]map[roots.ID]link.ServerPath{}
	for _, mp := range ms {
		if !mp.Shown {
			continue
		}
		folder, err := link.ParsePath(*mp.ServerFolder)
		if err != nil {
			return nil, err
		}
		if out[mp.ServerID] == nil {
			out[mp.ServerID] = map[roots.ID]link.ServerPath{}
		}
		out[mp.ServerID][mp.RootID] = folder
	}
	return out, nil
}

// pathsOf is where a server with folders lists each of fs, in fs's order.
func pathsOf(fs []files.File, folders map[roots.ID]link.ServerPath) ([]link.ServerPath, error) {
	var out []link.ServerPath
	for _, f := range fs {
		folder, ok := folders[f.RootID]
		if !ok {
			continue
		}
		rel, err := storing.ParseRel(f.Path)
		if err != nil {
			return nil, err
		}
		out = append(out, folder.Join(rel))
	}
	return out, nil
}

// topOf is the top item of the first of paths the server's index holds,
// "" for none.
func topOf(ctx context.Context, q *db.Queries, server link.ServerID, paths []link.ServerPath) (string, error) {
	keys := make([]string, len(paths))
	for i, p := range paths {
		keys[i] = p.Key()
	}
	rows, err := q.ListItemsAt(ctx, db.ListItemsAtParams{ServerID: server, PathKeys: keys})
	if err != nil {
		return "", err
	}
	tops := make(map[string]string, len(rows))
	for _, row := range rows {
		tops[row.PathKey] = row.TopKey
	}
	for _, k := range keys {
		if top, ok := tops[k]; ok {
			return top, nil
		}
	}
	return "", nil
}

// lookups are the paths Open in asked each server's sender to look up,
// with the Works waiting on each, and the paths a lookup found no item
// at, until missedWait passes.
type lookups struct {
	mu     sync.Mutex
	asked  map[link.ServerID]map[string]lookup
	missed map[link.ServerID]map[string]time.Time
}

type lookup struct {
	path  link.ServerPath
	works []works.WorkID
}

func newLookups() *lookups {
	return &lookups{asked: map[link.ServerID]map[string]lookup{}, missed: map[link.ServerID]map[string]time.Time{}}
}

// ask has server look up ps for work, and reports whether a path is newly
// asked.
func (l *lookups) ask(server link.ServerID, work works.WorkID, ps []link.ServerPath, now time.Time) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.asked[server] == nil {
		l.asked[server] = map[string]lookup{}
	}
	missed := l.missed[server]
	fresh := false
	for _, p := range ps {
		k := p.Key()
		if at, ok := missed[k]; ok {
			if now.Sub(at) < missedWait {
				continue
			}
			delete(missed, k)
		}
		a, ok := l.asked[server][k]
		fresh = fresh || !ok
		a.path = p
		if !slices.Contains(a.works, work) {
			a.works = append(a.works, work)
		}
		l.asked[server][k] = a
	}
	return fresh
}

// take is up to n of the paths asked of server, in key order.
func (l *lookups) take(server link.ServerID, n int) []link.ServerPath {
	l.mu.Lock()
	defer l.mu.Unlock()
	keys := make([]string, 0, len(l.asked[server]))
	for k := range l.asked[server] {
		keys = append(keys, k)
	}
	slices.Sort(keys)
	out := make([]link.ServerPath, 0, min(n, len(keys)))
	for _, k := range keys[:min(n, len(keys))] {
		out = append(out, l.asked[server][k].path)
	}
	return out
}

// answered settles each asked path es names, and answers the Works one
// found an item for.
func (l *lookups) answered(server link.ServerID, es []link.IndexEntry, now time.Time) []string {
	l.mu.Lock()
	defer l.mu.Unlock()
	var out []string
	for _, e := range es {
		k := e.Path.Key()
		a, ok := l.asked[server][k]
		if !ok {
			continue
		}
		delete(l.asked[server], k)
		if e.Gone {
			if l.missed[server] == nil {
				l.missed[server] = map[string]time.Time{}
			}
			l.missed[server][k] = now
			continue
		}
		for _, w := range a.works {
			if id := w.String(); !slices.Contains(out, id) {
				out = append(out, id)
			}
		}
	}
	slices.Sort(out)
	return out
}
