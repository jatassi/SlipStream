package node

import (
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

func TestParseDataDirMakesTheSocketPathAbsoluteAndShort(t *testing.T) {
	t.Chdir(t.TempDir())
	wd, err := filepath.Abs(".")
	if err != nil {
		t.Fatal(err)
	}
	d, err := ParseDataDir("nodes/../node-a")
	if err != nil {
		t.Fatal(err)
	}
	if want := filepath.Join(wd, "node-a", "run", "control.sock"); d.ControlSocket() != want {
		t.Fatalf("control socket = %s, want %s", d.ControlSocket(), want)
	}

	long := filepath.Join(wd, strings.Repeat("d", 104-len(wd)))
	_, err = ParseDataDir(long)
	sock := filepath.Join(long, "run", "control.sock")
	want := "data dir " + long + ": control socket path " + sock + " is " + strconv.Itoa(len(sock)) + " bytes, over the 103-byte limit"
	if err == nil || err.Error() != want {
		t.Fatalf("err = %v, want %s", err, want)
	}

	if _, err := ParseDataDir(""); err == nil || err.Error() != "data dir is empty" {
		t.Fatalf("empty: err = %v", err)
	}
}
