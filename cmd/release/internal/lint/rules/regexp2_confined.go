package rules

import (
	"path"
	"strconv"
	"strings"

	"github.com/jatassi/slipstream/cmd/release/internal/lint"
)

func init() {
	lint.Register(lint.Rule{
		Name:  "regexp2-confined",
		Doc:   "regexp2 is imported only by internal/regex, so every .NET pattern compiles with its \\p{IsBlock} rewrite and match limit",
		Check: checkRegexp2Confined,
	})
}

const (
	regexPackage = "internal/regex"
	regexp2      = "github.com/dlclark/regexp2"
)

func checkRegexp2Confined(repo *lint.Repo) ([]lint.Finding, error) {
	files, err := repo.GoSyntax()
	if err != nil {
		return nil, err
	}
	var findings []lint.Finding
	for _, f := range files {
		if path.Dir(f.Path) == regexPackage {
			continue
		}
		for _, imp := range f.Syntax.Imports {
			if p, _ := strconv.Unquote(imp.Path.Value); p == regexp2 || strings.HasPrefix(p, regexp2+"/") {
				findings = append(findings, lint.Finding{
					Path:    f.Path,
					Line:    repo.Fset.Position(imp.Pos()).Line,
					Message: "regexp2 imported outside internal/regex; compile with regex.Compile",
				})
			}
		}
	}
	return findings, nil
}
