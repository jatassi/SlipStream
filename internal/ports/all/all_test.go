package all

import (
	"slices"
	"strings"
	"testing"
	"unicode"

	"github.com/jatassi/slipstream/internal/ports/port"
)

// miscased are the GLOSSARY.md terms a description may use, in the case
// their _In UI_ lines rule out: Tracker keeps its capital, and release
// is lowercase in sentences.
var miscased = []string{"tracker", "trackers", "Release", "Releases"}

func TestEveryPortDescribesItsSiteInOneSentenceOfGlossaryTerms(t *testing.T) {
	for _, e := range Registry().Entries() {
		words := strings.FieldsFunc(e.Description, func(r rune) bool { return !unicode.IsLetter(r) && r != '-' })
		if !strings.HasSuffix(e.Description, ".") {
			t.Errorf("%s: %q doesn't end with a period", e.ID, e.Description)
		}
		for _, w := range words {
			if slices.Contains(miscased, w) {
				t.Errorf("%s: %q writes the glossary term %q in the wrong case", e.ID, e.Description, w)
			}
		}
		for _, p := range []port.Privacy{port.Private, port.SemiPrivate, port.Public} {
			if slices.Contains(words, string(p)) != (p == e.Privacy) {
				t.Errorf("%s: %q doesn't name its privacy, %s, alone", e.ID, e.Description, e.Privacy)
			}
		}
	}
}
