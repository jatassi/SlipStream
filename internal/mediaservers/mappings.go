package mediaservers

import (
	"context"
	"errors"
	"slices"
	"strings"

	"github.com/jatassi/slipstream/internal/mediaserverlink/link"
	"github.com/jatassi/slipstream/internal/mediaservers/internal/db"
	"github.com/jatassi/slipstream/internal/nodes"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/roots"
	"github.com/jatassi/slipstream/internal/storage"
)

// libraries reads the server's libraries with its credential: from the
// reaching Node while it is in step, else from the Controller at the
// fallback address when a fallback is available. The Node held the
// credential for the call, so it is recorded as a holder.
func (m *MediaServers) libraries(ctx context.Context, sv saved, names map[role.NodeID]nodes.Named) ([]link.Library, error) {
	r, f := m.routesOf(sv, names)
	row := sv.row
	if row.ReachingNodeID == m.cfg.Roster.Self() || m.cfg.Roster.InStep(row.ReachingNodeID) {
		c, err := m.check(ctx, r, row.Identity)
		if err != nil {
			return nil, err
		}
		err = m.cfg.Library.Write(ctx, func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
			return m.recordFetch(ctx, db.New(tx), row.ID, row.ReachingNodeID, row.SettingsVersion)
		})
		return c.Libraries, err
	}
	if row.Fallback != "available" {
		return nil, &UnreachedError{Label: r.label, Node: r.name, Address: r.address}
	}
	c, err := m.direct(f, row.Identity).Check(ctx)
	if err != nil {
		return nil, failure(err, f)
	}
	return c.Libraries, nil
}

func libraryViews(libs []link.Library) []LibraryView {
	out := make([]LibraryView, len(libs))
	for i, l := range libs {
		out[i] = LibraryView{Key: l.Key, Name: l.Name, Folders: []string{}}
		for _, f := range l.Folders {
			out[i].Folders = append(out[i].Folders, f.String())
		}
	}
	return out
}

// within is the library and library folder that folder sits inside.
func within(folder link.ServerPath, libs []link.Library) (link.Library, link.ServerPath, bool) {
	for _, l := range libs {
		for _, f := range l.Folders {
			if folder.Within(f) {
				return l, f, true
			}
		}
	}
	return link.Library{}, link.ServerPath{}, false
}

// prefill is root's mapping before the Admin saves one: the library
// folder whose path is the root's, else the one that shares the most
// trailing folder names with it, marked a guess. A root that shares none
// gets no folder. Pure.
func prefill(root roots.Root, libs []link.Library) (link.ServerPath, bool, bool) {
	at, err := link.ParsePath(root.Path)
	if err != nil {
		return link.ServerPath{}, false, false
	}
	var best link.ServerPath
	most := 0
	for _, l := range libs {
		for _, f := range l.Folders {
			if f.Key() == at.Key() && f.Windows() == at.Windows() {
				return f, false, true
			}
			if n := sharedTail(at, f); n > most {
				best, most = f, n
			}
		}
	}
	return best, true, most > 0
}

// sharedTail counts the trailing folder names a and b share, case folded,
// since a root and a server may see one share through different systems.
func sharedTail(a, b link.ServerPath) int {
	as, bs := names(a), names(b)
	n := 0
	for n < len(as) && n < len(bs) && strings.EqualFold(as[len(as)-1-n], bs[len(bs)-1-n]) {
		n++
	}
	return n
}

func names(p link.ServerPath) []string {
	return strings.FieldsFunc(p.String(), func(r rune) bool { return r == '/' || r == '\\' || r == ':' })
}

