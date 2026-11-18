package cues

import (
	"slices"
	"strings"
	"unicode"

	"github.com/saintfish/chardet"
)

// guess decodes b with the language's page chardet is most confident of,
// among the pages that decode b cleanly with the fewest odd symbols.
// chardet alone reads every language's charsets, so the table is the
// hint that keeps it to the wanted language's. With no chardet evidence for any clean page, the
// first clean page in table order wins.
func guess(b []byte, pages []codePage) (text []byte, page string, err error) {
	type decoded struct {
		page codePage
		text []byte
		odd  int
	}
	var clean []decoded
	for _, p := range pages {
		if text, _, err := decodeAs(b, p.name, p.enc); err == nil {
			clean = append(clean, decoded{p, text, oddRunes(text)})
		}
	}
	if len(clean) == 0 {
		return nil, "", errUndecodable
	}
	fewest := slices.MinFunc(clean, func(a, b decoded) int { return a.odd - b.odd }).odd
	clean = slices.DeleteFunc(clean, func(d decoded) bool { return d.odd > fewest })
	if len(clean) == 1 {
		return clean[0].text, clean[0].page.name, nil
	}
	results, _ := chardet.NewTextDetector().DetectAll(b)
	best, score := clean[0], -1
	for _, d := range clean {
		for _, r := range results {
			if slices.Contains(d.page.chardet, r.Charset) && r.Confidence > score {
				best, score = d, r.Confidence
			}
		}
	}
	return best.text, best.page.name, nil
}

// oddRunes counts the Latin-1 and Latin Extended symbols in text that
// subtitle prose doesn't carry. One is most often a letter read in the
// wrong code page, as ISO-8859-2's ś read as windows-1250 is ¶.
func oddRunes(text []byte) int {
	n := 0
	for _, r := range string(text) {
		if r >= 0x80 && r < 0x300 && !unicode.IsLetter(r) && !unicode.IsSpace(r) && !strings.ContainsRune("«»¡¿°·", r) {
			n++
		}
	}
	return n
}
