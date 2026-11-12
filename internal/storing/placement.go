package storing

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"sync"

	"github.com/zeebo/blake3"
)

var ErrOccupied = errors.New("another file is at the target path")

// recaseSuffix names a file or folder midway through a case-only rename:
// "<new name>.slipstream-recase" beside it.
const recaseSuffix = ".slipstream-recase"

type placer struct {
	d     Disk
	style PathStyle
	root  string
	// walks is the root's walk lock, held to write across a case-only
	// rename.
	walks *sync.RWMutex
}

// NeedsCopyError is a place asked without allowCopy whose link and clone
// both refused. Size is the bytes a copy writes.
type NeedsCopyError struct{ Size int64 }

func (e *NeedsCopyError) Error() string {
	return fmt.Sprintf("the file can't be linked or cloned there and needs a copy of %d bytes", e.Size)
}

// NeedsSpaceError is a clone or copy the target volume has no room for.
type NeedsSpaceError struct{}

func (*NeedsSpaceError) Error() string { return "the target volume has no room for the file" }

// place puts the file at from at rel, making its folders, by the first
// step of the ladder that takes it: a hard link, a clone where the link is
// refused across volumes, and a copy where the clone is unsupported and
// allowCopy lets it. Any other refusal stops the ladder with its own
// error. A file at rel with from's size and BLAKE3 counts as placed, by
// Link when it is from itself and else by Copy, so placing again
// converges; another file there is ErrOccupied, and is never removed.
func (p placer) place(ctx context.Context, from string, rel Rel, allowCopy bool) (Info, Step, error) {
	to := p.style.Under(p.root, rel)
	if err := p.folders(ctx, Rel{}, rel); err != nil {
		return nil, 0, err
	}
	switch same, err := p.identical(ctx, from, to); {
	case err == nil && same:
		return p.settled(ctx, from, to)
	case err == nil:
		return nil, 0, ErrOccupied
	case !errors.Is(err, fs.ErrNotExist):
		return nil, 0, err
	}
	step, err := p.climb(ctx, from, to, allowCopy)
	if err != nil {
		return nil, 0, err
	}
	placed, err := p.d.Stat(ctx, to)
	if err != nil {
		return nil, 0, fmt.Errorf("read the placed file: %w", err)
	}
	return placed, step, nil
}

func (p placer) climb(ctx context.Context, from, to string, allowCopy bool) (Step, error) {
	err := p.d.Link(ctx, from, to)
	switch {
	case err == nil:
		return Link, nil
	case !errors.Is(err, ErrCrossDevice):
		return 0, fmt.Errorf("link the file: %w", err)
	}
	switch err := p.d.Clone(ctx, from, to); {
	case err == nil:
		return Clone, nil
	case NoSpace(err):
		return 0, &NeedsSpaceError{}
	case !errors.Is(err, errors.ErrUnsupported):
		return 0, fmt.Errorf("clone the file: %w", err)
	}
	if !allowCopy {
		src, err := p.d.Stat(ctx, from)
		if err != nil {
			return 0, fmt.Errorf("read the source: %w", err)
		}
		return 0, &NeedsCopyError{Size: src.Size()}
	}
	switch err := p.copy(ctx, from, to); {
	case NoSpace(err):
		return 0, &NeedsSpaceError{}
	case err != nil:
		return 0, err
	}
	return Copy, nil
}

func (p placer) settled(ctx context.Context, from, to string) (Info, Step, error) {
	src, err := p.d.Stat(ctx, from)
	if err != nil {
		return nil, 0, fmt.Errorf("read the source: %w", err)
	}
	dst, err := p.d.Stat(ctx, to)
	if err != nil {
		return nil, 0, fmt.Errorf("read the placed file: %w", err)
	}
	if src.Same(dst) {
		return dst, Link, nil
	}
	return dst, Copy, nil
}

// move renames the file at from to to, making to's folders. A file at to
// with from's size and BLAKE3 counts as moved, and so does any file at to
// once from is gone, so moving again converges; another file there is
// ErrOccupied. A to that folds to from and holds the same bytes is from
// spelled in another case, which a volume that ignores case finds at to:
// it is renamed through recase, never removed as a copy of itself. With
// carry, from's companions follow it, after it, so a failed move changes
// nothing and a retry carries what a crash left. It answers the moved
// file and the companions carry left.
func (p placer) move(ctx context.Context, from, to Rel, carry bool) (Info, []Rel, error) {
	info, err := p.moveFile(ctx, from, to)
	if err != nil || !carry {
		return info, nil, err
	}
	left, err := p.carry(ctx, from, to)
	if err != nil {
		return nil, nil, err
	}
	return info, left, nil
}

func (p placer) moveFile(ctx context.Context, from, to Rel) (Info, error) {
	src, dst := p.style.Under(p.root, from), p.style.Under(p.root, to)
	if err := p.folders(ctx, from, to); err != nil {
		return nil, err
	}
	if _, err := p.d.Stat(ctx, src); errors.Is(err, fs.ErrNotExist) {
		return p.moved(ctx, dst)
	}
	switch same, err := p.identical(ctx, src, dst); {
	case errors.Is(err, fs.ErrNotExist):
		if err := p.d.Rename(ctx, src, dst); err != nil {
			return nil, fmt.Errorf("move the file into place: %w", err)
		}
	case err != nil:
		return nil, err
	case !same:
		return nil, ErrOccupied
	case Fold(from.s) == Fold(to.s):
		if err := p.recase(ctx, src, dst); err != nil {
			return nil, err
		}
	default:
		if err := p.d.Remove(ctx, src); err != nil {
			return nil, fmt.Errorf("remove the identical copy: %w", err)
		}
	}
	info, err := p.d.Stat(ctx, dst)
	if err != nil {
		return nil, fmt.Errorf("read the moved file: %w", err)
	}
	return info, nil
}

