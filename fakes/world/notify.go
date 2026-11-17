package world

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"slices"

	"github.com/jatassi/slipstream/fakes/capturescript"
	"github.com/jatassi/slipstream/fakes/fault"
	"github.com/jatassi/slipstream/fakes/notifysink"
	"github.com/jatassi/slipstream/fakes/scenario"
	"github.com/jatassi/slipstream/fakes/world/internal/setup"
	"github.com/jatassi/slipstream/internal/notifications/providers"
)

// captureScriptPackage is built into each run that names script, rather
// than shipped beside the binary under test: a release never carries it,
// and a build from the worktree always matches the fake it reports to.
const captureScriptPackage = "github.com/jatassi/slipstream/fakes/capturescript/cmd/capture-script"

// seedNotify adds the Scenario's notification connections on the
// Controller, last, so the seeded library sends none of them a message.
func (r *Run) seedNotify(ctx context.Context, _ *unwind) error {
	if len(r.scenario.Notify) == 0 {
		return nil
	}
	controller := r.scenario.Nodes[slices.IndexFunc(r.scenario.Nodes, func(n scenario.Node) bool { return n.Setup == scenario.SetupController })].Name
	sink, _ := r.table.Find(fault.Target(notifysink.Fake.Name), controller)
	report, _ := r.table.Find(fault.Target(capturescript.Fake.Name), controller)
	w := setup.NotifyWorld{Sink: sink.URL(""), Report: report.URL("")}
	if slices.Contains(r.scenario.Notify, providers.ID("script")) {
		var err error
		if w.Script, err = r.buildCaptureScript(ctx); err != nil {
			return err
		}
	}
	if err := setup.Notify(ctx, http.DefaultClient, r.scenario, r.ready(), r.claims, w); err != nil {
		return err
	}
	r.log.Info("notification connections added", "providers", r.scenario.Notify)
	return nil
}

// buildCaptureScript builds capture-script into the run's bin/, beside the
// copy of the binary under test. It needs only the standard library.
func (r *Run) buildCaptureScript(ctx context.Context) (string, error) {
	out := filepath.Join(r.runDir, "bin", "capture-script")
	if runtime.GOOS == "windows" {
		out += ".exe"
	}
	cmd := exec.CommandContext(ctx, "go", "build", "-o", out, captureScriptPackage)
	cmd.Env = append(os.Environ(), "CGO_ENABLED=0")
	if b, err := cmd.CombinedOutput(); err != nil {
		return "", fmt.Errorf("build capture-script from source: %w\n%s", err, b)
	}
	return out, nil
}
