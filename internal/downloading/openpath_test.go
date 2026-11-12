package downloading_test

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jatassi/slipstream/internal/downloading"
	"github.com/jatassi/slipstream/internal/transfers/source"
)

func TestOpenOfAPathReadsOnlyAFileTheNodeHolds(t *testing.T) {
	held, elsewhere := t.TempDir(), t.TempDir()
	n := downloading.NewNode(nil, downloading.NodeConfig{Holds: func(p string) bool { return strings.HasPrefix(p, held) }})
	file := filepath.Join(held, "Heat.mkv")
	if err := os.WriteFile(file, []byte("heat"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(elsewhere, "Heat.mkv"), []byte("heat"), 0o600); err != nil {
		t.Fatal(err)
	}
	if got, err := n.Open(source.Path{Abs: file}); err != nil || got != file {
		t.Fatalf("Open of a held file = %q, %v; want %q", got, err, file)
	}
	for _, c := range []struct {
		name string
		path string
		want source.Reason
	}{
		{"a file elsewhere", filepath.Join(elsewhere, "Heat.mkv"), source.Failed},
		{"a folder", held, source.Failed},
		{"a file that isn't there", filepath.Join(held, "Ronin.mkv"), source.Gone},
	} {
		_, err := n.Open(source.Path{Abs: c.path})
		if a, ok := errors.AsType[*source.Absent](err); !ok || a.Reason != c.want {
			t.Errorf("Open of %s = %v, want Absent reason %d", c.name, err, c.want)
		}
	}
}
