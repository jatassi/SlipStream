package rules

import (
	"go/ast"
	"go/build/constraint"
	"go/token"
	"path"
	"slices"
	"strconv"
	"strings"

	"github.com/jatassi/slipstream/cmd/release/internal/lint"
)

func init() {
	lint.Register(lint.Rule{
		Name:  "conformance-suites",
		Doc:   "a fake whose Spec says fake.OnlineSuite(suite) has conformance/<suite>/ with func Run, an online_test.go under //go:build online, and a test beside the fake that calls Run (spec #135, Guards)",
		Check: checkConformanceSuites,
	})
}

const (
	conformanceTree = "conformance"
	onlineTag       = "online"
)

type onlineSuite struct {
	file  string
	line  int
	dir   string
	suite string
}

func checkConformanceSuites(repo *lint.Repo) ([]lint.Finding, error) {
	module, err := modulePath(repo)
	if err != nil {
		return nil, err
	}
	files, err := repo.GoSyntax()
	if err != nil {
		return nil, err
	}
	fakePkg := path.Join(module, fakesTree, "fake")
	var suites []onlineSuite
	var findings []lint.Finding
	for _, f := range files {
		imports := importNames(f)
		ast.Inspect(f.Syntax, func(n ast.Node) bool {
			lit, ok := n.(*ast.CompositeLit)
			if !ok {
				return true
			}
			if pkg, name, ok := qualifiedName(imports, lit.Type); !ok || pkg != fakePkg || name != "Spec" {
				return true
			}
			call, ok := onlineCall(imports, fakePkg, lit)
			if !ok {
				return true
			}
			line := repo.Fset.Position(call.Pos()).Line
			suite, ok := stringArg(call)
			if !ok {
				findings = append(findings, lint.Finding{Path: f.Path, Line: line, Message: "fake.OnlineSuite takes a string literal naming its suite under conformance/"})
				return true
			}
			suites = append(suites, onlineSuite{file: f.Path, line: line, dir: path.Dir(f.Path), suite: suite})
			return true
		})
	}
	for _, s := range suites {
		for _, problem := range suiteProblems(files, module, s) {
			findings = append(findings, lint.Finding{Path: s.file, Line: s.line, Message: problem})
		}
	}
	return findings, nil
}

func onlineCall(imports map[string]string, fakePkg string, lit *ast.CompositeLit) (*ast.CallExpr, bool) {
	for _, elt := range lit.Elts {
		kv, ok := elt.(*ast.KeyValueExpr)
		if !ok {
			continue
		}
		if key, ok := kv.Key.(*ast.Ident); !ok || key.Name != "Online" {
			continue
		}
		call, ok := kv.Value.(*ast.CallExpr)
		if !ok {
			return nil, false
		}
		if pkg, name, ok := qualifiedName(imports, call.Fun); ok && pkg == fakePkg && name == "OnlineSuite" {
			return call, true
		}
	}
	return nil, false
}

func stringArg(call *ast.CallExpr) (string, bool) {
	if len(call.Args) != 1 {
		return "", false
	}
	lit, ok := call.Args[0].(*ast.BasicLit)
	if !ok || lit.Kind != token.STRING {
		return "", false
	}
	s, err := strconv.Unquote(lit.Value)
	return s, err == nil && s != ""
}

func suiteProblems(files []lint.GoFile, module string, s onlineSuite) []string {
	dir := path.Join(conformanceTree, s.suite)
	suitePkg := path.Join(module, dir)
	var hasRun, hasOnline, called bool
	for _, f := range files {
		switch {
		case path.Dir(f.Path) == dir && !strings.HasSuffix(f.Path, "_test.go"):
			hasRun = hasRun || declaresFunc(f, "Run")
		case f.Path == dir+"/online_test.go":
			hasOnline = buildsOnlyWith(f, onlineTag)
		case path.Dir(f.Path) == s.dir && strings.HasSuffix(f.Path, "_test.go"):
			called = called || callsFunc(f, suitePkg, "Run")
		}
	}
	var problems []string
	if !hasRun {
		problems = append(problems, dir+" declares no func Run")
	}
	if !hasOnline {
		problems = append(problems, dir+"/online_test.go is missing or not built only under //go:build "+onlineTag)
	}
	if !called {
		problems = append(problems, "no test in "+s.dir+" calls "+dir+".Run")
	}
	return problems
}

func declaresFunc(f lint.GoFile, name string) bool {
	return slices.ContainsFunc(f.Syntax.Decls, func(d ast.Decl) bool {
		fn, ok := d.(*ast.FuncDecl)
		return ok && fn.Recv == nil && fn.Name.Name == name
	})
}

// buildsOnlyWith reports whether f's //go:build line holds with tag set
// and fails without it.
func buildsOnlyWith(f lint.GoFile, tag string) bool {
	for _, group := range f.Syntax.Comments {
		if group.Pos() > f.Syntax.Package {
			break
		}
		for _, c := range group.List {
			if !constraint.IsGoBuild(c.Text) {
				continue
			}
			expr, err := constraint.Parse(c.Text)
			if err != nil {
				return false
			}
			return expr.Eval(func(t string) bool { return t == tag }) && !expr.Eval(func(string) bool { return false })
		}
	}
	return false
}

func callsFunc(f lint.GoFile, pkg, name string) bool {
	imports := importNames(f)
	found := false
	ast.Inspect(f.Syntax, func(n ast.Node) bool {
		if call, ok := n.(*ast.CallExpr); ok {
			if p, fn, ok := qualifiedName(imports, call.Fun); ok && p == pkg && fn == name {
				found = true
			}
		}
		return !found
	})
	return found
}
