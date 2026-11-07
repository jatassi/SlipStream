package sim

import (
	"os"
	"os/exec"
	"strings"
	"testing"
	"testing/synctest"

	"github.com/jatassi/slipstream/fakes/scenario"
)

// TestFailedWorldLogsEveryNode fails a world whose seedbox has no log file
// in a child process, and checks the child's output accounts for both Nodes.
func TestFailedWorldLogsEveryNode(t *testing.T) {
	if os.Getenv("SIM_FAILED_WORLD_CHILD") == "1" {
		synctest.Test(t, func(t *testing.T) {
			s, err := scenario.Load("home")
			if err != nil {
				t.Fatal(err)
			}
			w := Launch(t, s)
			seedbox := w.Node("seedbox")
			seedbox.Stop()
			if err := os.Remove(seedbox.logFile()); err != nil {
				t.Fatal(err)
			}
			t.Fail()
		})
		return
	}
	cmd := exec.CommandContext(t.Context(), os.Args[0], "-test.run=^TestFailedWorldLogsEveryNode$", "-test.v")
	cmd.Env = append(os.Environ(), "SIM_FAILED_WORLD_CHILD=1")
	out, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatalf("child passed, want it failed:\n%s", out)
	}
	for _, want := range []string{"lines of node home's log", "node seedbox has no log to show"} {
		if !strings.Contains(string(out), want) {
			t.Errorf("child output lacks %q:\n%s", want, out)
		}
	}
}
