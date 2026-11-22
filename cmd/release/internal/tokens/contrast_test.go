package tokens

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCheckContrast(t *testing.T) {
	if err := CheckContrast(repo); err != nil {
		t.Fatal(err)
	}
	root := t.TempDir()
	doc := strings.NewReplacer(
		`ink-3: "hsl(214 14% 71%)"`, `ink-3: "hsl(214 14% 61%)"`,
		"| `ink-3` | `hsl(var(--h) 14% 71%)` |", "| `ink-3` | `hsl(var(--h) 14% 61%)` |",
	).Replace(design(t))
	if err := os.WriteFile(filepath.Join(root, designPath), []byte(doc), 0o644); err != nil {
		t.Fatal(err)
	}
	want := "the ambient ceiling is 0.015, but DESIGN.md's ambient.ceiling records 0.053; update it, .impeccable/design.json and the Ambient Ceiling Rule"
	if err := CheckContrast(root); err == nil || err.Error() != want {
		t.Errorf("darker ink-3: got %v, want %s", err, want)
	}
}
