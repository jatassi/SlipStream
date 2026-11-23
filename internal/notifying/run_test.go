package notifying_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/jatassi/slipstream/internal/notifying"
)

// asScript is the argument that makes this test binary play a script. It
// is an argument because the runner hands the script no variable of ours.
const asScript = "notifying-test-script"

func TestMain(m *testing.M) {
	if len(os.Args) > 2 && os.Args[1] == asScript {
		// syscall.Exit skips the coverage hook of a -cover build, which
		// would add a GOCOVERDIR warning to the script's output.
		syscall.Exit(playScript(os.Args[2], os.Args[3:]))
	}
	os.Exit(m.Run())
}

type report struct {
	Argv  []string `json:"argv"`
	Env   []string `json:"env"`
	Event string   `json:"event"`
}

func playScript(play string, args []string) int {
	switch play {
	case "report":
		event, err := os.ReadFile(os.Getenv("NOTIFICATION_EVENT_FILE"))
		if err != nil {
			return 9
		}
		b, err := json.Marshal(report{Argv: os.Args, Env: os.Environ(), Event: string(event)})
		if err != nil || os.WriteFile(args[0], b, 0o600) != nil {
			return 9
		}
	case "fail":
		fmt.Println("it broke")
		return 3
	case "spam":
		fmt.Print(strings.Repeat("a", 7<<10) + strings.Repeat("b", 1<<10))
	case "child":
		grandchild := exec.CommandContext(context.Background(), os.Args[0], asScript, "beat", args[0])
		if grandchild.Start() != nil {
			return 9
		}
		time.Sleep(time.Hour)
	case "beat":
		f, err := os.OpenFile(args[0], os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600)
		if err != nil {
			return 9
		}
		for {
			_, _ = f.WriteString(".")
			time.Sleep(20 * time.Millisecond)
		}
	}
	return 0
}

func runner(t *testing.T) notifying.ProcessRunner {
	t.Helper()
	return notifying.ProcessRunner{Environ: append(os.Environ(), "SLIPSTREAM_DATA_DIR=/srv/slipstream", "SLIPSTREAM_API_KEY=hunter2"), TempDir: t.TempDir()}
}

func script(play string, args ...string) notifying.Script {
	exe, err := os.Executable()
	if err != nil {
		panic(err)
	}
	return notifying.Script{Path: exe, Args: append([]string{asScript, play}, args...), Event: "imported", EventFile: []byte(`{"event":"imported"}`), Stop: 30 * time.Second}
}

func TestAScriptGetsItsArgumentsTheEventAndACleanEnvironment(t *testing.T) {
	out := filepath.Join(t.TempDir(), "report.json")
	r := runner(t)
	s := script("report", out, "--flag", "two words")
	ran := r.Run(t.Context(), s)
	if ran.Exit != 0 || ran.TimedOut || ran.Missing || ran.Err != "" {
		t.Fatalf("Run = %+v", ran)
	}
	b, err := os.ReadFile(out)
	if err != nil {
		t.Fatal(err)
	}
	var got report
	if err := json.Unmarshal(b, &got); err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(got.Argv[1:], s.Args) {
		t.Errorf("argv = %q, want %q", got.Argv[1:], s.Args)
	}
	if got.Event != `{"event":"imported"}` {
		t.Errorf("the event file held %q", got.Event)
	}
	env := map[string]string{}
	for _, kv := range got.Env {
		name, value, _ := strings.Cut(kv, "=")
		if name == "" {
			continue // Windows keeps each drive's directory as =C:=C:\dir
		}
		env[strings.ToUpper(name)] = value
	}
	if env["NOTIFICATION_EVENT"] != "imported" || env["PATH"] == "" {
		t.Errorf("env = %q, want NOTIFICATION_EVENT=imported and PATH", got.Env)
	}
	for name := range env {
		if strings.HasPrefix(name, "SLIPSTREAM_") || !slices.Contains([]string{"PATH", "HOME", "TMPDIR", "USERPROFILE", "HOMEDRIVE", "HOMEPATH", "TEMP", "TMP", "SYSTEMROOT", "NOTIFICATION_EVENT", "NOTIFICATION_EVENT_FILE"}, name) {
			t.Errorf("the script got %s from the Node", name)
		}
	}
	if _, err := os.Stat(env["NOTIFICATION_EVENT_FILE"]); !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("the event file is still there after the run: %v", err)
	}
}

func TestAFailingScriptAnswersItsExitAndTail(t *testing.T) {
	ran := runner(t).Run(t.Context(), script("fail"))
	if ran.Exit != 3 || strings.TrimSpace(string(ran.Tail)) != "it broke" || ran.TimedOut {
		t.Fatalf("Run = %+v, want exit 3 with its output", ran)
	}
}

func TestOnlyTheLastKiBOfOutputComesBack(t *testing.T) {
	ran := runner(t).Run(t.Context(), script("spam"))
	if want := strings.Repeat("b", notifying.MaxTail); ran.Exit != 0 || string(ran.Tail) != want {
		t.Fatalf("Run = exit %d with a %d-byte tail, want the last %d bytes", ran.Exit, len(ran.Tail), notifying.MaxTail)
	}
}

func TestAMissingScriptIsMissing(t *testing.T) {
	ran := runner(t).Run(t.Context(), notifying.Script{Path: filepath.Join(t.TempDir(), "notify.sh"), Stop: time.Second})
	if !ran.Missing {
		t.Fatalf("Run = %+v, want missing", ran)
	}
}

func TestAScriptPastStopIsKilledWithEverythingItStarted(t *testing.T) {
	beat := filepath.Join(t.TempDir(), "beat")
	s := script("child", beat)
	s.Stop = 2 * time.Second
	start := time.Now()
	ran := runner(t).Run(t.Context(), s)
	if !ran.TimedOut || ran.Exit != -1 {
		t.Fatalf("Run = %+v, want timed out", ran)
	}
	if took := time.Since(start); took > 10*time.Second {
		t.Errorf("Run took %s past a 2s stop", took)
	}
	last, err := os.ReadFile(beat)
	if err != nil || len(last) == 0 {
		t.Fatalf("the grandchild never ran: %v", err)
	}
	time.Sleep(300 * time.Millisecond)
	if now, _ := os.ReadFile(beat); len(now) != len(last) {
		t.Fatalf("the grandchild still runs: %d beats, then %d", len(last), len(now))
	}
}
