package rules

import (
	"go/ast"
	"path"
	"regexp"
	"strconv"
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

var majorVersion = regexp.MustCompile(`^v\d+$`)

func importNames(f lint.GoFile) map[string]string {
	names := map[string]string{}
	for _, imp := range f.Syntax.Imports {
		p, _ := strconv.Unquote(imp.Path.Value)
		name := path.Base(p)
		if majorVersion.MatchString(name) {
			name = path.Base(path.Dir(p))
		}
		if imp.Name != nil {
			name = imp.Name.Name
		}
		names[name] = p
	}
	return names
}

func ownerAPIPackage(f lint.GoFile) bool {
	name := f.Syntax.Name.Name
	return strings.HasSuffix(name, "api") && name != "api"
}

func qualifiedName(imports map[string]string, expr ast.Expr) (pkg, name string, ok bool) {
	sel, isSel := expr.(*ast.SelectorExpr)
	if !isSel {
		return "", "", false
	}
	x, isIdent := sel.X.(*ast.Ident)
	if !isIdent || x.Obj != nil || imports[x.Name] == "" {
		return "", "", false
	}
	return imports[x.Name], sel.Sel.Name, true
}