// moved is the file at dst once its source is gone, finishing a case-only
// rename a crash cut short.
func (p placer) moved(ctx context.Context, dst string) (Info, error) {
	info, err := p.d.Stat(ctx, dst)
	if errors.Is(err, fs.ErrNotExist) {
		if p.d.Rename(ctx, dst+recaseSuffix, dst) == nil {
			info, err = p.d.Stat(ctx, dst)
		}
	}
	if err != nil {
		return nil, fmt.Errorf("neither the file nor its target is there: %w", err)
	}
	return info, nil
}

// folders makes to's folders. Where to spells one of from's folders in
// another case, under folders that fold alike, a volume that ignores case
// finds one folder by both names: it takes to's spelling through recase.
// A volume that keeps case has two folders, and to's is made beside
// from's.
func (p placer) folders(ctx context.Context, from, to Rel) error {
	dir := p.root
	was, segs := from.segments(), to.segments()
	along := true
	for i, name := range segs[:len(segs)-1] {
		next := p.style.Join(dir, name)
		along = along && i < len(was)-1 && Fold(was[i]) == Fold(name)
		if along && was[i] != name {
			if err := p.recaseFolder(ctx, p.style.Join(dir, was[i]), next); err != nil {
				return err
			}
		}
		if err := p.d.Mkdir(ctx, next); err != nil && !errors.Is(err, fs.ErrExist) {
			return fmt.Errorf("make the file's folder: %w", err)
		}
		dir = next
	}
	return nil
}

// recaseFolder gives the folder at was to's spelling when both names find
// the same folder, first finishing a rename to to a crash cut short.
func (p placer) recaseFolder(ctx context.Context, was, to string) error {
	switch _, err := p.d.Stat(ctx, to+recaseSuffix); {
	case err == nil:
		if err := p.d.Rename(ctx, to+recaseSuffix, to); err != nil {
			return fmt.Errorf("finish renaming the folder: %w", err)
		}
		return nil
	case !errors.Is(err, fs.ErrNotExist):
		return fmt.Errorf("read the folder: %w", err)
	}
	a, aerr := p.d.Stat(ctx, was)
	b, berr := p.d.Stat(ctx, to)
	switch {
	case errors.Is(aerr, fs.ErrNotExist), errors.Is(berr, fs.ErrNotExist):
		return nil
	case aerr != nil:
		return fmt.Errorf("read the folder: %w", aerr)
	case berr != nil:
		return fmt.Errorf("read the folder: %w", berr)
	case !a.Same(b):
		return nil
	}
	return p.recase(ctx, was, to)
}

// recase renames from to to, one name spelled in another case, through
// to+recaseSuffix: a volume that ignores case may take a direct rename as
// a rename onto itself. It holds the root's walk lock, so no Walk page
// lists the temporary name.
func (p placer) recase(ctx context.Context, from, to string) error {
	p.walks.Lock()
	defer p.walks.Unlock()
	if err := p.d.Rename(ctx, from, to+recaseSuffix); err != nil {
		return fmt.Errorf("rename to a temporary name: %w", err)
	}
	if err := p.d.Rename(ctx, to+recaseSuffix, to); err != nil {
		return fmt.Errorf("rename from the temporary name: %w", err)
	}
	return nil
}

// identical reports whether to holds the same bytes as from, by identity
// or by size and BLAKE3. It fails with fs.ErrNotExist when nothing is at
// to.
func (p placer) identical(ctx context.Context, from, to string) (bool, error) {
	at, err := p.d.Stat(ctx, to)
	if err != nil {
		return false, fmt.Errorf("read the target: %w", err)
	}
	src, err := p.d.Stat(ctx, from)
	if err != nil {
		return false, fmt.Errorf("read the source: %w", err)
	}
	if src.Same(at) {
		return true, nil
	}
	if src.Size() != at.Size() {
		return false, nil
	}
	a, err := p.sum(ctx, from)
	if err != nil {
		return false, err
	}
	b, err := p.sum(ctx, to)
	if err != nil {
		return false, err
	}
	return a == b, nil
}

func (p placer) sum(ctx context.Context, path string) ([32]byte, error) {
	var out [32]byte
	f, err := p.d.Open(ctx, path)
	if err != nil {
		return out, fmt.Errorf("hash the file: %w", err)
	}
	defer func() { _ = f.Close() }()
	h := blake3.New()
	if _, err := io.Copy(h, f); err != nil {
		return out, fmt.Errorf("hash %s: %w", path, err)
	}
	copy(out[:], h.Sum(nil))
	return out, nil
}

func (p placer) copy(ctx context.Context, from, to string) error {
	in, err := p.d.Open(ctx, from)
	if err != nil {
		return fmt.Errorf("copy the file: %w", err)
	}
	defer func() { _ = in.Close() }()
	part := to + ".part"
	out, err := p.d.Create(ctx, part)
	if err != nil {
		return fmt.Errorf("copy the file: %w", err)
	}
	_, err = io.Copy(out, in)
	if err == nil {
		err = out.Sync()
	}
	if closeErr := out.Close(); err == nil {
		err = closeErr
	}
	if err != nil {
		_ = p.d.Remove(ctx, part)
		return fmt.Errorf("copy the file: %w", err)
	}
	if err := p.d.Rename(ctx, part, to); err != nil {
		return fmt.Errorf("move the copy into place: %w", err)
	}
	return nil
}
