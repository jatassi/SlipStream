package rules

import (
	"go/ast"
	"go/token"
	"path"
	"slices"
	"strings"

	"github.com/jatassi/slipstream/cmd/release/internal/lint"
)

func init() {
	lint.Register(lint.Rule{
		Name:  "fakes-wired",
		Doc:   "every package directly under fakes/, and every fake package beneath fakes/contenttypes/<id>/, declares one var Fake fake.Spec and fakes/world lists it (spec #130, Lint rules; spec #133, Lints and CI checks)",
		Check: checkFakesWired,
	})
}

const (
	fakesTree = "fakes"
	worldTree = "fakes/world"
)

var fakeInfrastructure = []string{"fake", "scenario", "world", "testbin", "fault"}

type fakesIndex struct {
	repo     *lint.Repo
	specPkg  string
	first    map[string]lint.GoFile
	declared map[string][]lint.Finding
	listed   map[string]bool
}

func checkFakesWired(repo *lint.Repo) ([]lint.Finding, error) {
	module, err := modulePath(repo)
	if err != nil {
		return nil, err
	}
	files, err := repo.GoSyntax()
	if err != nil {
		return nil, err
	}
	idx := fakesIndex{
		repo:     repo,
		specPkg:  path.Join(module, fakesTree, "fake"),
		first:    map[string]lint.GoFile{},
		declared: map[string][]lint.Finding{},
		listed:   map[string]bool{},
	}
	for _, f := range files {
		if !strings.HasSuffix(f.Path, "_test.go") {
			idx.add(f)
		}
	}
	var findings []lint.Finding
	for dir, first := range idx.first {
		decls := idx.declared[dir]
		if len(decls) == 0 {
			findings = append(findings, lint.Finding{
				Path:    first.Path,
				Line:    repo.Fset.Position(first.Syntax.Package).Line,
				Message: dir + " declares no var Fake fake.Spec",
			})
			continue
		}
		for _, again := range decls[1:] {
			again.Message = dir + " declares var Fake fake.Spec more than once"
			findings = append(findings, again)
		}
		if !idx.listed[path.Join(module, dir)+".Fake"] {
			decls[0].Message = dir + ".Fake is not listed in " + worldTree
			findings = append(findings, decls[0])
		}
	}
	return findings, nil
}

func (idx fakesIndex) add(f lint.GoFile) {
	dir := path.Dir(f.Path)
	isFake := isFakePackage(dir)
	if !isFake && dir != worldTree {
		return
	}
	if _, seen := idx.first[dir]; isFake && !seen {
		idx.first[dir] = f
	}
	imports := importNames(f)
	for _, decl := range f.Syntax.Decls {
		gen, ok := decl.(*ast.GenDecl)
		if !ok || gen.Tok != token.VAR {
			continue
		}
		for _, spec := range gen.Specs {
			vs, _ := spec.(*ast.ValueSpec)
			idx.addVars(f, isFake, imports, vs)
		}
	}
}

func isFakePackage(dir string) bool {
	if fake := scopeOf(dir).fake; fake != "" {
		return fake == dir
	}
	return path.Dir(dir) == fakesTree && dir != testContentTypes && !slices.Contains(fakeInfrastructure, path.Base(dir))
}

func (idx fakesIndex) addVars(f lint.GoFile, isFake bool, imports map[string]string, vs *ast.ValueSpec) {
	dir := path.Dir(f.Path)
	for i, name := range vs.Names {
		var value ast.Expr
		if i < len(vs.Values) {
			value = vs.Values[i]
		}
		switch {
		case isFake && name.Name == "Fake" && idx.declaresSpec(imports, vs.Type, value):
			idx.declared[dir] = append(idx.declared[dir], lint.Finding{
				Path: f.Path,
				Line: idx.repo.Fset.Position(name.Pos()).Line,
			})
		case dir == worldTree && name.Name == "fakes":
			idx.list(imports, value)
		}
	}
}

func (idx fakesIndex) declaresSpec(imports map[string]string, typ, value ast.Expr) bool {
	if typ != nil {
		return idx.isSpec(imports, typ)
	}
	lit, ok := value.(*ast.CompositeLit)
	return ok && idx.isSpec(imports, lit.Type)
}

func (idx fakesIndex) list(imports map[string]string, value ast.Expr) {
	lit, ok := value.(*ast.CompositeLit)
	if !ok {
		return
	}
	slice, ok := lit.Type.(*ast.ArrayType)
	if !ok || slice.Len != nil || !idx.isSpec(imports, slice.Elt) {
		return
	}
	for _, elt := range lit.Elts {
		if pkg, name, ok := qualifiedName(imports, elt); ok {
			idx.listed[pkg+"."+name] = true
		}
	}
}

func (idx fakesIndex) isSpec(imports map[string]string, expr ast.Expr) bool {
	pkg, name, ok := qualifiedName(imports, expr)
	return ok && pkg == idx.specPkg && name == "Spec"
}
