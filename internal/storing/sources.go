package storing

import (
	"context"
	"errors"
	"io/fs"
	"maps"
	"slices"
	"strings"

	"github.com/jatassi/slipstream/internal/media"
	storagev1 "github.com/jatassi/slipstream/internal/pb/slipstream/storage/v1"
	"github.com/jatassi/slipstream/internal/rpc"
	"github.com/jatassi/slipstream/internal/storage"
)

// listLimit is the most video files one ListVideos lists, so a folder an
// Admin picks by mistake, a whole drive of downloads, can't hold the Node
// for minutes.
const listLimit = 5000

// sourcePath is where src's bytes lie on this Node: a payload where its
// client keeps it, or a path that passes the fence.
func (s *Store) sourcePath(ctx context.Context, src Source) (string, error) {
	switch src := src.(type) {
	case Payload:
		p, err := s.cfg.Payloads.Path(src.InfoHash, src.FileIndex)
		if err != nil {
			return "", rpc.Errorf(rpc.CodeUnavailable, "%v", err)
		}
		return p, nil
	case NodePath:
		return s.fence(ctx, src.Abs)
	}
	panic("storing: a source of no kind")
}

func (s *Store) ListVideos(ctx context.Context, req *storagev1.ListVideosRequest) (*storagev1.ListVideosResponse, error) {
	videos, err := videosOf(req.GetVideoExtensions())
	if err != nil {
		return nil, err
	}
	l, err := s.listVideos(ctx, req.GetPath(), videos)
	if err != nil {
		return nil, wireError(err)
	}
	res := &storagev1.ListVideosResponse{Path: l.path}
	if !l.root.IsZero() {
		res.RootId, res.Rel = l.root.Bytes(), l.rel.String()
	}
	for _, f := range l.files {
		res.Files = append(res.Files, &storagev1.Listed{Path: f.path, Size: f.info.Size(), Identity: identityOf(f.info), Seeded: f.seeded})
	}
	return res, nil
}

func (s *Store) RemoveSource(ctx context.Context, req *storagev1.RemoveSourceRequest) (*storagev1.RemoveSourceResponse, error) {
	if err := s.removeSource(ctx, req.GetPath()); err != nil {
		return nil, wireError(err)
	}
	return &storagev1.RemoveSourceResponse{}, nil
}

type videoList struct {
	path  string
	root  storage.UUID
	rel   Rel
	files []listedVideo
}

type listedVideo struct {
	path   string
	info   Info
	seeded bool
}

// listVideos lists the video files under the fenced folder path by
// media.Counts with videos, depth first and each folder's names in byte
// order. It enters no linked folder and lists no linked file, so nothing
// it lists lies outside the fence.
func (s *Store) listVideos(ctx context.Context, path string, videos media.Videos) (videoList, error) {
	dir, err := s.fence(ctx, path)
	if err != nil {
		return videoList{}, err
	}
	l := videoList{path: dir}
	if l.root, l.rel, err = s.rootOf(ctx, dir); err != nil {
		return videoList{}, err
	}
	seeding := s.seeding()
	var walk func(folder string, segs []string) error
	walk = func(folder string, segs []string) error {
		entries, err := s.entries(ctx, folder)
		if err != nil {
			return s.folderError(err)
		}
		slices.SortFunc(entries, func(a, b Entry) int { return strings.Compare(a.Name, b.Name) })
		for _, e := range entries {
			p, at := s.host.Style.Join(folder, e.Name), append(slices.Clone(segs), e.Name)
			switch {
			case e.Link, e.Dir && (media.SkipsFolder(e.Name) || !e.Readable):
			case e.Dir:
				if err := walk(p, at); err != nil {
					return err
				}
			case media.Counts(strings.Join(at, "/"), videos):
				if err := s.listVideo(ctx, &l, p, seeding); err != nil {
					return err
				}
			}
		}
		return nil
	}
	return l, walk(dir, nil)
}

