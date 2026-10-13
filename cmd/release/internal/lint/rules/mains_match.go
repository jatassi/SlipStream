package rules

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/parser"
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
	mains := map[string]bool{}
	for _, f := range files {
		dir := path.Dir(f.Path)
		if (dir != releaseMainDir && dir != e2eMainDir) || strings.HasSuffix(f.Path, "_test.go") {
			continue
		}
		if path.Base(f.Path) == "main.go" {
			mains[dir] = true
			continue
		}
		findings = append(findings, lint.Finding{Path: f.Path, Line: 1, Message: "a main holds only main.go"})
	}
	for _, dir := range []string{releaseMainDir, e2eMainDir} {
		if !mains[dir] {
			findings = append(findings, lint.Finding{Path: dir + "/main.go", Line: 1, Message: "does not exist"})
		}
	}
	if len(mains) < 2 {
		return findings, nil
	}
	release, err := parseMain(repo, releaseMainDir+"/main.go")
	if err != nil {
		return nil, err
	}
	e2e, err := parseMain(repo, e2eMainDir+"/main.go")
	if err != nil {
		return nil, err
	}
	if removeTestContentTypes(module, e2e.syntax) == 0 {
		findings = append(findings, lint.Finding{Path: e2e.path, Line: 1, Message: "registers no test Content Type"})
	}
	if differs := firstDifference(e2e, release, e2e.syntax, release.syntax); differs != nil {
		findings = append(findings, lint.Finding{
			Path:    e2e.path,
			Line:    e2e.fset.Position(differs.Pos()).Line,
			Message: "differs from " + release.path + " beyond registering test Content Types",
		})
	}
	return findings, nil
}

func parseMain(repo *lint.Repo, file string) (mainFile, error) {
	src, err := repo.Read(file)
	if err != nil {
		return mainFile{}, err
	}
	fset := token.NewFileSet()
	syntax, err := parser.ParseFile(fset, file, src, parser.ParseComments)
	if err != nil {
		return mainFile{}, fmt.Errorf("parse %s: %w", file, err)
	}
	ast.SortImports(fset, syntax)
	return mainFile{file, fset, syntax}, nil
}

func removeTestContentTypes(module string, f *ast.File) int {
	imports := importNames(lint.GoFile{Syntax: f})
	testTypesPrefix := path.Join(module, testContentTypes) + "/"
	testTypes := map[string]bool{}
	for name, p := range imports {
		if strings.HasPrefix(p, testTypesPrefix) {
			testTypes[name] = true
		}
	}
	nodeMain := path.Join(module, "internal/node")
	removed := 0
	ast.Inspect(f, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}
		if pkg, name, ok := qualifiedName(imports, call.Fun); !ok || pkg != nodeMain || name != "Main" {
			return true
		}
		before := len(call.Args)
		call.Args = slices.DeleteFunc(call.Args, func(arg ast.Expr) bool {
			sel, ok := arg.(*ast.SelectorExpr)
			if !ok || sel.Sel.Name != "Def" {
				return false
			}
			x, ok := sel.X.(*ast.Ident)
			return ok && testTypes[x.Name]
		})
		removed += before - len(call.Args)
		return true
	})
	f.Decls = slices.DeleteFunc(f.Decls, func(decl ast.Decl) bool {
		gen, ok := decl.(*ast.GenDecl)
		if !ok || gen.Tok != token.IMPORT {
			return false
		}
		gen.Specs = slices.DeleteFunc(gen.Specs, func(spec ast.Spec) bool {
			imp, _ := spec.(*ast.ImportSpec)
			p, _ := strconv.Unquote(imp.Path.Value)
			return strings.HasPrefix(p, testTypesPrefix)
		})
		return len(gen.Specs) == 0
	})
	return removed
}

func firstDifference(e2e, release mainFile, a, b ast.Node) ast.Node {
	if printed(e2e.fset, a) == printed(release.fset, b) {
		return nil
	}
	as, bs := mainChildren(a), mainChildren(b)
	for i := range min(len(as), len(bs)) {
		if d := firstDifference(e2e, release, as[i], bs[i]); d != nil {
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
