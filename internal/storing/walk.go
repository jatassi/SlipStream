package storing

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"slices"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/jatassi/slipstream/internal/media"
	"github.com/jatassi/slipstream/internal/storage"
)

// walkLimit is the most entries one Walk page lists.
const walkLimit = 1000

// walkPage is one page of a walk on the Node. next is the last entry the
// page finished, by its segments, and nil once the walk is done.
type walkPage struct {
	files    []walkedFile
	unread   []Unread
	next     []string
	caseFold bool
}

type walkedFile struct {
	rel  Rel
	info Info
}

// walk lists up to limit video files under root after the entry after
// names, depth first, each folder's names in byte order, so the cursor
// compares one segment at a time and never as a string ("A B/x" sorts
// before "A/y" as a string, after it in the walk). It skips by
// media.SkipsFolder and media.Counts, enters no linked folder, sizes a
// linked file by its target, and keeps no state between pages (ADR 0003).
// Each page resolves the root afresh and needs its last judgment Ready.
// Every Disk call is timed under the root's folder, so one hung call
// fails the page and later pages fail fast while the kernel holds it. A
// page also ends early once it has run Timings.Stat, so a slow share
// pages instead of failing.
func (s *Store) walk(ctx context.Context, root storage.UUID, after []string, limit int) (walkPage, error) {
	dir, err := s.ready(ctx, root)
	if err != nil {
		return walkPage{}, err
	}
	walks := s.walksOf(root)
	walks.RLock()
	defer walks.RUnlock()
	fold, err := s.caseFold(ctx, dir)
	if err != nil {
		return walkPage{}, s.folderError(err)
	}
	w := &walker{s: s, ctx: ctx, root: dir, limit: limit, until: time.Now().Add(s.cfg.Timings.Stat)}
	w.page.caseFold = fold
	more, err := w.folder(dir, nil, after)
	if err != nil {
		return walkPage{}, s.folderError(err)
	}
	if !more {
		w.page.next = w.at
	}
	return w.page, nil
}

// ready is root's folder when its marker names it now and the last pass
// judged it Ready at the intent's place, else *NotReady.
func (s *Store) ready(ctx context.Context, root storage.UUID) (string, error) {
	s.mu.Lock()
	r, known := s.rootLocked(root)
	j, judged := s.judged[root]
	s.mu.Unlock()
	switch {
	case !known || !judged || j.place != r.place:
		return "", &NotReady{Unknown: true}
	case j.state != Ready:
		return "", &NotReady{State: j.state}
	}
	return s.resolve(ctx, root, false)
}

// caseFold reports whether dir resolves names ignoring case: its marker,
// asked for in upper case, is the same file.
func (s *Store) caseFold(ctx context.Context, dir string) (bool, error) {
	return timed(ctx, s, dir, func(ctx context.Context) (bool, error) {
		marker, err := s.cfg.Disk.Stat(ctx, s.host.Style.Join(dir, MarkerName))
		if err != nil {
			return false, err
		}
		upper, err := s.cfg.Disk.Stat(ctx, s.host.Style.Join(dir, strings.ToUpper(MarkerName)))
		switch {
		case errors.Is(err, fs.ErrNotExist):
			return false, nil
		case err != nil:
			return false, err
		}
		return marker.Same(upper), nil
	})
}

type walker struct {
	s   *Store
	ctx context.Context //nolint:containedctx // one page's context, for every call the page makes
	// root is the root's folder, which keys every timed call.
	root  string
	limit int
	until time.Time
	page  walkPage
	// at is the last entry finished.
	at []string
}

