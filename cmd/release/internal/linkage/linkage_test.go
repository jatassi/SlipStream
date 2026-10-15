package linkage_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/jatassi/slipstream/cmd/release/internal/linkage"
)

func TestCheckRejectsForeignImports(t *testing.T) {
	for _, tc := range []struct{ fixture, want string }{
		{
			"testdata/dynamic-linux.elf",
			"testdata/dynamic-linux.elf: linux node binary imports what cmd/release/internal/linkage/policy.json does not list: " +
				"interpreter /lib64/ld-linux-x86-64.so.2, library libc.so.6",
		},
		{
			"testdata/foreign-dll.exe",
			"testdata/foreign-dll.exe: windows node binary imports what cmd/release/internal/linkage/policy.json does not list: " +
				"library libcrypto-3-x64.dll",
		},
		{
			"testdata/foreign-dylib.macho",
			"testdata/foreign-dylib.macho: darwin node binary imports what cmd/release/internal/linkage/policy.json does not list: " +
				"weak library /opt/homebrew/opt/openssl@3/lib/libssl.3.dylib",
		},
	} {
		t.Run(tc.fixture, func(t *testing.T) {
			err := linkage.Check(tc.fixture, linkage.Node)
			if err == nil || err.Error() != tc.want {
				t.Errorf("Check error = %v\nwant %s", err, tc.want)
			}
		})
	}
}

func TestCheckHoldsTheTrampolineToItsOwnList(t *testing.T) {
	for _, tc := range []struct{ fixture, want string }{
		{
			"testdata/foreign-dylib.macho",
			"testdata/foreign-dylib.macho: darwin trampoline binary imports what cmd/release/internal/linkage/policy.json does not list: " +
				"weak library /opt/homebrew/opt/openssl@3/lib/libssl.3.dylib",
		},
		{
			"testdata/dynamic-linux.elf",
			"testdata/dynamic-linux.elf: cmd/release/internal/linkage/policy.json lists no linux trampoline binary",
		},
	} {
		t.Run(tc.fixture, func(t *testing.T) {
			err := linkage.Check(tc.fixture, linkage.Trampoline)
			if err == nil || err.Error() != tc.want {
				t.Errorf("Check error = %v\nwant %s", err, tc.want)
			}
		})
	}
}

func TestCheckAssertsTheWindowsSubsystem(t *testing.T) {
	if err := linkage.Check("testdata/gui.exe", linkage.Trampoline); err != nil {
		t.Errorf("a GUI trampoline: %v", err)
	}
	want := "testdata/gui.exe: windows node binary has the GUI subsystem, want console"
	if err := linkage.Check("testdata/gui.exe", linkage.Node); err == nil || err.Error() != want {
		t.Errorf("a GUI Node: %v, want %s", err, want)
	}
	console := build(t, "windows", "amd64", "./testdata/hello")
	want = console + ": windows trampoline binary has the console subsystem, want GUI"
	if err := linkage.Check(console, linkage.Trampoline); err == nil || err.Error() != want {
		t.Errorf("a console trampoline: %v, want %s", err, want)
	}
}

func TestCheckPassesPureGoBinaries(t *testing.T) {
	for _, target := range []struct{ goos, goarch string }{
		{"linux", "amd64"},
		{"linux", "arm64"},
		{"windows", "amd64"},
		{"darwin", "arm64"},
	} {
		t.Run(target.goos+"-"+target.goarch, func(t *testing.T) {
			if err := linkage.Check(build(t, target.goos, target.goarch, "./testdata/hello"), linkage.Node); err != nil {
				t.Error(err)
			}
		})
	}
}

// TestCheckPassesTheTrampoline builds slipstream-app as release build does.
func TestCheckPassesTheTrampoline(t *testing.T) {
	for _, target := range []struct{ goos, goarch, ldflags string }{
		{"windows", "amd64", "-s -w -H windowsgui"},
		{"darwin", "arm64", "-s -w"},
	} {
		t.Run(target.goos+"-"+target.goarch, func(t *testing.T) {
			out := build(t, target.goos, target.goarch, "../../../slipstream-app", "-buildvcs=false", "-ldflags", target.ldflags)
			if err := linkage.Check(out, linkage.Trampoline); err != nil {
				t.Error(err)
			}
		})
	}
}

func TestCheckRejectsNonBinary(t *testing.T) {
	err := linkage.Check("policy.json", linkage.Node)
	want := "policy.json: not an ELF, PE or 64-bit Mach-O binary"
	if err == nil || err.Error() != want {
		t.Errorf("Check error = %v, want %s", err, want)
	}
}

func build(t *testing.T, goos, goarch, pkg string, flags ...string) string {
	t.Helper()
	out := filepath.Join(t.TempDir(), "bin")
	args := append(append([]string{"build", "-trimpath"}, flags...), "-o", out, pkg)
	cmd := exec.CommandContext(t.Context(), "go", args...)
	cmd.Env = append(os.Environ(), "CGO_ENABLED=0", "GOFLAGS=", "GOOS="+goos, "GOARCH="+goarch)
	if msg, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("go build: %v\n%s", err, msg)
	}
	return out
}
