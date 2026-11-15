package rules

import (
	"encoding/json"
	"fmt"
	"go/ast"
	"os"
	"slices"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/jatassi/slipstream/cmd/release/internal/lint"
)

func init() {
	lint.Register(lint.Rule{
		Name:  "house-words",
		Doc:   "string literals and doc tags in product code use none of the glossary words WRITING.md rule 21 keeps out of copy, beyond the counts web/lint/house-words-baseline.json still allows",
		Check: checkHouseWords,
	})
}

const (
	houseWordsList     = "web/lint/house-words.json"
	houseWordsBaseline = "web/lint/house-words-baseline.json"
)

// houseWordsCounting set to "off" lints against an empty baseline, so every
// match is reported and web/lint/house-words-baseline.js can count them.
const houseWordsCounting = "HOUSE_WORDS_BASELINE"

type houseBaseline map[string]map[string]int

func loadHouseBaseline(repo *lint.Repo) (houseBaseline, error) {
	if os.Getenv(houseWordsCounting) == "off" {
		return houseBaseline{}, nil
	}
	src, err := repo.Read(houseWordsBaseline)
	if err != nil {
		return nil, err
	}
	var b houseBaseline
	if err := json.Unmarshal(src, &b); err != nil {
		return nil, fmt.Errorf("%s: %w", houseWordsBaseline, err)
	}
	return b, nil
}

func checkHouseWords(repo *lint.Repo) ([]lint.Finding, error) {
	words, err := loadBannedCopy(repo, houseWordsList)
	if err != nil {
		return nil, err
	}
	baseline, err := loadHouseBaseline(repo)
	if err != nil {
		return nil, err
	}
	files, err := productFiles(repo)
	if err != nil {
		return nil, err
	}
	tracked, err := repo.Files()
	if err != nil {
		return nil, err
	}
	var findings []lint.Finding
	counted := map[string]bool{}
	for _, f := range files {
		if ast.IsGenerated(f.Syntax) {
			continue
		}
		counted[f.Path] = true
		lines := houseWordLines(repo, f, words)
		for _, w := range words {
			findings = append(findings, againstBaseline(f.Path, lines[w.Name], baseline[f.Path][w.Name], houseWordsBaseline, strconv.Quote(w.Name), houseWordMessage(w.Name))...)
		}
	}
	for file, counts := range baseline {
		if !slices.Contains(tracked, file) {
			findings = append(findings, goneEntry(houseWordsBaseline, file))
			continue
		}
		findings = append(findings, uncountedHouseWords(file, counts, words, counted[file])...)
	}
	return findings, nil
}

func houseWordLines(repo *lint.Repo, f lint.GoFile, words []bannedCopy) map[string][]int {
	lines := map[string][]int{}
	for _, lit := range copyLiterals(f.Syntax, quotesThirdParty[f.Path]) {
		if isKey(lit.text) {
			continue
		}
		for _, w := range words {
			if w.in(lit.text) {
				lines[w.Name] = append(lines[w.Name], repo.Fset.Position(lit.pos).Line)
			}
		}
	}
	return lines
}

// uncountedHouseWords checks the baseline entries of file that the product
// files left unchecked: names the list dropped, and Go files no longer
// linted. The web side counts the rest.
func uncountedHouseWords(file string, counts map[string]int, words []bannedCopy, counted bool) []lint.Finding {
	var findings []lint.Finding
	for name, allowed := range counts {
		switch {
		case !slices.ContainsFunc(words, func(w bannedCopy) bool { return w.Name == name }):
			findings = append(findings, lint.Finding{Path: houseWordsBaseline, Line: 1, Message: fmt.Sprintf("%s counts %q, which %s doesn't list. Remove it", file, name, houseWordsList)})
		case strings.HasSuffix(file, ".go") && !counted:
			findings = append(findings, againstBaseline(file, nil, allowed, houseWordsBaseline, strconv.Quote(name), houseWordMessage(name))...)
		}
	}
	return findings
}

func goneEntry(ledger, file string) lint.Finding {
	return lint.Finding{Path: ledger, Line: 1, Message: file + " no longer exists. Remove its entry"}
}

// isKey holds for a string with no space that starts lowercase, such as a
// route, a JSON member or a query key, which no person reads. A one-word
// label such as "Cells" starts with a capital.
func isKey(s string) bool {
	first, _ := utf8.DecodeRuneInString(s)
	return !unicode.IsUpper(first) && !strings.ContainsFunc(s, unicode.IsSpace)
}

func houseWordMessage(name string) string {
	return fmt.Sprintf("%q is a glossary word. Say what it means for the reader, per WRITING.md rule 21", name)
}

// againstBaseline reports the matches at lines once they outnumber what the
// ledger allows, and a ledger that allows more than file has, so the
// ledger only shrinks.
func againstBaseline(file string, lines []int, allowed int, ledger, what, message string) []lint.Finding {
	has := len(lines)
	if has < allowed {
		fix := fmt.Sprintf("Lower it to %d", has)
		if has == 0 {
			fix = "Remove it"
		}
		return []lint.Finding{{Path: file, Line: 1, Message: fmt.Sprintf("%s allows %d %s here and the file has %d. %s", ledger, allowed, what, has, fix)}}
	}
	if has == allowed {
		return nil
	}
	if allowed > 0 {
		message += fmt.Sprintf(". %s allows %d here and the file has %d", ledger, allowed, has)
	}
	findings := make([]lint.Finding, 0, has)
	for _, line := range lines {
		findings = append(findings, lint.Finding{Path: file, Line: line, Message: message})
	}
	return findings
}
