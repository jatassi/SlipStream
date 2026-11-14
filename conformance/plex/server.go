package plex

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"slices"

	"github.com/jatassi/slipstream/conformance/internal/mediaservers"
	"github.com/jatassi/slipstream/internal/mediaserverlink/link"
	"github.com/jatassi/slipstream/internal/providers/plex"
)

// server drives a Plex through the wire client, the way the Plex kind
// does: a folder refresh per change, the destination before the source,
// an item refresh for a removed Sidecar.
type server struct {
	c     *plex.Client
	tg    Target
	scans map[string]scanMark
}

// scanMark is what a section looked like when Scan asked for a scan.
// scannedAt is whole seconds, so a scan that starts and ends in the second
// of the last one shows only as refreshing seen true.
type scanMark struct {
	scannedAt int64
	refreshed bool
}

var _ mediaservers.Server = (*server)(nil)

func (s *server) Reach(ctx context.Context) (link.Identity, error) {
	id, err := s.c.Identity(ctx)
	if err != nil {
		return link.Identity{}, err
	}
	v, err := link.ParseVersion(id.Version)
	if err != nil {
		return link.Identity{}, err
	}
	return link.Identity{ID: id.MachineIdentifier, Version: v}, nil
}

func (s *server) Libraries(ctx context.Context) ([]link.Library, error) {
	sections, err := s.c.Sections(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]link.Library, len(sections))
	for i, sec := range sections {
		out[i] = link.Library{Key: string(sec.Key), Name: sec.Title}
		for _, l := range sec.Locations {
			p, err := link.ParsePath(l.Path)
			if err != nil {
				return nil, err
			}
			out[i].Folders = append(out[i].Folders, p)
		}
	}
	return out, nil
}

func folders(paths []link.ServerPath) []link.ServerPath {
	var out []link.ServerPath
	for _, p := range paths {
		if !slices.ContainsFunc(out, func(f link.ServerPath) bool { return f.Key() == p.Dir().Key() }) {
			out = append(out, p.Dir())
		}
	}
	return out
}

func (s *server) refresh(ctx context.Context, library string, fs []link.ServerPath) error {
	for _, f := range fs {
		if err := s.c.RefreshPath(ctx, plex.ID(library), f.String()); err != nil {
			return err
		}
	}
	return nil
}

func (s *server) Tell(ctx context.Context, library string, paths []link.ServerPath) error {
	return s.refresh(ctx, library, folders(paths))
}

// Moved refreshes the destination folders first, then each source folder
// that differs. A source refreshed first deletes the item when
// autoEmptyTrash is on (recorded run 2026-11-14, case E).
func (s *server) Moved(ctx context.Context, library string, from, to []link.ServerPath) error {
	dest := folders(to)
	if err := s.refresh(ctx, library, dest); err != nil {
		return err
	}
	src := slices.DeleteFunc(folders(from), func(f link.ServerPath) bool {
		return slices.ContainsFunc(dest, func(d link.ServerPath) bool { return d.Key() == f.Key() })
	})
	return s.refresh(ctx, library, src)
}

func (s *server) SidecarRemoved(ctx context.Context, library string, video, _ link.ServerPath) error {
	it, ok, err := s.find(ctx, library, video)
	if err != nil {
		return err
	}
	if !ok {
		return fmt.Errorf("no item at %s", video)
	}
	return s.c.RefreshItem(ctx, it.RatingKey)
}

func (s *server) Scan(ctx context.Context, library string) error {
	sec, ok, err := s.c.Section(ctx, plex.ID(library))
	if err != nil {
		return err
	}
	if !ok {
		return fmt.Errorf("no section %s", library)
	}
	s.scans[library] = scanMark{scannedAt: sec.ScannedAt}
	return s.c.RefreshSection(ctx, plex.ID(library))
}

// ScanEnded is graft 3's rule on Plex: the section isn't refreshing and
// its scan ended after the request.
func (s *server) ScanEnded(ctx context.Context, library string) (bool, error) {
	sec, ok, err := s.c.Section(ctx, plex.ID(library))
	if err != nil {
		return false, err
	}
	if !ok {
		return false, fmt.Errorf("no section %s", library)
	}
	mark := s.scans[library]
	if sec.Refreshing {
		mark.refreshed = true
		s.scans[library] = mark
		return false, nil
	}
	return mark.refreshed || sec.ScannedAt > mark.scannedAt, nil
}

