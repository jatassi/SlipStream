package capturescript_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/jatassi/slipstream/fakes/capturescript"
	"github.com/jatassi/slipstream/fakes/fake/faketest"
	"github.com/jatassi/slipstream/fakes/scenario"
	"github.com/jatassi/slipstream/fakes/testbin"
	"github.com/jatassi/slipstream/internal/notifying"
)

func TestARunOfTheScriptIsCapturedWithItsCleanEnvironment(t *testing.T) {
	f := faketest.Loopback(t, capturescript.Fake, scenario.Scenario{Seed: "script"}, faketest.As("home"))
	bin := testbin.Build(t, t.TempDir(), "./fakes/capturescript/cmd/capture-script")
	runner := notifying.ProcessRunner{Environ: append(os.Environ(), "SLIPSTREAM_DATA_DIR=/srv/slipstream"), TempDir: t.TempDir()}
	ran := runner.Run(t.Context(), notifying.Script{
		Path: bin, Args: []string{"--report", f.Address.URL(""), "--exit", "3", "--say", "it broke"},
		Event: "imported", EventFile: []byte(`{"event":"imported"}`), Stop: 30 * time.Second,
	})
	if ran.Exit != 3 || string(ran.Tail) != "it broke" {
		t.Fatalf("Run = %+v, want exit 3 with its output", ran)
	}
	records := f.Records(t)
	if len(records) != 1 || records[0].Kind != capturescript.Script.Kind || records[0].Node != "home" {
		t.Fatalf("captured %+v, want one script.run from home", records)
	}
	var run capturescript.Run
	if err := json.Unmarshal(records[0].Data, &run); err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(run.Argv[1:], []string{"--report", f.Address.URL(""), "--exit", "3", "--say", "it broke"}) || run.Exit != 3 {
		t.Errorf("captured argv %q, exit %d", run.Argv, run.Exit)
	}
	if run.Env["NOTIFICATION_EVENT"] != "imported" || string(run.Event) != `{"event":"imported"}` {
		t.Errorf("captured env %v and event %s", run.Env, run.Event)
	}
	for name := range run.Env {
		if strings.HasPrefix(strings.ToUpper(name), "SLIPSTREAM_") {
			t.Errorf("the script got %s", name)
		}
	}
	event, err := os.ReadFile(filepath.Join(f.RunDir, records[0].Files["event.json"]))
	if err != nil || string(event) != `{"event":"imported"}` {
		t.Errorf("kept event file %q, %v", event, err)
	}
}
