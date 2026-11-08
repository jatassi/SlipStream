// Package artwork is the real TMDB and TVDB image files the Fake world
// serves, each named by the path its host serves it at and recorded once
// by fakes/scenario/record.
package artwork

import (
	"embed"
	"io/fs"
	"slices"
	"strings"
)

//go:embed all:files
var files embed.FS

// Store is a directory of image files: a TMDB file at the root as TMDB
// names it, and a TVDB file under tvdb at its path after /banners.
type Store struct {
	fsys fs.FS
}

// New is the store of the files in fsys.
func New(fsys fs.FS) Store { return Store{fsys: fsys} }

// Embedded is the store checked in under files.
var Embedded = New(must(fs.Sub(files, "files")))

func must(fsys fs.FS, err error) fs.FS {
	if err != nil {
		panic("artwork: " + err.Error())
	}
	return fsys
}

const (
	tvdbRoot = "/banners/"
	tvdbDir  = "tvdb"
)

// Name is the store's name for the file served at p: a TMDB file path
// such as /abc123.jpg, or a TVDB artwork path such as
// /banners/v4/series/1/posters/a.jpg.
func Name(p string) (string, bool) {
	name, ok := strings.CutPrefix(p, "/")
	if rest, tvdb := strings.CutPrefix(p, tvdbRoot); tvdb {
		name = tvdbDir + "/" + rest
	} else if strings.Contains(name, "/") {
		return "", false
	}
	return name, ok && fs.ValidPath(name) && !strings.Contains("/"+name, "/.")
}

// Lookup is the bytes of the file served at p.
func (s Store) Lookup(p string) ([]byte, bool) {
	name, ok := Name(p)
	if !ok {
		return nil, false
	}
	data, err := fs.ReadFile(s.fsys, name)
	return data, err == nil
}

// Paths is the path every file in the store is served at, sorted.
func (s Store) Paths() []string {
	var out []string
	_ = fs.WalkDir(s.fsys, ".", func(name string, e fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if e.IsDir() || strings.HasPrefix(e.Name(), ".") {
			return nil
		}
		if rest, ok := strings.CutPrefix(name, tvdbDir+"/"); ok {
			out = append(out, tvdbRoot+rest)
		} else {
			out = append(out, "/"+name)
		}
		return nil
	})
	slices.Sort(out)
	return out
}
