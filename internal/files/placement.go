package files

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/zeebo/blake3"
)

// Linker makes hard links. A filesystem without them refuses, and Place
// then copies.
type Linker interface {
	Link(oldname, newname string) error
}

type OSLinker struct{}

func (OSLinker) Link(oldname, newname string) error {
	return os.Link(oldname, newname) //nolint:wrapcheck // Place asks only whether it linked
}

var ErrOccupied = errors.New("another file is at the target path")

// Place puts the file at from at to, making to's folder: a hard link, else
// a copy to to+".part" renamed into place. A file at to with from's size
// and BLAKE3 counts as placed, so placing again converges; another file
// there is ErrOccupied.
func Place(l Linker, from, to string) error {
	if err := os.MkdirAll(filepath.Dir(to), 0o755); err != nil { //nolint:gosec // media servers read the library
		return fmt.Errorf("make the file's folder: %w", err)
	}
	switch same, err := identical(from, to); {
	case errors.Is(err, os.ErrNotExist):
	case err != nil:
		return err
	case same:
		return nil
	default:
		return ErrOccupied
	}
	if l.Link(from, to) == nil {
		return nil
	}
	return copyInto(from, to)
}

// Move renames the staged file at from to to, making to's folder. A file
// at to with from's size and BLAKE3 counts as moved, and so does any file
// at to once from is gone, so moving again converges; another file there
// is ErrOccupied.
func Move(from, to string) error {
	if _, err := os.Stat(from); errors.Is(err, os.ErrNotExist) {
		if _, err := os.Stat(to); err != nil {
			return fmt.Errorf("neither the staged file nor the target is there: %w", err)
		}
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(to), 0o755); err != nil { //nolint:gosec // media servers read the library
		return fmt.Errorf("make the file's folder: %w", err)
	}
	switch same, err := identical(from, to); {
	case errors.Is(err, os.ErrNotExist):
	case err != nil:
		return err
	case same:
		if err := os.Remove(from); err != nil {
			return fmt.Errorf("remove the staged copy: %w", err)
		}
		return nil
	default:
		return ErrOccupied
	}
	if err := os.Rename(from, to); err != nil {
		return fmt.Errorf("move the staged file into place: %w", err)
	}
	return nil
}

// identical reports whether to holds the same bytes as from, by size and
// BLAKE3. It fails with os.ErrNotExist when nothing is at to.
func identical(from, to string) (bool, error) {
	at, err := os.Stat(to)
	if err != nil {
		return false, fmt.Errorf("read the target: %w", err)
	}
	src, err := os.Stat(from)
	if err != nil {
		return false, fmt.Errorf("read the source: %w", err)
	}
	if os.SameFile(src, at) {
		return true, nil
	}
	if src.Size() != at.Size() {
		return false, nil
	}
	a, err := sum(from)
	if err != nil {
		return false, err
	}
	b, err := sum(to)
	if err != nil {
		return false, err
	}
	return a == b, nil
}

func sum(path string) ([32]byte, error) {
	var out [32]byte
	f, err := os.Open(path) //nolint:gosec // a download or a library file the Grab placed
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

func copyInto(from, to string) error {
	in, err := os.Open(from) //nolint:gosec // the download the Grab imports
	if err != nil {
		return fmt.Errorf("copy the file: %w", err)
	}
	defer func() { _ = in.Close() }()
	part := to + ".part"
	out, err := os.Create(part) //nolint:gosec // beside the target the Grab planned
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
		_ = os.Remove(part)
		return fmt.Errorf("copy the file: %w", err)
	}
	if err := os.Rename(part, to); err != nil {
		return fmt.Errorf("move the copy into place: %w", err)
	}
	return nil
}
