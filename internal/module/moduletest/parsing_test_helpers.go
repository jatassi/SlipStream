package moduletest

import (
	"testing"

	"github.com/slipstream/slipstream/internal/module"
)

// ParsingFixture defines a test case for file parsing.
type ParsingFixture struct {
	Filename      string
	ExpectedTitle string
	ExpectedYear  int
	ExpectMatch   bool
	MinConfidence float64
}

// RunParsingTest validates that a module's FileParser handles the given fixtures correctly.
// For each fixture, it calls TryMatch and checks match/no-match expectations, title,
// year, and minimum confidence.
func RunParsingTest(t *testing.T, parser module.FileParser, fixtures []ParsingFixture) {
	t.Helper()
	for _, fx := range fixtures {
		t.Run(fx.Filename, func(t *testing.T) {
			confidence, result := parser.TryMatch(fx.Filename)

			if fx.ExpectMatch {
				assertMatchFound(t, fx, confidence, result)
			} else if result != nil && confidence > 0 {
				t.Errorf("expected no match for %q but got confidence=%.2f title=%q", fx.Filename, confidence, result.Title)
			}
		})
	}
}

func assertMatchFound(t *testing.T, fx ParsingFixture, confidence float64, result *module.ParseResult) {
	t.Helper()
	if result == nil {
		t.Fatalf("expected match for %q but got nil", fx.Filename)
	}
	if confidence < fx.MinConfidence {
		t.Errorf("confidence %.2f < min %.2f for %q", confidence, fx.MinConfidence, fx.Filename)
	}
	if fx.ExpectedTitle != "" && result.Title != fx.ExpectedTitle {
		t.Errorf("title: got %q, want %q", result.Title, fx.ExpectedTitle)
	}
	if fx.ExpectedYear != 0 && result.Year != fx.ExpectedYear {
		t.Errorf("year: got %d, want %d", result.Year, fx.ExpectedYear)
	}
}
