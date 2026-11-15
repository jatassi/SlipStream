package rules

import (
	"encoding/json"
	"fmt"
	"go/ast"
	"slices"
	"strings"

	"github.com/jatassi/slipstream/cmd/release/internal/lint"
)

func init() {
	lint.Register(lint.Rule{
		Name:  "problem-detail-from-error",
		Doc:   "product code writes a problem's detail as a sentence for a person and never builds it from an error's text, beyond the counts web/lint/problem-detail-baseline.json still allows (WRITING.md rule 27)",
		Check: checkProblemDetail,
	})
}

// detailArgs is the argument of each api.Code method that becomes the
// problem's detail.
var detailArgs = map[string]int{"Because": 0, "Explained": 0, "On": 1, "Logged": 1}

const problemDetailBaseline = "web/lint/problem-detail-baseline.json"

func checkProblemDetail(repo *lint.Repo) ([]lint.Finding, error) {
	src, err := repo.Read(problemDetailBaseline)
	if err != nil {
		return nil, err
	}
	var known map[string]int
	if err := json.Unmarshal(src, &known); err != nil {
		return nil, fmt.Errorf("%s: %w", problemDetailBaseline, err)
	}
	files, err := productFiles(repo)
	if err != nil {
		return nil, err
	}
	var findings []lint.Finding
	counted := map[string]bool{}
	for _, f := range files {
		counted[f.Path] = true
		var lines []int
		ast.Inspect(f.Syntax, func(n ast.Node) bool {
			for _, detail := range problemDetails(n) {
				if errorText(detail) {
					lines = append(lines, repo.Fset.Position(detail.Pos()).Line)
				}
			}
			return true
		})
		findings = append(findings, againstBaseline(f.Path, lines, known[f.Path], problemDetailBaseline, "details built from an error", problemDetailMessage)...)
	}
	tracked, err := repo.Files()
	if err != nil {
		return nil, err
	}
	for file, allowed := range known {
		switch {
		case !slices.Contains(tracked, file):
			findings = append(findings, goneEntry(problemDetailBaseline, file))
		case !counted[file]:
			findings = append(findings, againstBaseline(file, nil, allowed, problemDetailBaseline, "details built from an error", problemDetailMessage)...)
		}
	}
	return findings, nil
}

// problemDetails are the expressions n makes a problem's detail or a field
// error's message from.
func problemDetails(n ast.Node) []ast.Expr {
	switch n := n.(type) {
	case *ast.CallExpr:
		if sel, ok := n.Fun.(*ast.SelectorExpr); ok {
			if i, ok := detailArgs[sel.Sel.Name]; ok && i < len(n.Args) {
				return []ast.Expr{n.Args[i]}
			}
		}
	case *ast.CompositeLit:
		if isFieldError(n.Type) {
			return fieldErrorMessage(n)
		}
		if list, ok := n.Type.(*ast.ArrayType); ok && isFieldError(list.Elt) {
			var messages []ast.Expr
			for _, elt := range n.Elts {
				if lit, ok := elt.(*ast.CompositeLit); ok && lit.Type == nil {
					messages = append(messages, fieldErrorMessage(lit)...)
				}
			}
			return messages
		}
	}
	return nil
}

func fieldErrorMessage(lit *ast.CompositeLit) []ast.Expr {
	for _, elt := range lit.Elts {
		if kv, ok := elt.(*ast.KeyValueExpr); ok {
			if key, ok := kv.Key.(*ast.Ident); ok && key.Name == "Message" {
				return []ast.Expr{kv.Value}
			}
		}
	}
	return nil
}

func isFieldError(t ast.Expr) bool {
	if sel, ok := t.(*ast.SelectorExpr); ok {
		t = sel.Sel
	}
	id, ok := t.(*ast.Ident)
	return ok && id.Name == "FieldError"
}

const problemDetailMessage = "a problem detail is a sentence for a person. Log the error and write the detail, per WRITING.md rule 27"

// errorText holds for an expression that calls Error() or formats an error
// variable with package fmt.
func errorText(expr ast.Expr) bool {
	found := false
	ast.Inspect(expr, func(n ast.Node) bool {
		if call, ok := n.(*ast.CallExpr); ok && !found {
			found = callsError(call) || formatsError(call)
		}
		return !found
	})
	return found
}

func callsError(call *ast.CallExpr) bool {
	sel, ok := call.Fun.(*ast.SelectorExpr)
	return ok && sel.Sel.Name == "Error" && len(call.Args) == 0
}

func formatsError(call *ast.CallExpr) bool {
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok || !strings.HasPrefix(sel.Sel.Name, "Sprint") {
		return false
	}
	if pkg, ok := sel.X.(*ast.Ident); !ok || pkg.Name != "fmt" {
		return false
	}
	return slices.ContainsFunc(call.Args, func(arg ast.Expr) bool {
		id, ok := arg.(*ast.Ident)
		return ok && (id.Name == "err" || strings.HasSuffix(id.Name, "Err"))
	})
}
