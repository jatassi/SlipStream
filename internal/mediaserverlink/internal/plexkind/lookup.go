package plexkind

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/jatassi/slipstream/internal/mediaserverlink/link"
	"github.com/jatassi/slipstream/internal/providers/plex"
)

// lookup reads the item at each path, in the section whose folders hold
// it. A path no item holds is learned gone.
func (c *conn) lookup(ctx context.Context, j link.Job, w link.Lookup) (link.Advanced, error) {
	libs, err := c.libraries(ctx)
	if err != nil {
		return link.Advanced{}, err
	}
	a := link.Advanced{Job: j, Done: true}
	left := w.Paths
	for _, l := range libs {
		var in, out []link.ServerPath
		for _, p := range left {
			if within(p, l.Folders) {
				in = append(in, p)
			} else {
				out = append(out, p)
			}
		}
		left = out
		found, err := c.at(ctx, l.Key, in)
		if err != nil {
			return link.Advanced{}, err
		}
		for _, p := range in {
			if m, ok := found[p.Key()]; ok {
				a.Learned = append(a.Learned, entry(l.Key, p, m))
			} else {
				a.Learned = append(a.Learned, link.IndexEntry{Path: p, Library: l.Key, Gone: true})
			}
		}
	}
	for _, p := range left {
		a.Learned = append(a.Learned, link.IndexEntry{Path: p, Gone: true})
	}
	return a, nil
}

func within(p link.ServerPath, folders []link.ServerPath) bool {
	for _, f := range folders {
		if p.Within(f) {
			return true
		}
	}
	return false
}

// index reads one page of a section's items. The cursor is the listing
// type's place in plex.ItemTypes and the offset in it.
func (c *conn) index(ctx context.Context, j link.Job, w link.Index) (link.Advanced, error) {
	typ, start, err := cursorOf(w.Cursor)
	if err != nil {
		return link.Advanced{}, err
	}
	size := c.t.IndexPage
	page, err := c.c.SectionPage(ctx, plex.ID(w.Library), plex.ItemTypes[typ], start, size)
	if err != nil {
		return link.Advanced{}, err
	}
	a := link.Advanced{Job: j}
	for _, m := range page.Items {
		for _, f := range files(m) {
			a.Learned = append(a.Learned, entry(w.Library, f, m))
		}
	}
	switch {
	case start+size < page.Total:
		w.Cursor = cursor(typ, start+size)
	case typ+1 < len(plex.ItemTypes):
		w.Cursor = cursor(typ+1, 0)
	default:
		w.Cursor = ""
	}
	a.Job.Work, a.Done = w, w.Cursor == ""
	return a, nil
}

func cursor(typ, start int) string { return strconv.Itoa(typ) + ":" + strconv.Itoa(start) }

func cursorOf(s string) (typ, start int, err error) {
	if s == "" {
		return 0, 0, nil
	}
	t, o, ok := strings.Cut(s, ":")
	typ, terr := strconv.Atoi(t)
	start, oerr := strconv.Atoi(o)
	if !ok || terr != nil || oerr != nil || typ < 0 || typ >= len(plex.ItemTypes) || start < 0 {
		return 0, 0, fmt.Errorf("%q isn't a Plex index cursor", s)
	}
	return typ, start, nil
}
