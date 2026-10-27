package rules

import (
	"path"
	"strconv"
	"strings"

	"github.com/jatassi/slipstream/cmd/release/internal/lint"
)

func init() {
	lint.Register(lint.Rule{
		Name:  "sides",
		Doc:   "the Node's searching reaches no Controller store, and only the Node runtime names its generated service (S26 (Indexers and Trackers), Architecture)",
		Check: checkSides,
	})
}

// searchingSide is the Node's half of searching. Controller packages reach
// it through searching.Remote, so they name no generated search type.
const searchingSide = "internal/searching"

var (
	searchingForbids = []string{"internal/indexers/...", "internal/storage/..."}
	searchWireUsers  = []string{"internal/searching/...", "internal/node", "internal/pb/..."}
)

func checkSides(repo *lint.Repo) ([]lint.Finding, error) {
	module, err := modulePath(repo)
	if err != nil {
		return nil, err
	}
	files, err := repo.GoSyntax()
	if err != nil {
		return nil, err
	}
	wire := path.Join(module, "internal/pb/slipstream/search")
	var findings []lint.Finding
	for _, f := range files {
		dir := path.Dir(f.Path)
		node := inTree(dir, searchingSide)
		for _, imp := range f.Syntax.Imports {
			p, _ := strconv.Unquote(imp.Path.Value)
			pkg, ok := strings.CutPrefix(p, module+"/")
			var message string
			switch {
			case !ok:
			case node && (inDirs(pkg, searchingForbids) || strings.HasSuffix(pkg, "/internal/db")):
				message = pkg + " imported by the Node's searching; a job carries what the Node needs"
			case inTree(p, wire) && !inDirs(dir, searchWireUsers):
				message = "the generated search service imported outside the Node runtime; call it through searching.Remote"
			}
			if message != "" {
				findings = append(findings, lint.Finding{Path: f.Path, Line: repo.Fset.Position(imp.Pos()).Line, Message: message})
			}
		}
	}
	return findings, nil
}
