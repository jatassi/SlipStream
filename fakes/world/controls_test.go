package world_test

import (
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/jatassi/slipstream/fakes/events"
	"github.com/jatassi/slipstream/fakes/fake"
	"github.com/jatassi/slipstream/fakes/fault"
	"github.com/jatassi/slipstream/fakes/world"
	"github.com/jatassi/slipstream/fakes/world/worldtest"
)

func TestControlsOverTheSocketArmAfterRulesInLogOrder(t *testing.T) {
	t.Parallel()
	runDir := filepath.Join(t.TempDir(), "run")
	ctx := worldtest.Context(t)
	run, err := world.LaunchWith(ctx, world.Options{Binary: stub{}.build(t), Scenario: stubScenario(), RunDir: runDir, Origin: worldtest.Origin(t)}, []fake.Spec{echo})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _, _ = run.Stop(ctx) })
	m := run.Manifest()
	remote, err := world.Dial(runDir)
	if err != nil {
		t.Fatal(err)
	}

	rule := fault.Rule{ID: "after-step", Target: "echo", Fault: fault.Status{Code: 503}, Times: 1, After: &events.Query{Kind: events.StepStart}}
	if _, err := remote.SetFault(ctx, rule); err != nil {
		t.Fatal(err)
	}
	if got := getFake(ctx, t, m); got != "echo to world\n" {
		t.Fatalf("before any step the echo fake answered %q", got)
	}
	step, err := remote.StartStep(ctx, world.StepStart{Verb: "exec", Args: []string{"--", "curl", "{fakes.echo.url}"}})
	if err != nil || step.N != 1 {
		t.Fatalf("StartStep = %+v, %v; want step 1", step, err)
	}
	if got := getFake(ctx, t, m); got != "Service Unavailable\n" {
		t.Fatalf("after the step started the echo fake answered %q, want the 503", got)
	}
	if got := getFake(ctx, t, m); got != "echo to world\n" {
		t.Fatalf("after the one 503 the echo fake answered %q", got)
	}
	if _, err := remote.EndStep(ctx, 1, world.StepEnd{Exit: 0, Files: map[string]string{"stdout": "journal/1.out"}}); err != nil {
		t.Fatal(err)
	}
	if _, err := remote.SetFault(world.InStep(ctx, 1), fault.Rule{ID: "gone", Target: "echo", Fault: fault.Status{Code: 404}}); err != nil {
		t.Fatal(err)
	}
	if _, err := remote.ClearFault(ctx, ""); err != nil {
		t.Fatal(err)
	}

	want := []string{
		`5 fault.set world {"id":"after-step","target":"echo","times":1,"after":{"kind":"step.start"},"fault":"status=503"}`,
		`6 step.start world {"n":1,"verb":"exec","args":["--","curl","{fakes.echo.url}"]}`,
		`7 fault.fired world {"rule":"after-step","target":"echo","fault":"status=503","match":1}`,
		`8 step.end world {"n":1,"exit":0} map[stdout:journal/1.out]`,
		`9 fault.set world {"fault":"status=404","id":"gone","step":1,"target":"echo"}`,
		`10 fault.cleared world {"removed":true}`,
	}
	if got := logged(t, runDir)[4:]; !reflect.DeepEqual(got, want) {
		t.Fatalf("events.jsonl holds\n%s\nwant\n%s", strings.Join(got, "\n"), strings.Join(want, "\n"))
	}
}

func TestControlsRefuse(t *testing.T) {
	t.Parallel()
	runDir := filepath.Join(t.TempDir(), "run")
	ctx := worldtest.Context(t)
	run, err := world.LaunchWith(ctx, world.Options{Binary: stub{}.build(t), Scenario: stubScenario(), RunDir: runDir, Origin: worldtest.Origin(t)}, []fake.Spec{echo})
	if err != nil {
		t.Fatal(err)
	}
	remote, err := world.Dial(runDir)
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range []struct {
		name string
		err  func() error
		want string
	}{
		{"a fault the front can't play", func() error {
			_, err := remote.SetFault(ctx, fault.Rule{ID: "x", Target: "echo", Fault: fault.BadTLS(fault.Expired)})
			return err
		}, "tls=expired is a TLS fault and echo carries no TLS"},
		{"a rule without an id", func() error {
			_, err := remote.SetFault(ctx, fault.Rule{Target: "echo", Fault: fault.Hang{}})
			return err
		}, `a rule needs an id, and "_" clears every rule`},
		{"a step that is not open", func() error {
			_, err := remote.EndStep(ctx, 9, world.StepEnd{})
			return err
		}, "step 9 is not open"},
		{"a step's file outside the run dir", func() error {
			_, err := remote.EndStep(ctx, 9, world.StepEnd{Files: map[string]string{"stdout": "../x.out"}})
			return err
		}, "step 9's stdout file ../x.out is not inside the run dir"},
		{"a step that is neither a verb nor a note", func() error {
			_, err := remote.StartStep(ctx, world.StepStart{})
			return err
		}, "a step is a verb with its arguments or a note"},
	} {
		if err := c.err(); err == nil || err.Error() != c.want {
			t.Errorf("%s: %v, want %s", c.name, err, c.want)
		}
	}
	if _, err := run.Stop(ctx); err != nil {
		t.Fatal(err)
	}
	if _, err := world.Dial(runDir); err == nil || err.Error() != "no run is live in "+runDir+": it has no manifest.json" {
		t.Fatalf("Dial on a stopped run = %v", err)
	}
}
