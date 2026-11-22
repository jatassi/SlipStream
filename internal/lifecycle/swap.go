package lifecycle

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/jatassi/slipstream/internal/atomicfile"
)

// SwapFiles is the live files a restore replaces, in swap order. library.db
// is last, so it moves only once the files that open it are in place.
var SwapFiles = []string{"library.key", "backup.key", "backups.json", library}

const (
	library = "library.db"
	// Restored marks a staged file, which the swap moves in.
	Restored = ".restore"
	// Aside marks a live file the swap moved out, which Aside deletes.
	Aside = ".pre-restore"
)

// sidecars are the files SQLite keeps beside library.db while it is open
// or after a crash. They move with it, so the aside library opens whole as
// library.db.pre-restore, and no restored library.db opens beside a WAL
// that isn't its own.
var sidecars = []string{"-wal", "-shm"}

// Step is one rename of the swap, of names in the data dir.
type Step struct{ From, To string }

// Plan is the renames that finish the swap of the files present, in
// order. It reads nothing but present, so a swap stopped after any prefix
// of its steps plans exactly the rest. For each live file with a .restore
// file:
//
//   - library.db's -wal and -shm, when present, move aside first;
//   - the live file, when present, moves aside, replacing an older one;
//   - the .restore file moves in.
func Plan(present func(name string) bool) []Step {
	moved := map[string]bool{}
	has := func(name string) bool {
		if p, ok := moved[name]; ok {
			return p
		}
		return present(name)
	}
	var steps []Step
	move := func(from, to string) {
		steps = append(steps, Step{From: from, To: to})
		moved[from], moved[to] = false, true
	}
	for _, f := range SwapFiles {
		if !has(f + Restored) {
			continue
		}
		if f == library {
			for _, s := range sidecars {
				if has(f + s) {
					move(f+s, f+Aside+s)
				}
			}
		}
		if has(f) {
			move(f, f+Aside)
		}
		move(f+Restored, f)
	}
	return steps
}

// Swap runs Plan's steps for the files in the data dir, syncing the dir
// after each. before sees each step first, and an error from it stops the
// swap there, as a crash would.
func (d Dir) Swap(before func(Step) error) error {
	for {
		steps := Plan(d.present)
		if len(steps) == 0 {
			return nil
		}
		s := steps[0]
		if err := before(s); err != nil {
			return err
		}
		if err := atomicfile.Rename(filepath.Join(d.Path, s.From), filepath.Join(d.Path, s.To), d.Retry); err != nil {
			return err
		}
		if err := atomicfile.SyncDir(d.Path); err != nil {
			return err
		}
	}
}

// present is whether name is in the data dir. A stat that fails for
// another reason counts as present, so the rename reports it.
func (d Dir) present(name string) bool {
	_, err := os.Lstat(filepath.Join(d.Path, name))
	return !errors.Is(err, fs.ErrNotExist)
}

// Aside deletes the files the swap moved aside, once the record is
// Written.
func (d Dir) Aside() error { return d.remove(Aside) }

// Leftovers deletes the .restore files a staging left without writing its
// record, when no record exists.
func (d Dir) Leftovers() error { return d.remove(Restored) }

func (d Dir) remove(suffix string) error {
	var names []string
	for _, s := range sidecars {
		names = append(names, library+suffix+s)
	}
	for _, f := range SwapFiles {
		names = append(names, f+suffix)
	}
	for _, name := range names {
		if err := atomicfile.Remove(filepath.Join(d.Path, name), d.Retry); err != nil {
			return err
		}
	}
	return atomicfile.SyncDir(d.Path)
}

// WipeFiles is what Wipe and join deletes, in order: library.db with its
// sidecars first, so no start finds a library whose key is gone, then its
// key, backup.key with the fence secret, and the Backups index.
var WipeFiles = []string{library, library + sidecars[0], library + sidecars[1], "library.key", "backup.key", "backups.json"}

// Wipe deletes each of WipeFiles that is present, calling before with its
// name first: an error stops there, as a crash would. Any order of stops
// converges, since the record stays until Finish.
func (d Dir) Wipe(before func(name string) error) error {
	for _, name := range WipeFiles {
		if !d.present(name) {
			continue
		}
		if err := before(name); err != nil {
			return err
		}
		if err := atomicfile.Remove(filepath.Join(d.Path, name), d.Retry); err != nil {
			return err
		}
	}
	return atomicfile.SyncDir(d.Path)
}
