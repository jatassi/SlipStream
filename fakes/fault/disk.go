package fault

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"slices"
	"strings"
	"syscall"

	"github.com/jatassi/slipstream/internal/storing"
)

type (
	// Full fills the folder's volume: free reads 0, and new folders and
	// file writes fail with ENOSPC.
	Full struct{}
	// Unmounted makes the folder a mount point with nothing mounted: it
	// lists empty, and everything beneath it, its marker too, is absent.
	Unmounted struct{}
	// Hung holds every call on the folder until the rule clears, as a
	// share that stopped answering, or the call's context ends.
	Hung struct{}
	// ReadOnly fails writes with EROFS and reads the folder unwritable.
	ReadOnly struct{}
	// Volume makes the folder a volume of its own, Total bytes with Free
	// of them free.
	Volume struct{ Total, Free int64 }
)

func (Full) Layer() Layer      { return Disk }
func (Unmounted) Layer() Layer { return Disk }
func (Hung) Layer() Layer      { return Disk }
func (ReadOnly) Layer() Layer  { return Disk }
func (Volume) Layer() Layer    { return Disk }

func (Full) String() string      { return "full" }
func (Unmounted) String() string { return "unmounted" }
func (Hung) String() string      { return "hung" }
func (ReadOnly) String() string  { return "readonly" }
func (f Volume) String() string  { return "volume=" + size(f.Total) + ",free=" + size(f.Free) }

func volume(v string) (Fault, error) {
	total, free, ok := strings.Cut(v, ",free=")
	if !ok {
		return nil, errors.New("want volume=<total>,free=<free>")
	}
	t, err := byteCount(total)
	if err != nil {
		return nil, err
	}
	f, err := byteCount(free)
	if err != nil {
		return nil, err
	}
	if t == 0 || f > t {
		return nil, fmt.Errorf("free %s of %s: want a positive total and free no more than it", free, total)
	}
	return Volume{Total: t, Free: f}, nil
}

// diskFront is the one front of a disk Plan. A disk rule's Target is a
// folder in its Node's own path style, and covers it and everything
// beneath it.
var diskFront = Front{Layers: Disk}

// NewDiskPlan is a Plan for one Node's disk rules, which its Disk method plays.
func NewDiskPlan() *Plan { return NewPlan([]Front{diskFront}, nil) }

func checkDisk(r Rule, fronts []Front) error {
	switch {
	case !slices.ContainsFunc(fronts, func(f Front) bool { return f.Layers&Disk != 0 }):
		return fmt.Errorf("%s is a disk fault, which plays only in the simulation tier", r.Fault)
	case r.Target == "":
		return fmt.Errorf("disk rule %q names no folder", r.ID)
	case r.Origin != "" || r.Skip > 0 || r.Times > 0 || r.After != nil:
		return fmt.Errorf("%s holds a folder until cleared and takes no origin, skip, times or after", r.Fault)
	}
	return nil
}

// beneath reports whether path is folder or lies beneath it, in either
// path style.
func beneath(folder, path string) bool {
	if !strings.HasPrefix(path, folder) {
		return false
	}
	return len(path) == len(folder) || isSep(path[len(folder)]) || isSep(folder[len(folder)-1])
}

func isSep(c byte) bool { return c == '/' || c == '\\' }

// played is what the armed disk rules make of one path.
type played struct {
	// absent: the path lies beneath an unmounted folder.
	absent bool
	// empty: the path is an unmounted folder.
	empty    bool
	readOnly bool
	full     bool
	// volume is the Volume of the deepest folder holding the path, mounted
	// at mount.
	volume *Volume
	mount  string
}

// diskPlay reads what the rules covering path make of it, first waiting
// while a Hung one holds it.
func (p *Plan) diskPlay(ctx context.Context, op, path string) (played, error) {
	for {
		p.mu.Lock()
		out, hung := p.playedLocked(path)
		if !hung {
			p.mu.Unlock()
			return out, nil
		}
		changed := p.changedLocked()
		p.mu.Unlock()
		select {
		case <-changed:
		case <-ctx.Done():
			return played{}, &fs.PathError{Op: op, Path: path, Err: ctx.Err()}
		}
	}
}

func (p *Plan) playedLocked(path string) (out played, hung bool) {
	for _, e := range p.rules {
		folder := string(e.Target)
		if !e.armed || e.Fault.Layer() != Disk || !beneath(folder, path) {
			continue
		}
		switch f := e.Fault.(type) {
		case Hung:
			hung = true
		case Unmounted:
			out.empty = out.empty || folder == path
			out.absent = out.absent || folder != path
		case ReadOnly:
			out.readOnly = true
		case Full:
			out.full = true
		case Volume:
			if out.volume == nil || len(folder) > len(out.mount) {
				out.volume, out.mount = &f, folder
			}
		}
	}
	return out, hung
}

