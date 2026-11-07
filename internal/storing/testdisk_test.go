package storing

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"
)

// styleHere is this machine's path dialect, which real temp folders use.
var styleHere = map[bool]PathStyle{false: Unix, true: Windows}[runtime.GOOS == "windows"]

// testDisk is Disk on real temp folders, with the faults a test sets per
// path and a count of the calls made.
type testDisk struct {
	host Host
	// downloads is the Node's downloads folders, which no root may overlap.
	downloads []string

	mu sync.Mutex
	// hung paths make every call on them wait until the channel closes.
	hung       map[string]chan struct{}
	denied     map[string]bool
	broken     map[string]bool
	readOnly   map[string]bool
	unreadable map[string]bool
	noLinks    bool
	// volumes is the volume of each folder and everything under it; the
	// longest folder wins.
	volumes map[string]Volume
	places  []Place
	calls   map[string]int
}

func newTestDisk(t *testing.T) *testDisk {
	t.Helper()
	return &testDisk{
		host:       Host{Style: styleHere, Account: "media", Data: t.TempDir()},
		downloads:  []string{t.TempDir()},
		hung:       map[string]chan struct{}{},
		denied:     map[string]bool{},
		broken:     map[string]bool{},
		readOnly:   map[string]bool{},
		unreadable: map[string]bool{},
		volumes:    map[string]Volume{},
		calls:      map[string]int{},
	}
}

func (d *testDisk) hang(path string) (release func()) {
	ch := make(chan struct{})
	d.mu.Lock()
	d.hung[path] = ch
	d.mu.Unlock()
	return func() {
		d.mu.Lock()
		delete(d.hung, path)
		d.mu.Unlock()
		close(ch)
	}
}

func (d *testDisk) flag(m map[string]bool, path string) {
	d.mu.Lock()
	defer d.mu.Unlock()
	m[path] = true
}

func (d *testDisk) count(op, path string) int {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.calls[op+" "+path]
}

func (d *testDisk) fault(ctx context.Context, op, path string) error {
	d.mu.Lock()
	d.calls[op+" "+path]++
	hung, denied, broken := d.hung[path], d.denied[path], d.broken[path]
	d.mu.Unlock()
	if hung != nil {
		select {
		case <-hung:
		case <-ctx.Done():
			return fmt.Errorf("%s %s: %w", op, path, ctx.Err())
		}
	}
	switch {
	case denied:
		return &fs.PathError{Op: op, Path: path, Err: fs.ErrPermission}
	case broken:
		return &fs.PathError{Op: op, Path: path, Err: errors.New("input/output error")}
	}
	return nil
}

func (d *testDisk) Stat(ctx context.Context, path string) (Info, error) {
	if err := d.fault(ctx, "stat", path); err != nil {
		return nil, err
	}
	return OSDisk{}.Stat(ctx, path)
}

func (d *testDisk) ReadDir(ctx context.Context, dir string) ([]Entry, error) {
	if err := d.fault(ctx, "readdir", dir); err != nil {
		return nil, err
	}
	des, err := os.ReadDir(dir)
	if err != nil {
		return nil, err //nolint:wrapcheck // the OS's own error, as a Disk answers it
	}
	d.mu.Lock()
	defer d.mu.Unlock()
	var out []Entry
	for _, de := range des {
		full := filepath.Join(dir, de.Name())
		fi, err := os.Stat(full)
		out = append(out, Entry{Name: de.Name(), Dir: err == nil && fi.IsDir(), Hidden: hidden(d.host.Style, de.Name(), 0), Readable: !d.unreadable[full]})
	}
	return out, nil
}

func (d *testDisk) Mkdir(ctx context.Context, path string) error {
	if err := d.fault(ctx, "mkdir", path); err != nil {
		return err
	}
	return os.Mkdir(path, 0o750) //nolint:wrapcheck // the OS's own error, as a Disk answers it
}

func (d *testDisk) Open(ctx context.Context, path string) (io.ReadCloser, error) {
	if err := d.fault(ctx, "open", path); err != nil {
		return nil, err
	}
	return os.Open(path) //nolint:wrapcheck // the OS's own error, as a Disk answers it
}

func (d *testDisk) OpenAt(ctx context.Context, path string) (ReadAtCloser, error) {
	if err := d.fault(ctx, "open", path); err != nil {
		return nil, err
	}
	return OSDisk{}.OpenAt(ctx, path)
}

func (d *testDisk) OpenWrite(ctx context.Context, path string) (WriteAtFile, error) {
	if err := d.fault(ctx, "create", path); err != nil {
		return nil, err
	}
	return OSDisk{}.OpenWrite(ctx, path)
}

func (d *testDisk) Create(ctx context.Context, path string) (File, error) {
	if err := d.fault(ctx, "create", path); err != nil {
		return nil, err
	}
	d.mu.Lock()
	ro := d.readOnly[filepath.Dir(path)]
	d.mu.Unlock()
	if ro {
		return nil, &fs.PathError{Op: "open", Path: path, Err: errors.New("read-only file system")}
	}
	return os.Create(path) //nolint:wrapcheck // the OS's own error, as a Disk answers it
}

func (d *testDisk) Remove(ctx context.Context, path string) error {
	if err := d.fault(ctx, "remove", path); err != nil {
		return err
	}
	return os.Remove(path) //nolint:wrapcheck // the OS's own error, as a Disk answers it
}

func (d *testDisk) RemoveAll(ctx context.Context, path string) error {
	if err := d.fault(ctx, "remove", path); err != nil {
		return err
	}
	return OSDisk{}.RemoveAll(ctx, path)
}

func (d *testDisk) Rename(ctx context.Context, from, to string) error {
	if err := d.fault(ctx, "rename", to); err != nil {
		return err
	}
	return os.Rename(from, to) //nolint:wrapcheck // the OS's own error, as a Disk answers it
}

func (d *testDisk) Link(ctx context.Context, from, to string) error {
	if err := d.fault(ctx, "link", to); err != nil {
		return err
	}
	d.mu.Lock()
	refuse := d.noLinks
	d.mu.Unlock()
	if refuse {
		return &os.LinkError{Op: "link", Old: from, New: to, Err: errors.New("operation not permitted")}
	}
	return os.Link(from, to) //nolint:wrapcheck // the OS's own error, as a Disk answers it
}

func (d *testDisk) Writable(ctx context.Context, dir string) (bool, error) {
	if err := d.fault(ctx, "writable", dir); err != nil {
		return false, err
	}
	if _, err := os.Stat(dir); err != nil {
		return false, err //nolint:wrapcheck // the OS's own error, as a Disk answers it
	}
	d.mu.Lock()
	defer d.mu.Unlock()
	return !d.readOnly[dir], nil
}

func (d *testDisk) Volume(ctx context.Context, path string) (Volume, error) {
	if err := d.fault(ctx, "volume", path); err != nil {
		return Volume{}, err
	}
	d.mu.Lock()
	defer d.mu.Unlock()
	best, found := "", false
	var v Volume
	for dir, vol := range d.volumes {
		if (path == dir || strings.HasPrefix(path, dir+string(filepath.Separator))) && len(dir) >= len(best) {
			best, v, found = dir, vol, true
		}
	}
	if !found {
		return Volume{}, fmt.Errorf("no volume holds %s", path)
	}
	v.SampledAt = time.Now()
	return v, nil
}

func (d *testDisk) Places(context.Context) ([]Place, error) { return d.places, nil }

func (d *testDisk) Host() Host { return d.host }
