package rules

import (
	"go/ast"
	"go/token"
	"path"
	"slices"
	"strconv"

	"github.com/jatassi/slipstream/cmd/release/internal/lint"
)

func init() {
	lint.Register(lint.Rule{
		Name:  "config-writer",
		Doc:   "only config.Edit writes config.yaml, so every surface keeps the file's comments and passes the loader (spec #139, Architecture checks)",
		Check: checkConfigWriter,
	})
}

const (
	configWriter  = "internal/node/internal/config/edit.go"
	configPackage = "internal/node/internal/config"
)

// fileWriters are the calls that create, replace or write a file named by
// one of their arguments.
var fileWriters = map[string][]string{
	"os":                  {"WriteFile", "Create", "OpenFile", "Rename"},
	"internal/atomicfile": {"Write"},
}

func checkConfigWriter(repo *lint.Repo) ([]lint.Finding, error) {
	module, err := modulePath(repo)
	if err != nil {
		return nil, err
	}
	files, err := productFiles(repo)
	if err != nil {
		return nil, err
	}
	var findings []lint.Finding
	for _, f := range files {
		if f.Path == configWriter {
			continue
		}
		imports := importNames(f)
		ast.Inspect(f.Syntax, func(n ast.Node) bool {
			call, ok := n.(*ast.CallExpr)
			if !ok || !writesAFile(module, imports, call) || !namesConfigYAML(module, imports, call.Args) {
				return true
			}
			_, name, _ := qualifiedName(imports, call.Fun)
			findings = append(findings, lint.Finding{
				Path:    f.Path,
				Line:    repo.Fset.Position(call.Pos()).Line,
				Message: name + " writes config.yaml; change keys through config.Edit",
			})
			return true
		})
	}
	return findings, nil
}

func writesAFile(module string, imports map[string]string, call *ast.CallExpr) bool {
	pkg, name, ok := qualifiedName(imports, call.Fun)
	if !ok {
		return false
	}
	for writer, names := range fileWriters {
		if writer != "os" {
			writer = path.Join(module, writer)
		}
		if pkg == writer && slices.Contains(names, name) {
			return true
		}
	}
	return false
}

// namesConfigYAML reports an argument that spells config.yaml or names it
// through config.SourceYAML.
func namesConfigYAML(module string, imports map[string]string, args []ast.Expr) bool {
	found := false
	for _, arg := range args {
		ast.Inspect(arg, func(n ast.Node) bool {
			switch n := n.(type) {
			case *ast.BasicLit:
				s, err := strconv.Unquote(n.Value)
				found = found || n.Kind == token.STRING && err == nil && path.Base(s) == "config.yaml"
			case *ast.SelectorExpr:
				pkg, name, ok := qualifiedName(imports, n)
				found = found || ok && pkg == path.Join(module, configPackage) && name == "SourceYAML"
			}
			return !found
		})
	}
	return found
}
