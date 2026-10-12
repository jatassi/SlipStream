package buildinfo_test

import (
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/jatassi/slipstream/internal/buildinfo"
)

var want = buildinfo.Stamp{
	Version:      "v2.0.0-rc.1-3-g4de06ac",
	Commit:       "4de06ac1c0ffee4de06ac1c0ffee4de06ac1c0ff",
	Fingerprint:  "clean",
	Target:       "linux-amd64",
	NativeBundle: "9f2c1d",
	Libtorrent:   `2.0.11"rc'\\1`,
}

func build(t *testing.T, goos string, ldflags ...string) string {
	t.Helper()
	out := filepath.Join(t.TempDir(), "stamped-"+goos)
	if goos == "windows" {
		out += ".exe"
	}
	args := append([]string{"build", "-trimpath", "-o", out}, ldflags...)
	cmd := exec.Command("go", append(args, "./testdata/stamped")...)
	cmd.Env = append(os.Environ(), "CGO_ENABLED=0", "GOOS="+goos, "GOARCH="+runtime.GOARCH)
	if msg, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("go build: %v\n%s", err, msg)
	}
	return out
}

func stampFlag(t *testing.T, s buildinfo.Stamp) string {
	t.Helper()
	value, err := buildinfo.Encode(s)
	if err != nil {
		t.Fatal(err)
	}
	return "-ldflags=-s -w -X=github.com/jatassi/slipstream/internal/buildinfo.stamp=" + value
}

func TestEncodeRejectsWhitespace(t *testing.T) {
	_, err := buildinfo.Encode(buildinfo.Stamp{Version: "v1 beta"})
	want := `stamp {"version":"v1 beta","commit":"","fingerprint":"","target":"","nativeBundle":"","libtorrent":""} contains whitespace`
	if err == nil || err.Error() != want {
		t.Errorf("Encode err = %v, want %s", err, want)
	}
}

func TestReadFileFindsStampInStrippedBinaries(t *testing.T) {
	for _, goos := range []string{"linux", "windows", "darwin"} {
		t.Run(goos, func(t *testing.T) {
			got, err := buildinfo.ReadFile(build(t, goos, stampFlag(t, want)))
			if err != nil {
				t.Fatal(err)
			}
			if got != want {
				t.Errorf("ReadFile = %+v, want %+v", got, want)
			}
		})
	}
}

func TestReadFileReportsUnstampedBinary(t *testing.T) {
	_, err := buildinfo.ReadFile(build(t, "linux", "-ldflags=-s -w"))
	if !errors.Is(err, buildinfo.ErrUnstamped) {
		t.Errorf("ReadFile err = %v, want ErrUnstamped", err)
	}
}

func TestCurrentMatchesLinkedStamp(t *testing.T) {
	if testing.Short() {
		t.Skip("runs a built binary")
	}
	host := build(t, runtime.GOOS, stampFlag(t, want))
	out, err := exec.Command(host).Output()
	if err != nil {
		t.Fatal(err)
	}
	var got buildinfo.Stamp
	if err := json.Unmarshal(out, &got); err != nil {
		t.Fatalf("%v: %s", err, out)
	}
	if got != want {
		t.Errorf("Current = %+v, want %+v", got, want)
	}
}
