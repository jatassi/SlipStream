package rules

import (
	"path"
	"strconv"
	"strings"

	"github.com/jatassi/slipstream/cmd/release/internal/lint"
)

func init() {
	lint.Register(lint.Rule{
		Name:  "huma-confined",
		Doc:   "Huma is imported only by internal/api (spec #129, Lint rules)",
		Check: checkHumaConfined,
	})
}

const (
	apiMechanism = "internal/api"
	humaModule   = "github.com/danielgtaylor/huma"
)

func checkHumaConfined(repo *lint.Repo) ([]lint.Finding, error) {
	files, err := repo.GoSyntax()
	if err != nil {
		return nil, err
	}
	var findings []lint.Finding
	for _, f := range files {
		if path.Dir(f.Path) == apiMechanism {
			continue
		}
		for _, imp := range f.Syntax.Imports {
			if p, _ := strconv.Unquote(imp.Path.Value); p == humaModule || strings.HasPrefix(p, humaModule+"/") {
				findings = append(findings, lint.Finding{
					Path:    f.Path,
					Line:    repo.Fset.Position(imp.Pos()).Line,
					Message: "huma imported outside internal/api; register operations with api.Register",
				})
			}
		}
	}
	return findings, nil
}