// find pages through the section's movies and episodes for the item with
// a file at p. Plex has no lookup by path.
func (s *server) find(ctx context.Context, library string, p link.ServerPath) (plex.Metadata, bool, error) {
	const size = 100
	for _, typ := range []plex.ItemType{plex.TypeMovie, plex.TypeEpisode} {
		for start := 0; ; start += size {
			page, err := s.c.SectionPage(ctx, plex.ID(library), typ, start, size)
			if err != nil {
				return plex.Metadata{}, false, err
			}
			for _, m := range page.Items {
				if has(m, p) {
					return m, true, nil
				}
			}
			if start+size >= page.Total {
				break
			}
		}
	}
	return plex.Metadata{}, false, nil
}

func has(m plex.Metadata, p link.ServerPath) bool {
	for _, md := range m.Media {
		for _, part := range md.Parts {
			if f, err := link.ParsePath(part.File); err == nil && f.Key() == p.Key() {
				return true
			}
		}
	}
	return false
}

func item(m plex.Metadata) mediaservers.Item {
	it := mediaservers.Item{Key: string(m.RatingKey), PlayCount: m.ViewCount}
	for _, md := range m.Media {
		for _, part := range md.Parts {
			if f, err := link.ParsePath(part.File); err == nil {
				it.Files = append(it.Files, f)
			}
			for _, st := range part.Streams {
				if st.StreamType == plex.StreamSubtitle && st.Key != "" {
					it.Sidecars++
				}
			}
		}
	}
	for _, c := range m.Collections {
		it.Collections = append(it.Collections, c.Tag)
	}
	return it
}

func (s *server) ItemAt(ctx context.Context, library string, p link.ServerPath) (mediaservers.Item, bool, error) {
	m, ok, err := s.find(ctx, library, p)
	if err != nil || !ok {
		return mediaservers.Item{}, false, err
	}
	return item(m), true, nil
}

func (s *server) Item(ctx context.Context, key string) (mediaservers.Item, bool, error) {
	m, ok, err := s.c.Item(ctx, plex.ID(key))
	if err != nil || !ok {
		return mediaservers.Item{}, false, err
	}
	return item(m), true, nil
}

// Watched scrobbles item and tags it with the collection, as recorded run
// 2026-11-14 seeded its items. SlipStream itself writes neither.
func (s *server) Watched(ctx context.Context, key, name string) error {
	if err := s.send(ctx, http.MethodGet, "/:/scrobble", url.Values{"key": {key}, "identifier": {"com.plexapp.plugins.library"}}); err != nil {
		return err
	}
	return s.send(ctx, http.MethodPut, "/library/metadata/"+key, url.Values{
		"type": {"1"}, "id": {key}, "collection[0].tag.tag": {name}, "collection.locked": {"1"},
	})
}

func (s *server) send(ctx context.Context, method, path string, q url.Values) error {
	u, err := url.Parse(s.tg.Address)
	if err != nil {
		return fmt.Errorf("the Plex address: %w", err)
	}
	u = u.JoinPath(path)
	u.RawQuery = q.Encode()
	req, err := http.NewRequestWithContext(ctx, method, u.String(), http.NoBody)
	if err != nil {
		return fmt.Errorf("%s %s: %w", method, path, err)
	}
	req.Header.Set("X-Plex-Token", s.tg.Token)
	resp, err := s.tg.HTTP.Do(req)
	if err != nil {
		return fmt.Errorf("%s %s: %w", method, path, err)
	}
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("%s %s responded with status %d", method, path, resp.StatusCode)
	}
	return nil
}

func (s *server) Sessions(ctx context.Context) ([]link.Session, error) {
	sessions, err := s.c.Sessions(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]link.Session, 0, len(sessions))
	for _, sess := range sessions {
		ls := link.Session{Key: string(sess.SessionKey), Decision: link.DecisionDirectPlay}
		if sess.User != nil {
			ls.User = sess.User.Title
		}
		if sess.Player != nil {
			ls.Player, ls.Product = sess.Player.Title, sess.Player.Product
		}
		if len(sess.Media) > 0 && len(sess.Media[0].Parts) > 0 {
			if ls.Path, err = link.ParsePath(sess.Media[0].Parts[0].File); err != nil {
				return nil, err
			}
		}
		if t := sess.Transcode; t != nil {
			ls.Decision, ls.Hardware = link.DecisionCopy, t.TranscodeHwRequested
			if t.VideoDecision == "transcode" {
				ls.Decision = link.DecisionTranscode
			}
		}
		out = append(out, ls)
	}
	return out, nil
}
