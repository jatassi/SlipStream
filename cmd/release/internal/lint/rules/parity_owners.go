package rules

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/jatassi/slipstream/cmd/release/internal/lint"
)

const parityMatrix = "docs/parity-matrix.md"

var (
	parityRow = regexp.MustCompile(`^[|] P-[0-9]+ `)
	specIssue = regexp.MustCompile(`github[.]com/jatassi/SlipStream/issues/[0-9]+\)`)
)

func init() {
	lint.Register(lint.Rule{
		Name:  "parity-owners",
		Doc:   "every kept or superseded parity row names one spec and is planned or shipped, every excepted or out-of-scope row has Status - (docs/parity-matrix.md legend, spec #126)",
		Check: checkParityOwners,
	})
}

func checkParityOwners(repo *lint.Repo) ([]lint.Finding, error) {
	src, err := repo.Read(parityMatrix)
	if err != nil {
		return nil, err
	}
	var findings []lint.Finding
	for i, row := range strings.Split(string(src), "\n") {
		line := i + 1
		if !parityRow.MatchString(row) {
			continue
		}
		report := func(format string, args ...any) {
			findings = append(findings, lint.Finding{Path: parityMatrix, Line: line, Message: fmt.Sprintf(format, args...)})
		}
		cells := strings.Split(row, " | ")
		cell := func(i int) string {
			if i < len(cells) {
				return cells[i]
			}
			return ""
		}
		id, disposition, coveredBy := cell(0)[2:], cell(4), cell(5)
		status := strings.TrimSuffix(cell(6), " |")
		switch {
		case strings.HasPrefix(disposition, "kept"), strings.HasPrefix(disposition, "superseded"):
			if links := len(specIssue.FindAllString(coveredBy, -1)); links != 1 {
				report("%s: expected one spec in Covered by, found %d", id, links)
			}
			if status != "planned" && status != "shipped" {
				report("%s: Status is %s, expected planned or shipped", id, status)
			}
		case strings.HasPrefix(disposition, "excepted"), strings.HasPrefix(disposition, "out of scope"):
			if status != "-" {
				report("%s: Status is %s, expected -", id, status)
			}
		default:
			report("%s: unknown disposition %s", id, disposition)
		}
	}
	return findings, nil
}
