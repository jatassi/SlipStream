// Command fileitems rewrites the test files of the packages it is given
// from a library file of one Item to one of many: files.File{ItemID: x}
// becomes files.File{Items: []works.ItemID{x}}, files.Found{Item: x}
// becomes files.Found{Items: []works.ItemID{x}}, and a read of ItemID on a
// value it knows is a files.File becomes Items[0]. It lists every other
// read of ItemID on a files.File it finds in a file it doesn't rewrite.
//
//	go run ./scripts/codemod/fileitems ./...
package main

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
)

const filesPath = "github.com/jatassi/slipstream/internal/files"

func main() {
	roots := os.Args[1:]
	if len(roots) == 0 {
		roots = []string{"./..."}
	}
	failed := false
	for _, root := range roots {
		dir := strings.TrimSuffix(root, "/...")
		err := filepath.WalkDir(dir, func(p string, d fs.DirEntry, err error) error { //nolint:gosec // the codemod walks the roots its caller names
			switch {
			case err != nil:
				return err
			case d.IsDir() && (d.Name() == "node_modules" || strings.HasPrefix(d.Name(), ".") && p != dir):
				return filepath.SkipDir
			case d.IsDir() || !strings.HasSuffix(p, ".go"):
				return nil
			}
			return rewrite(p)
		})
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			failed = true
		}
	}
	if failed {
		os.Exit(1)
	}
}

// edit replaces src[from:to] with text.
type edit struct {
	from, to int
	text     string
}

func rewrite(path string) error {
	src, err := os.ReadFile(path) //nolint:gosec // the codemod rewrites the files under the roots it is given
	if err != nil {
		return fmt.Errorf("read %s: %w", path, err)
	}
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, path, src, parser.ParseComments)
	if err != nil {
		return fmt.Errorf("parse %s: %w", path, err)
	}
	qual, ok := importName(f, filesPath)
	if !ok && f.Name.Name != "files" {
		return nil
	}
	rw := &rewriter{fset: fset, f: f, qual: qual, test: strings.HasSuffix(path, "_test.go")}
	rw.known = knownFiles(f, rw.isType)
	ast.Inspect(f, rw.visit)
	if len(rw.edits) == 0 {
		return nil
	}
	out := apply(src, rw.edits)
	if _, ok := importName(f, "github.com/jatassi/slipstream/internal/works"); !ok && bytes.Contains(out, []byte("[]works.ItemID{")) {
		out = addImport(out, fset, f)
	}
	formatted, err := format.Source(out)
	if err != nil {
		return fmt.Errorf("%s: %w", path, err)
	}
	if err := os.WriteFile(path, formatted, 0o644); err != nil { //nolint:gosec // a source file keeps the mode every other one has
		return fmt.Errorf("write %s: %w", path, err)
	}
	return nil
}

// rewriter gathers the edits of one file, f, which imports the files
// package as qual.
type rewriter struct {
	fset  *token.FileSet
	f     *ast.File
	qual  string
	test  bool
	known func(ast.Expr) bool
	edits []edit
}

// isType reports whether e, or what it points to, is the files type name.
func (rw *rewriter) isType(e ast.Expr, name string) bool {
	if star, ok := e.(*ast.StarExpr); ok {
		e = star.X
	}
	switch t := e.(type) {
	case *ast.SelectorExpr:
		x, ok := t.X.(*ast.Ident)
		return ok && x.Name == rw.qual && t.Sel.Name == name
	case *ast.Ident:
		return rw.f.Name.Name == "files" && t.Name == name
	}
	return false
}

func (rw *rewriter) off(p token.Pos) int { return rw.fset.Position(p).Offset }

func (rw *rewriter) visit(n ast.Node) bool {
	switch n := n.(type) {
	case *ast.CompositeLit:
		rw.literal(n)
	case *ast.SelectorExpr:
		rw.read(n)
	}
	return true
}

// literal wraps the Item key of a files.File or files.Found literal in a
// slice, renamed Items.
func (rw *rewriter) literal(n *ast.CompositeLit) {
	key, ok := rw.itemKey(n.Type)
	if !ok {
		return
	}
	for _, el := range n.Elts {
		kv, ok := el.(*ast.KeyValueExpr)
		if !ok {
			continue
		}
		if id, isID := kv.Key.(*ast.Ident); !isID || id.Name != key {
			continue
		}
		if !rw.test {
			report(rw.fset, kv.Pos(), "a "+key+" key")
			continue
		}
		rw.edits = append(rw.edits,
			edit{rw.off(kv.Key.Pos()), rw.off(kv.Key.End()), "Items"},
			edit{rw.off(kv.Value.Pos()), rw.off(kv.Value.Pos()), "[]works.ItemID{"},
			edit{rw.off(kv.Value.End()), rw.off(kv.Value.End()), "}"})
	}
}

