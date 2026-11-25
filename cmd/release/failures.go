package main

import (
	"cmp"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
)

// failure is a test that failed, as CI's annotations, the step summary and
// failures.json name it.
type failure struct {
	Tier   string `json:"tier"`
	Target string `json:"target"`
	// Package is the Go package, or the spec or test file from the repo root.
	Package string `json:"package"`
	// File and Line are where the failure points, from the repo root; Line
	// is 0 when the tier doesn't say.
	File string `json:"file"`
	Line int    `json:"line"`
	// Test is empty for a package or file that failed outside its tests,
	// as in its build.
	Test    string      `json:"test"`
	Message string      `json:"message"`
	Rerun   rerunResult `json:"rerun"`
	// selector picks the test for its tier's rerun, empty for a failure
	// that can't run again alone.
	selector string
}

func (f failure) key() string { return f.Package + "\x00" + f.Test }

func (f failure) title() string {
	title := f.Package
	if f.Test != "" {
		title += ": " + f.Test
	}
	if f.Rerun != notRerun {
		title += " (" + string(f.Rerun) + ")"
	}
	return title
}

type rerunResult string

const (
	notRerun    rerunResult = ""
	flaky       rerunResult = "flaky"
	failedAgain rerunResult = "failed again"
)

// results is what one run of a tier's tests reports.
type results struct {
	failed []failure
	// passed holds the key of each test that passed.
	passed map[string]bool
}

func (r *results) pass(key string) {
	if r.passed == nil {
		r.passed = map[string]bool{}
	}
	r.passed[key] = true
}

func (r *results) add(o results) {
	r.failed = append(r.failed, o.failed...)
	for k := range o.passed {
		r.pass(k)
	}
}

const (
	// maxReruns is the most failed tests CI runs again. More failures than
	// that come from one cause, which a rerun would only repeat.
	maxReruns = 20
	// maxAnnotations is how many error annotations GitHub shows for a step.
	maxAnnotations = 10
)

func inCI() bool { return os.Getenv("CI") == "true" }

// ciTarget names the target the failures came from, as GOOS and GOARCH
// set it for the go command.
func ciTarget() string {
	return cmp.Or(os.Getenv("GOOS"), runtime.GOOS) + "-" + cmp.Or(os.Getenv("GOARCH"), runtime.GOARCH)
}

// rerunFailed runs the failed tests once more in CI when 1 to maxReruns
// failed, and labels each that ran again flaky when it passed then. A flaky
// test still fails its tier.
func rerunFailed(first results, rerun func([]failure) results) []failure {
	failed := first.failed
	var picked []failure
	for _, f := range failed {
		if f.selector != "" {
			picked = append(picked, f)
		}
	}
	if !inCI() || len(picked) == 0 || len(failed) > maxReruns {
		return failed
	}
	fmt.Printf("\nRunning the %d failed tests once more.\n", len(picked))
	again := rerun(picked)
	for i := range failed {
		switch {
		case failed[i].selector == "":
		case again.passed[failed[i].key()]:
			failed[i].Rerun = flaky
		default:
			failed[i].Rerun = failedAgain
		}
	}
	return failed
}

// reportFailures shows the failures as annotations and a step summary in
// CI, and writes them to path as JSON when path is set.
func reportFailures(failures []failure, path string) error {
	for i := range failures {
		failures[i].Message = cmp.Or(failures[i].Message, "The test failed with no message.")
	}
	if inCI() && len(failures) > 0 {
		if _, err := os.Stdout.WriteString(annotations(failures)); err != nil {
			return fmt.Errorf("print the annotations: %w", err)
		}
		if path := os.Getenv("GITHUB_STEP_SUMMARY"); path != "" {
			if err := appendSummary(path, summary(failures)); err != nil {
				return err
			}
		}
	}
	if path == "" {
		return nil
	}
	if failures == nil {
		failures = []failure{}
	}
	b, err := json.MarshalIndent(failures, "", "\t")
	if err != nil {
		return fmt.Errorf("encode the failed tests: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("write the failed tests: %w", err)
	}
	if err := os.WriteFile(path, append(b, '\n'), 0o644); err != nil {
		return fmt.Errorf("write the failed tests: %w", err)
	}
	return nil
}

func appendSummary(path, text string) error {
	f, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0)
	if err != nil {
		return fmt.Errorf("open the step summary: %w", err)
	}
	_, err = f.WriteString(text)
	if closeErr := f.Close(); err == nil {
		err = closeErr
	}
	if err != nil {
		return fmt.Errorf("write the step summary: %w", err)
	}
	return nil
}

// annotations is a GitHub error annotation for each of the first
// maxAnnotations failures.
func annotations(failures []failure) string {
	var w strings.Builder
	for _, f := range failures[:min(len(failures), maxAnnotations)] {
		props := []string{}
		if f.File != "" {
			props = append(props, "file="+escapeProperty(f.File))
		}
		if f.Line > 0 {
			props = append(props, "line="+strconv.Itoa(f.Line))
		}
		props = append(props, "title="+escapeProperty(f.title()))
		fmt.Fprintf(&w, "::error %s::%s\n", strings.Join(props, ","), escapeData(f.Message))
	}
	return w.String()
}

func summary(failures []failure) string {
	var w strings.Builder
	fmt.Fprintf(&w, "### Failed tests on %s\n\n| Test | Where | Result | Failure |\n|---|---|---|---|\n", failures[0].Target)
	hasFlaky := false
	for _, f := range failures {
		where := f.File
		if f.Line > 0 {
			where += ":" + strconv.Itoa(f.Line)
		}
		result := cmp.Or(string(f.Rerun), "failed")
		hasFlaky = hasFlaky || f.Rerun == flaky
		fmt.Fprintf(&w, "| %s | %s | %s | %s |\n", markdownCell(f.Package+" "+f.Test), markdownCell(where), result, markdownCell(f.Message))
	}
	w.WriteString("\n")
	if hasFlaky {
		w.WriteString("A flaky test failed and then passed when it ran again. It still fails the job.\n")
	}
	if len(failures) > maxReruns {
		fmt.Fprintf(&w, "%d tests failed. CI runs failed tests again only when %d or fewer fail.\n", len(failures), maxReruns)
	}
	if more := len(failures) - maxAnnotations; more > 0 {
		fmt.Fprintf(&w, "GitHub shows %d annotations for a step. The other %d failed tests appear only in this table.\n", maxAnnotations, more)
	}
	w.WriteString("\n")
	return w.String()
}

func markdownCell(s string) string {
	return strings.ReplaceAll(strings.TrimSpace(s), "|", `\|`)
}

var dataEscapes = strings.NewReplacer("%", "%25", "\r", "%0D", "\n", "%0A")

var propertyEscapes = strings.NewReplacer("%", "%25", "\r", "%0D", "\n", "%0A", ":", "%3A", ",", "%2C")

func escapeData(s string) string { return dataEscapes.Replace(s) }

func escapeProperty(s string) string { return propertyEscapes.Replace(s) }

var ansiEscape = regexp.MustCompile(`\x1b\[[0-9;]*[A-Za-z]`)

// firstLine is a failure message's first line with text, without the
// terminal colors Playwright and Vitest add.
func firstLine(s string) string {
	for line := range strings.Lines(ansiEscape.ReplaceAllString(s, "")) {
		if line = strings.TrimSpace(line); line != "" {
			return line
		}
	}
	return ""
}
