package main

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

func TestDevReadsTheSetupCodeFromTheNodeLog(t *testing.T) {
	dir := t.TempDir()
	log := filepath.Join(dir, "home.log")
	records := `{"time":"2026-10-12T00:00:00Z","level":"INFO","msg":"listening","addr":"127.0.0.2:1"}
{"time":"2026-10-12T00:00:00Z","level":"INFO","msg":"setup code","code":"7K2M-Q9XD-4HBT","hint":"open this Node in a browser"}
`
	if err := os.WriteFile(log, []byte(records), 0o644); err != nil {
		t.Fatal(err)
	}
	manifest := filepath.Join(dir, "manifest.json")
	if err := os.WriteFile(manifest, []byte(`{"nodes":[{"url":"http://127.0.0.2:1","log":`+strconv.Quote(log)+`}]}`), 0o644); err != nil {
		t.Fatal(err)
	}
	var out strings.Builder
	if origin, err := announceWorld(&out, manifest); err != nil || origin != "http://127.0.0.2:1" {
		t.Fatalf("announceWorld = %q, %v, want http://127.0.0.2:1", origin, err)
	}
	if want := "Node http://127.0.0.2:1, manifest " + manifest + "\nSetup code 7K2M-Q9XD-4HBT\n"; out.String() != want {
		t.Errorf("announceWorld printed %q, want %q", out.String(), want)
	}
}
