package host

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseVersion(t *testing.T) {
	for in, want := range map[string]version{
		"4.18.0-553.el8.x86_64": {4, 18},
		"6.18.44-fc-v114":       {6, 18},
		"2.28":                  {2, 28},
		"5.4":                   {5, 4},
	} {
		if got, err := parseVersion(in); err != nil || got != want {
			t.Errorf("parseVersion(%q) = %v, %v; want %v", in, got, err, want)
		}
	}
	if _, err := parseVersion("glibc"); err == nil || err.Error() != `"glibc" is not a version` {
		t.Errorf("parseVersion(glibc) = %v", err)
	}
}

func TestOnNetworkShareChecksTheNearestExistingAncestor(t *testing.T) {
	dir := t.TempDir()
	if remote, err := OnNetworkShare(filepath.Join(dir, "not", "yet")); remote || err != nil {
		t.Fatalf("missing dir under a local temp dir: %v, %v; want local", remote, err)
	}
	file := filepath.Join(dir, "file")
	if err := os.WriteFile(file, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := OnNetworkShare(filepath.Join(file, "below")); err == nil {
		t.Fatal("a dir below a file: no error")
	}
}
