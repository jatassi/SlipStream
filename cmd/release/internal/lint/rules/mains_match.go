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

type mainFile struct {
	path   string
	fset   *token.FileSet
	syntax *ast.File
}

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
	stripped, removed := newTestTypeStripper(module, e2e).strip(e2e.Syntax)
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

// strip returns n's copy and how many node.Main arguments it removed; a nil
// copy means n goes entirely.
func (s testTypeStripper) strip(n ast.Node) (ast.Node, int) {
	switch n := n.(type) {
	case *ast.File:
		c := *n
		var removed int
		c.Decls, removed = stripEach(s, n.Decls)
		return &c, removed
	case *ast.GenDecl:
		if n.Tok != token.IMPORT {
			return n, 0
		}
		c := *n
		c.Specs, _ = stripEach(s, n.Specs)
		if len(c.Specs) == 0 {
			return nil, 0
		}
		return &c, 0
	case *ast.ImportSpec:
		if p, _ := strconv.Unquote(n.Path.Value); strings.HasPrefix(p, s.prefix) {
			return nil, 0
		}
		return n, 0
	case *ast.FuncDecl:
		if n.Body == nil {
			return n, 0
		}
		c := *n
		body, removed := s.strip(n.Body)
		c.Body = body.(*ast.BlockStmt)
		return &c, removed
	case *ast.BlockStmt:
		c := *n
		var removed int
		c.List, removed = stripEach(s, n.List)
		return &c, removed
	case *ast.ExprStmt:
		c := *n
		x, removed := s.strip(n.X)
		c.X = x.(ast.Expr)
		return &c, removed
	case *ast.CallExpr:
		c := *n
		var removed int
		c.Args, removed = stripEach(s, n.Args)
		if pkg, name, ok := qualifiedName(s.imports, n.Fun); ok && pkg == s.nodeMain && name == "Main" {
			before := len(c.Args)
			c.Args = slices.DeleteFunc(c.Args, s.isTestDef)
			removed += before - len(c.Args)
		}
		return &c, removed
	}
	return n, 0
}

func (s testTypeStripper) isTestDef(arg ast.Expr) bool {
	sel, ok := arg.(*ast.SelectorExpr)
	if !ok || sel.Sel.Name != "Def" {
		return false
	}
	x, ok := sel.X.(*ast.Ident)
	return ok && s.testTypes[x.Name]
}

func stripEach[T ast.Node](s testTypeStripper, nodes []T) ([]T, int) {
	out := make([]T, 0, len(nodes))
	total := 0
	for _, n := range nodes {
		c, removed := s.strip(n)
		total += removed
		if c != nil {
			out = append(out, c.(T))
		}
	}
	return out, total
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
