package rules

import (
	"bytes"
	"go/ast"
	"go/printer"
	"go/token"
	"path"
	"slices"
	"strconv"
	"strings"

	"github.com/jatassi/slipstream/cmd/release/internal/lint"
)

func init() {
	lint.Register(lint.Rule{
		Name:  "mains-match",
		Doc:   "cmd/slipstream and cmd/slipstream-e2e each hold only main.go, and differ only by the test Content Types the e2e main registers (spec #133, Lints and CI checks)",
		Check: checkMainsMatch,
	})
}

const (
	releaseMainDir = "cmd/slipstream"
	e2eMainDir     = "cmd/slipstream-e2e"
)

func checkMainsMatch(repo *lint.Repo) ([]lint.Finding, error) {
	module, err := modulePath(repo)
	if err != nil {
		return nil, err
	}
	files, err := repo.GoSyntax()
	if err != nil {
		return nil, err
	}
	var findings []lint.Finding
	mains := map[string]lint.GoFile{}
	for _, f := range files {
		dir := path.Dir(f.Path)
		if (dir != releaseMainDir && dir != e2eMainDir) || strings.HasSuffix(f.Path, "_test.go") {
			continue
		}
		if path.Base(f.Path) == "main.go" {
			mains[dir] = f
			continue
		}
		findings = append(findings, lint.Finding{Path: f.Path, Line: 1, Message: "a main holds only main.go"})
	}
	for _, dir := range []string{releaseMainDir, e2eMainDir} {
		if _, ok := mains[dir]; !ok {
			findings = append(findings, lint.Finding{Path: dir + "/main.go", Line: 1, Message: "does not exist"})
		}
	}
	if len(mains) < 2 {
		return findings, nil
	}
	release, e2e := mains[releaseMainDir], mains[e2eMainDir]
	stripped, removed := newTestTypeStripper(module, e2e).file(e2e.Syntax)
	if removed == 0 {
		findings = append(findings, lint.Finding{Path: e2e.Path, Line: 1, Message: "registers no test Content Type"})
	}
	if differs := firstDifference(repo.Fset, stripped, release.Syntax); differs != nil {
		findings = append(findings, lint.Finding{
			Path:    e2e.Path,
			Line:    repo.Fset.Position(differs.Pos()).Line,
			Message: "differs from " + release.Path + " beyond registering test Content Types",
		})
	}
	return findings, nil
}

// testTypeStripper copies a main without its test Content Types' imports
// and node.Main arguments, leaving the shared syntax tree as it is.
type testTypeStripper struct {
	imports   map[string]string
	prefix    string
	testTypes map[string]bool
	nodeMain  string
}

func newTestTypeStripper(module string, f lint.GoFile) testTypeStripper {
	s := testTypeStripper{
		imports:   importNames(f),
		prefix:    path.Join(module, testContentTypes) + "/",
		testTypes: map[string]bool{},
		nodeMain:  path.Join(module, "internal/node"),
	}
	for name, p := range s.imports {
		if strings.HasPrefix(p, s.prefix) {
			s.testTypes[name] = true
		}
	}
	return s
}

func (s testTypeStripper) file(f *ast.File) (out *ast.File, removed int) {
	c := *f
	c.Decls = nil
	for _, d := range f.Decls {
		switch d := d.(type) {
		case *ast.GenDecl:
			if d.Tok != token.IMPORT {
				c.Decls = append(c.Decls, d)
				continue
			}
			g := *d
			g.Specs = slices.DeleteFunc(slices.Clone(d.Specs), s.isTestImport)
			if len(g.Specs) > 0 {
				c.Decls = append(c.Decls, &g)
			}
		case *ast.FuncDecl:
			fn := *d
			if d.Body != nil {
				var n int
				fn.Body, n = s.block(d.Body)
				removed += n
			}
			c.Decls = append(c.Decls, &fn)
		default:
			c.Decls = append(c.Decls, d)
		}
	}
	return &c, removed
}

func (s testTypeStripper) block(b *ast.BlockStmt) (out *ast.BlockStmt, removed int) {
	c := *b
	c.List = make([]ast.Stmt, len(b.List))
	for i, stmt := range b.List {
		c.List[i] = stmt
		if e, ok := stmt.(*ast.ExprStmt); ok {
			x := *e
			var n int
			x.X, n = s.expr(e.X)
			removed += n
			c.List[i] = &x
		}
	}
	return &c, removed
}

func (s testTypeStripper) expr(e ast.Expr) (out ast.Expr, removed int) {
	call, ok := e.(*ast.CallExpr)
	if !ok {
		return e, 0
	}
	c := *call
	c.Args = make([]ast.Expr, len(call.Args))
	for i, arg := range call.Args {
		var n int
		c.Args[i], n = s.expr(arg)
		removed += n
	}
	if pkg, name, ok := qualifiedName(s.imports, call.Fun); ok && pkg == s.nodeMain && name == "Main" {
		before := len(c.Args)
		c.Args = slices.DeleteFunc(c.Args, s.isTestDef)
		removed += before - len(c.Args)
	}
	return &c, removed
}

func (s testTypeStripper) isTestImport(spec ast.Spec) bool {
	imp, ok := spec.(*ast.ImportSpec)
	if !ok {
		return false
	}
	p, _ := strconv.Unquote(imp.Path.Value)
	return strings.HasPrefix(p, s.prefix)
}

func (s testTypeStripper) isTestDef(arg ast.Expr) bool {
	sel, ok := arg.(*ast.SelectorExpr)
	if !ok || sel.Sel.Name != "Def" {
		return false
	}
	x, ok := sel.X.(*ast.Ident)
	return ok && s.testTypes[x.Name]
}

func firstDifference(fset *token.FileSet, a, b ast.Node) ast.Node {
	if printed(fset, a) == printed(fset, b) {
		return nil
	}
	as, bs := mainChildren(a), mainChildren(b)
	for i := range min(len(as), len(bs)) {
		if d := firstDifference(fset, as[i], bs[i]); d != nil {
			return d
		}
	}
	if len(as) > len(bs) {
		return as[len(bs)]
	}
	return a
}

func mainChildren(n ast.Node) []ast.Node {
	var children []ast.Node
	switch n := n.(type) {
	case *ast.File:
		children = append(children, n.Name)
		for _, d := range n.Decls {
			children = append(children, d)
		}
	case *ast.GenDecl:
		for _, s := range n.Specs {
			children = append(children, s)
		}
	case *ast.FuncDecl:
		children = append(children, n.Name, n.Type)
		if n.Body != nil {
			children = append(children, n.Body)
		}
	case *ast.BlockStmt:
		for _, s := range n.List {
			children = append(children, s)
		}
	}
	return children
}

func printed(fset *token.FileSet, n ast.Node) string {
	var buf bytes.Buffer
	if err := printer.Fprint(&buf, fset, n); err != nil {
		return err.Error()
	}
	return buf.String()
}
