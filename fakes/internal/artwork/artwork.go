// Package artwork is the real TMDB image files the Fake world serves, each
// named by its TMDB file name and recorded once by fakes/tmdb/record.
package artwork

import (
	"embed"
	"io/fs"
	"path"
	"slices"
	"strings"
)

//go:embed all:files
var files embed.FS

// Store is a directory of image files named as TMDB names them.
type Store struct {
	fsys fs.FS
}

// New is the store of the files at the root of fsys.
func New(fsys fs.FS) Store { return Store{fsys: fsys} }

// Embedded is the store checked in under files.
var Embedded = New(must(fs.Sub(files, "files")))

func must(fsys fs.FS, err error) fs.FS {
	if err != nil {
		panic("artwork: " + err.Error())
	}
	return fsys
}

// Lookup is the bytes of the file at TMDB file path p, such as
// /abc123.jpg.
func (s Store) Lookup(p string) ([]byte, bool) {
	name, ok := strings.CutPrefix(p, "/")
	if !ok || !fs.ValidPath(name) || strings.Contains(name, "/") || strings.HasPrefix(name, ".") {
		return nil, false
	}
	data, err := fs.ReadFile(s.fsys, name)
	return data, err == nil
}

// Paths is the TMDB file path of every file in the store, sorted.
func (s Store) Paths() []string {
	entries, _ := fs.ReadDir(s.fsys, ".")
	var out []string
	for _, e := range entries {
		if !e.IsDir() && !strings.HasPrefix(e.Name(), ".") {
			out = append(out, path.Join("/", e.Name()))
		}
	}
	slices.Sort(out)
	return out
}
