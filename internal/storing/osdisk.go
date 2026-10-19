package storing

import (
	"context"
	"io"
	"io/fs"
	"os"
	"path/filepath"
)

// OSDisk is Disk on this machine, through the OS's own calls: no cgo and
// no df.
type OSDisk struct{ host Host }

// NewOSDisk is this machine's Disk; data and downloads are the folders no
// root may overlap.
func NewOSDisk(data, downloads string) (OSDisk, error) {
	account, err := account()
	if err != nil {
		return OSDisk{}, err
	}
	service, err := service()
	if err != nil {
		return OSDisk{}, err
	}
	return OSDisk{Host{Style: nativeStyle, Account: account, Service: service, Data: data, Downloads: downloads}}, nil
}

func (d OSDisk) Host() Host { return d.host }

type osInfo struct{ fs.FileInfo }

func (i osInfo) Dir() bool { return i.IsDir() }

func (i osInfo) Same(o Info) bool {
	other, ok := o.(osInfo)
	return ok && os.SameFile(i.FileInfo, other.FileInfo)
}

func (OSDisk) Stat(_ context.Context, path string) (Info, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, err //nolint:wrapcheck // a PathError names the call and the path; callers classify it with errors.Is
	}
	return osInfo{info}, nil
}

func (OSDisk) ReadDir(_ context.Context, dir string) ([]Entry, error) {
	des, err := os.ReadDir(dir)
	if err != nil {
		return nil, err //nolint:wrapcheck // a PathError names the call and the path; callers classify it with errors.Is
	}
	entries := make([]Entry, 0, len(des))
	for _, de := range des {
		path := filepath.Join(dir, de.Name())
		isDir := de.IsDir()
		if de.Type()&(fs.ModeSymlink|fs.ModeIrregular) != 0 {
			info, err := os.Stat(path)
			isDir = err == nil && info.IsDir()
		}
		entries = append(entries, Entry{
			Name:     de.Name(),
			Dir:      isDir,
			Hidden:   hidden(nativeStyle, de.Name(), attrsOf(de)),
			Readable: !isDir || listable(path),
		})
	}
	return entries, nil
}

func (OSDisk) Mkdir(_ context.Context, path string) error {
	return os.Mkdir(path, 0o755) //nolint:gosec,wrapcheck // media servers read the library; a PathError names the call and the path
}

func (OSDisk) Open(_ context.Context, path string) (io.ReadCloser, error) {
	f, err := os.Open(path) //nolint:gosec // the Store resolves path under a root
	if err != nil {
		return nil, err //nolint:wrapcheck // a PathError names the call and the path; callers classify it with errors.Is
	}
	return f, nil
}

func (OSDisk) Create(_ context.Context, path string) (File, error) {
	f, err := os.Create(path) //nolint:gosec // the Store resolves path under a root
	if err != nil {
		return nil, err //nolint:wrapcheck // a PathError names the call and the path; callers classify it with errors.Is
	}
	return f, nil
}

func (OSDisk) Remove(_ context.Context, path string) error {
	return os.Remove(path) //nolint:wrapcheck // a PathError names the call and the path; callers classify it with errors.Is
}

func (OSDisk) Rename(_ context.Context, from, to string) error {
	return os.Rename(from, to) //nolint:wrapcheck // a LinkError names the call and both paths
}

func (OSDisk) Link(_ context.Context, from, to string) error {
	return os.Link(from, to) //nolint:wrapcheck // a LinkError names the call and both paths
}

// placesAt is a place for each path whose volume reads; one that doesn't,
// such as a card reader with no card, is no place to start browsing.
func placesAt(ctx context.Context, d Disk, paths []string) []Place {
	var places []Place
	for _, p := range paths {
		if v, err := d.Volume(ctx, p); err == nil {
			places = append(places, Place{Name: v.Name, Path: p, Kind: v.Kind, Free: v.Free, Total: v.Total})
		}
	}
	return places
}
