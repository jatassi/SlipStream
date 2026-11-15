package rules

import (
	"encoding/json"
	"fmt"
	"regexp"
	"slices"
	"strings"

	"github.com/jatassi/slipstream/cmd/release/internal/lint"
)

func init() {
	lint.Register(lint.Rule{
		Name:  "glossary-format",
		Doc:   "each GLOSSARY.md entry has one non-empty _In UI_ line before its _Avoid_ line and a definition of at most two sentences, unless web/lint/glossary-long.json still lists it",
		Check: checkGlossaryFormat,
	})
}

const (
	glossaryFile = "GLOSSARY.md"
	glossaryLong = "web/lint/glossary-long.json"
)

var (
	glossaryTerm = regexp.MustCompile(`^\*\*(.+)\*\*:$`)
	codeSpan     = regexp.MustCompile("`[^`]*`")
	// sentenceEnd is a stop that a space and a lowercase letter don't
	// follow, so "e.g. a" and "4.6.0" end nothing.
	sentenceEnd = regexp.MustCompile(`[.!?]["')]*(?:$|\s+[^\sa-z])`)
)

type glossaryEntry struct {
	term       string
	line       int
	definition []string
	inUI       []int
	inUIEmpty  bool
	avoid      int
}

func checkGlossaryFormat(repo *lint.Repo) ([]lint.Finding, error) {
	src, err := repo.Read(glossaryFile)
	if err != nil {
		return nil, err
	}
	listed, err := repo.Read(glossaryLong)
	if err != nil {
		return nil, err
	}
	var long []string
	if err := json.Unmarshal(listed, &long); err != nil {
		return nil, fmt.Errorf("%s: %w", glossaryLong, err)
	}
	entries := glossaryEntries(string(src))
	var findings []lint.Finding
	add := func(e glossaryEntry, line int, format string, args ...any) {
		findings = append(findings, lint.Finding{Path: glossaryFile, Line: line, Message: fmt.Sprintf("**%s** "+format, append([]any{e.term}, args...)...)})
	}
	for _, e := range entries {
		switch {
		case len(e.inUI) == 0:
			add(e, e.line, `has no _In UI_ line. Add one before _Avoid_, or "_In UI_: none" for a term no screen shows`)
		case len(e.inUI) > 1:
			add(e, e.inUI[1], "has %d _In UI_ lines. Keep one", len(e.inUI))
		case e.inUIEmpty:
			add(e, e.inUI[0], `has an empty _In UI_ line. Write the words a screen uses, or "none"`)
		case e.avoid != 0 && e.avoid < e.inUI[0]:
			add(e, e.inUI[0], "has its _In UI_ line after _Avoid_. Move it above")
		}
		n := sentences(strings.Join(e.definition, " "))
		allowed := slices.Contains(long, e.term)
		if n > 2 && !allowed {
			add(e, e.line, "has %d sentences. Define it in two and move the rules to its spec or ADR", n)
		}
		if n <= 2 && allowed {
			add(e, e.line, "has %d sentences now. Remove it from %s", n, glossaryLong)
		}
	}
	for _, term := range long {
		if !slices.ContainsFunc(entries, func(e glossaryEntry) bool { return e.term == term }) {
			findings = append(findings, lint.Finding{Path: glossaryLong, Line: 1, Message: fmt.Sprintf("%s has no entry %q. Remove it", glossaryFile, term)})
		}
	}
	return findings, nil
}

func glossaryEntries(src string) []glossaryEntry {
	var entries []glossaryEntry
	var cur *glossaryEntry
	for i, line := range strings.Split(strings.ReplaceAll(src, "\r\n", "\n"), "\n") {
		if m := glossaryTerm.FindStringSubmatch(line); m != nil {
			entries = append(entries, glossaryEntry{term: m[1], line: i + 1})
			cur = &entries[len(entries)-1]
			continue
		}
		if cur == nil {
			continue
		}
		switch {
		case strings.HasPrefix(line, "#") || strings.TrimSpace(line) == "":
			cur = nil
		case strings.HasPrefix(line, "_In UI_:"):
			cur.inUI = append(cur.inUI, i+1)
			cur.inUIEmpty = strings.TrimSpace(strings.TrimPrefix(line, "_In UI_:")) == ""
		case strings.HasPrefix(line, "_Avoid_:"):
			cur.avoid = i + 1
		case len(cur.inUI) == 0 && cur.avoid == 0:
			cur.definition = append(cur.definition, line)
		}
	}
	return entries
}

func sentences(definition string) int {
	return len(sentenceEnd.FindAllStringIndex(codeSpan.ReplaceAllString(strings.TrimSpace(definition), "x"), -1))
}
