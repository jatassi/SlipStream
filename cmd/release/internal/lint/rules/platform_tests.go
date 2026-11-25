package rules

import (
	"fmt"
	"go/ast"
	"go/build/constraint"
	"go/token"
	"maps"
	"path"
	"regexp"
	"slices"
	"strings"

	"github.com/jatassi/slipstream/cmd/release/internal/lint"
	"github.com/jatassi/slipstream/cmd/release/internal/platform"
)

func init() {
	lint.Register(lint.Rule{
		Name:  "platform-tests",
		Doc:   "a package with OS-specific code has a test in the platform suite, which the train and the merge check run on Windows, macOS and linux-arm64, and each entry of the suite picks a test its package declares",
		Check: func(repo *lint.Repo) ([]lint.Finding, error) { return checkPlatformTests(repo, platform.Tests) },
	})
}

const platformRegistry = "cmd/release/internal/platform/platform.go"

// osNames are the GOOS values a file name or a build constraint can name,
// and unix, which a constraint matches on every Unix.
var osNames = []string{
	"aix", "android", "darwin", "dragonfly", "freebsd", "hurd", "illumos", "ios", "js", "linux",
	"netbsd", "openbsd", "plan9", "solaris", "unix", "wasip1", "windows", "zos",
}

var archNames = []string{
	"386", "amd64", "arm", "arm64", "loong64", "mips", "mips64", "mips64le", "mipsle",
	"ppc64", "ppc64le", "riscv64", "s390x", "wasm",
}

// osSignal is where a file shows it holds code for some OSes only.
type osSignal struct {
	file string
	line int
}

func checkPlatformTests(repo *lint.Repo, tests []platform.Test) ([]lint.Finding, error) {
	files, err := repo.GoSyntax()
	if err != nil {
		return nil, err
	}
	osPackages := map[string]osSignal{}
	declared := map[string][]string{}
	for _, f := range files {
		dir := path.Dir(f.Path)
		if strings.HasSuffix(f.Path, "_test.go") {
			declared[dir] = append(declared[dir], topLevelTests(f)...)
		}
		if _, seen := osPackages[dir]; seen || ast.IsGenerated(f.Syntax) {
			continue
		}
		if line, ok := osSpecific(repo.Fset, f); ok {
			osPackages[dir] = osSignal{f.Path, line}
		}
	}
	var findings []lint.Finding
	registered := map[string]bool{}
	for _, t := range tests {
		registered[t.Package] = true
		for _, c := range t.Covers {
			registered[c] = true
			if _, ok := osPackages[c]; !ok {
				findings = append(findings, lint.Finding{Path: platformRegistry, Line: 1, Message: fmt.Sprintf("The entry for %s covers %s, which has no OS-specific code. Remove it from Covers.", t.Package, c)})
			}
		}
		run, err := regexp.Compile(t.Run)
		switch {
		case err != nil:
			findings = append(findings, lint.Finding{Path: platformRegistry, Line: 1, Message: fmt.Sprintf("The entry for %s runs %q, which isn't a regular expression.", t.Package, t.Run)})
		case !slices.ContainsFunc(declared[t.Package], run.MatchString):
			findings = append(findings, lint.Finding{Path: platformRegistry, Line: 1, Message: fmt.Sprintf("The entry for %s runs %q, which matches none of its tests. Name a test the package declares.", t.Package, t.Run)})
		}
	}
	for _, dir := range slices.Sorted(maps.Keys(osPackages)) {
		if !registered[dir] {
			s := osPackages[dir]
			findings = append(findings, lint.Finding{Path: s.file, Line: s.line, Message: fmt.Sprintf("%s has OS-specific code and no test in the platform suite. Add a test that runs this code to %s.", dir, platformRegistry)})
		}
	}
	return findings, nil
}

// osSpecific reports the first line where f shows it holds code for some
// OSes only: an OS in its name or its build constraint, for any file, or a
// comparison of runtime.GOOS with a literal outside a test file. A test's
// own GOOS checks mostly skip or adjust what it expects, so they leave the
// package's code the same on every OS.
func osSpecific(fset *token.FileSet, f lint.GoFile) (int, bool) {
	if namesOS(path.Base(f.Path)) {
		return 1, true
	}
	if line, ok := osConstraint(fset, f); ok {
		return line, true
	}
	if strings.HasSuffix(f.Path, "_test.go") {
		return 0, false
	}
	return goosComparison(fset, f)
}

func osConstraint(fset *token.FileSet, f lint.GoFile) (int, bool) {
	for _, group := range f.Syntax.Comments {
		if group.Pos() > f.Syntax.Package {
			break
		}
		for _, c := range group.List {
			if expr, err := constraint.Parse(c.Text); err == nil && constraint.IsGoBuild(c.Text) && constrainsOS(expr) {
				return fset.Position(c.Pos()).Line, true
			}
		}
	}
	return 0, false
}

func goosComparison(fset *token.FileSet, f lint.GoFile) (int, bool) {
	imports := importNames(f)
	isGOOS := func(e ast.Expr) bool {
		pkg, name, ok := qualifiedName(imports, e)
		return ok && pkg == "runtime" && name == "GOOS"
	}
	line := 0
	ast.Inspect(f.Syntax, func(n ast.Node) bool {
		switch n := n.(type) {
		case *ast.BinaryExpr:
			if (n.Op == token.EQL || n.Op == token.NEQ) && (isGOOS(n.X) && isString(n.Y) || isString(n.X) && isGOOS(n.Y)) {
				line = fset.Position(n.Pos()).Line
			}
		case *ast.SwitchStmt:
			if n.Tag != nil && isGOOS(n.Tag) {
				line = fset.Position(n.Pos()).Line
			}
		}
		return line == 0
	})
	return line, line > 0
}

// namesOS reports whether a file name ends in _GOOS or _GOOS_GOARCH, before
// any _test, as go build reads it, or in _unix.
func namesOS(name string) bool {
	parts := strings.Split(strings.TrimSuffix(strings.TrimSuffix(name, ".go"), "_test"), "_")
	n := len(parts)
	return n > 1 && slices.Contains(osNames, parts[n-1]) ||
		n > 2 && slices.Contains(osNames, parts[n-2]) && slices.Contains(archNames, parts[n-1])
}

func constrainsOS(expr constraint.Expr) bool {
	switch e := expr.(type) {
	case *constraint.TagExpr:
		return slices.Contains(osNames, e.Tag)
	case *constraint.NotExpr:
		return constrainsOS(e.X)
	case *constraint.AndExpr:
		return constrainsOS(e.X) || constrainsOS(e.Y)
	case *constraint.OrExpr:
		return constrainsOS(e.X) || constrainsOS(e.Y)
	}
	return false
}

func isString(e ast.Expr) bool {
	lit, ok := e.(*ast.BasicLit)
	return ok && lit.Kind == token.STRING
}

// topLevelTests is the Test functions a test file declares, the names go
// test -run matches at the top level.
func topLevelTests(f lint.GoFile) []string {
	var tests []string
	for _, d := range f.Syntax.Decls {
		fn, ok := d.(*ast.FuncDecl)
		if ok && fn.Recv == nil && strings.HasPrefix(fn.Name.Name, "Test") && fn.Name.Name != "TestMain" {
			tests = append(tests, fn.Name.Name)
		}
	}
	return tests
}
