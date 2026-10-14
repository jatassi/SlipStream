package rules

import (
	"go/ast"
	"go/token"
	"maps"
	"slices"
	"strconv"
	"strings"

	"github.com/jatassi/slipstream/cmd/release/internal/lint"
)

func init() {
	lint.Register(lint.Rule{
		Name:  "deadline-waits",
		Doc:   "the end-to-end tier waits on conditions under the test's deadline and skips only with an issue link; the simulation tier and synctest tests wait on fake time (spec #130, Lint rules; spec #132, Lints and CI checks; spec #135, Guards)",
		Check: checkDeadlineWaits,
	})
}

var endToEndTrees = []string{"e2e/", "fakes/world/", "fakes/events/"}

// pollingWaits are the tier's two polling waits, each under its context's
// deadline: worldtest.Until on a condition and events.Wait on a record.
var pollingWaits = []string{"fakes/world/worldtest/until.go", "fakes/events/wait.go"}

const (
	// simTree runs inside a testing/synctest bubble, whose waits are fake
	// time, as is every file importing synctest.
	simTree      = "fakes/world/sim/"
	synctestPath = "testing/synctest"
	issueLink    = "https://github.com/jatassi/SlipStream/issues/"

	useUntil   = "wait on a condition with worldtest.Until or on a record with events.Wait"
	useContext = "take the context from worldtest.Context"
)

type deadlineWait struct {
	pkg         string
	names       []string
	replacement string
}

var deadlineWaits = []deadlineWait{
	{pkg: "time", replacement: useUntil, names: []string{"Sleep", "After", "Tick", "NewTimer", "NewTicker", "AfterFunc"}},
	{pkg: "context", replacement: useContext, names: []string{"Background", "TODO"}},
}

var (
	testingPointers = []string{"T", "B", "F"}
	skipMethods     = []string{"Skip", "Skipf", "SkipNow"}
)

func checkDeadlineWaits(repo *lint.Repo) ([]lint.Finding, error) {
	files, err := repo.GoSyntax()
	if err != nil {
		return nil, err
	}
	var findings []lint.Finding
	report := func(f lint.GoFile, n ast.Node, message string) {
		findings = append(findings, lint.Finding{Path: f.Path, Line: repo.Fset.Position(n.Pos()).Line, Message: message})
	}
	for _, f := range files {
		if !underAny(f.Path, endToEndTrees) || slices.Contains(pollingWaits, f.Path) || strings.HasPrefix(f.Path, simTree) {
			continue
		}
		imports := importNames(f)
		if slices.Contains(slices.Collect(maps.Values(imports)), synctestPath) {
			continue
		}
		ast.Inspect(f.Syntax, func(n ast.Node) bool {
			if message, ok := deadlineFinding(imports, n); ok {
				report(f, n, message)
			}
			return true
		})
	}
	return findings, nil
}

func deadlineFinding(imports map[string]string, n ast.Node) (string, bool) {
	switch n := n.(type) {
	case *ast.CallExpr:
		if skip, ok := unlinkedSkip(imports, n); ok {
			return skip + " without an issue link; a flaky test gets an issue first", true
		}
	case *ast.SelectorExpr:
		if pkg, name, ok := qualifiedName(imports, n); ok {
			if w, ok := bannedWait(pkg, name); ok {
				return pkg + "." + name + " in the end-to-end tier; " + w.replacement, true
			}
		} else if recv, ok := testingReceiver(imports, n.X); ok && n.Sel.Name == "Context" {
			return recv + ".Context in the end-to-end tier; " + useContext, true
		}
	}
	return "", false
}

func bannedWait(pkg, name string) (deadlineWait, bool) {
	for _, w := range deadlineWaits {
		if w.pkg == pkg && slices.Contains(w.names, name) {
			return w, true
		}
	}
	return deadlineWait{}, false
}

func unlinkedSkip(imports map[string]string, call *ast.CallExpr) (string, bool) {
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok || !slices.Contains(skipMethods, sel.Sel.Name) {
		return "", false
	}
	x, ok := sel.X.(*ast.Ident)
	if !ok {
		return "", false
	}
	if _, ok := testingReceiver(imports, x); !ok {
		return "", false
	}
	if len(call.Args) > 0 {
		if lit, ok := call.Args[0].(*ast.BasicLit); ok && lit.Kind == token.STRING {
			msg, _ := strconv.Unquote(lit.Value)
			if strings.Contains(msg, issueLink) {
				return "", false
			}
		}
	}
	return x.Name + "." + sel.Sel.Name, true
}

func testingReceiver(imports map[string]string, x ast.Expr) (string, bool) {
	ident, ok := x.(*ast.Ident)
	if !ok || ident.Obj == nil {
		return "", false
	}
	var typ ast.Expr
	switch decl := ident.Obj.Decl.(type) {
	case *ast.Field:
		typ = decl.Type
	case *ast.ValueSpec:
		typ = decl.Type
	}
	if star, ok := typ.(*ast.StarExpr); ok {
		if pkg, name, ok := qualifiedName(imports, star.X); ok && pkg == "testing" && slices.Contains(testingPointers, name) {
			return "(*testing." + name + ")", true
		}
		return "", false
	}
	if pkg, name, ok := qualifiedName(imports, typ); ok && pkg == "testing" && name == "TB" {
		return "testing.TB", true
	}
	return "", false
}
