package rules

import (
	"go/ast"
	"path"
	"regexp"
	"strconv"
	"strings"

	"github.com/jatassi/slipstream/cmd/release/internal/lint"
)

var nonProductTrees = []string{"fakes/", "cmd/release/", "cmd/fakeworld/"}

func productFiles(repo *lint.Repo) ([]lint.GoFile, error) {
	files, err := repo.GoSyntax()
	if err != nil {
		return nil, err
	}
	var product []lint.GoFile
	for _, f := range files {
		if strings.HasSuffix(f.Path, "_test.go") || (underAny(f.Path, nonProductTrees) && !isContentTypePackage(path.Dir(f.Path))) {
			continue
		}
		product = append(product, f)
	}
	return product, nil
}

const (
	shippedContentTypes = "internal/contenttypes"
	testContentTypes    = "fakes/contenttypes"
)

// contentTypeScope is the Content Type scope rule (spec #133, Packages) for
// the package in dir. id names the Content Type whose tree holds dir, or is
// empty. fake is the fake package beneath that type holding dir: any
// directory under fakes/contenttypes/<id>/ other than internal, and its
// subtree. A Content Type package has an id and no fake.
func contentTypeScope(dir string) (id, fake string) {
	for _, root := range []string{shippedContentTypes, testContentTypes} {
		rest, ok := strings.CutPrefix(dir, root+"/")
		if !ok {
			continue
		}
		elems := strings.Split(rest, "/")
		if root == testContentTypes && len(elems) > 1 && elems[1] != "internal" {
			return elems[0], path.Join(root, elems[0], elems[1])
		}
		return elems[0], ""
	}
	return "", ""
}

func isContentTypePackage(dir string) bool {
	id, fake := contentTypeScope(dir)
	return id != "" && fake == ""
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

const contentTypePkg = "internal/contenttype"

// isDefineCall reports whether call is contenttype.Define, with or without
// explicit type arguments.
func isDefineCall(module string, imports map[string]string, call *ast.CallExpr) bool {
	fun := call.Fun
	switch generic := fun.(type) {
	case *ast.IndexExpr:
		fun = generic.X
	case *ast.IndexListExpr:
		fun = generic.X
	}
	pkg, name, ok := qualifiedName(imports, fun)
	return ok && pkg == path.Join(module, contentTypePkg) && name == "Define"
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
