package storing

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"strings"

	"github.com/jatassi/slipstream/internal/media"
	"github.com/jatassi/slipstream/internal/storage"
)

// Planned is one rename a plan proposes within a root.
type Planned struct{ From, To Rel }

// Occupant is a file at a planned rename's target, its name spelled as
// the Node spells it.
type Occupant struct {
	Path Rel
	Seen Seen
}

// Surveyed is a Node's answer for one Planned: the files whose names
// equal its target's without case, which may be its source in another
// case, and its source's companions when asked.
type Surveyed struct {
	Occupants  []Occupant
	Companions []Rel
}

type surveyed struct {
	occupants  []onDisk
	companions []Rel
}

type onDisk struct {
	path Rel
	info Info
}

// survey answers each planned rename under root, listing each folder
// once. A target folder that isn't there holds nothing.
func (s *Store) survey(ctx context.Context, root storage.UUID, planned []Planned, withCompanions bool, videos media.Videos) ([]surveyed, error) {
	dir, err := s.resolve(ctx, root, false)
	if err != nil {
		return nil, err
	}
	listed := map[Rel][]Entry{}
	list := func(folder Rel) ([]Entry, error) {
		if es, ok := listed[folder]; ok {
			return es, nil
		}
		es, err := s.entries(ctx, s.host.Style.Under(dir, folder))
		if err != nil && !errors.Is(err, fs.ErrNotExist) && !errors.Is(err, errNotAFolder) {
			return nil, s.folderError(err)
		}
		listed[folder] = es
		return es, nil
	}
	out := make([]surveyed, len(planned))
	for i, p := range planned {
		folder, _ := p.To.split()
		es, err := list(folder)
		if err != nil {
			return nil, err
		}
		if out[i].occupants, err = s.occupants(ctx, dir, p.To, es); err != nil {
			return nil, err
		}
		if !withCompanions {
			continue
		}
		from, video := p.From.split()
		if es, err = list(from); err != nil {
			return nil, err
		}
		for _, c := range companions(es, video, videos) {
			out[i].companions = append(out[i].companions, from.join(c))
		}
	}
	return out, nil
}

// occupants are the files among to's folder's entries whose names equal
// to's without case. One gone since the listing holds nothing.
func (s *Store) occupants(ctx context.Context, dir string, to Rel, entries []Entry) ([]onDisk, error) {
	folder, name := to.split()
	var out []onDisk
	for _, e := range entries {
		if Fold(e.Name) != Fold(name) {
			continue
		}
		at := folder.join(e.Name)
		info, err := timed(ctx, s, dir, func(ctx context.Context) (Info, error) {
			return s.cfg.Disk.Stat(ctx, s.host.Style.Under(dir, at))
		})
		switch {
		case errors.Is(err, fs.ErrNotExist):
		case err != nil:
			return nil, s.folderError(err)
		default:
			out = append(out, onDisk{at, info})
		}
	}
	return out, nil
}

// prune removes folder and each parent below root while it is empty, and
// answers the folders removed, innermost first. It stops at a held folder,
// a kept folder or a parent of one, a folder that holds anything, and the
// root; a folder already gone is passed over, so a repeat converges.
func (s *Store) prune(ctx context.Context, root storage.UUID, folder Rel, keep []Rel) ([]Rel, error) {
	dir, err := s.resolve(ctx, root, false)
	if err != nil {
		return nil, err
	}
	var removed []Rel
	for f := folder; f != (Rel{}) && !held(f) && !kept(f, keep); f, _ = f.split() {
		path := s.host.Style.Under(dir, f)
		es, err := s.entries(ctx, path)
		switch {
		case errors.Is(err, fs.ErrNotExist):
			continue
		case errors.Is(err, errNotAFolder):
			return removed, nil
		case err != nil:
			return nil, s.folderError(err)
		case len(es) > 0:
			return removed, nil
		}
		if err := s.cfg.Disk.Remove(ctx, path); err != nil {
			return nil, s.refused(ctx, dir, fmt.Errorf("remove an empty folder: %w", err))
		}
		removed = append(removed, f)
	}
	return removed, nil
}

// held is a folder SlipStream keeps in a root, or one inside it.
func held(f Rel) bool {
	top, _, _ := strings.Cut(f.s, "/")
	return top == HoldingDir || top == StagingDir
}

// kept is f among keep, or a parent of one, without case.
func kept(f Rel, keep []Rel) bool {
	folded := Fold(f.s)
	for _, k := range keep {
		if k := Fold(k.s); k == folded || strings.HasPrefix(k, folded+"/") {
			return true
		}
	}
	return false
}
