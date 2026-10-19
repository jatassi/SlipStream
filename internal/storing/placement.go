package storing

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"

	"github.com/zeebo/blake3"
)

var ErrOccupied = errors.New("another file is at the target path")

type placer struct {
	d     Disk
	style PathStyle
	root  string
}

// place puts the file at from at rel, making its folders: a hard link,
// else a copy to rel+".part" renamed into place. A file at rel with from's
// size and BLAKE3 counts as placed, so placing again converges; another
// file there is ErrOccupied. It answers the placed file's size and whether
// it is a link to from.
func (p placer) place(ctx context.Context, from string, rel Rel) (size int64, linked bool, err error) {
	to := p.style.Under(p.root, rel)
	if err := p.folders(ctx, rel); err != nil {
		return 0, false, err
	}
	switch same, err := p.identical(ctx, from, to); {
	case errors.Is(err, fs.ErrNotExist):
		if p.d.Link(ctx, from, to) != nil {
			if err := p.copy(ctx, from, to); err != nil {
				return 0, false, err
			}
		}
	case err != nil:
		return 0, false, err
	case !same:
		return 0, false, ErrOccupied
	}
	src, err := p.d.Stat(ctx, from)
	if err != nil {
		return 0, false, fmt.Errorf("read the source: %w", err)
	}
	dst, err := p.d.Stat(ctx, to)
	if err != nil {
		return 0, false, fmt.Errorf("read the placed file: %w", err)
	}
	return dst.Size(), src.Same(dst), nil
}

// move renames the file at from to to, making to's folders. A file at to
// with from's size and BLAKE3 counts as moved, and so does any file at to
// once from is gone, so moving again converges; another file there is
// ErrOccupied. It answers the moved file's size.
func (p placer) move(ctx context.Context, from, to Rel) (int64, error) {
	src, dst := p.style.Under(p.root, from), p.style.Under(p.root, to)
	if _, err := p.d.Stat(ctx, src); errors.Is(err, fs.ErrNotExist) {
		info, err := p.d.Stat(ctx, dst)
		if err != nil {
			return 0, fmt.Errorf("neither the file nor its target is there: %w", err)
		}
		return info.Size(), nil
	}
	if err := p.folders(ctx, to); err != nil {
		return 0, err
	}
	switch same, err := p.identical(ctx, src, dst); {
	case errors.Is(err, fs.ErrNotExist):
		if err := p.d.Rename(ctx, src, dst); err != nil {
			return 0, fmt.Errorf("move the file into place: %w", err)
		}
	case err != nil:
		return 0, err
	case !same:
		return 0, ErrOccupied
	default:
		if err := p.d.Remove(ctx, src); err != nil {
			return 0, fmt.Errorf("remove the identical copy: %w", err)
		}
	}
	info, err := p.d.Stat(ctx, dst)
	if err != nil {
		return 0, fmt.Errorf("read the moved file: %w", err)
	}
	return info.Size(), nil
}

func (p placer) folders(ctx context.Context, rel Rel) error {
	dir := p.root
	segs := rel.segments()
	for _, name := range segs[:len(segs)-1] {
		dir = p.style.Join(dir, name)
		if err := p.d.Mkdir(ctx, dir); err != nil && !errors.Is(err, fs.ErrExist) {
			return fmt.Errorf("make the file's folder: %w", err)
		}
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
