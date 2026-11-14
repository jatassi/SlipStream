package mbkind

import (
	"context"
	"slices"

	"github.com/jatassi/slipstream/internal/mediaserverlink/link"
	"github.com/jatassi/slipstream/internal/providers/jellyfin"
)

// users are the server's users, administrators first, so users[0] is the
// one whose view a lookup reads.
func (c *conn) users(ctx context.Context) ([]jellyfin.User, error) {
	us, err := c.c.Users(ctx)
	if err != nil {
		return nil, err
	}
	slices.SortStableFunc(us, func(a, b jellyfin.User) int {
		switch {
		case a.Policy.IsAdministrator == b.Policy.IsAdministrator:
			return 0
		case a.Policy.IsAdministrator:
			return -1
		}
		return 1
	})
	if len(us) == 0 || !us[0].Policy.IsAdministrator {
		return nil, failed("the server has no administrator")
	}
	return us, nil
}

// files are an item's file paths: its own and each version's.
func files(it jellyfin.Item) []link.ServerPath {
	var out []link.ServerPath
	for _, raw := range append([]string{it.Path}, sourcePaths(it)...) {
		p, err := link.ParsePath(raw)
		if err == nil && !slices.ContainsFunc(out, func(q link.ServerPath) bool { return q.Key() == p.Key() }) {
			out = append(out, p)
		}
	}
	return out
}

func sourcePaths(it jellyfin.Item) []string {
	out := make([]string, len(it.MediaSources))
	for i, ms := range it.MediaSources {
		out[i] = ms.Path
	}
	return out
}

func key(it jellyfin.Item) link.ItemKey {
	return link.ItemKey{Item: it.ID, Top: it.Top()}
}

// want is the paths a lookup asks for, in each library.
type want map[string][]link.ServerPath

func (w want) add(library string, p link.ServerPath) { w[library] = append(w[library], p) }

// found is the item at each path a lookup found, by link.ServerPath.Key.
type found map[string]jellyfin.Item

func (f found) at(p link.ServerPath) (jellyfin.Item, bool) {
	it, ok := f[p.Key()]
	return it, ok
}

// find is the items with a file at the paths w names, as user sees them,
// with user's data. Emby looks each path up; Jellyfin, which has no
// lookup by path, pages through each library once.
func (c *conn) find(ctx context.Context, user string, w want) (found, error) {
	out := found{}
	for library, paths := range w {
		if c.f.flavor == jellyfin.Emby {
			for _, p := range paths {
				items, err := c.c.ItemsAt(ctx, user, p.String())
				if err != nil {
					return nil, err
				}
				if len(items) > 0 {
					out[p.Key()] = items[0]
				}
			}
			continue
		}
		if err := c.page(ctx, user, library, paths, out); err != nil {
			return nil, err
		}
	}
	return out, nil
}

func (c *conn) page(ctx context.Context, user, library string, paths []link.ServerPath, out found) error {
	size := c.d.Timings.IndexPage
	left := len(paths)
	for start := 0; left > 0; start += size {
		page, err := c.c.Items(ctx, jellyfin.Query{User: user, Parent: library, Recursive: true, Types: jellyfin.VideoTypes, Start: start, Limit: size})
		if err != nil {
			return err
		}
		for _, it := range page.Items {
			for _, f := range files(it) {
				if _, dup := out[f.Key()]; !dup && slices.ContainsFunc(paths, func(p link.ServerPath) bool { return p.Key() == f.Key() }) {
					out[f.Key()] = it
					left--
				}
			}
		}
		if start+size >= page.Total {
			return nil
		}
	}
	return nil
}

// collections are the names of the collections each item is in, by item
// id.
func (c *conn) collections(ctx context.Context, user string) (of map[string][]string, ids map[string]string, err error) {
	page, err := c.c.Items(ctx, jellyfin.Query{User: user, Recursive: true, Types: []string{jellyfin.TypeBoxSet}})
	if err != nil {
		return nil, nil, err
	}
	of, ids = map[string][]string{}, map[string]string{}
	for _, col := range page.Items {
		ids[col.Name] = col.ID
		in, err := c.c.Items(ctx, jellyfin.Query{User: user, Parent: col.ID})
		if err != nil {
			return nil, nil, err
		}
		for _, it := range in.Items {
			of[it.ID] = append(of[it.ID], col.Name)
		}
	}
	for _, names := range of {
		slices.Sort(names)
	}
	return of, ids, nil
}
