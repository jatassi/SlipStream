package rules

import (
	"errors"
	"fmt"
	"go/ast"
	"path"
	"slices"
	"strings"

	"github.com/jatassi/slipstream/cmd/release/internal/lint"
)

func init() {
	lint.Register(lint.Rule{
		Name:  "config-sections",
		Doc:   "every section default is called by the composition root, or passed to contenttype.Define, and called nowhere else (spec #127, Architecture lints; spec #133, Lints and CI checks)",
		Check: checkConfigSections,
	})
}

const compositionRoot = "internal/node/config.go"

var sectionDefaults = []string{"DefaultEndpoints", "DefaultTimings", "DefaultSettings"}

type sectionDefault struct {
	file string
	line int
}

type sectionCall struct {
	callee, target string
	line           int
}

func checkConfigSections(repo *lint.Repo) ([]lint.Finding, error) {
	module, err := modulePath(repo)
	if err != nil {
		return nil, err
	}
	files, err := productFiles(repo)
	if err != nil {
		return nil, err
	}
	var findings []lint.Finding
	declared := map[string]sectionDefault{}
	wired := map[string]bool{}
	for _, f := range files {
		pkg := path.Join(module, path.Dir(f.Path))
		for _, decl := range f.Syntax.Decls {
			if fn, ok := decl.(*ast.FuncDecl); ok && fn.Recv == nil && slices.Contains(sectionDefaults, fn.Name.Name) {
				declared[pkg+"."+fn.Name.Name] = sectionDefault{f.Path, repo.Fset.Position(fn.Pos()).Line}
			}
		}
		for _, target := range definedEndpoints(module, f, pkg) {
			wired[target] = true
		}
		for _, c := range sectionCalls(repo, f, pkg) {
			if f.Path == compositionRoot {
				wired[c.target] = true
				continue
			}
			findings = append(findings, lint.Finding{
				Path:    f.Path,
				Line:    c.line,
				Message: fmt.Sprintf("%s called outside %s; read the section from the Node's Config", c.callee, compositionRoot),
			})
		}
	}
	for target, d := range declared {
		if !wired[target] {
			name := target[strings.LastIndex(target, ".")+1:]
			findings = append(findings, lint.Finding{
				Path:    d.file,
				Line:    d.line,
				Message: fmt.Sprintf("%s is never called by %s; wire the section into node.Config", name, compositionRoot),
			})
		}
	}
	return findings, nil
}

func sectionCalls(repo *lint.Repo, f lint.GoFile, pkg string) []sectionCall {
	imports := importNames(f)
	var calls []sectionCall
	ast.Inspect(f.Syntax, func(n ast.Node) bool {
		if call, ok := n.(*ast.CallExpr); ok {
			if callee, target, ok := sectionDefaultCall(call, pkg, imports); ok {
				calls = append(calls, sectionCall{callee, target, repo.Fset.Position(call.Pos()).Line})
			}
		}
		return true
	})
	return calls
}

func definedEndpoints(module string, f lint.GoFile, pkg string) []string {
	imports := importNames(f)
	var targets []string
	ast.Inspect(f.Syntax, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok || len(call.Args) < 3 || !isDefineCall(module, imports, call) {
			return true
		}
		switch arg := call.Args[2].(type) {
		case *ast.Ident:
			if arg.Name == "DefaultEndpoints" {
				targets = append(targets, pkg+".DefaultEndpoints")
			}
		case *ast.SelectorExpr:
			if p, name, ok := qualifiedName(imports, arg); ok && name == "DefaultEndpoints" {
				targets = append(targets, p+".DefaultEndpoints")
			}
		}
		return true
	})
	return targets
}

func sectionDefaultCall(call *ast.CallExpr, pkg string, imports map[string]string) (callee, target string, ok bool) {
	switch fun := call.Fun.(type) {
	case *ast.Ident:
		if slices.Contains(sectionDefaults, fun.Name) {
			return fun.Name, pkg + "." + fun.Name, true
		}
	case *ast.SelectorExpr:
		x, isIdent := fun.X.(*ast.Ident)
		if isIdent && x.Obj == nil && imports[x.Name] != "" && slices.Contains(sectionDefaults, fun.Sel.Name) {
			return x.Name + "." + fun.Sel.Name, imports[x.Name] + "." + fun.Sel.Name, true
		}
	}
	return "", "", false
}

func modulePath(repo *lint.Repo) (string, error) {
	mod, err := repo.Read("go.mod")
	if err != nil {
		return "", err
	}
	for line := range strings.Lines(string(mod)) {
		if name, ok := strings.CutPrefix(strings.TrimSpace(line), "module "); ok {
			return strings.Trim(strings.TrimSpace(name), `"`), nil
		}
	}
	return "", errors.New("go.mod declares no module")
}
