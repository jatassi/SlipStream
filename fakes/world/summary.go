package world

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"slices"
	"strings"

	"github.com/jatassi/slipstream/fakes/events"
	"github.com/jatassi/slipstream/internal/buildinfo"
)

// SummaryFile is the summary fakeworld summary writes into a run dir.
const SummaryFile = "summary.md"

// journaled is one step as the event log tells it.
type journaled struct {
	StepStart
	N     int
	Seq   uint64
	Exit  *int
	Files map[string]string
}

// Command is the step as a shell line: the fakeworld verb and its
// arguments after the run dir, or a Go test's note.
func (s journaled) Command() string {
	if s.Note != "" {
		return "note: " + s.Note
	}
	words := append([]string{"fakeworld", s.Verb}, s.Args...)
	for i, w := range words[2:] {
		words[i+2] = shellQuote(w)
	}
	return strings.Join(words, " ")
}

func shellQuote(s string) string {
	if s != "" && !strings.ContainsAny(s, " \t\n'\"\\$`|&;<>(){}*?[]#~") {
		return s
	}
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}

// steps reads each step of a run's journal from its records, in order.
func steps(recs []events.Record) ([]journaled, error) {
	var out []journaled
	for _, r := range recs {
		switch r.Kind {
		case events.StepStart:
			var s stepStarted
			if err := json.Unmarshal(r.Data, &s); err != nil {
				return nil, fmt.Errorf("%s %d: %w", r.Kind, r.Seq, err)
			}
			out = append(out, journaled{StepStart: s.StepStart, N: s.N, Seq: r.Seq})
		case events.StepEnd:
			var e stepEnded
			if err := json.Unmarshal(r.Data, &e); err != nil {
				return nil, fmt.Errorf("%s %d: %w", r.Kind, r.Seq, err)
			}
			if i := slices.IndexFunc(out, func(s journaled) bool { return s.N == e.N }); i >= 0 {
				out[i].Exit, out[i].Files = &e.Exit, r.Files
			}
		}
	}
	return out, nil
}

// Summary is the run in runDir as summary.md tells it, read from its
// files alone: the build stamp, the Scenario, the last doctor result, the
// steps, the cue and fault timeline, each ERROR a Node logged and the
// proof files.
func Summary(runDir string) (string, error) {
	recs, err := events.Read(runDir)
	if err != nil {
		return "", err
	}
	m, err := ReadLastManifest(runDir)
	if err != nil {
		return "", err
	}
	journal, err := steps(recs)
	if err != nil {
		return "", err
	}
	proofs, err := proofFiles(runDir)
	if err != nil {
		return "", err
	}
	var b strings.Builder
	fmt.Fprintf(&b, "# Run %s\n\n", m.RunID)
	fmt.Fprintf(&b, "- Build: %s\n", buildLine(runDir, m))
	fmt.Fprintf(&b, "- Scenario: %s (seed %s, sha256 %s)\n", m.Scenario.Name, m.Scenario.Seed, m.Scenario.SHA256)
	fmt.Fprintf(&b, "- Origin: %s\n", originLine(recs))
	b.WriteString("\n## Doctor\n\n")
	writeDoctor(&b, runDir, journal)
	b.WriteString("\n## Steps\n\n")
	writeSteps(&b, journal)
	b.WriteString("\n## Cues and faults\n\n")
	writeTimeline(&b, recs)
	b.WriteString("\n## Node errors\n\n")
	writeNodeErrors(&b, recs)
	b.WriteString("\n## Proof files\n\n")
	if len(proofs) == 0 {
		b.WriteString("None.\n")
	}
	for _, p := range proofs {
		fmt.Fprintf(&b, "- %s\n", p)
	}
	return b.String(), nil
}

func buildLine(runDir string, m Manifest) string {
	line := fmt.Sprintf("%s, sha256 %s", m.Binary.Version, m.Binary.SHA256)
	rel, err := filepath.Rel(m.RunDir, m.Binary.Path)
	if err != nil {
		return line + "; its copy is not in the run dir"
	}
	stamp, err := buildinfo.ReadFile(filepath.Join(runDir, rel))
	switch {
	case errors.Is(err, fs.ErrNotExist):
		return line + "; " + filepath.ToSlash(rel) + " is not in this bundle"
	case errors.Is(err, buildinfo.ErrUnstamped):
		return line + "; unstamped"
	case err != nil:
		return line + "; " + err.Error()
	}
	return fmt.Sprintf("%s, commit %s, fingerprint %s, target %s", line, stamp.Commit, stamp.Fingerprint, stamp.Target)
}

