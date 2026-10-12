package rules

import (
	"slices"
	"strings"

	"github.com/jatassi/slipstream/cmd/release/internal/lint"
)

func init() {
	lint.Register(lint.Rule{
		Name:  "query-order",
		Doc:   "every :many query and every LIMIT in queries/*.sql has a top-level ORDER BY (spec #128, Typed ids, UTC times and row order)",
		Check: checkQueryOrder,
	})
}

type query struct {
	name string
	line int
	many bool
	body strings.Builder
}

func checkQueryOrder(repo *lint.Repo) ([]lint.Finding, error) {
	files, err := repo.Files()
	if err != nil {
		return nil, err
	}
	var findings []lint.Finding
	for _, f := range files {
		if !lint.OwnerSQL(f, "queries") {
			continue
		}
		src, err := repo.Read(f)
		if err != nil {
			return nil, err
		}
		for _, q := range splitQueries(string(src)) {
			words := topLevelWords(q.body.String())
			if containsPhrase(words, "ORDER", "BY") {
				continue
			}
			if q.many {
				findings = append(findings, lint.Finding{Path: f, Line: q.line, Message: q.name + " returns many rows without a top-level ORDER BY"})
			}
			if containsPhrase(words, "LIMIT") {
				findings = append(findings, lint.Finding{Path: f, Line: q.line, Message: q.name + " uses LIMIT without a top-level ORDER BY"})
			}
		}
	}
	return findings, nil
}

func splitQueries(src string) []*query {
	var queries []*query
	for i, line := range strings.Split(src, "\n") {
		if rest, ok := strings.CutPrefix(strings.TrimSpace(line), "-- name:"); ok {
			fields := strings.Fields(rest)
			q := &query{line: i + 1}
			if len(fields) > 0 {
				q.name = fields[0]
			}
			q.many = len(fields) > 1 && fields[1] == ":many"
			queries = append(queries, q)
			continue
		}
		if len(queries) > 0 {
			q := queries[len(queries)-1]
			q.body.WriteString(line)
			q.body.WriteByte('\n')
		}
	}
	return queries
}

func topLevelWords(sql string) []string {
	var words []string
	depth := 0
	for i := 0; i < len(sql); {
		c := sql[i]
		switch {
		case strings.HasPrefix(sql[i:], "--"):
			i += skipPast(sql[i:], "\n")
		case strings.HasPrefix(sql[i:], "/*"):
			i += 2 + skipPast(sql[i+2:], "*/")
		case c == '\'' || c == '"' || c == '`':
			i += 1 + skipPast(sql[i+1:], string(c))
		case c == '[':
			i += 1 + skipPast(sql[i+1:], "]")
		case c == '(':
			depth++
			i++
		case c == ')':
			depth--
			i++
		case isWordByte(c):
			start := i
			for i < len(sql) && isWordByte(sql[i]) {
				i++
			}
			if depth == 0 {
				words = append(words, strings.ToUpper(sql[start:i]))
			}
		default:
			i++
		}
	}
	return words
}

// A doubled quote inside a string is two quoted runs back to back, which
// leaves the scan in the same place.
func skipPast(s, end string) int {
	if i := strings.Index(s, end); i >= 0 {
		return i + len(end)
	}
	return len(s)
}

func isWordByte(c byte) bool {
	return c == '_' || '0' <= c && c <= '9' || 'a' <= c && c <= 'z' || 'A' <= c && c <= 'Z'
}

func containsPhrase(words []string, phrase ...string) bool {
	for i := range words {
		if i+len(phrase) <= len(words) && slices.Equal(words[i:i+len(phrase)], phrase) {
			return true
		}
	}
	return false
}