// Disk is base with p's disk rules played on it.
func (p *Plan) Disk(base storing.Disk) storing.Disk { return &disk{base: base, plan: p} }

type disk struct {
	base storing.Disk
	plan *Plan
}

func failed(op, path string, err error) error { return &fs.PathError{Op: op, Path: path, Err: err} }

// read plays the rules on a call that reads path.
func (d *disk) read(ctx context.Context, op, path string) (played, error) {
	pl, err := d.plan.diskPlay(ctx, op, path)
	if err == nil && pl.absent {
		err = failed(op, path, fs.ErrNotExist)
	}
	return pl, err
}

// write plays the rules on a call that writes at each of paths.
func (d *disk) write(ctx context.Context, op string, paths ...string) (played, error) {
	var all played
	for _, path := range paths {
		pl, err := d.read(ctx, op, path)
		if err != nil {
			return played{}, err
		}
		if pl.readOnly {
			return played{}, failed(op, path, syscall.EROFS)
		}
		all.full = all.full || pl.full
	}
	return all, nil
}

func (d *disk) Stat(ctx context.Context, path string) (storing.Info, error) {
	if _, err := d.read(ctx, "stat", path); err != nil {
		return nil, err
	}
	return d.base.Stat(ctx, path)
}

func (d *disk) ReadDir(ctx context.Context, dir string) ([]storing.Entry, error) {
	pl, err := d.read(ctx, "readdir", dir)
	if err != nil || pl.empty {
		return nil, err
	}
	return d.base.ReadDir(ctx, dir)
}

func (d *disk) Mkdir(ctx context.Context, path string) error {
	pl, err := d.write(ctx, "mkdir", path)
	if err == nil && pl.full {
		err = failed("mkdir", path, syscall.ENOSPC)
	}
	if err != nil {
		return err
	}
	return d.base.Mkdir(ctx, path)
}

func (d *disk) Open(ctx context.Context, path string) (io.ReadCloser, error) {
	if _, err := d.read(ctx, "open", path); err != nil {
		return nil, err
	}
	return d.base.Open(ctx, path)
}

func (d *disk) Create(ctx context.Context, path string) (storing.File, error) {
	pl, err := d.write(ctx, "open", path)
	if err != nil {
		return nil, err
	}
	f, err := d.base.Create(ctx, path)
	if err != nil || !pl.full {
		return f, err
	}
	return fullFile{File: f, path: path}, nil
}

// fullFile is a file on a full volume: it opened, and no byte fits.
type fullFile struct {
	storing.File
	path string
}

func (f fullFile) Write([]byte) (int, error) { return 0, failed("write", f.path, syscall.ENOSPC) }

func (d *disk) Remove(ctx context.Context, path string) error {
	if _, err := d.write(ctx, "remove", path); err != nil {
		return err
	}
	return d.base.Remove(ctx, path)
}

func (d *disk) Rename(ctx context.Context, from, to string) error {
	if _, err := d.write(ctx, "rename", from, to); err != nil {
		return err
	}
	return d.base.Rename(ctx, from, to)
}

func (d *disk) Link(ctx context.Context, from, to string) error {
	if _, err := d.write(ctx, "link", from, to); err != nil {
		return err
	}
	return d.base.Link(ctx, from, to)
}

func (d *disk) Writable(ctx context.Context, dir string) (bool, error) {
	pl, err := d.read(ctx, "access", dir)
	if err != nil || pl.readOnly {
		return false, err
	}
	return d.base.Writable(ctx, dir)
}

func (d *disk) Volume(ctx context.Context, path string) (storing.Volume, error) {
	pl, err := d.read(ctx, "statfs", path)
	if err != nil {
		return storing.Volume{}, err
	}
	v, err := d.base.Volume(ctx, path)
	if err != nil {
		return storing.Volume{}, err
	}
	if f := pl.volume; f != nil {
		v.ID, v.Name, v.Mount, v.Total, v.Free = "sim:"+pl.mount, pl.mount, pl.mount, f.Total, f.Free
	}
	if pl.full {
		v.Free = 0
	}
	return v, nil
}

func (d *disk) Places(ctx context.Context) ([]storing.Place, error) {
	return d.base.Places(ctx)
}

func (d *disk) Host() storing.Host { return d.base.Host() }
