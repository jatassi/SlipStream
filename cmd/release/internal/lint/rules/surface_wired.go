package rules

import (
	"go/ast"
	"go/token"
	"path"
	"slices"

	"github.com/jatassi/slipstream/cmd/release/internal/lint"
)

func init() {
	lint.Register(lint.Rule{
		Name:  "surface-wired",
		Doc:   "surface.Mount calls every <owner>api.Routes and surface.Events lists every declared event (spec #129, Lint rules)",
		Check: checkSurfaceWired,
	})
}

const surfaceTree = "internal/node/surface"

var eventDeclarers = []string{"AdminEvent", "AccountEvent", "EveryoneEvent"}

type surfaceEntry struct {
	file, label string
	line        int
}

type surfaceIndex struct {
	repo           *lint.Repo
	livePkg        string
	routes, events map[string]surfaceEntry
	wired          map[string]bool
}

func checkSurfaceWired(repo *lint.Repo) ([]lint.Finding, error) {
	module, err := modulePath(repo)
	if err != nil {
		return nil, err
	}
	files, err := productFiles(repo)
	if err != nil {
		return nil, err
	}
	idx := surfaceIndex{
		repo:    repo,
		livePkg: path.Join(module, "internal/live"),
		routes:  map[string]surfaceEntry{},
		events:  map[string]surfaceEntry{},
		wired:   map[string]bool{},
	}
	for _, f := range files {
		idx.add(f, path.Join(module, path.Dir(f.Path)))
	}
	var findings []lint.Finding
	for _, declared := range []map[string]surfaceEntry{idx.routes, idx.events} {
		for key, e := range declared {
			if !idx.wired[key] {
				findings = append(findings, lint.Finding{Path: e.file, Line: e.line, Message: e.label})
			}
		}
	}
	return findings, nil
}

func (idx surfaceIndex) add(f lint.GoFile, pkg string) {
	imports := importNames(f)
	inSurface := path.Dir(f.Path) == surfaceTree
	for _, decl := range f.Syntax.Decls {
		switch d := decl.(type) {
		case *ast.FuncDecl:
			idx.addFunc(f, pkg, imports, inSurface, d)
		case *ast.GenDecl:
			if d.Tok == token.VAR {
				idx.addVars(f, pkg, imports, inSurface, d)
			}
		}
	}
}

func (idx surfaceIndex) addFunc(f lint.GoFile, pkg string, imports map[string]string, inSurface bool, fn *ast.FuncDecl) {
	if routesFunc(f, fn) {
		idx.routes[pkg+".Routes"] = idx.entry(f, fn.Name, f.Syntax.Name.Name+".Routes is never mounted in internal/node/surface")
	}
	if inSurface && fn.Recv == nil && fn.Name.Name == "Mount" {
		ast.Inspect(fn.Body, func(n ast.Node) bool {
			if call, ok := n.(*ast.CallExpr); ok {
				idx.wire(imports, call.Fun)
			}
			return true
		})
	}
}

func (idx surfaceIndex) addVars(f lint.GoFile, pkg string, imports map[string]string, inSurface bool, decl *ast.GenDecl) {
	for _, spec := range decl.Specs {
		vs, _ := spec.(*ast.ValueSpec)
		for i, name := range vs.Names[:min(len(vs.Names), len(vs.Values))] {
			if idx.declaresEvent(imports, vs.Values[i]) {
				idx.events[pkg+"."+name.Name] = idx.entry(f, name, f.Syntax.Name.Name+"."+name.Name+" is not listed in surface.Events")
			}
			if lit, ok := vs.Values[i].(*ast.CompositeLit); ok && inSurface && name.Name == "Events" {
				for _, elt := range lit.Elts {
					idx.wire(imports, elt)
				}
			}
		}
	}
}

func (idx surfaceIndex) entry(f lint.GoFile, at ast.Node, label string) surfaceEntry {
	return surfaceEntry{file: f.Path, label: label, line: idx.repo.Fset.Position(at.Pos()).Line}
}

func (idx surfaceIndex) wire(imports map[string]string, expr ast.Expr) {
	if pkg, name, ok := qualifiedName(imports, expr); ok {
		idx.wired[pkg+"."+name] = true
	}
}

func (idx surfaceIndex) declaresEvent(imports map[string]string, value ast.Expr) bool {
	call, ok := value.(*ast.CallExpr)
	if !ok {
		return false
	}
	generic, ok := call.Fun.(*ast.IndexExpr)
	if !ok {
		return false
	}
	pkg, name, ok := qualifiedName(imports, generic.X)
	return ok && pkg == idx.livePkg && slices.Contains(eventDeclarers, name)
}
