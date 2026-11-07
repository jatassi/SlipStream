package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/jatassi/slipstream/internal/ports/provenance"
)

const module = "github.com/jatassi/slipstream"

var portID = regexp.MustCompile(`^[a-z][a-z0-9]*$`)

// newPort writes a port's boilerplate: one Go file per C# file, each
// with its header and package clause, the upstream.json that maps them,
// the registry file and an empty fixture with the cases every port
// needs.
func newPort(ctx context.Context, out io.Writer, repo string, tree *provenance.Tree, ref, site, id, family string) error {
	if site == "" {
		return errors.New("new needs --site, the site's C# class")
	}
	if id == "" {
		id = strings.NewReplacer(" ", "", ".", "", "-", "").Replace(strings.ToLower(site))
	}
	if !portID.MatchString(id) {
		return fmt.Errorf("the port id %q isn't a Go package name; pass one with --id", id)
	}
	dir := "internal/ports/" + id
	if _, err := os.Stat(filepath.Join(repo, filepath.FromSlash(dir))); !errors.Is(err, fs.ErrNotExist) { //nolint:gosec // G703: a directory in the repository the maintainer names
		return fmt.Errorf("%s already exists", dir)
	}
	head, err := resolve(ctx, tree, ref)
	if err != nil {
		return err
	}
	files, err := siteFiles(ctx, tree, head.SHA, site)
	if err != nil {
		return err
	}
	u := provenance.Unit{ID: "port:" + id, Kind: provenance.Port, Dir: dir, Manifest: provenance.Manifest{
		Commit: head.SHA, Date: head.Date, Family: family, Files: map[string][]string{},
	}}
	for _, f := range files {
		gofile := goFile(site, id, f)
		u.Files[gofile] = append(u.Files[gofile], f)
		if err := writeNew(repo, path.Join(dir, gofile), "package "+id+"\n"); err != nil {
			return err
		}
	}
	if err := provenance.Write(repo, u); err != nil {
		return err
	}
	if err := writeNew(repo, "internal/ports/all/"+id+".go", fmt.Sprintf("package all\n\nimport \"%s/%s\"\n\nfunc init() { add(%s.Port) }\n", module, dir, id)); err != nil {
		return err
	}
	if err := writeNew(repo, "fakes/site/ports/"+id+"/fixture.yaml", emptyFixture); err != nil {
		return err
	}
	_, err = fmt.Fprintf(out, "Wrote %s from %s at %s. Translate each C# file into its Go file, then fill fakes/site/ports/%s/fixture.yaml.\n", dir, site, head.SHA, id)
	return err //nolint:wrapcheck // the one write of the command's output
}

const emptyFixture = `source: synthetic
settings: {}
cases:
  - name: search
    exchanges: []
  - name: empty
    exchanges: []
  - name: auth
    exchanges: []
  - name: limit
    exchanges: []
`

// siteFiles are the site's C# files at commit: every file in its folder
// under Definitions when it has one, else its one file.
func siteFiles(ctx context.Context, tree *provenance.Tree, commit, site string) ([]string, error) {
	files, err := tree.Files(ctx, commit, definitionsDir+site+"/")
	if err != nil || len(files) > 0 {
		return files, err
	}
	file := definitionsDir + site + ".cs"
	if files, err = tree.Files(ctx, commit, file); err != nil {
		return nil, err
	}
	if len(files) == 0 {
		return nil, fmt.Errorf("prowlarr has neither %s nor a folder %s/ at %s", file, strings.TrimSuffix(file, ".cs"), commit)
	}
	return files, nil
}

// goFile names the Go file for a C# file: the C# name without the site's
// prefix, lowercased, so HDBitsParser.cs is parser.go and HDBits.cs is
// hdbits.go.
func goFile(site, id, file string) string {
	name := strings.TrimSuffix(path.Base(file), ".cs")
	switch {
	case strings.EqualFold(name, site):
		name = id
	case len(name) > len(site) && strings.EqualFold(name[:len(site)], site):
		name = name[len(site):]
	}
	return strings.ToLower(name) + ".go"
}

func writeNew(repo, file, content string) error {
	name := filepath.Join(repo, filepath.FromSlash(file))
	if err := os.MkdirAll(filepath.Dir(name), 0o755); err != nil { //nolint:gosec // G301: a source directory in the repository
		return fmt.Errorf("make %s: %w", path.Dir(file), err)
	}
	f, err := os.OpenFile(name, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644) //nolint:gosec // G302,G304: a new source file in the repository
	if err != nil {
		return fmt.Errorf("write %s: %w", file, err)
	}
	if _, err := f.WriteString(content); err != nil {
		_ = f.Close()
		return fmt.Errorf("write %s: %w", file, err)
	}
	if err := f.Close(); err != nil {
		return fmt.Errorf("write %s: %w", file, err)
	}
	return nil
}
