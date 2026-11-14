package mediaserver

import (
	"os"
	"slices"
	"time"

	"github.com/jatassi/slipstream/internal/mediaserverlink/link"
)

// batch is Emby's one timer over every report until it fires.
type batch struct {
	paths []link.ServerPath
	timer *time.Timer
}

// Update is one path report. Deleted is the report's update type, which
// only DeletesAtOnce reads.
type Update struct {
	Path    link.ServerPath
	Deleted bool
}

// Report takes path reports, as Jellyfin's and Emby's /Library/Media/Updated
// do. After the monitor delay each path's folder is scanned.
func (s *Server) Report(updates []Update) {
	s.lock()
	defer s.mu.Unlock()
	for _, u := range updates {
		p := u.Path
		if _, _, ok := s.libraryOf(p); !ok {
			continue
		}
		if u.Deleted && s.b.DeletesAtOnce {
			s.scanNearest(p)
			continue
		}
		if s.b.RestartsOnRepeat {
			if t, ok := s.pending[p.Key()]; ok {
				t.Stop()
			}
			s.pending[p.Key()] = time.AfterFunc(s.delay, func() { s.fire(p) })
			continue
		}
		if s.batch == nil {
			b := &batch{}
			b.timer = time.AfterFunc(s.delay, func() { s.fireBatch(b) })
			s.batch = b
		}
		if !slices.ContainsFunc(s.batch.paths, func(q link.ServerPath) bool { return q.Key() == p.Key() }) {
			s.batch.paths = append(s.batch.paths, p)
		}
	}
}

func (s *Server) fire(p link.ServerPath) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.pending, p.Key())
	s.scanNearest(p)
}

func (s *Server) fireBatch(b *batch) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.batch != b {
		return
	}
	s.batch = nil
	for _, p := range b.paths {
		s.scanNearest(p)
	}
}

// scanNearest scans the folder a report of p resolves to, inside its
// library folder: the nearest item's with ResolvesNearestItem, else the
// nearest folder on disk.
func (s *Server) scanNearest(p link.ServerPath) {
	l, root, ok := s.libraryOf(p)
	if !ok {
		return
	}
	if s.b.ResolvesNearestItem {
		if dir, ok := s.nearestItem(l, root, p); ok {
			s.scanFolder(l, dir)
		}
		return
	}
	dir := p
	if !s.isDir(dir) {
		dir = p.Dir()
	}
	for !s.isDir(dir) && dir.Key() != root.Key() {
		dir = dir.Dir()
	}
	s.scanFolder(l, dir)
}

// nearestItem walks up from p to the first path the server holds an item
// at, and answers that item's folder: a file's item, a folder holding an
// item, or root once a scan read it with content.
func (s *Server) nearestItem(l *library, root, p link.ServerPath) (link.ServerPath, bool) {
	for d := p; ; d = d.Dir() {
		if it, _, ok := s.partAt(d); ok {
			return it.Folder, true
		}
		for _, it := range s.items {
			if it.Folder.Key() == d.Key() {
				return d, true
			}
		}
		if d.Key() == root.Key() {
			return root, l.scanned[root.Key()]
		}
	}
}

func (s *Server) isDir(p link.ServerPath) bool {
	local, ok := s.disk.local(p)
	if !ok {
		return false
	}
	info, err := os.Stat(local)
	return err == nil && info.IsDir()
}

func (s *Server) dropPending() {
	for k, t := range s.pending {
		t.Stop()
		delete(s.pending, k)
	}
	if s.batch != nil {
		s.batch.timer.Stop()
		s.batch = nil
	}
}
