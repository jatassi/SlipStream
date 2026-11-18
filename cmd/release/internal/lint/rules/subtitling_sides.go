package rules

import (
	"path"
	"strconv"
	"strings"

	"github.com/jatassi/slipstream/cmd/release/internal/lint"
)

func init() {
	lint.Register(lint.Rule{
		Name:  "subtitling-sides",
		Doc:   "the Subtitles Role's holder and the subtitle providers reach no Controller store and no part of internal/subtitles but Timing sync and the formats, so a holder keeps nothing between calls (S46 (Subtitles), The Subtitles Role)",
		Check: checkSubtitlingSides,
	})
}

var (
	subtitlingSide = []string{"internal/subtitling/...", "internal/providers/..."}
	// subtitlingForbids are the Controller's stores and its subtitles
	// owner; subtitlingAllows are the pure leaves under it both sides run.
	subtitlingForbids = []string{"internal/storage/...", "internal/subtitles/..."}
	subtitlingAllows  = []string{"internal/subtitles/timing/...", "internal/subtitles/cues/..."}
)

// checkSubtitlingSides reads non-test files only: a test may build a
// store to stand up its Nodes.
func checkSubtitlingSides(repo *lint.Repo) ([]lint.Finding, error) {
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
		if strings.HasSuffix(f.Path, "_test.go") || !inDirs(path.Dir(f.Path), subtitlingSide) {
			continue
		}
		for _, imp := range f.Syntax.Imports {
			p, _ := strconv.Unquote(imp.Path.Value)
			pkg, ok := strings.CutPrefix(p, module+"/")
			if !ok || inDirs(pkg, subtitlingAllows) || !inDirs(pkg, subtitlingForbids) {
				continue
			}
			findings = append(findings, lint.Finding{Path: f.Path, Line: repo.Fset.Position(imp.Pos()).Line, Message: pkg + " imported by the Subtitles Role's holder side; each call carries what the holder needs"})
		}
	}
	return findings, nil
}
