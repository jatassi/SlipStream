package rules

import (
	"go/ast"
	"path"

	"github.com/jatassi/slipstream/cmd/release/internal/lint"
)

func init() {
	lint.Register(lint.Rule{
		Name:  "targets-apply-confined",
		Doc:   "targets.Apply is called only by internal/files, which refills every Work a Change touches in the same Write (S17 (Targets, Profiles and Editions), Changes that leave files unwanted)",
		Check: checkTargetsApplyConfined,
	})
}

const (
	targetsPackage = "internal/targets"
	filesPackage   = "internal/files"
)

func checkTargetsApplyConfined(repo *lint.Repo) ([]lint.Finding, error) {
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
		if dir := path.Dir(f.Path); dir == targetsPackage || dir == filesPackage {
			continue
		}
		imports := importNames(f)
		ast.Inspect(f.Syntax, func(n ast.Node) bool {
			sel, ok := n.(*ast.SelectorExpr)
			if !ok || sel.Sel.Name != "Apply" {
				return true
			}
			if x, ok := sel.X.(*ast.Ident); ok && imports[x.Name] == module+"/"+targetsPackage {
				findings = append(findings, lint.Finding{
					Path:    f.Path,
					Line:    repo.Fset.Position(sel.Pos()).Line,
					Message: "targets.Apply outside internal/files; save a Change with files.Change, which refills the Works it touches",
				})
			}
			return true
		})
	}
	return findings, nil
}