// mappingView is root's mapping on server as saved, or pre-filled when
// none is.
func (m *MediaServers) mappingView(server string, root roots.Root, w world, mp *db.MediaserversMapping, libs []link.Library) MappingView {
	v := MappingView{
		RootID: root.ID.String(), RootName: w.words[root.ID],
		RootNode: NodeRef{ID: root.Node.String(), Name: w.names[root.Node].Name}, Shown: true,
	}
	if mp == nil {
		v.Words = noMappingWords(server, v.RootName)
		folder, guess, ok := prefill(root, libs)
		if !ok {
			return v
		}
		l, lf, _ := within(folder, libs)
		v.ServerFolder, v.Guess = folder.String(), guess
		v.Library = &Library{Key: l.Key, Name: l.Name, Folder: lf.String()}
		return v
	}
	v.Shown, v.Guess = mp.Shown, mp.Guess
	v.Confirmed, v.Scanned = !mp.ConfirmedAt.IsZero(), !mp.ScannedAt.IsZero()
	if mp.Shown {
		v.ServerFolder = deref(mp.ServerFolder)
		v.Library = &Library{Key: deref(mp.LibraryKey), Name: deref(mp.LibraryName), Folder: deref(mp.LibraryFolder)}
		if !v.Confirmed {
			v.Words = server + " holds changes for files in " + v.RootName + " until its Path mapping is confirmed. Check the folder, then save the mappings."
		}
	}
	return v
}

func (m *MediaServers) mappingViews(server link.ServerID, name string, w world, libs []link.Library) []MappingView {
	out := []MappingView{}
	for _, root := range w.roots {
		var mp *db.MediaserversMapping
		if saved, ok := w.mappings[server][root.ID]; ok {
			mp = &saved
		}
		out = append(out, m.mappingView(name, root, w, mp, libs))
	}
	return out
}

func (m *MediaServers) worldOf(ctx context.Context) (world, error) {
	var w world
	err := m.cfg.Library.Read(ctx, func(ctx context.Context, r storage.Reader[storage.Library]) error {
		var err error
		w, err = m.world(ctx, r)
		return err
	})
	return w, err
}

// Libraries is the server's libraries and every library root's mapping,
// pre-filled where none is saved.
func (s service) Libraries(ctx context.Context, id link.ServerID) (Libraries, error) {
	sv, names, err := s.m.loaded(ctx, id)
	if err != nil {
		return Libraries{}, err
	}
	libs, err := s.m.libraries(ctx, sv, names)
	if err != nil {
		return Libraries{}, err
	}
	w, err := s.m.worldOf(ctx)
	if err != nil {
		return Libraries{}, err
	}
	return Libraries{Libraries: libraryViews(libs), Mappings: s.m.mappingViews(id, sv.row.Name, w, libs)}, nil
}

// checkMappings refuses each shown mapping whose folder isn't inside one
// of the server's library folders, and names the folders it has. Pure.
func checkMappings(server string, ms []Mapping, words map[roots.ID]string, libs []link.Library) error {
	var outside []Outside
	for i, mp := range ms {
		if !mp.Shown {
			continue
		}
		if p, err := link.ParsePath(mp.Folder); err == nil {
			if _, _, ok := within(p, libs); ok {
				continue
			}
		}
		outside = append(outside, Outside{Index: i, Root: words[mp.Root], Folder: mp.Folder})
	}
	if len(outside) == 0 {
		return nil
	}
	var found []string
	for _, l := range libs {
		for _, f := range l.Folders {
			found = append(found, f.String())
		}
	}
	return &OutsideLibrariesError{Server: server, Outside: outside, Found: found}
}

// SaveMappings saves the server's mappings, the Admin's word for each, as
// confirmed at each root's place now. A root left out has no mapping. A
// mapping newly confirmed or newly shown queues a whole library refresh
// of its root, which covers the files the server didn't hear about.
func (s service) SaveMappings(ctx context.Context, id link.ServerID, ms []Mapping) ([]MappingView, error) {
	sv, names, err := s.m.loaded(ctx, id)
	if err != nil {
		return nil, err
	}
	w, err := s.m.worldOf(ctx)
	if err != nil {
		return nil, err
	}
	places := map[roots.ID]int64{}
	for _, root := range w.roots {
		places[root.ID] = root.Place
	}
	for _, mp := range ms {
		if _, ok := places[mp.Root]; !ok {
			return nil, ErrRootNotFound
		}
	}
	libs, err := s.m.libraries(ctx, sv, names)
	if err != nil {
		return nil, err
	}
	if err := checkMappings(sv.row.Name, ms, w.words, libs); err != nil {
		return nil, err
	}
	if err := s.m.storeMappings(ctx, id, ms, places, libs); err != nil {
		return nil, err
	}
	s.m.changed(ctx, id)
	if w, err = s.m.worldOf(ctx); err != nil {
		return nil, err
	}
	return s.m.mappingViews(id, sv.row.Name, w, libs), nil
}

