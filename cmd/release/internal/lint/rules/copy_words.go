package rules

import (
	"encoding/json"
	"fmt"
	"go/ast"
	"go/token"
	"reflect"
	"regexp"
	"strconv"
	"strings"

	"github.com/jatassi/slipstream/cmd/release/internal/lint"
)

func init() {
	lint.Register(lint.Rule{
		Name:  "copy-words",
		Doc:   "string literals and doc tags in product code use none of the words and dashes WRITING.md bans",
		Check: checkCopyWords,
	})
}

const copyWordsList = "web/lint/copy-words.json"

// quotesThirdParty names, per file, the function whose strings are another
// program's own text, which SlipStream replays word for word.
var quotesThirdParty = map[string]string{
	"internal/torrent/unit3d/unit3d.go": "rejected",
}

type bannedCopy struct {
	Name    string `json:"name"`
	Pattern string `json:"pattern"`
	Alone   bool   `json:"alone"`
	re      *regexp.Regexp
	whole   *regexp.Regexp
}

func (b bannedCopy) in(s string) bool {
	return b.re.MatchString(s) && (!b.Alone || !b.whole.MatchString(strings.TrimSpace(s)))
}

func loadBannedCopy(repo *lint.Repo) ([]bannedCopy, error) {
	src, err := repo.Read(copyWordsList)
	if err != nil {
		return nil, err
	}
	var words []bannedCopy
	if err := json.Unmarshal(src, &words); err != nil {
		return nil, fmt.Errorf("%s: %w", copyWordsList, err)
	}
	for i := range words {
		if words[i].re, err = regexp.Compile("(?i)" + words[i].Pattern); err != nil {
			return nil, fmt.Errorf("%s: %s: %w", copyWordsList, words[i].Name, err)
		}
		words[i].whole = regexp.MustCompile("(?i)^(?:" + words[i].Pattern + ")$")
	}
	return words, nil
}

func checkCopyWords(repo *lint.Repo) ([]lint.Finding, error) {
	words, err := loadBannedCopy(repo)
	if err != nil {
		return nil, err
	}
	files, err := productFiles(repo)
	if err != nil {
		return nil, err
	}
	var findings []lint.Finding
	for _, f := range files {
		for _, lit := range copyLiterals(f.Syntax, quotesThirdParty[f.Path]) {
			for _, w := range words {
				if w.in(lit.text) {
					findings = append(findings, lint.Finding{
						Path:    f.Path,
						Line:    repo.Fset.Position(lit.pos).Line,
						Message: fmt.Sprintf("%q is banned in user-facing strings. Rewrite it per WRITING.md", w.Name),
					})
				}
			}
		}
	}
	return findings, nil
}

type copyLiteral struct {
	text string
	pos  token.Pos
}

func copyLiterals(file *ast.File, skipFunc string) []copyLiteral {
	tags := map[*ast.BasicLit]bool{}
	var lits []copyLiteral
	for _, decl := range file.Decls {
		if fn, ok := decl.(*ast.FuncDecl); ok && fn.Name.Name == skipFunc {
			continue
		}
		ast.Inspect(decl, func(n ast.Node) bool {
			lits = appendCopyLiteral(lits, tags, n)
			return true
		})
	}
	return lits
}

func appendCopyLiteral(lits []copyLiteral, tags map[*ast.BasicLit]bool, n ast.Node) []copyLiteral {
	switch n := n.(type) {
	case *ast.Field:
		if n.Tag == nil {
			return lits
		}
		tags[n.Tag] = true
		tag, _ := strconv.Unquote(n.Tag.Value)
		if doc := reflect.StructTag(tag).Get("doc"); doc != "" {
			return append(lits, copyLiteral{doc, n.Tag.Pos()})
		}
	case *ast.BasicLit:
		if n.Kind == token.STRING && !tags[n] {
			s, _ := strconv.Unquote(n.Value)
			return append(lits, copyLiteral{s, n.Pos()})
		}
	}
	return lits
}
