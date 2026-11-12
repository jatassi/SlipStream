package storing

import (
	"context"
	"errors"
	"io/fs"
	"slices"
	"strings"
)

// OutsideError is a path the Node refuses to read or change: outside every
// library root it holds and every download folder once each link on it is
// followed.
type OutsideError struct{ Path string }

func (e *OutsideError) Error() string {
	return e.Path + " isn't in a library root or a download folder on this Node"
}

// Holds is whether the absolute path lies in a root the Store holds or in
// one of the Node's download folders, the built-in client's and each
// external client's, once every link and junction on it is followed. It
// is the Node's one fence: every request that names an absolute path to
// read or change passes it, and so does a Transfer that sends one.
func (s *Store) Holds(ctx context.Context, path string) bool {
	_, err := s.fence(ctx, path)
	return err == nil
}

// fence is path clean when Holds, else *OutsideError.
func (s *Store) fence(ctx context.Context, path string) (string, error) {
	p, err := s.host.Style.Clean(path)
	if err != nil {
		return "", &OutsideError{Path: path}
	}
	real, err := s.real(ctx, p)
	if err != nil {
		return "", &OutsideError{Path: path}
	}
	s.mu.Lock()
	folders := make([]string, 0, len(s.roots))
	for _, r := range s.roots {
		folders = append(folders, r.path)
	}
	s.mu.Unlock()
	folders = append(folders, s.cfg.Downloads()...)
	same := s.identity(ctx)
	// The folders compare by file identity, as a folder check's do, so
	// case, separators and a linked root can't take a path outside them.
	if !slices.ContainsFunc(folders, func(dir string) bool { return within(s.host.Style, real, dir, same) }) {
		return "", &OutsideError{Path: path}
	}
	return p, nil
}

// real is the clean path p with every link on it followed. The part of p
// that isn't there yet holds no link, so it follows its deepest folder
// that is.
func (s *Store) real(ctx context.Context, p string) (string, error) {
	at := p
	for {
		resolved, err := timed(ctx, s, at, func(ctx context.Context) (string, error) { return s.cfg.Disk.Resolve(ctx, at) })
		if err == nil {
			if rest := strings.TrimLeft(p[len(at):], `/\`); rest != "" {
				return s.host.Style.Join(resolved, rest), nil
			}
			return resolved, nil
		}
		if !errors.Is(err, fs.ErrNotExist) {
			return "", err
		}
		parent, ok := s.host.Style.Parent(at)
		if !ok {
			return "", err
		}
		at = parent
	}
}
