package rules

import (
	"fmt"
	"go/ast"
	"path"
	"strings"

	"github.com/jatassi/slipstream/cmd/release/internal/lint"
)

func init() {
	lint.Register(lint.Rule{
		Name:  "install-marker",
		Doc:   "install.Read and install.Write are called only from ProcessEnv and cmd/release (spec #181, Lints and linkage)",
		Check: checkInstallMarker,
	})
}

const installPkg = "internal/install"

type markerUse struct{ dir, fn string }

var markerUses = map[string][]markerUse{
	"Read":  {{dir: "internal/node", fn: "ProcessEnv"}, {dir: "cmd/release/..."}},
	"Write": {{dir: "cmd/release/..."}},
}

var markerAdvice = map[string]string{
	"Read":  "ProcessEnv reads the marker once; use Env.Install",
	"Write": "only whoever places a binary writes its marker",
}

func checkInstallMarker(repo *lint.Repo) ([]lint.Finding, error) {
	module, err := modulePath(repo)
	if err != nil {
		return nil, err
	}
	files, err := repo.GoSyntax()
	if err != nil {
		return nil, err
	}
	var findings []lint.Finding
	for _, f := range files {
		dir := path.Dir(f.Path)
		if strings.HasSuffix(f.Path, "_test.go") || dir == installPkg {
			continue
		}
		imports := importNames(f)
		for _, decl := range f.Syntax.Decls {
			fn := ""
			if d, ok := decl.(*ast.FuncDecl); ok && d.Recv == nil {
				fn = d.Name.Name
			}
			ast.Inspect(decl, func(n ast.Node) bool {
				pkg, name, ok := qualifiedName(imports, asExpr(n))
				if !ok || pkg != path.Join(module, installPkg) || markerUses[name] == nil || markerAllowed(name, dir, fn) {
					return true
				}
				findings = append(findings, lint.Finding{
					Path:    f.Path,
					Line:    repo.Fset.Position(n.Pos()).Line,
					Message: fmt.Sprintf("install.%s named outside its owners; %s", name, markerAdvice[name]),
				})
				return true
			})
		}
	}
	return findings, nil
}

func asExpr(n ast.Node) ast.Expr {
	e, _ := n.(ast.Expr)
	return e
}

func markerAllowed(name, dir, fn string) bool {
	for _, u := range markerUses[name] {
		if inDirs(dir, []string{u.dir}) && (u.fn == "" || u.fn == fn) {
			return true
		}
	}
	return false
}
