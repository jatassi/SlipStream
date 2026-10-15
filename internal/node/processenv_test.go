package node

import (
	"testing"

	"github.com/jatassi/slipstream/internal/install"
)

func TestProcessEnvWithoutAMarkerIsDevWithNoDefaultDataDir(t *testing.T) {
	t.Setenv("SLIPSTREAM_DATA_DIR", "")
	_, err := ProcessEnv("run", "")
	if want := "no data dir: pass --data-dir or set SLIPSTREAM_DATA_DIR"; err == nil || err.Error() != want || ExitCode(err) != ExitUsage {
		t.Fatalf("no data dir: %v (exit %d), want %s (exit %d)", err, ExitCode(err), want, ExitUsage)
	}
	env, err := ProcessEnv("run", shortTempDir(t))
	if err != nil || env.Install != install.Dev {
		t.Fatalf("with a data dir: install %s, %v; want dev", env.Install, err)
	}
}
