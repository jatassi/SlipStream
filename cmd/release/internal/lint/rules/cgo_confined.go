package rules

import (
	"path"
	"slices"
	"strconv"

	"github.com/jatassi/slipstream/cmd/release/internal/lint"
)

var nativePackages = []string{
	"internal/torrent/libtorrent",
}

func init() {
	lint.Register(lint.Rule{
		Name:  "cgo-confined",
		Doc:   `import "C" appears only in the declared native packages (spec #126, Architecture-lint runner)`,
		Check: checkCgoConfined,
	})
}

func checkCgoConfined(repo *lint.Repo) ([]lint.Finding, error) {
	files, err := repo.GoSyntax()
	if err != nil {
		return nil, err
	}
	var findings []lint.Finding
	for _, f := range files {
		if slices.Contains(nativePackages, path.Dir(f.Path)) {
			continue
		}
		for _, imp := range f.Syntax.Imports {
			if p, _ := strconv.Unquote(imp.Path.Value); p == "C" {
				findings = append(findings, lint.Finding{
					Path:    f.Path,
					Line:    repo.Fset.Position(imp.Pos()).Line,
					Message: `import "C" outside the declared native packages`,
				})
			}
		}
	}
	return findings, nil
}
