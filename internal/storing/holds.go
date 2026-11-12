package storing

import (
	"context"
	"slices"
)

// Holds is whether the absolute path lies in a root the Store holds or in
// one of the Node's downloads folders. Its folders compare by file
// identity, as a folder check's do, so case and separators can't take a
// path outside them.
func (s *Store) Holds(ctx context.Context, path string) bool {
	p, err := s.host.Style.Clean(path)
	if err != nil {
		return false
	}
	s.mu.Lock()
	folders := make([]string, 0, len(s.roots))
	for _, r := range s.roots {
		folders = append(folders, r.path)
	}
	s.mu.Unlock()
	folders = append(folders, s.cfg.Downloads()...)
	same := s.identity(ctx)
	return slices.ContainsFunc(folders, func(dir string) bool { return within(s.host.Style, p, dir, same) })
}
