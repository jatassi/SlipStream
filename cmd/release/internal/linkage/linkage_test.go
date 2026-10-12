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
			"testdata/dynamic-linux.elf: linux binary imports what cmd/release/internal/linkage/policy.json does not list: " +
				"interpreter /lib64/ld-linux-x86-64.so.2, library libc.so.6",
		},
		{
			"testdata/foreign-dll.exe",
			"testdata/foreign-dll.exe: windows binary imports what cmd/release/internal/linkage/policy.json does not list: " +
				"library libcrypto-3-x64.dll",
		},
		{
			"testdata/foreign-dylib.macho",
			"testdata/foreign-dylib.macho: darwin binary imports what cmd/release/internal/linkage/policy.json does not list: " +
				"weak library /opt/homebrew/opt/openssl@3/lib/libssl.3.dylib",
		},
	} {
		t.Run(tc.fixture, func(t *testing.T) {
			err := linkage.Check(tc.fixture)
			if err == nil || err.Error() != tc.want {
				t.Errorf("Check error = %v\nwant %s", err, tc.want)
			}
		})
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
			out := filepath.Join(t.TempDir(), "hello")
			cmd := exec.CommandContext(t.Context(), "go", "build", "-trimpath", "-o", out, "./testdata/hello")
			cmd.Env = append(os.Environ(), "CGO_ENABLED=0", "GOOS="+target.goos, "GOARCH="+target.goarch)
			if msg, err := cmd.CombinedOutput(); err != nil {
				t.Fatalf("go build: %v\n%s", err, msg)
			}
			if err := linkage.Check(out); err != nil {
				t.Error(err)
			}
		})
	}
}

func TestCheckRejectsNonBinary(t *testing.T) {
	err := linkage.Check("policy.json")
	want := "policy.json: not an ELF, PE or 64-bit Mach-O binary"
	if err == nil || err.Error() != want {
		t.Errorf("Check error = %v, want %s", err, want)
	}
}
