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

// Report takes path reports, as Jellyfin's and Emby's /Library/Media/Updated
// do, whatever the update type. After the monitor delay each path's
// nearest folder on disk is scanned.
func (s *Server) Report(paths []link.ServerPath) {
	s.lock()
	defer s.mu.Unlock()
	for _, p := range paths {
		l, _, ok := s.libraryOf(p)
		if !ok || (s.b.FreshRootIgnores && !l.scanned) {
			continue
		}
		if s.b.RestartsOnRepeat {
			if t, ok := s.pending[p.Key()]; ok {
				t.Stop()
			}
			s.pending[p.Key()] = time.AfterFunc(s.b.MonitorDelay, func() { s.fire(p) })
			continue
		}
		if s.batch == nil {
			b := &batch{}
			b.timer = time.AfterFunc(s.b.MonitorDelay, func() { s.fireBatch(b) })
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

// scanNearest scans the nearest folder of p that exists on disk, inside
// its library folder.
func (s *Server) scanNearest(p link.ServerPath) {
	l, folder, ok := s.libraryOf(p)
	if !ok {
		return
	}
	dir := p
	if !s.isDir(dir) {
		dir = p.Dir()
	}
	for !s.isDir(dir) && dir.Key() != folder.Key() {
		dir = dir.Dir()
	}
	s.scanFolder(l, dir)
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
