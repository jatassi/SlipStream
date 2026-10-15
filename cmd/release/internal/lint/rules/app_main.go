package rules

import (
	"path"
	"slices"
	"strconv"
	"strings"

	"github.com/jatassi/slipstream/cmd/release/internal/lint"
)

func init() {
	lint.Register(lint.Rule{
		Name:  "app-main",
		Doc:   "cmd/slipstream-app holds only main.go and its per-OS launch files, and imports std and golang.org/x/sys only (spec #181, The Windows trampoline)",
		Check: checkAppMain,
	})
}

const appMainDir = "cmd/slipstream-app"

// appMainFiles are the trampoline's files. Windows' launch sets a
// CreationFlags field that exists only in Windows' syscall package, so
// each OS's launch is its own file.
var appMainFiles = []string{"main.go", "launch_windows.go", "launch_darwin.go"}

func checkAppMain(repo *lint.Repo) ([]lint.Finding, error) {
	files, err := repo.Files()
	if err != nil {
		return nil, err
	}
	var findings []lint.Finding
	hasMain := false
	for _, f := range files {
		if path.Dir(f) != appMainDir {
			continue
		}
		hasMain = hasMain || path.Base(f) == "main.go"
		if !slices.Contains(appMainFiles, path.Base(f)) {
			findings = append(findings, lint.Finding{Path: f, Line: 1, Message: "the trampoline holds only " + strings.Join(appMainFiles, ", ")})
		}
	}
	if !hasMain {
		findings = append(findings, lint.Finding{Path: appMainDir + "/main.go", Line: 1, Message: "does not exist"})
	}
	syntax, err := repo.GoSyntax()
	if err != nil {
		return nil, err
	}
	for _, f := range syntax {
		if path.Dir(f.Path) != appMainDir {
			continue
		}
		for _, imp := range f.Syntax.Imports {
			p, _ := strconv.Unquote(imp.Path.Value)
			if !trampolineImport(p) {
				findings = append(findings, lint.Finding{
					Path:    f.Path,
					Line:    repo.Fset.Position(imp.Pos()).Line,
					Message: "imports " + p + "; the trampoline imports only std and golang.org/x/sys",
				})
			}
		}
	}
	return findings, nil
}

func trampolineImport(p string) bool {
	first, _, _ := strings.Cut(p, "/")
	std := p != "C" && !strings.Contains(first, ".")
	return std || p == "golang.org/x/sys" || strings.HasPrefix(p, "golang.org/x/sys/")
}
