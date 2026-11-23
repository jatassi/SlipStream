package worldtest

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

// A hold that outlasts removeAll's wait names the process that holds the
// file, so a failed cleanup says what to look at.
func TestAHeldFileNamesItsHolder(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	path := filepath.Join(dir, "held")
	if err := os.WriteFile(path, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	f, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = f.Close() }()
	names, err := holders(path)
	if err != nil {
		t.Fatal(err)
	}
	self := fmt.Sprintf("(PID %d)", os.Getpid())
	if !slices.ContainsFunc(names, func(n string) bool { return strings.HasSuffix(n, self) }) {
		t.Fatalf("the file's holders are %q, want this test's process %s", names, self)
	}
}