// storeMappings replaces the server's mappings with ms in one Write.
func (m *MediaServers) storeMappings(ctx context.Context, id link.ServerID, ms []Mapping, places map[roots.ID]int64, libs []link.Library) error {
	return m.cfg.Library.Write(ctx, func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
		q := db.New(tx)
		before, err := q.ListMappings(ctx, id)
		if err != nil {
			return err
		}
		now := storage.Now()
		for _, old := range before {
			if slices.ContainsFunc(ms, func(mp Mapping) bool { return mp.Root == old.RootID }) {
				continue
			}
			if err := q.DeleteMapping(ctx, db.DeleteMappingParams{ServerID: id, RootID: old.RootID}); err != nil {
				return err
			}
		}
		for _, mp := range ms {
			if err := m.saveMapping(ctx, q, id, mp, places[mp.Root], libs, before, now); err != nil {
				return err
			}
		}
		return nil
	})
}

func (m *MediaServers) saveMapping(ctx context.Context, q *db.Queries, id link.ServerID, mp Mapping, place int64, libs []link.Library, before []db.MediaserversMapping, now storage.Time) error {
	p := db.UpsertMappingParams{ServerID: id, RootID: mp.Root, Shown: mp.Shown, ConfirmedAt: now, ConfirmedPlace: &place}
	if mp.Shown {
		folder, err := link.ParsePath(mp.Folder)
		if err != nil {
			return err
		}
		l, lf, _ := within(folder, libs)
		f, lfs := folder.String(), lf.String()
		p.ServerFolder, p.LibraryKey, p.LibraryName, p.LibraryFolder = &f, &l.Key, &l.Name, &lfs
	}
	if err := q.UpsertMapping(ctx, p); err != nil {
		return err
	}
	i := slices.IndexFunc(before, func(b db.MediaserversMapping) bool { return b.RootID == mp.Root })
	fresh := i < 0 || !before[i].Shown || before[i].ConfirmedAt.IsZero()
	if !mp.Shown || !fresh {
		return nil
	}
	return q.QueueLibrary(ctx, db.QueueLibraryParams{ID: link.ChangeID{UUID: storage.NewUUID()}, ServerID: id, RootID: mp.Root, At: now})
}

// unconfirm clears the confirmation of every mapping whose root moved to
// another place since the Admin confirmed it: its changes wait until the
// Admin confirms the mapping again.
func (m *MediaServers) unconfirm(ctx context.Context) {
	var servers []link.ServerID
	err := m.cfg.Library.Write(ctx, func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
		servers = nil
		all, err := roots.All(ctx, tx)
		if err != nil {
			return err
		}
		places := map[roots.ID]int64{}
		for _, root := range all {
			places[root.ID] = root.Place
		}
		q := db.New(tx)
		mappings, err := q.ListAllMappings(ctx)
		if err != nil {
			return err
		}
		for _, mp := range mappings {
			if mp.ConfirmedPlace == nil || *mp.ConfirmedPlace == places[mp.RootID] {
				continue
			}
			if err := q.Unconfirm(ctx, db.UnconfirmParams{ServerID: mp.ServerID, RootID: mp.RootID}); err != nil {
				return err
			}
			if !slices.Contains(servers, mp.ServerID) {
				servers = append(servers, mp.ServerID)
			}
		}
		return nil
	})
	switch {
	case err != nil && !errors.Is(err, context.Canceled):
		m.cfg.Log.Error("unconfirm the Path mappings of re-homed roots", "err", err.Error())
	case len(servers) > 0:
		m.cfg.Log.Info("unconfirmed the Path mappings of re-homed roots", "servers", len(servers))
		m.changed(ctx, servers...)
	}
}
