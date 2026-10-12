package rules

import (
	"strings"

	"github.com/jatassi/slipstream/cmd/release/internal/lint"
)

var nonProductTrees = []string{"fakes/", "cmd/release/"}

func productFiles(repo *lint.Repo) ([]lint.GoFile, error) {
	files, err := repo.GoSyntax()
	if err != nil {
		return nil, err
	}
	var product []lint.GoFile
	for _, f := range files {
		if strings.HasSuffix(f.Path, "_test.go") || underAny(f.Path, nonProductTrees) {
			continue
		}
		product = append(product, f)
	}
	return product, nil
}

func underAny(file string, trees []string) bool {
	for _, tree := range trees {
		if strings.HasPrefix(file, tree) {
			return true
		}
	}
	return false
}