// itemKey is the key naming the one Item of a literal of type t.
func (rw *rewriter) itemKey(t ast.Expr) (string, bool) {
	switch {
	case rw.isType(t, "File"):
		return "ItemID", true
	case rw.isType(t, "Found"):
		return "Item", true
	}
	return "", false
}

// read makes a read of ItemID on a known files.File read Items[0].
func (rw *rewriter) read(n *ast.SelectorExpr) {
	if n.Sel.Name != "ItemID" || !rw.known(n.X) {
		return
	}
	if !rw.test {
		report(rw.fset, n.Pos(), "a read of ItemID")
		return
	}
	rw.edits = append(rw.edits, edit{rw.off(n.Sel.Pos()), rw.off(n.Sel.End()), "Items[0]"})
}

// knownFiles reports whether an expression of f names a files.File: an
// identifier declared with that type, assigned a files.File literal or
// ranged over a slice of them, or an index of such a slice.
func knownFiles(f *ast.File, isType func(ast.Expr, string) bool) func(ast.Expr) bool {
	k := &known{singles: map[string]bool{}, lists: map[string]bool{}, isType: isType}
	ast.Inspect(f, k.visit)
	return k.names
}

// known is the identifiers of a file holding one files.File, singles, and
// those holding a slice of them, lists.
type known struct {
	singles, lists map[string]bool
	isType         func(ast.Expr, string) bool
}

func (k *known) visit(n ast.Node) bool {
	switch n := n.(type) {
	case *ast.Field:
		for _, name := range n.Names {
			k.mark(name.Name, k.isType(n.Type, "File"), k.isList(n.Type))
		}
	case *ast.ValueSpec:
		k.spec(n)
	case *ast.AssignStmt:
		k.assign(n)
	case *ast.RangeStmt:
		over, ok := n.X.(*ast.Ident)
		if v, isID := n.Value.(*ast.Ident); ok && isID && k.lists[over.Name] {
			k.singles[v.Name] = true
		}
	}
	return true
}

func (k *known) spec(n *ast.ValueSpec) {
	for i, name := range n.Names {
		var value ast.Expr
		if i < len(n.Values) {
			value = n.Values[i]
		}
		k.mark(name.Name, n.Type != nil && k.isType(n.Type, "File") || k.isSingle(value), n.Type != nil && k.isList(n.Type) || k.isList(value))
	}
}

func (k *known) assign(n *ast.AssignStmt) {
	if len(n.Lhs) != len(n.Rhs) {
		return
	}
	for i, l := range n.Lhs {
		if id, ok := l.(*ast.Ident); ok {
			k.mark(id.Name, k.isSingle(n.Rhs[i]), k.isList(n.Rhs[i]))
		}
	}
}

func (k *known) mark(name string, single, list bool) {
	k.singles[name] = k.singles[name] || single
	k.lists[name] = k.lists[name] || list
}

func (k *known) isList(e ast.Expr) bool {
	switch t := e.(type) {
	case *ast.ArrayType:
		return k.isType(t.Elt, "File")
	case *ast.CompositeLit:
		a, ok := t.Type.(*ast.ArrayType)
		return ok && k.isType(a.Elt, "File")
	}
	return false
}

func (k *known) isSingle(e ast.Expr) bool {
	if u, ok := e.(*ast.UnaryExpr); ok && u.Op == token.AND {
		e = u.X
	}
	c, ok := e.(*ast.CompositeLit)
	return ok && k.isType(c.Type, "File")
}

func (k *known) names(e ast.Expr) bool {
	switch x := e.(type) {
	case *ast.Ident:
		return k.singles[x.Name]
	case *ast.IndexExpr:
		id, ok := x.X.(*ast.Ident)
		return ok && k.lists[id.Name]
	case *ast.ParenExpr:
		star, ok := x.X.(*ast.StarExpr)
		if !ok {
			return false
		}
		id, isID := star.X.(*ast.Ident)
		return isID && k.singles[id.Name]
	}
	return false
}

// importName is the name f imports path under.
func importName(f *ast.File, path string) (string, bool) {
	for _, im := range f.Imports {
		if p, _ := strconv.Unquote(im.Path.Value); p == path {
			if im.Name != nil {
				return im.Name.Name, true
			}
			return filepath.Base(path), true
		}
	}
	return "", false
}

// addImport adds the works import to src, parsed as f, after its first.
func addImport(src []byte, fset *token.FileSet, f *ast.File) []byte {
	at := fset.Position(f.Imports[0].End()).Offset
	return slices.Concat(src[:at], []byte("\n\t\"github.com/jatassi/slipstream/internal/works\""), src[at:])
}

func apply(src []byte, edits []edit) []byte {
	slices.SortFunc(edits, func(a, b edit) int { return b.from - a.from })
	out := slices.Clone(src)
	for _, e := range edits {
		out = slices.Concat(out[:e.from], []byte(e.text), out[e.to:])
	}
	return out
}

func report(fset *token.FileSet, p token.Pos, what string) {
	fmt.Printf("%s: %s on a library file, left for a hand edit\n", fset.Position(p), what)
}
