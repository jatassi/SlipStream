package lint_test

import (
	"strings"
	"testing"

	"github.com/jatassi/slipstream/cmd/release/internal/lint"
)

func rule(name string, findings ...lint.Finding) lint.Rule {
	return lint.Rule{Name: name, Check: func(*lint.Repo) ([]lint.Finding, error) { return findings, nil }}
}

func TestRunPrintsSortedFindingsAndFailsOnlyWithFindings(t *testing.T) {
	var out strings.Builder
	err := lint.Run(&out, lint.Open(t.Context(), t.TempDir()), []lint.Rule{
		rule("b-rule",
			lint.Finding{Path: "z.go", Line: 2, Message: "late"},
			lint.Finding{Path: "a.go", Line: 10, Message: "ten"},
		),
		rule("a-rule", lint.Finding{Path: "a.go", Line: 9, Message: "nine"}),
		rule("clean"),
	})
	want := "a.go:9: [a-rule] nine\na.go:10: [b-rule] ten\nz.go:2: [b-rule] late\n"
	if out.String() != want {
		t.Errorf("output:\n%s\nwant:\n%s", out.String(), want)
	}
	if err == nil || err.Error() != "3 findings" {
		t.Errorf("err = %v, want 3 findings", err)
	}

	var clean strings.Builder
	if err := lint.Run(&clean, lint.Open(t.Context(), t.TempDir()), []lint.Rule{rule("clean")}); err != nil || clean.String() != "" {
		t.Errorf("a clean run = %q, %v; want no output and no error", clean.String(), err)
	}
}
