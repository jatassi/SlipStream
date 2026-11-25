package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

func goResults(t *testing.T, recorded []byte) results {
	t.Helper()
	l := newGoTestLog(io.Discard)
	if _, err := l.Write(recorded); err != nil {
		t.Fatal(err)
	}
	if err := l.flush(); err != nil {
		t.Fatal(err)
	}
	return l.results()
}

func playwrightResults(t *testing.T, recorded []byte) results {
	t.Helper()
	var r playwrightReport
	if err := json.Unmarshal(recorded, &r); err != nil {
		t.Fatal(err)
	}
	return r.results()
}

func vitestResults(t *testing.T, recorded []byte) results {
	t.Helper()
	path := filepath.Join(t.TempDir(), "report.json")
	if err := os.WriteFile(path, recorded, 0o644); err != nil {
		t.Fatal(err)
	}
	r, err := readVitestReport(path, "/r")
	if err != nil {
		t.Fatal(err)
	}
	return r
}

func goRerunArgs(f []failure) []string {
	var args []string
	for _, r := range rerunRuns(f) {
		args = append(args, r.args()...)
	}
	return args
}

func selectors(f []failure) []string {
	var s []string
	for _, x := range f {
		s = append(s, x.selector)
	}
	return s
}

func manyGoFailures(n int) []byte {
	var b bytes.Buffer
	for i := range n {
		test := fmt.Sprintf("TestCase%02d", i)
		fmt.Fprintf(&b, `{"Action":"output","Package":"p","Test":%q,"Output":"    p_test.go:%d: wrong\n"}`+"\n", test, i+1)
		fmt.Fprintf(&b, `{"Action":"fail","Package":"p","Test":%q}`+"\n", test)
	}
	b.WriteString(`{"Action":"fail","Package":"p"}` + "\n")
	return b.Bytes()
}

// The recordings are go test -json, Playwright's and Vitest's JSON reports
// of a first run and of the rerun of its failed tests, in which one test
// passes.
func TestFailedTestsReachTheAnnotations(t *testing.T) {
	t.Setenv("CI", "true")
	read := func(name string) []byte {
		b, err := os.ReadFile(filepath.Join("testdata", "failures", name))
		if err != nil {
			t.Fatal(err)
		}
		return b
	}
	for _, c := range []struct {
		name         string
		parse        func(*testing.T, []byte) results
		first, again []byte
		rerunArgs    func([]failure) []string
		// wantRerun is what the rerun gets; nil when there is none.
		wantRerun []string
		want      []string
	}{
		{
			name: "go", parse: goResults, first: read("go-first.jsonl"), again: read("go-again.jsonl"), rerunArgs: goRerunArgs,
			wantRerun: []string{"-run", "^(TestCounts|TestPanics|TestShapes)$", "github.com/jatassi/slipstream/widget"},
			want: []string{
				"::error file=widget/widget_test.go,line=13,title=github.com/jatassi/slipstream/widget%3A TestCounts (flaky)::count = 2, want 3",
				"::error file=widget/widget_test.go,line=19,title=github.com/jatassi/slipstream/widget%3A TestShapes/circle (failed again)::radius = 0, want 1",
				"::error title=github.com/jatassi/slipstream/widget%3A TestPanics (failed again)::panic: assignment to entry in nil map [recovered, repanicked]",
				"::error title=github.com/jatassi/slipstream/stall%3A TestHangs::panic: test timed out after 5s",
			},
		},
		{
			name: "playwright", parse: playwrightResults, first: read("playwright-first.json"), again: read("playwright-again.json"), rerunArgs: selectors,
			wantRerun: []string{
				"[chromium-desktop] › work.spec.ts › Search finds the release",
				"[chromium-desktop] › work.spec.ts › between Nodes › a Move keeps the files",
			},
			want: []string{
				"::error file=web/e2e/work.spec.ts,line=7,title=web/e2e/work.spec.ts%3A [chromium-desktop] › Search finds the release (flaky)::Error: expect(received).toBe(expected) // Object.is equality",
				"::error file=web/e2e/work.spec.ts,line=12,title=web/e2e/work.spec.ts%3A [chromium-desktop] › between Nodes › a Move keeps the files (failed again)::Error: expect(received).toEqual(expected) // deep equality",
			},
		},
		{
			name: "vitest", parse: vitestResults, first: read("vitest-first.json"), again: read("vitest-again.json"), rerunArgs: vitestRerunArgs,
			wantRerun: []string{"src/queue.test.ts", "-t", "^(Queue > orders by age|drops a finished grab)$"},
			want: []string{
				"::error file=web/src/broken.test.ts,title=web/src/broken.test.ts::Cannot find module './missing-module' imported from /r/web/src/broken.test.ts",
				"::error file=web/src/queue.test.ts,line=8,title=web/src/queue.test.ts%3A Queue > orders by age (flaky)::AssertionError: expected [ 2, 1 ] to deeply equal [ 1, 2 ]",
				"::error file=web/src/queue.test.ts,line=13,title=web/src/queue.test.ts%3A drops a finished grab (failed again)::AssertionError: expected 'done' to be 'gone' // Object.is equality",
			},
		},
		{
			name: "more than the reruns take", parse: goResults, first: manyGoFailures(maxReruns + 1),
			want: func() []string {
				var w []string
				for i := range maxAnnotations {
					w = append(w, fmt.Sprintf("::error file=p/p_test.go,line=%d,title=p%%3A TestCase%02d::wrong", i+1, i))
				}
				return w
			}(),
		},
	} {
		t.Run(c.name, func(t *testing.T) {
			var gotRerun []string
			failed := rerunFailed(c.parse(t, c.first), func(f []failure) results {
				gotRerun = c.rerunArgs(f)
				return c.parse(t, c.again)
			})
			if !slices.Equal(gotRerun, c.wantRerun) {
				t.Errorf("the rerun gets %q, want %q", gotRerun, c.wantRerun)
			}
			got := strings.Split(strings.TrimSuffix(annotations(failed), "\n"), "\n")
			slices.Sort(got)
			want := slices.Sorted(slices.Values(c.want))
			if !slices.Equal(got, want) {
				t.Errorf("annotations:\n%s\nwant:\n%s", strings.Join(got, "\n"), strings.Join(want, "\n"))
			}
			if rows := strings.Count(summary(failed), "\n| ") - 1; rows != len(failed) {
				t.Errorf("the summary has %d rows, want one for each of the %d failed tests:\n%s", rows, len(failed), summary(failed))
			}
		})
	}
}
