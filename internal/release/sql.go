package release

import (
	"fmt"
	"strings"
)

// QualitySQL is an SQLite expression whose value is the key of
// File(title, nil, ResolutionOf(width, height)).Quality, where title,
// width and height are SQL expressions. It is for a migration, which
// can't run Go, to guess the quality of files it already holds.
func QualitySQL(title, width, height string) string {
	t := "lower(" + title + ")"
	for _, r := range []rune{apostrophe, rightQuote} {
		t = fmt.Sprintf("replace(%s, %s, '')", t, quote(string(r)))
	}
	for _, r := range separators {
		if r != ' ' {
			t = fmt.Sprintf("replace(%s, %s, ' ')", t, quote(string(r)))
		}
	}
	t = fmt.Sprintf("replace(replace(replace(%s, ' ', ' |'), '| ', ''), '|', '')", t)

	var b strings.Builder
	fmt.Fprintf(&b, "(WITH n (t, w, h) AS (SELECT ' ' || %s || ' ', %s, %s)\n SELECT CASE\n", t, width, height)
	for _, c := range classes {
		fmt.Fprintf(&b, "  WHEN w >= %d OR h >= %d THEN %s\n", c.width, c.height, qualityAtSQL(c.res))
	}
	fmt.Fprintf(&b, "  ELSE %s\n END FROM n)", qualityAtSQL(R480p))
	return b.String()
}

func qualityAtSQL(r Resolution) string {
	var b strings.Builder
	b.WriteString("CASE")
	for _, s := range sourceWords {
		q, _ := QualityOf(Known(s.source), Known(r))
		var named []string
		for _, w := range s.words {
			named = append(named, fmt.Sprintf("instr(t, %s) > 0", quote(phrase(w))))
		}
		fmt.Fprintf(&b, "\n    WHEN %s THEN %s", strings.Join(named, " OR "), quote(q.String()))
	}
	q, _ := QualityOf(Fact[Source]{}, Known(r))
	fmt.Fprintf(&b, "\n    ELSE %s END", quote(q.String()))
	return b.String()
}

func quote(s string) string { return "'" + strings.ReplaceAll(s, "'", "''") + "'" }
