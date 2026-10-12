package rules

import (
	"go/ast"
	"path"
	"slices"
	"strconv"
	"strings"

	"github.com/jatassi/slipstream/cmd/release/internal/lint"
)

func init() {
	lint.Register(lint.Rule{
		Name:  "sql-confined",
		Doc:   "database/sql, the SQLite driver, goose and raw statements appear only in internal/storage and sqlc's generated code (spec #128, Connections, writes and transactions)",
		Check: checkSQLConfined,
	})
}

const storageMechanism = "internal/storage"

var rawStatements = []string{"ExecContext", "QueryContext", "QueryRowContext", "PrepareContext"}

func checkSQLConfined(repo *lint.Repo) ([]lint.Finding, error) {
	files, err := productFiles(repo)
	if err != nil {
		return nil, err
	}
	var findings []lint.Finding
	for _, f := range files {
		if path.Dir(f.Path) == storageMechanism || sqlcGenerated(f) {
			continue
		}
		for _, imp := range f.Syntax.Imports {
			if p, _ := strconv.Unquote(imp.Path.Value); confinedImport(p) {
				findings = append(findings, lint.Finding{
					Path:    f.Path,
					Line:    repo.Fset.Position(imp.Pos()).Line,
					Message: "import " + p + " outside internal/storage and generated queries",
				})
			}
		}
		ast.Inspect(f.Syntax, func(n ast.Node) bool {
			call, ok := n.(*ast.CallExpr)
			if !ok {
				return true
			}
			if sel, ok := call.Fun.(*ast.SelectorExpr); ok && slices.Contains(rawStatements, sel.Sel.Name) {
				findings = append(findings, lint.Finding{
					Path:    f.Path,
					Line:    repo.Fset.Position(sel.Sel.Pos()).Line,
					Message: sel.Sel.Name + " outside generated queries; add the statement to queries/*.sql",
				})
			}
			return true
		})
	}
	return findings, nil
}

func confinedImport(p string) bool {
	within := func(root string) bool { return p == root || strings.HasPrefix(p, root+"/") }
	return p == "database/sql" || within("modernc.org/sqlite") || within("github.com/pressly/goose")
}
