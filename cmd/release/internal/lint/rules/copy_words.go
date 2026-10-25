package rules

import (
	"fmt"
	"go/ast"
	"go/token"
	"regexp"
	"strconv"

	"github.com/jatassi/slipstream/cmd/release/internal/lint"
)

func init() {
	lint.Register(lint.Rule{
		Name:  "copy-words",
		Doc:   "string literals in product code use none of the words and dashes WRITING.md bans",
		Check: checkCopyWords,
	})
}

type bannedCopy struct {
	name    string
	pattern string
}

var bannedCopyWords = []bannedCopy{
	{"em dash", `\S\s*—\s*\S`},
	{"en dash", `\S\s*–\s*\S`},
	{"please", `\bplease\b`},
	{"sorry", `\bsorry\b`},
	{"oops", `\boops\b`},
	{"whoops", `\bwhoops\b`},
	{"something went wrong", `\bsomething\s+went\s+wrong\b`},
	{"an error occurred", `\ban\s+error\s+occurred\b`},
	{"all set", `\ball\s+set\b`},
	{"all good", `\ball\s+good\b`},
	{"additionally", `\badditionally\b`},
	{"delve", `\bdelve\b`},
	{"leverage", `\bleverage\b`},
	{"utilize", `\butilize\b`},
	{"seamless", `\bseamless(?:ly)?\b`},
	{"robust", `\brobust\b`},
	{"crucial", `\bcrucial\b`},
	{"pivotal", `\bpivotal\b`},
	{"showcase", `\bshowcase\b`},
	{"testament", `\btestament\b`},
	{"vibrant", `\bvibrant\b`},
	{"meticulous", `\bmeticulous(?:ly)?\b`},
	{"multifaceted", `\bmultifaceted\b`},
}

// quotesThirdParty holds files whose strings are another program's own text,
// which SlipStream matches or replays word for word.
var quotesThirdParty = map[string]bool{
	"internal/torrent/unit3d/unit3d.go": true,
}

var bannedCopyPatterns = compileBannedCopy(bannedCopyWords)

func compileBannedCopy(words []bannedCopy) []*regexp.Regexp {
	res := make([]*regexp.Regexp, len(words))
	for i, w := range words {
		res[i] = regexp.MustCompile("(?i)" + w.pattern)
	}
	return res
}

func checkCopyWords(repo *lint.Repo) ([]lint.Finding, error) {
	files, err := productFiles(repo)
	if err != nil {
		return nil, err
	}
	var findings []lint.Finding
	for _, f := range files {
		if quotesThirdParty[f.Path] {
			continue
		}
		for _, lit := range copyLiterals(f.Syntax) {
			s, err := strconv.Unquote(lit.Value)
			if err != nil {
				return nil, fmt.Errorf("%s: unquote %s: %w", f.Path, lit.Value, err)
			}
			for i, re := range bannedCopyPatterns {
				if re.MatchString(s) {
					findings = append(findings, lint.Finding{
						Path:    f.Path,
						Line:    repo.Fset.Position(lit.Pos()).Line,
						Message: fmt.Sprintf("%q is banned in user-facing strings; rewrite it per WRITING.md", bannedCopyWords[i].name),
					})
				}
			}
		}
	}
	return findings, nil
}

func copyLiterals(file *ast.File) []*ast.BasicLit {
	tags := map[*ast.BasicLit]bool{}
	var lits []*ast.BasicLit
	ast.Inspect(file, func(n ast.Node) bool {
		switch n := n.(type) {
		case *ast.ImportSpec:
			return false
		case *ast.Field:
			if n.Tag != nil {
				tags[n.Tag] = true
			}
		case *ast.BasicLit:
			if n.Kind == token.STRING && !tags[n] {
				lits = append(lits, n)
			}
		}
		return true
	})
	return lits
}