// listVideo adds the video file p to l, unless it went before its stat.
func (s *Store) listVideo(ctx context.Context, l *videoList, p string, seeding map[string]bool) error {
	info, err := timed(ctx, s, l.path, func(ctx context.Context) (Info, error) { return s.cfg.Disk.Stat(ctx, p) })
	switch {
	case errors.Is(err, fs.ErrNotExist):
		return nil
	case err != nil:
		return s.folderError(err)
	}
	if len(l.files) == listLimit {
		return rpc.Errorf(rpc.CodeFailedPrecondition, "%s holds more than %d video files. Choose a folder inside it", l.path, listLimit)
	}
	l.files = append(l.files, listedVideo{path: p, info: info, seeded: seeding[s.key(p)]})
	return nil
}

// rootOf is the root the Store holds that the clean folder dir lies in,
// with dir beneath it, or the zero id when none holds it. The root is the
// nearest folder above dir, links followed, that is a root by identity.
func (s *Store) rootOf(ctx context.Context, dir string) (storage.UUID, Rel, error) {
	resolved, err := s.real(ctx, dir)
	if err != nil {
		return storage.UUID{}, Rel{}, s.folderError(err)
	}
	s.mu.Lock()
	roots := slices.Clone(s.roots)
	s.mu.Unlock()
	same := s.identity(ctx)
	var names []string
	for at, ok := resolved, true; ok; at, ok = s.host.Style.Parent(at) {
		for _, r := range roots {
			if !same(at, r.path) {
				continue
			}
			slices.Reverse(names)
			rel, err := ParseRel(strings.Join(names, "/"))
			if err != nil {
				return storage.UUID{}, Rel{}, rpc.Errorf(rpc.CodeFailedPrecondition, "%s can't be a path in a library root: %v", dir, err)
			}
			return r.id, rel, nil
		}
		segs := s.host.Style.Segments(at)
		names = append(names, segs[len(segs)-1].Name)
	}
	return storage.UUID{}, Rel{}, nil
}

// seeding is where each file of a torrent the Node's clients hold lies, by
// key.
func (s *Store) seeding() map[string]bool {
	out := map[string]bool{}
	for _, p := range s.cfg.Torrents() {
		if c, err := s.host.Style.Clean(p); err == nil {
			out[s.key(c)] = true
		}
	}
	return out
}

// key is the clean path p as two paths to one file on this Node compare:
// without case on Windows.
func (s *Store) key(p string) string {
	if s.host.Style == Windows {
		return Fold(p)
	}
	return p
}

// removeSource removes the file at path, once a Move placed it. A file
// already gone is removed.
func (s *Store) removeSource(ctx context.Context, path string) error {
	p, err := s.fence(ctx, path)
	if err != nil {
		return err
	}
	info, err := timed(ctx, s, p, func(ctx context.Context) (Info, error) { return s.cfg.Disk.Stat(ctx, p) })
	switch {
	case errors.Is(err, fs.ErrNotExist):
		return nil
	case err != nil:
		return s.folderError(err)
	case info.Dir():
		return &FolderError{Kind: FolderNotAFolder, Text: p + " is a folder"}
	}
	if _, err := timed(ctx, s, p, func(ctx context.Context) (struct{}, error) { return struct{}{}, s.cfg.Disk.Remove(ctx, p) }); err != nil && !errors.Is(err, fs.ErrNotExist) {
		return s.folderError(err)
	}
	return nil
}

// importPlaces are where browsing for an import starts: each root the
// Store holds and each of the Node's download folders.
func (s *Store) importPlaces(ctx context.Context) Listing {
	s.mu.Lock()
	roots, sampled := slices.Clone(s.roots), maps.Clone(s.sampled)
	s.mu.Unlock()
	l := Listing{Service: s.host.Service}
	for _, r := range roots {
		v := sampled[r.id]
		l.Places = append(l.Places, Place{Name: r.path, Path: r.path, Kind: v.Kind, Free: v.Free, Total: v.Total, Root: r.id})
	}
	for _, d := range s.cfg.Downloads() {
		p := Place{Name: d, Path: d}
		if v, err := timed(ctx, s, d, func(ctx context.Context) (Volume, error) { return s.cfg.Disk.Volume(ctx, d) }); err == nil {
			p.Kind, p.Free, p.Total = v.Kind, v.Free, v.Total
		}
		l.Places = append(l.Places, p)
	}
	return l
}
