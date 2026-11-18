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
		Name:  "subtitles-pure",
		Doc:   "Timing sync, the subtitle formats, the Speech Map format and languages do no IO and read no clock or randomness, so both sides of a Subtitles call compute the same answer (S46 (Subtitles), Packages and seams)",
		Check: checkSubtitlesPure,
	})
}

// pureTrees are the pure packages and every package below them. A folder
// that doesn't exist yet has no files to check.
var pureTrees = []string{"internal/subtitles/timing/...", "internal/subtitles/cues/...", "internal/media/speech/...", "internal/language/..."}

// impureImports are refused with every package below them. os itself may
// be imported for its error values, and any other os name is refused.
var impureImports = []string{"os", "net", "math/rand", "crypto/rand"}

var impureModuleImports = []string{"internal/storage", "internal/rpc", "internal/link"}

var clockReads = []string{"Now", "Since", "Until"}

func checkSubtitlesPure(repo *lint.Repo) ([]lint.Finding, error) {
	module, err := modulePath(repo)
	if err != nil {
		return nil, err
	}
	files, err := repo.GoSyntax()
	if err != nil {
		return nil, err
	}
	var findings []lint.Finding
	report := func(f lint.GoFile, n ast.Node, message string) {
		findings = append(findings, lint.Finding{Path: f.Path, Line: repo.Fset.Position(n.Pos()).Line, Message: message})
	}
	for _, f := range files {
		if !purePackage(f.Path) {
			continue
		}
		for _, imp := range f.Syntax.Imports {
			if p, _ := strconv.Unquote(imp.Path.Value); impure(module, p) {
				report(f, imp, p+" imported by a pure package; its caller does the IO and passes the result")
			}
		}
		imports := importNames(f)
		ast.Inspect(f.Syntax, func(n ast.Node) bool {
			if sel, ok := n.(*ast.SelectorExpr); ok {
				if message, ok := impureName(imports, sel); ok {
					report(f, sel, message)
				}
			}
			return true
		})
	}
	return findings, nil
}

func impure(module, p string) bool {
	return p != "os" && (slices.ContainsFunc(impureImports, func(tree string) bool { return inTree(p, tree) }) ||
		slices.ContainsFunc(impureModuleImports, func(tree string) bool { return inTree(p, path.Join(module, tree)) }))
}

func impureName(imports map[string]string, sel *ast.SelectorExpr) (string, bool) {
	pkg, name, ok := qualifiedName(imports, sel)
	switch {
	case !ok:
	case pkg == "os" && !strings.HasPrefix(name, "Err"):
		return "os." + name + " in a pure package; only os error values are allowed", true
	case pkg == "time" && slices.Contains(clockReads, name):
		return "time." + name + " in a pure package; take the time from the caller", true
	}
	return "", false
}

// purePackage is a pure package's non-test file. A package named like
// timingtest is a test helper, which may draw seeded randomness.
func purePackage(file string) bool {
	dir := path.Dir(file)
	return !strings.HasSuffix(file, "_test.go") && !strings.HasSuffix(path.Base(dir), "test") && inDirs(dir, pureTrees)
}
