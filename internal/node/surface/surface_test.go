package surface_test

import (
	"bytes"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jatassi/slipstream/internal/node/surface"
)

var update = flag.Bool("update", false, "rewrite the committed openapi.json")

// committed is where the web build's Orval config reads the document.
var committed = filepath.Join("..", "..", "..", "web", "openapi.json")

func TestDocument(t *testing.T) {
	got := surface.Document()
	if !bytes.Equal(got, surface.Document()) {
		t.Fatal("Document() rendered different bytes on a second call")
	}
	if *update {
		if err := os.WriteFile(committed, got, 0o644); err != nil {
			t.Fatal(err)
		}
		return
	}
	want, err := os.ReadFile(committed)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("openapi.json is stale; run go test ./internal/node/surface -run TestDocument -update\n%s", lineDiff(string(want), string(got)))
	}
}

// lineDiff lists the lines only in want with - and only in got with +, in
// order, from their longest common subsequence.
func lineDiff(want, got string) string {
	a, b := strings.Split(want, "\n"), strings.Split(got, "\n")
	lcs := make([][]int, len(a)+1)
	for i := range lcs {
		lcs[i] = make([]int, len(b)+1)
	}
	for i := len(a) - 1; i >= 0; i-- {
		for j := len(b) - 1; j >= 0; j-- {
			if a[i] == b[j] {
				lcs[i][j] = lcs[i+1][j+1] + 1
			} else {
				lcs[i][j] = max(lcs[i+1][j], lcs[i][j+1])
			}
		}
	}
	var out strings.Builder
	i, j := 0, 0
	for i < len(a) || j < len(b) {
		switch {
		case i < len(a) && j < len(b) && a[i] == b[j]:
			i, j = i+1, j+1
		case j < len(b) && (i == len(a) || lcs[i][j+1] >= lcs[i+1][j]):
			fmt.Fprintf(&out, "+%d: %s\n", j+1, b[j])
			j++
		default:
			fmt.Fprintf(&out, "-%d: %s\n", i+1, a[i])
			i++
		}
	}
	return out.String()
}