// folder walks dir, the folder at segs under the root, past after, the
// cursor's rest within it. It answers false once the page is full.
func (w *walker) folder(dir string, segs, after []string) (bool, error) {
	entries, err := timed(w.ctx, w.s, w.root, func(ctx context.Context) ([]Entry, error) { return w.s.cfg.Disk.ReadDir(ctx, dir) })
	switch {
	case len(segs) > 0 && errors.Is(err, fs.ErrNotExist):
		return true, nil
	case len(segs) > 0 && errors.Is(err, fs.ErrPermission):
		w.unread(segs, UnreadDenied)
		return true, nil
	case err != nil:
		return false, err
	}
	slices.SortFunc(entries, func(a, b Entry) int { return strings.Compare(a.Name, b.Name) })
	for _, e := range entries {
		var rest []string
		if len(after) > 0 {
			switch c := strings.Compare(e.Name, after[0]); {
			case c < 0, c == 0 && len(after) == 1:
				continue
			case c == 0:
				rest = after[1:]
			}
			after = nil
		}
		more, err := w.entry(dir, append(slices.Clone(segs), e.Name), e, rest)
		if err != nil || !more {
			return more, err
		}
	}
	return true, nil
}

// entry walks e, at segs under the root in dir, past after.
func (w *walker) entry(dir string, segs []string, e Entry, after []string) (bool, error) {
	path := w.s.host.Style.Join(dir, e.Name)
	switch {
	case e.Dir && (e.Link || media.SkipsFolder(e.Name)):
	case e.Dir:
		if _, reason := named(segs); reason != 0 {
			w.unread(segs, reason)
			break
		}
		if !e.Readable {
			w.unread(segs, UnreadDenied)
			break
		}
		if more, err := w.folder(path, segs, after); err != nil || !more {
			return more, err
		}
	case media.Counts(strings.Join(segs, "/")):
		if err := w.file(path, segs); err != nil {
			return false, err
		}
	}
	return w.finished(segs), nil
}

func (w *walker) file(path string, segs []string) error {
	rel, reason := named(segs)
	if reason != 0 {
		w.unread(segs, reason)
		return nil
	}
	info, err := timed(w.ctx, w.s, w.root, func(ctx context.Context) (Info, error) { return w.s.cfg.Disk.Stat(ctx, path) })
	switch {
	case errors.Is(err, fs.ErrNotExist):
	case errors.Is(err, fs.ErrPermission):
		w.unread(segs, UnreadDenied)
	case err != nil:
		return err
	case !info.Dir():
		w.page.files = append(w.page.files, walkedFile{rel, info})
	}
	return nil
}

// finished moves the cursor past segs and answers whether the page has
// room and time for more.
func (w *walker) finished(segs []string) bool {
	w.at = segs
	return len(w.page.files) < w.limit && time.Now().Before(w.until)
}

func (w *walker) unread(segs []string, reason UnreadReason) {
	w.page.unread = append(w.page.unread, Unread{Path: strings.ToValidUTF8(strings.Join(segs, "/"), "�"), Reason: reason})
}

// named is segs as a Rel, or why a Rel can't carry them.
func named(segs []string) (Rel, UnreadReason) {
	s := strings.Join(segs, "/")
	if !utf8.ValidString(s) {
		return Rel{}, UnreadNotUTF8
	}
	rel, err := ParseRel(s)
	if err != nil {
		return Rel{}, UnreadRefused
	}
	return rel, 0
}

// probe reads rel's media record under root with the configured prober,
// bounded by ProbeTimeout. A file gone since the walk is a *FolderError
// NotFound.
func (s *Store) probe(ctx context.Context, root storage.UUID, rel Rel) (media.Record, error) {
	dir, err := s.resolve(ctx, root, false)
	if err != nil {
		return media.Record{}, err
	}
	path := s.host.Style.Under(dir, rel)
	if _, err := timed(ctx, s, dir, func(ctx context.Context) (Info, error) { return s.cfg.Disk.Stat(ctx, path) }); err != nil {
		return media.Record{}, s.folderError(err)
	}
	ctx, cancel := context.WithTimeout(ctx, s.cfg.ProbeTimeout)
	defer cancel()
	r, err := s.cfg.Prober.Probe(ctx, path)
	if err != nil {
		return media.Record{}, fmt.Errorf("probe %s: %w", rel, err)
	}
	return r, nil
}
