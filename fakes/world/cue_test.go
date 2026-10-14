package world_test

import (
	"bytes"
	"errors"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/jatassi/slipstream/fakes/fake"
	"github.com/jatassi/slipstream/fakes/internal/fixture"
	"github.com/jatassi/slipstream/fakes/testbin"
	"github.com/jatassi/slipstream/fakes/world"
	"github.com/jatassi/slipstream/fakes/world/worldtest"
)

func TestFakeworldCuePlaysAFakesCueInItsStep(t *testing.T) {
	t.Parallel()
	runDir := filepath.Join(t.TempDir(), "run")
	ctx := worldtest.Context(t)
	run, err := world.LaunchWith(ctx, world.Options{Binary: stub{}.build(t), Scenario: stubScenario(), RunDir: runDir, Origin: worldtest.Origin(t)}, []fake.Spec{fixture.Fake})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _, _ = run.Stop(ctx) })
	m := run.Manifest()
	if want := []fake.CueSpec{fixture.Greet}; !reflect.DeepEqual(m.Fakes[0].Cues, want) {
		t.Fatalf("the manifest's fixture cues = %+v, want %+v", m.Fakes[0].Cues, want)
	}
	whoami := m.Fakes[0].URL + "/whoami"

	code, stdout, stderr := fakeworld(t, "cue", runDir, "fixture-greet", `{"text":"hi"}`)
	const record = `"kind":"cue","node":"world","data":{"args":{"text":"hi"},"fake":"fixture","name":"fixture-greet","result":{"was":"fixture"},"step":1}}`
	if code != 0 || !strings.HasSuffix(strings.TrimSpace(stdout), record) || stderr != "" {
		t.Fatalf("fakeworld cue = %d %q %q, want the record ending %s", code, stdout, stderr, record)
	}
	if got := get(ctx, t, m, whoami); got != "hi" {
		t.Fatalf("after the cue whoami answered %q, want hi", got)
	}

	code, _, stderr = fakeworld(t, "cue", runDir, "fixture-greet", `{"txt":"hi"}`)
	if want := "fakeworld cue: cue fixture-greet: json: unknown field \"txt\"\n"; code != 1 || stderr != want {
		t.Fatalf("fakeworld cue with an unknown argument = %d %q, want exit 1 with %q", code, stderr, want)
	}
	code, _, stderr = fakeworld(t, "cue", runDir, "nope")
	if want := "fakeworld cue: no fake declares cue nope\n"; code != 1 || stderr != want {
		t.Fatalf("fakeworld cue nope = %d %q, want exit 1 with %q", code, stderr, want)
	}

	want := []string{
		`step.start world {"n":1,"verb":"cue","args":["fixture-greet","{\"text\":\"hi\"}"]}`,
		`cue world {"args":{"text":"hi"},"fake":"fixture","name":"fixture-greet","result":{"was":"fixture"},"step":1}`,
		`step.end world {"n":1,"exit":0}`,
		`step.start world {"n":2,"verb":"cue","args":["fixture-greet","{\"txt\":\"hi\"}"]}`,
		`step.end world {"n":2,"exit":1}`,
		`step.start world {"n":3,"verb":"cue","args":["nope"]}`,
		`step.end world {"n":3,"exit":1}`,
	}
	var got []string
	for _, line := range besideFronts(t, runDir) {
		if strings.HasPrefix(line, "step.") || strings.HasPrefix(line, "cue ") {
			got = append(got, line)
		}
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("events.jsonl's steps and cues are\n%s\nwant\n%s", strings.Join(got, "\n"), strings.Join(want, "\n"))
	}
}

func fakeworld(t *testing.T, args ...string) (code int, stdout, stderr string) {
	t.Helper()
	cmd := exec.CommandContext(worldtest.Context(t), testbin.Fakeworld(t), args...)
	var out, errOut bytes.Buffer
	cmd.Stdout, cmd.Stderr = &out, &errOut
	if err := cmd.Run(); err != nil {
		if _, exited := errors.AsType[*exec.ExitError](err); !exited {
			t.Fatal(err)
		}
	}
	return cmd.ProcessState.ExitCode(), out.String(), errOut.String()
}
