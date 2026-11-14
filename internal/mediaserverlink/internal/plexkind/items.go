package plexkind

import (
	"context"
	"slices"
	"time"

	"github.com/jatassi/slipstream/internal/mediaserverlink/link"
	"github.com/jatassi/slipstream/internal/providers/plex"
)

// place is one folder of one section.
type place struct {
	library string
	folder  link.ServerPath
}

// places are the distinct folders of cs's paths, in the order cs names
// them.
func places(cs []link.JobChange, path func(link.JobChange) link.ServerPath) []place {
	var out []place
	for _, ch := range cs {
		p := place{ch.Library, path(ch).Dir()}
		if !slices.ContainsFunc(out, p.same) {
			out = append(out, p)
		}
	}
	return out
}

func (p place) same(o place) bool { return p.library == o.library && p.folder.Key() == o.folder.Key() }

func newPath(ch link.JobChange) link.ServerPath { return ch.Path }

func oldPath(ch link.JobChange) link.ServerPath { return ch.From }

func (c *conn) refreshFolders(ctx context.Context, ps []place) error {
	for _, p := range ps {
		if err := c.c.RefreshPath(ctx, plex.ID(p.library), p.folder.String()); err != nil {
			return err
		}
	}
	return nil
}

// files are the files of m's versions. Plex stores each as an absolute
// path, so a part it can't parse names no file SlipStream knows.
func files(m plex.Metadata) []link.ServerPath {
	var out []link.ServerPath
	for _, md := range m.Media {
		for _, part := range md.Parts {
			if p, err := link.ParsePath(part.File); err == nil {
				out = append(out, p)
			}
		}
	}
	return out
}

func keyOf(m plex.Metadata) link.ItemKey {
	top := m.GrandparentRatingKey
	if top == "" {
		top = m.RatingKey
	}
	return link.ItemKey{Item: string(m.RatingKey), Top: string(top)}
}

func entry(library string, p link.ServerPath, m plex.Metadata) link.IndexEntry {
	return link.IndexEntry{Path: p, Library: library, Item: keyOf(m)}
}

// each pages through every item of section, calling visit for each until
// visit answers false. Plex has no lookup by path.
func (c *conn) each(ctx context.Context, section string, visit func(plex.Metadata) bool) error {
	size := c.t.IndexPage
	for _, typ := range plex.ItemTypes {
		for start := 0; ; start += size {
			page, err := c.c.SectionPage(ctx, plex.ID(section), typ, start, size)
			if err != nil {
				return err
			}
			for _, m := range page.Items {
				if !visit(m) {
					return nil
				}
			}
			if start+size >= page.Total {
				break
			}
		}
	}
	return nil
}

// at is the item holding a file at each of paths in section, by path key.
// A path with no item is missing from it.
func (c *conn) at(ctx context.Context, section string, paths []link.ServerPath) (map[string]plex.Metadata, error) {
	found := map[string]plex.Metadata{}
	if len(paths) == 0 {
		return found, nil
	}
	want := map[string]bool{}
	for _, p := range paths {
		want[p.Key()] = true
	}
	err := c.each(ctx, section, func(m plex.Metadata) bool {
		for _, f := range files(m) {
			if want[f.Key()] {
				found[f.Key()] = m
			}
		}
		return len(found) < len(want)
	})
	return found, err
}

// under is the items with a file in one of folders, by folder key.
func (c *conn) under(ctx context.Context, section string, folders []link.ServerPath) (map[string][]plex.Metadata, error) {
	found := map[string][]plex.Metadata{}
	err := c.each(ctx, section, func(m plex.Metadata) bool {
		for _, f := range folders {
			if slices.ContainsFunc(files(m), func(p link.ServerPath) bool { return p.Dir().Key() == f.Key() }) {
				found[f.Key()] = append(found[f.Key()], m)
			}
		}
		return true
	})
	return found, err
}

// holds is whether any item of section has a file under folder.
func (c *conn) holds(ctx context.Context, section string, folder link.ServerPath) (bool, error) {
	held := false
	err := c.each(ctx, section, func(m plex.Metadata) bool {
		held = slices.ContainsFunc(files(m), func(p link.ServerPath) bool { return p.Within(folder) })
		return !held
	})
	return held, err
}

// groups are cs's indexes by section, in the order cs names the sections.
func groups(cs []link.JobChange, keep func(link.JobChange) bool) (libraries []string, members map[string][]int) {
	members = map[string][]int{}
	for i, ch := range cs {
		if !keep(ch) {
			continue
		}
		if _, ok := members[ch.Library]; !ok {
			libraries = append(libraries, ch.Library)
		}
		members[ch.Library] = append(members[ch.Library], i)
	}
	return libraries, members
}

// snapshotOf is what an item carries: the account owner's watch data and
// the item's collections.
func snapshotOf(m plex.Metadata) link.Snapshot {
	u := link.UserData{
		Played: m.ViewCount > 0, PlayCount: m.ViewCount,
		Position: time.Duration(m.ViewOffset) * time.Millisecond, Rating: m.UserRating,
	}
	if m.LastViewedAt != 0 {
		u.LastPlayed = time.Unix(m.LastViewedAt, 0).UTC()
	}
	s := link.Snapshot{Item: keyOf(m), Users: []link.UserData{u}}
	for _, t := range m.Collections {
		s.Collections = append(s.Collections, t.Tag)
	}
	return s
}

func owner(s link.Snapshot) link.UserData {
	if len(s.Users) == 0 {
		return link.UserData{}
	}
	return s.Users[0]
}

// lost is what now didn't carry from was. A lost watch state covers the
// play count and last played date.
func lost(was, now link.Snapshot) []link.Loss {
	w, n := owner(was), owner(now)
	var out []link.Loss
	switch {
	case w.Played && !n.Played, w.Position > 0 && n.Position == 0 && !n.Played:
		out = append(out, link.LostWatchState)
	default:
		if n.PlayCount < w.PlayCount {
			out = append(out, link.LostPlayCount)
		}
		if !w.LastPlayed.IsZero() && n.LastPlayed.IsZero() {
			out = append(out, link.LostLastPlayed)
		}
	}
	if w.Rating != 0 && n.Rating != w.Rating {
		out = append(out, link.LostRating)
	}
	for _, col := range was.Collections {
		if !slices.Contains(now.Collections, col) {
			out = append(out, link.LostCollections)
			break
		}
	}
	return out
}
