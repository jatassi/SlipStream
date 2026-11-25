package rules

import (
	"path"
	"strconv"
	"strings"

	"github.com/jatassi/slipstream/cmd/release/internal/lint"
)

func init() {
	lint.Register(lint.Rule{
		Name:  "netfacts-sides",
		Doc:   "a Node's network facts reach no Controller owner, so a Node's facts come from what the call or the intent carries (S51 (Reachability diagnostic), netfacts)",
		Check: checkNetfactsSides,
	})
}

var (
	netfactsSide    = []string{"internal/netfacts/..."}
	netfactsForbids = []string{
		"internal/reachability/...", "internal/nodes/...", "internal/mediaservers/...",
		"internal/identity/...", "internal/tasks/...", "internal/health/...",
	}
)

// checkNetfactsSides reads non-test files only: a test may stand up a
// Controller around a Node.
func checkNetfactsSides(repo *lint.Repo) ([]lint.Finding, error) {
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
		if strings.HasSuffix(f.Path, "_test.go") || !inDirs(path.Dir(f.Path), netfactsSide) {
			continue
		}
		for _, imp := range f.Syntax.Imports {
			p, _ := strconv.Unquote(imp.Path.Value)
			pkg, ok := strings.CutPrefix(p, module+"/")
			if !ok || !inDirs(pkg, netfactsForbids) {
				continue
			}
			findings = append(findings, lint.Finding{Path: f.Path, Line: repo.Fset.Position(imp.Pos()).Line, Message: pkg + " imported by a Node's network facts; take what a Node needs from the call or the intent"})
		}
	}
	return findings, nil
}
