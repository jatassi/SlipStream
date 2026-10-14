package main_test

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/jatassi/slipstream/fakes/events"
)

func self(t *testing.T) string {
	t.Helper()
	exe, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	return exe
}

func jsonText(t *testing.T, v any) string {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

// journal is the run's step and fault records as seq-free lines.
func journal(t *testing.T, runDir string) []string {
	t.Helper()
	recs, err := events.Read(runDir)
	if err != nil {
		t.Fatal(err)
	}
	var out []string
	for _, r := range recs {
		if !strings.HasPrefix(r.Kind, "step.") && r.Kind != events.FaultSet && r.Kind != events.FaultCleared {
			continue
		}
		line := fmt.Sprintf("%s %s", r.Kind, r.Data)
		if r.Files != nil {
			line += fmt.Sprint(" ", r.Files)
		}
		out = append(out, line)
	}
	return out
}

func readFile(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

func TestExecJournals(t *testing.T) {
	t.Parallel()
	runDir, m := startWorld(t)
	t.Cleanup(func() { fakeworld(t, "stop", runDir) })
	exe := self(t)

	echo := []string{"--", exe, commandArg, "echo", "exit=0", "{nodes.home.url}"}
	if got, want := fakeworld(t, append([]string{"exec", runDir}, echo...)...), (result{0, m.Nodes[0].URL + "\n", "stderr of the step\n"}); got != want {
		t.Fatalf("exec echo = %+v, want %+v", got, want)
	}
	set := fakeworld(t, "fault", "set", runDir, "tmdb-503", "tmdb", "status=503", "--times", "1")
	var setRec events.Record
	if err := json.Unmarshal([]byte(set.stdout), &setRec); set.code != 0 || err != nil || setRec.Kind != events.FaultSet {
		t.Fatalf("fault set = %+v, want its fault.set record", set)
	}
	get := []string{"--proof", "search-and-grab/held", "--", exe, commandArg, "get", "{ca.file}", "{fakes.tmdb.url}/3/configuration"}
	if got, want := fakeworld(t, append([]string{"exec", runDir}, get...)...), (result{0, "503\n", ""}); got != want {
		t.Fatalf("exec get = %+v, want %+v", got, want)
	}
	wait := fakeworld(t, "fault", "wait", runDir, "--rule", "tmdb-503")
	var fired events.Record
	if err := json.Unmarshal([]byte(wait.stdout), &fired); wait.code != 0 || err != nil || fired.Node != "world" || string(fired.Data) != `{"rule":"tmdb-503","target":"tmdb","fault":"status=503","match":1}` {
		t.Fatalf("fault wait = %+v, want the tmdb-503 firing", wait)
	}
	failing := []string{"--", exe, commandArg, "echo", "exit=4", "{apiKey}"}
	if got, want := fakeworld(t, append([]string{"exec", runDir}, failing...)...), (result{4, m.APIKey + "\n", "stderr of the step\n"}); got != want {
		t.Fatalf("exec of a failing command = %+v, want %+v", got, want)
	}

	want := []string{
		`step.start {"n":1,"verb":"exec","args":` + jsonText(t, echo) + `}`,
		`step.end {"n":1,"exit":0} map[stderr:journal/1.err stdout:journal/1.out]`,
		`step.start {"n":2,"verb":"fault set","args":["tmdb-503","tmdb","status=503","--times","1"]}`,
		`fault.set {"fault":"status=503","id":"tmdb-503","step":2,"target":"tmdb","times":1}`,
		`step.end {"n":2,"exit":0}`,
		`step.start {"n":3,"verb":"exec","args":` + jsonText(t, get) + `}`,
		`step.end {"n":3,"exit":0} map[stderr:proof/search-and-grab/held/3.err stdout:proof/search-and-grab/held/3.out]`,
		`step.start {"n":4,"verb":"fault wait","args":["--rule","tmdb-503"]}`,
		`step.end {"n":4,"exit":0}`,
		`step.start {"n":5,"verb":"exec","args":` + jsonText(t, failing) + `}`,
		`step.end {"n":5,"exit":4} map[stderr:journal/5.err stdout:journal/5.out]`,
	}
	if got := journal(t, runDir); !reflect.DeepEqual(got, want) {
		t.Fatalf("the journal holds\n%s\nwant\n%s", strings.Join(got, "\n"), strings.Join(want, "\n"))
	}
	for path, want := range map[string]string{
		"journal/1.out":                    m.Nodes[0].URL + "\n",
		"journal/1.err":                    "stderr of the step\n",
		"proof/search-and-grab/held/3.out": "503\n",
		"journal/5.out":                    m.APIKey + "\n",
	} {
		if got := readFile(t, filepath.Join(runDir, filepath.FromSlash(path))); got != want {
			t.Errorf("%s = %q, want %q", path, got, want)
		}
	}

	list := fakeworld(t, "fault", "list", runDir)
	var live events.Record
	if err := json.Unmarshal([]byte(list.stdout), &live); list.code != 0 || err != nil || live.Seq != setRec.Seq {
		t.Fatalf("fault list = %+v, want the tmdb-503 fault.set at seq %d", list, setRec.Seq)
	}
	fakeworld(t, "stop", runDir)
	stopped := fakeworld(t, "fault", "list", runDir)
	if stopped != list {
		t.Fatalf("fault list on the stopped run = %+v, want %+v", stopped, list)
	}
}

func TestTokens(t *testing.T) {
	t.Parallel()
	runDir, m := startWorld(t)
	t.Cleanup(func() { fakeworld(t, "stop", runDir) })
	exe := self(t)
	touched := filepath.Join(t.TempDir(), "touched")

	got := fakeworld(t, "exec", runDir, "--", exe, commandArg, "touch", touched, "{nodes.nas.url}")
	if wantErr := "fakeworld: exec: {nodes.nas.url} names nothing in the manifest\nusage: fakeworld <command>\n"; got.code != 3 || got.stdout != "" || !strings.HasPrefix(got.stderr, wantErr) {
		t.Fatalf("exec with an unknown path = %+v, want exit 3 with %q", got, wantErr)
	}
	if _, err := os.Stat(touched); err == nil {
		t.Fatal("the command ran despite an unknown path")
	}
	if steps := journal(t, runDir); len(steps) != 0 {
		t.Fatalf("an unknown path journaled %q", steps)
	}

	literal := fakeworld(t, "exec", runDir, "--", exe, commandArg, "echo", "exit=0", "{{apiKey}", `{"name":"{nodes.home.name}"}`, "{{{apiKey}}")
	if want := (result{0, "{apiKey}\n" + `{"name":"home"}` + "\n{" + m.APIKey + "}\n", "stderr of the step\n"}); literal != want {
		t.Fatalf("exec with literal braces = %+v, want %+v", literal, want)
	}
}
