package rules

import (
	"go/ast"
	"path"
	"strconv"
	"strings"

	"github.com/jatassi/slipstream/cmd/release/internal/lint"
)

func init() {
	lint.Register(lint.Rule{
		Name:  "sidecar-writer",
		Doc:   "only the Keeper (keeper*.go) and the map drain (maps.go) of internal/subtitles reach SidecarService, so every Sidecar byte effect is settled from rows, one file at a time (S46 (Subtitles), The Keeper)",
		Check: checkSidecarWriter,
	})
}

// checkSidecarWriter reads non-test files only, and not subtitlestest,
// whose world builds the calls the Controller is given.
func checkSidecarWriter(repo *lint.Repo) ([]lint.Finding, error) {
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
		if !sidecarWriterScope(f.Path) {
			continue
		}
		storing := importName(f.Syntax, module+"/internal/storing")
		ast.Inspect(f.Syntax, func(n ast.Node) bool {
			if msg := sidecarReach(n, storing); msg != "" {
				findings = append(findings, lint.Finding{Path: f.Path, Line: repo.Fset.Position(n.Pos()).Line, Message: msg + " outside the Keeper and the map drain; write rows for the Keeper to settle"})
			}
			return true
		})
	}
	return findings, nil
}

// sidecarWriterScope is whether the rule reads p: internal/subtitles but
// its tests, subtitlestest, the Keeper and the map drain.
func sidecarWriterScope(p string) bool {
	dir, base := path.Split(p)
	dir = path.Clean(dir)
	switch {
	case strings.HasSuffix(base, "_test.go"), !inDirs(dir, []string{"internal/subtitles/..."}), inDirs(dir, []string{"internal/subtitles/subtitlestest/..."}):
		return false
	}
	return dir != "internal/subtitles" || !strings.HasPrefix(base, "keeper") && base != "maps.go"
}

// importName is the name f imports pkg under, "" when it doesn't.
func importName(f *ast.File, pkg string) string {
	for _, imp := range f.Imports {
		if p, _ := strconv.Unquote(imp.Path.Value); p == pkg {
			if imp.Name != nil {
				return imp.Name.Name
			}
			return path.Base(pkg)
		}
	}
	return ""
}

// sidecarReach names how n reaches SidecarService, "" when it doesn't: a
// storing.SidecarCalls name or a call of cfg.Storage.
func sidecarReach(n ast.Node, storing string) string {
	switch n := n.(type) {
	case *ast.SelectorExpr:
		if x, ok := n.X.(*ast.Ident); ok && storing != "" && x.Name == storing && strings.HasPrefix(n.Sel.Name, "SidecarCalls") {
			return storing + "." + n.Sel.Name
		}
	case *ast.CallExpr:
		if s, ok := n.Fun.(*ast.SelectorExpr); ok && s.Sel.Name == "Storage" && endsInCfg(s.X) {
			return "cfg.Storage called"
		}
	}
	return ""
}

func endsInCfg(e ast.Expr) bool {
	switch e := e.(type) {
	case *ast.Ident:
		return e.Name == "cfg"
	case *ast.SelectorExpr:
		return e.Sel.Name == "cfg"
	}
	return false
}
