package rules

import (
	"go/ast"
	"path"
	"slices"

	"github.com/jatassi/slipstream/cmd/release/internal/lint"
)

func init() {
	lint.Register(lint.Rule{
		Name:  "routes-placement",
		Doc:   "api.Register, api.RegisterStream and api.RegisterDownload are referenced only from Routes in an <owner>api package (spec #129, Lint rules)",
		Check: checkRoutesPlacement,
	})
}

var registerFuncs = []string{"Register", "RegisterStream", "RegisterDownload"}

func checkRoutesPlacement(repo *lint.Repo) ([]lint.Finding, error) {
	module, err := modulePath(repo)
	if err != nil {
		return nil, err
	}
	files, err := productFiles(repo)
	if err != nil {
		return nil, err
	}
	apiPkg := path.Join(module, apiMechanism)
	var findings []lint.Finding
	for _, f := range files {
		imports := importNames(f)
		for _, decl := range f.Syntax.Decls {
			if routesFunc(f, decl) {
				continue
			}
			ast.Inspect(decl, func(n ast.Node) bool {
				sel, ok := n.(*ast.SelectorExpr)
				if !ok {
					return true
				}
				if pkg, name, ok := qualifiedName(imports, sel); ok && pkg == apiPkg && slices.Contains(registerFuncs, name) {
					findings = append(findings, lint.Finding{
						Path:    f.Path,
						Line:    repo.Fset.Position(sel.Pos()).Line,
						Message: "api." + name + " outside <owner>api.Routes",
					})
				}
				return true
			})
		}
	}
	return findings, nil
}

func routesFunc(f lint.GoFile, decl ast.Decl) bool {
	fn, ok := decl.(*ast.FuncDecl)
	return ok && fn.Recv == nil && fn.Name.Name == "Routes" && ownerAPIPackage(f)
}