func originLine(recs []events.Record) string {
	var l launched
	if len(recs) == 0 || recs[0].Kind != events.RunLaunched || json.Unmarshal(recs[0].Data, &l) != nil {
		return "unknown"
	}
	if l.Origin.Kind == FromTest {
		return "test, `" + l.Origin.Command + "`"
	}
	return string(l.Origin.Kind)
}

func writeDoctor(b *strings.Builder, runDir string, journal []journaled) {
	var last *journaled
	for k := len(journal) - 1; k >= 0 && last == nil; k-- {
		if journal[k].Verb == "doctor" {
			last = &journal[k]
		}
	}
	if last == nil {
		b.WriteString("doctor never ran on this run.\n")
		return
	}
	fmt.Fprintf(b, "Step %d, exit %s:\n\n", last.N, exitText(last.Exit))
	report, err := os.ReadFile(filepath.Join(runDir, filepath.FromSlash(last.Files["stdout"])))
	if last.Files["stdout"] == "" || err != nil {
		b.WriteString("Its report was not kept.\n")
		return
	}
	fmt.Fprintf(b, "```text\n%s```\n", report)
}

func exitText(exit *int) string {
	if exit == nil {
		return "none (it never ended)"
	}
	return fmt.Sprint(*exit)
}

func writeSteps(b *strings.Builder, journal []journaled) {
	if len(journal) == 0 {
		b.WriteString("None.\n")
		return
	}
	b.WriteString("| n | command | exit | proof |\n|---|---|---|---|\n")
	for _, s := range journal {
		proof := ""
		for _, f := range s.Files {
			if strings.HasPrefix(f, "proof/") {
				proof = path.Dir(f)
			}
		}
		fmt.Fprintf(b, "| %d | %s | %s | %s |\n", s.N, cell(s.Command()), exitText(s.Exit), proof)
	}
}

// cell is s as a code span in a table cell.
func cell(s string) string {
	s = strings.ReplaceAll(s, "|", `\|`)
	if strings.Contains(s, "`") {
		return "`` " + s + " ``"
	}
	return "`" + s + "`"
}

func writeTimeline(b *strings.Builder, recs []events.Record) {
	n := 0
	for _, r := range recs {
		if r.Kind != events.Cue && !strings.HasPrefix(r.Kind, "fault.") {
			continue
		}
		if n++; n == 1 {
			b.WriteString("| seq | kind | node | data |\n|---|---|---|---|\n")
		}
		fmt.Fprintf(b, "| %d | %s | %s | %s |\n", r.Seq, r.Kind, r.Node, cell(string(r.Data)))
	}
	if n == 0 {
		b.WriteString("None.\n")
	}
}

// writeNodeErrors lists each node.error with the step that was running
// when the Node logged it, the latest started of any still open.
func writeNodeErrors(b *strings.Builder, recs []events.Record) {
	var open []int
	n := 0
	for _, r := range recs {
		switch r.Kind {
		case events.StepStart, events.StepEnd:
			var s struct {
				N int `json:"n"`
			}
			_ = json.Unmarshal(r.Data, &s)
			if r.Kind == events.StepStart {
				open = append(open, s.N)
			} else {
				open = slices.DeleteFunc(open, func(k int) bool { return k == s.N })
			}
		case events.NodeError:
			n++
			step := "no step"
			if len(open) > 0 {
				step = fmt.Sprintf("step %d", open[len(open)-1])
			}
			fmt.Fprintf(b, "- seq %d, %s, node %s: %s\n", r.Seq, step, r.Node, cell(string(r.Data)))
		}
	}
	if n == 0 {
		b.WriteString("None.\n")
	}
}

// proofFiles are the files under the run's proof dir, relative to the run
// dir with forward slashes, sorted.
func proofFiles(runDir string) ([]string, error) {
	var out []string
	err := filepath.WalkDir(filepath.Join(runDir, "proof"), func(p string, d fs.DirEntry, err error) error {
		if errors.Is(err, fs.ErrNotExist) {
			return fs.SkipAll
		}
		if err != nil || d.IsDir() {
			return err
		}
		rel, err := filepath.Rel(runDir, p)
		if err != nil {
			return err //nolint:wrapcheck // the walk below names the proof dir
		}
		out = append(out, filepath.ToSlash(rel))
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("proof files: %w", err)
	}
	slices.Sort(out)
	return out, nil
}
