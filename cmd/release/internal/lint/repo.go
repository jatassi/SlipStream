package lint

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"
)

type Target struct {
	GOOS, GOARCH string
}

func (t Target) String() string { return t.GOOS + "/" + t.GOARCH }

type GoFile struct {
	Path   string
	Syntax *ast.File
}

type Package struct {
	ImportPath string
	Dir        string
	GoFiles    []string
	CgoFiles   []string
	Imports    []string
	Module     *struct{ Path string }
}

type Repo struct {
	Root string
	Fset *token.FileSet

	ctx context.Context

	files  []string
	goSrc  []GoFile
	parsed bool
	deps   map[string][]Package
}

func Open(ctx context.Context, root string) *Repo {
	return &Repo{Root: root, Fset: token.NewFileSet(), ctx: ctx, deps: map[string][]Package{}}
}

func (r *Repo) Files() ([]string, error) {
	if r.files != nil {
		return r.files, nil
	}
	out, err := r.command("git", "ls-files", "-z").Output()
	if err != nil {
		return nil, fmt.Errorf("git ls-files: %w", err)
	}
	r.files = strings.FieldsFunc(string(out), func(c rune) bool { return c == 0 })
	return r.files, nil
}

func (r *Repo) Read(rel string) ([]byte, error) {
	data, err := os.ReadFile(filepath.Join(r.Root, filepath.FromSlash(rel)))
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", rel, err)
	}
	return data, nil
}

func (r *Repo) GoSyntax() ([]GoFile, error) {
	if r.parsed {
		return r.goSrc, nil
	}
	files, err := r.Files()
	if err != nil {
		return nil, err
	}
	for _, f := range files {
		if path.Ext(f) != ".go" || ignoredByGo(f) {
			continue
		}
		src, err := r.Read(f)
		if err != nil {
			return nil, err
		}
		syntax, err := parser.ParseFile(r.Fset, f, src, parser.ParseComments)
		if err != nil {
			return nil, fmt.Errorf("parse Go: %w", err)
		}
		r.goSrc = append(r.goSrc, GoFile{f, syntax})
	}
	r.parsed = true
	return r.goSrc, nil
}

func OwnerSQL(file, folder string) bool {
	return path.Ext(file) == ".sql" && path.Base(path.Dir(file)) == folder && !ignoredByGo(file)
}

func ignoredByGo(file string) bool {
	for _, elem := range strings.Split(path.Dir(file), "/") {
		if elem == "testdata" || strings.HasPrefix(elem, "_") || strings.HasPrefix(elem, ".") {
			return true
		}
	}
	return false
}

func (r *Repo) Deps(pkg string, t Target) ([]Package, error) {
	key := pkg + " " + t.String()
	if pkgs, ok := r.deps[key]; ok {
		return pkgs, nil
	}
	cmd := r.command("go", "list", "-deps", "-json", "-tags", "release", pkg)
	cmd.Env = append(os.Environ(), "GOOS="+t.GOOS, "GOARCH="+t.GOARCH, "CGO_ENABLED=1")
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("go list %s for %s: %w\n%s", pkg, t, err, stderr.Bytes())
	}
	var pkgs []Package
	dec := json.NewDecoder(bytes.NewReader(out))
	for dec.More() {
		var p Package
		if err := dec.Decode(&p); err != nil {
			return nil, fmt.Errorf("go list %s for %s: %w", pkg, t, err)
		}
		pkgs = append(pkgs, p)
	}
	r.deps[key] = pkgs
	return pkgs, nil
}

func (r *Repo) command(name string, args ...string) *exec.Cmd {
	cmd := exec.CommandContext(r.ctx, name, args...)
	cmd.Dir = r.Root
	return cmd
}
